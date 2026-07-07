package plugin

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	graftikLogger "graftik-wails/internal/logger"
)

type Instance struct {
	Manifest *Manifest
	Cmd      *exec.Cmd
	Port     int
	cancel   context.CancelFunc
	done     chan struct{}
}

type Manager struct {
	mu         sync.RWMutex
	instances  map[string]*Instance
	luaPlugins map[string]*LuaPlugin
	pluginsDir string
	httpClient *http.Client
	log        graftikLogger.Logger
}

func NewManager(pluginsDir string, log graftikLogger.Logger) *Manager {
	if log == nil {
		panic("plugin: logger is required")
	}
	return &Manager{
		instances:  make(map[string]*Instance),
		luaPlugins: make(map[string]*LuaPlugin),
		pluginsDir: pluginsDir,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		log:        log,
	}
}

func (m *Manager) Discover(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.log.Debug("plugin: discovering plugins", "dir", m.pluginsDir)

	if err := os.MkdirAll(m.pluginsDir, 0755); err != nil {
		return fmt.Errorf("create plugins dir: %w", err)
	}

	entries, err := os.ReadDir(m.pluginsDir)
	if err != nil {
		return fmt.Errorf("read plugins dir: %w", err)
	}

	m.log.Debug("plugin: scanned entries", "count", len(entries))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip temp install dirs
		if strings.HasPrefix(entry.Name(), ".install-") {
			os.RemoveAll(filepath.Join(m.pluginsDir, entry.Name()))
			continue
		}

		if err := m.loadLuaPluginDir(entry.Name()); err != nil {
			m.log.Error("plugin: load error", "dir", entry.Name(), "err", err)
		}
	}

	return nil
}

func (m *Manager) loadLuaPluginDir(dirName string) error {
	m.log.Debug("plugin: loading lua plugin dir", "dir", dirName)
	dir := filepath.Join(m.pluginsDir, dirName)

	cfgPath := filepath.Join(dir, "plugin.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read config %s: %v", cfgPath, err)
	}

	var cfg DiscoveryConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config %s: %v", cfgPath, err)
	}

	if cfg.Type != "lua" {
		if cfg.Command != "" {
			m.log.Info("plugin: skipped", "name", cfg.Name)
		}
		return nil
	}

	p, err := LoadLuaPlugin(dir, m.log)
	if err != nil {
		return fmt.Errorf("load lua %s: %v", cfg.Name, err)
	}

	m.luaPlugins[p.Manifest.ID] = p
	m.log.Info("plugin: loaded", "name", p.Manifest.Name, "version", p.Manifest.Version)
	return nil
}

func (m *Manager) InstallPlugin(zipData []byte) (*PluginInfo, error) {
	m.log.Debug("plugin: installing plugin", "bytes", len(zipData))

	if err := os.MkdirAll(m.pluginsDir, 0755); err != nil {
		return nil, fmt.Errorf("create plugins dir: %w", err)
	}

	tmpDir, err := os.MkdirTemp(m.pluginsDir, ".install-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("read zip: %w", err)
	}

	for _, f := range reader.File {
		cleanName := filepath.ToSlash(filepath.Clean(f.Name))
		if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
			return nil, fmt.Errorf("invalid zip entry: %s", f.Name)
		}
		targetPath := filepath.Join(tmpDir, cleanName)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return nil, fmt.Errorf("create dir %s: %w", cleanName, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return nil, fmt.Errorf("create parent dir for %s: %w", cleanName, err)
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open zip entry %s: %w", f.Name, err)
		}

		out, err := os.Create(targetPath)
		if err != nil {
			rc.Close()
			return nil, fmt.Errorf("create %s: %w", cleanName, err)
		}

		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("extract %s: %w", cleanName, err)
		}
	}

	cfgData, err := os.ReadFile(filepath.Join(tmpDir, "plugin.json"))
	if err != nil {
		return nil, fmt.Errorf("plugin.json not found in zip")
	}
	var cfg DiscoveryConfig
	if err := json.Unmarshal(cfgData, &cfg); err != nil {
		return nil, fmt.Errorf("invalid plugin.json: %w", err)
	}
	if cfg.ID == "" {
		return nil, fmt.Errorf("plugin.json missing id")
	}
	if cfg.Type != "lua" {
		return nil, fmt.Errorf("unsupported plugin type: %s", cfg.Type)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "main.lua")); err != nil {
		return nil, fmt.Errorf("main.lua not found in zip")
	}

	targetDir := filepath.Join(m.pluginsDir, cfg.ID)

	m.mu.Lock()
	if old, ok := m.luaPlugins[cfg.ID]; ok {
		old.Close()
		delete(m.luaPlugins, cfg.ID)
	}
	m.mu.Unlock()

	if err := os.RemoveAll(targetDir); err != nil {
		return nil, fmt.Errorf("remove existing plugin dir: %w", err)
	}

	if err := os.Rename(tmpDir, targetDir); err != nil {
		return nil, fmt.Errorf("move plugin dir: %w", err)
	}

	p, err := LoadLuaPlugin(targetDir, m.log)
	if err != nil {
		return nil, fmt.Errorf("load plugin: %w", err)
	}

	m.mu.Lock()
	m.luaPlugins[cfg.ID] = p
	m.mu.Unlock()

	info := &PluginInfo{
		ID:      p.Manifest.ID,
		Name:    p.Manifest.Name,
		Version: p.Manifest.Version,
		Status:  "active",
		Menu:    p.Manifest.MenuEntries,
		UI:      p.Manifest.UI,
	}

	m.log.Info("plugin: installed", "name", p.Manifest.Name, "version", p.Manifest.Version)
	return info, nil
}

func (m *Manager) RemovePlugin(pluginID string) error {
	m.log.Debug("plugin: removing plugin", "id", pluginID)
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.luaPlugins[pluginID]
	if !ok {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	p.Close()
	delete(m.luaPlugins, pluginID)

	dir := filepath.Join(m.pluginsDir, pluginID)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove plugin dir: %w", err)
	}

	m.log.Info("plugin: removed", "id", pluginID)
	return nil
}

func (m *Manager) ExecuteAction(pluginID, action, argsJSON string) error {
	m.log.Debug("plugin: executing action", "action", action, "pluginID", pluginID)
	m.mu.RLock()
	p, ok := m.luaPlugins[pluginID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("plugin %s not found", pluginID)
	}
	var args map[string]any
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Errorf("invalid args JSON: %w", err)
		}
	}
	err := p.ExecuteAction(action, args)
	m.log.Debug("plugin: action completed", "action", action, "pluginID", pluginID, "err", err)
	return err
}

func (m *Manager) Plugins() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]PluginInfo, 0, len(m.instances)+len(m.luaPlugins))

	for _, inst := range m.instances {
		status := "active"
		select {
		case <-inst.done:
			status = "stopped"
		default:
		}
		result = append(result, PluginInfo{
			ID:      inst.Manifest.ID,
			Name:    inst.Manifest.Name,
			Version: inst.Manifest.Version,
			Status:  status,
			Menu:    inst.Manifest.MenuEntries,
		})
	}

	for _, p := range m.luaPlugins {
		result = append(result, PluginInfo{
			ID:      p.Manifest.ID,
			Name:    p.Manifest.Name,
			Version: p.Manifest.Version,
			Status:  "active",
			Menu:    p.Manifest.MenuEntries,
			UI:      p.Manifest.UI,
		})
	}

	return result
}

func (m *Manager) Shutdown() {
	m.log.Debug("plugin: shutting down manager")
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, inst := range m.instances {
		m.log.Debug("plugin: stopping instance", "id", id)
		if inst.cancel != nil {
			inst.cancel()
		}
		if inst.Cmd != nil && inst.Cmd.Process != nil {
			if err := inst.Cmd.Process.Kill(); err != nil {
				m.log.Error("plugin: failed to kill instance process", "id", id, "error", err)
			}
		}
		delete(m.instances, id)
	}

	for id, p := range m.luaPlugins {
		p.Close()
		delete(m.luaPlugins, id)
	}
}

// --- Exec plugin support (deprecated, kept for backward compat) ---

func (m *Manager) LaunchExecPlugins(ctx context.Context, hostPort int) {
	entries, err := os.ReadDir(m.pluginsDir)
	if err != nil {
		m.log.Error("plugin: failed to read plugins dir", "dir", m.pluginsDir, "error", err)
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		cfgPath := filepath.Join(m.pluginsDir, entry.Name(), "plugin.json")
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			m.log.Debug("plugin: skipped dir (no plugin.json)", "dir", entry.Name(), "error", err)
			continue
		}
		var cfg DiscoveryConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			m.log.Debug("plugin: skipped dir (invalid plugin.json)", "dir", entry.Name(), "error", err)
			continue
		}
		if cfg.Type == "lua" || cfg.Command == "" {
			continue
		}
		if err := m.launch(ctx, &cfg, hostPort); err != nil {
			m.log.Info("plugin: launch failed", "name", cfg.Name, "err", err)
		}
	}
}

func (m *Manager) launch(ctx context.Context, cfg *DiscoveryConfig, hostPort int) error {
	m.log.Debug("plugin: launching", "id", cfg.ID, "command", cfg.Command)
	dir := filepath.Join(m.pluginsDir, cfg.ID)
	exePath := filepath.Join(dir, cfg.Command)

	args := make([]string, len(cfg.Args))
	for i, a := range cfg.Args {
		args[i] = strings.ReplaceAll(a, "{{HOST_PORT}}", fmt.Sprintf("%d", hostPort))
	}

	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, exePath, args...)
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start %s: %w", cfg.Command, err)
	}

	go io.Copy(os.Stderr, stderr)

	port, err := readPort(stdout, 10*time.Second)
	if err != nil {
		cmd.Process.Kill()
		cancel()
		return fmt.Errorf("read port: %w", err)
	}

	manifest, err := m.fetchManifest(port)
	if err != nil {
		cmd.Process.Kill()
		cancel()
		return fmt.Errorf("fetch manifest: %w", err)
	}

	inst := &Instance{
		Manifest: manifest,
		Cmd:      cmd,
		Port:     port,
		cancel:   cancel,
		done:     make(chan struct{}),
	}

	go func() {
		cmd.Wait()
		close(inst.done)
	}()

	m.mu.Lock()
	m.instances[manifest.ID] = inst
	m.mu.Unlock()

	m.log.Info("plugin: launched", "name", manifest.Name, "version", manifest.Version, "port", port)
	return nil
}

func readPort(r io.Reader, timeout time.Duration) (int, error) {
	type portResult struct {
		port int
		err  error
	}

	ch := make(chan portResult, 1)

	go func() {
		buf := make([]byte, 1024)
		n, err := r.Read(buf)
		if err != nil {
			ch <- portResult{0, err}
			return
		}
		line := strings.TrimSpace(string(buf[:n]))
		var port int
		if _, err := fmt.Sscanf(line, "PORT:%d", &port); err != nil {
			ch <- portResult{0, fmt.Errorf("unexpected output %q: %w", line, err)}
			return
		}
		ch <- portResult{port, nil}
	}()

	select {
	case r := <-ch:
		return r.port, r.err
	case <-time.After(timeout):
		return 0, fmt.Errorf("timeout waiting for plugin port")
	}
}

func (m *Manager) fetchManifest(port int) (*Manifest, error) {
	resp, err := m.httpClient.Get(fmt.Sprintf("http://127.0.0.1:%d/api/describe", port))
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("describe returned status %d", resp.StatusCode)
	}

	var manifest Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	return &manifest, nil
}
