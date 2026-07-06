package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"graftik-wails/internal"
	"graftik-wails/internal/data"
	"graftik-wails/internal/hls"
	graftikLogger "graftik-wails/internal/logger"
	"graftik-wails/internal/media"
	"graftik-wails/internal/plugin"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx             context.Context
	Service         *internal.PlayerService
	store           *data.PlayerDataStore
	thumbnailStore  *data.ThumbnailDataStore
	ffmpegDir       string
	videoServer     *VideoServer
	hlsEngine       *hls.Engine
	currentStreamID string
	pluginManager   *plugin.Manager
	pluginsDir      string
	updateETag      string
	log             graftikLogger.Logger
}

func NewApp(log graftikLogger.Logger) (*App, error) {
	userDataDir, err := os.UserConfigDir()
	appDataDir := ""
	if err == nil {
		appDataDir = filepath.Join(userDataDir, "graftik-video-player")
	}

	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create app data dir: %w", err)
	}

	dbPath := filepath.Join(appDataDir, "player.db")

	store, err := data.NewPlayerDataStore(appDataDir, dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create player data store: %w", err)
	}

	thumbnailStore := data.NewThumbnailDataStore(appDataDir)

	pluginsDir := filepath.Join(appDataDir, "plugins")

	return &App{
		Service:        internal.NewPlayerService(store, thumbnailStore, log),
		store:          store,
		thumbnailStore: thumbnailStore,
		pluginManager:  plugin.NewManager(pluginsDir),
		pluginsDir:     pluginsDir,
		log:            log,
	}, nil
}

func (a *App) Logger() graftikLogger.Logger {
	return a.log
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Defer frontend log sink until frontend signals it's ready
	wailsRuntime.EventsOn(a.ctx, "frontend-ready", func(_ ...any) {
		sink := graftikLogger.SyncFrontendSink(a.ctx, wailsRuntime.EventsEmit)
		a.log.(*graftikLogger.DefaultLogger).SetFrontendSink(sink)
	})

	userDataDir, err := os.UserConfigDir()
	if err != nil {
		a.log.Error("failed to get user config dir", "error", err)
		return
	}
	appDataDir := filepath.Join(userDataDir, "graftik-video-player")
	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		a.log.Error("failed to create app data dir", "error", err)
		return
	}

	if err := a.store.Initialize(); err != nil {
		a.log.Error("failed to run migrations", "error", err)
		return
	}

	if err := a.store.AddDefaultPlaylist(); err != nil {
		a.log.Error("failed to create default playlist", "error", err)
		return
	}

	// Apply log config from preferences if available
	if preferences := a.store.GetPreferences(); preferences != nil {
		level := graftikLogger.LevelDebug
		if preferences.LogLevel != "" {
			level = graftikLogger.ParseLevel(preferences.LogLevel)
		}
		if !preferences.Debug {
			level = graftikLogger.LevelInfo
		}
		a.log.(*graftikLogger.DefaultLogger).SetLevel(level)

		if preferences.LogToFile {
			logPath := filepath.Join(appDataDir, "logs", "app.log")
			if err := a.log.(*graftikLogger.DefaultLogger).AddFileHandler(logPath); err != nil {
				a.log.Warn("failed to enable file logging", "path", logPath, "error", err)
			}
		}

		if w, h := preferences.WindowWidth, preferences.WindowHeight; w > 0 && h > 0 {
			wailsRuntime.WindowSetSize(ctx, w, h)
		}
	}

	// Start dedicated video file server with mux
	a.videoServer, err = NewVideoServer(a.log)
	if err != nil {
		a.log.Error("App startup: failed to start video server", "error", err)
	}
	if a.videoServer != nil {
		a.log.Debug("App startup: video server started", "port", a.videoServer.Port())
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	// Locate ffmpeg/ffprobe - first check bundled, then PATH
	a.ffmpegDir = a.findFFmpegDir()
	ffmpegPath := filepath.Join(a.ffmpegDir, "ffmpeg"+ext)
	ffprobePath := filepath.Join(a.ffmpegDir, "ffprobe"+ext)

	a.log.Debug("App startup: ffmpeg paths", "ffmpeg", ffmpegPath, "ffprobe", ffprobePath)

	// Init HLS engine
	hlsDir := filepath.Join(os.TempDir(), "graftik-hls")
	a.hlsEngine = hls.NewEngine(ffmpegPath, hlsDir)

	a.Service.SetContext(ctx)
	a.Service.SetFFmpegPaths(ffmpegPath, ffprobePath)
	a.Service.SetHlsEngine(a.hlsEngine)

	// Register HLS routes on video server
	if a.videoServer != nil {
		a.videoServer.RegisterHLS(a.hlsEngine.BaseDir())
	}

	// Wire plugin manager logging
	a.pluginManager.SetLogFn(func(format string, args ...any) {
		a.log.Info(fmt.Sprintf(format, args...))
	})

	// Wire up Lua plugin host callbacks
	plugin.SetAddToPlaylistFn(func(path, title string) {
		if a.store == nil || a.ctx == nil {
			return
		}
		items := a.store.InitNewPlaylistItems([]string{path})
		if title != "" && len(items) > 0 {
			items[0].Title = title
		}
		if playlistID := a.store.GetCurrentPlaylistID(); playlistID != "" && len(items) > 0 {
			items[0].PlaylistID = playlistID
		}
		wailsRuntime.EventsEmit(a.ctx, "add-playlist-item", items)
	})

	plugin.SetEventSink(func(event string, data string) {
		if a.ctx == nil {
			return
		}
		wailsRuntime.EventsEmit(a.ctx, event, data)
	})

	// Listen for plugin action requests from frontend (no-arg actions)
	wailsRuntime.EventsOn(a.ctx, "run-plugin-action", func(optionalData ...any) {
		if len(optionalData) < 1 {
			return
		}
		payload, ok := optionalData[0].(map[string]any)
		if !ok {
			return
		}
		pluginID, _ := payload["pluginId"].(string)
		action, _ := payload["action"].(string)
		if pluginID == "" || action == "" {
			return
		}
		if err := a.pluginManager.ExecuteAction(pluginID, action, ""); err != nil {
			a.log.Error("plugin action failed", "plugin", pluginID, "action", action, "error", err)
		}
	})

	// Discover Lua plugins
	if err := a.pluginManager.Discover(ctx); err != nil {
		a.log.Error("plugin discovery failed", "error", err)
	}

	// Background update check on startup
	go func() {
		time.Sleep(5 * time.Second)
		info, err := a.CheckForUpdates()
		if err != nil {
			a.log.Warn("update check failed", "error", err)
			return
		}
		if info != nil && info.HasUpdate {
			wailsRuntime.EventsEmit(ctx, "update-available", info.LatestVersion)
		}
	}()
}

func (a *App) shutdown(ctx context.Context) {
	a.pluginManager.Shutdown()

	// Stop all HLS streams and clean up
	if a.hlsEngine != nil {
		a.hlsEngine.Shutdown()
	}

	if a.store != nil {
		// Save window size
		w, h := wailsRuntime.WindowGetSize(ctx)
		if w > 0 && h > 0 {
			a.store.UpdateSettings(map[string]any{
				"windowWidth":  float64(w),
				"windowHeight": float64(h),
			})
		}

		a.store.Close()
	}
}

func (a *App) GetVideoServerPort() int {
	if a.videoServer == nil {
		return 0
	}
	return a.videoServer.Port()
}

func (a *App) GetStreamURL(playlistItemID string) *data.StreamURLResult {
	a.log.Debug("GetStreamURL starting", "playlistItemID", playlistItemID)

	if a.store == nil || a.hlsEngine == nil {
		a.log.Debug("GetStreamURL: store or hlsEngine nil")
		return nil
	}

	item := a.store.GetPlaylistItem(playlistItemID)
	if item == nil {
		a.log.Debug("GetStreamURL: playlist item not found", "playlistItemID", playlistItemID)
		return nil
	}

	a.log.Debug("GetStreamURL: playlist item found", "id", item.ID, "path", item.Path, "title", item.Title)

	// Stop previous HLS stream if any
	if a.currentStreamID != "" {
		a.log.Debug("GetStreamURL: stopping previous stream", "streamID", a.currentStreamID)
		a.hlsEngine.StopStream(a.currentStreamID)
		a.currentStreamID = ""
	}

	// For native extensions - skip probe and go directly
	if media.IsNativeExtension(item.Path) {
		url := fmt.Sprintf("http://127.0.0.1:%d/api/video?path=%s",
			a.videoServer.Port(), url.QueryEscape(item.Path))
		a.log.Debug("GetStreamURL: native extension, serving directly", "url", url)
		return &data.StreamURLResult{URL: url}
	}

	// Probe to determine remux vs transcode
	info, err := media.Probe(a.Service.FFprobePath(), item.Path)
	if err != nil {
		a.log.Error("GetStreamURL: media probe failed", "path", item.Path, "error", err)
		return nil
	}

	a.log.Debug("GetStreamURL: probe result", "path", item.Path, "action", info.Action, "actionLabel", info.ActionLabel)

	// If natively playable, serve directly
	if info.Action == "native" {
		url := fmt.Sprintf("http://127.0.0.1:%d/api/video?path=%s",
			a.videoServer.Port(), url.QueryEscape(item.Path))
		a.log.Debug("GetStreamURL: native playable, serving directly", "url", url)
		return &data.StreamURLResult{URL: url}
	}

	// Detect HW encoder for transcodes
	if info.Action == "sw_transcode" {
		hw := media.DetectHWEncoder(a.Service.FFmpegPath())
		if hw != "" {
			info.Action = "hw_transcode"
			info.HWEncoder = hwEncoderShortLabel(hw)
			info.ActionLabel = fmt.Sprintf("HW Transcode (%s)", info.HWEncoder)
			a.log.Debug("GetStreamURL: hw encoder detected", "encoder", hw)
		} else {
			a.log.Debug("GetStreamURL: no hw encoder, using sw transcode")
		}
	}

	// Start HLS stream
	streamID, err := a.hlsEngine.StartStream(item.Path, info)
	if err != nil {
		a.log.Error("GetStreamURL: hls stream start failed", "path", item.Path, "error", err)
		return nil
	}

	a.currentStreamID = streamID

	a.log.Debug("GetStreamURL: hls stream started", "streamID", streamID)

	return &data.StreamURLResult{
		URL:      fmt.Sprintf("http://127.0.0.1:%d/hls/%s/stream.m3u8", a.videoServer.Port(), streamID),
		StreamID: streamID,
	}
}

func hwEncoderShortLabel(name string) string {
	switch name {
	case "h264_nvenc":
		return "NVENC"
	case "h264_qsv":
		return "QSV"
	case "h264_amf":
		return "AMF"
	}
	return name
}

func (a *App) StopHLSStream(streamID string) {
	if a.hlsEngine == nil {
		return
	}
	a.hlsEngine.StopStream(streamID)
	if a.currentStreamID == streamID {
		a.currentStreamID = ""
	}
}

func (a *App) GetPlugins() []plugin.PluginInfo {
	return a.pluginManager.Plugins()
}

func (a *App) ExecutePluginAction(pluginID, action, argsJSON string) error {
	return a.pluginManager.ExecuteAction(pluginID, action, argsJSON)
}

func (a *App) PickPluginFile() (string, error) {
	if a.ctx == nil {
		return "", nil
	}
	file, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Plugin ZIP",
		Filters: []wailsRuntime.FileFilter{
			{
				DisplayName: "Plugin ZIP",
				Pattern:     "*.zip",
			},
		},
	})
	if err != nil {
		return "", err
	}
	return file, nil
}

func (a *App) PickDirectory() (string, error) {
	if a.ctx == nil {
		return "", nil
	}
	dir, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Directory",
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

func (a *App) GetPluginFile(pluginID, fileName string) (string, error) {
	if strings.Contains(pluginID, "..") || strings.Contains(pluginID, "/") || strings.Contains(pluginID, "\\") {
		return "", fmt.Errorf("invalid plugin id")
	}
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
		return "", fmt.Errorf("invalid file name")
	}
	fullPath := filepath.Join(a.pluginsDir, pluginID, fileName)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("read plugin file: %w", err)
	}
	return string(data), nil
}

func (a *App) InstallPluginFromFile(filePath string) (*plugin.PluginInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return a.pluginManager.InstallPlugin(data)
}

type progressReader struct {
	reader     io.Reader
	total      int64
	read       int64
	onProgress func(pct int)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)
	if pr.total > 0 && pr.onProgress != nil {
		pct := int(pr.read * 100 / pr.total)
		pr.onProgress(pct)
	}
	return n, err
}

func (a *App) InstallPluginFromURL(url string) (*plugin.PluginInfo, error) {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "plugin-install-log", `{"message":"Downloading plugin..."}`)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	pr := &progressReader{
		reader: resp.Body,
		total:  resp.ContentLength,
		onProgress: func(pct int) {
			if a.ctx != nil {
				wailsRuntime.EventsEmit(a.ctx, "plugin-install-progress", fmt.Sprintf(`{"percent":%d}`, pct))
			}
		},
	}

	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "plugin-install-log", `{"message":"Installing plugin..."}`)
	}

	data, err := io.ReadAll(pr)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	info, err := a.pluginManager.InstallPlugin(data)
	if err != nil {
		return nil, err
	}

	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "plugin-install-complete", fmt.Sprintf(`{"id":"%s","name":"%s","version":"%s"}`, info.ID, info.Name, info.Version))
	}

	a.log.Info("plugin installed from url", "url", url, "name", info.Name, "version", info.Version)
	return info, nil
}

func (a *App) findFFmpegDir() string {
	exeDir, err := os.Executable()
	if err == nil {
		appDir := filepath.Dir(exeDir)

		binDir := filepath.Join(appDir, "bin")
		if platformBin("ffmpeg", binDir) != "" {
			return binDir
		}

		resBin := filepath.Join(appDir, "..", "Resources", "bin")
		if resolved, _ := filepath.Abs(resBin); resolved != "" {
			if platformBin("ffmpeg", resolved) != "" {
				return resolved
			}
		}
	}

	if path, err := exec.LookPath("ffmpeg"); err == nil {
		return filepath.Dir(path)
	}

	return "."
}

func platformBin(name, dir string) string {
	candidates := []string{name, name + ".exe"}
	for _, c := range candidates {
		path := filepath.Join(dir, c)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func (a *App) CreateAppMenu() *menu.Menu {
	appMenu := menu.NewMenu()

	fileMenu := appMenu.AddSubmenu("File")

	fileMenu.AddText("Add Video", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
		if a.ctx != nil {
			a.openFileDialog(a.ctx)
		}
	})

	fileMenu.AddSeparator()

	fileMenu.AddText("New Playlist", nil, func(_ *menu.CallbackData) {
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "open-new-playlist")
		}
	})

	fileMenu.AddText("Choose Playlist", nil, func(_ *menu.CallbackData) {
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "open-choose-playlist")
		}
	})

	fileMenu.AddSeparator()

	fileMenu.AddText("Plugins...", keys.CmdOrCtrl("p"), func(_ *menu.CallbackData) {
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "open-plugin-panel")
		}
	})

	fileMenu.AddSeparator()

	fileMenu.AddText("Exit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		if a.ctx != nil {
			wailsRuntime.Quit(a.ctx)
		}
	})

	helpMenu := appMenu.AddSubmenu("Help")

	helpMenu.AddText("Check for Updates", nil, func(_ *menu.CallbackData) {
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "check-for-updates")
		}
	})

	helpMenu.AddSeparator()

	helpMenu.AddText("About", nil, func(_ *menu.CallbackData) {
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "show-about")
		}
	})

	return appMenu
}

func (a *App) openFileDialog(ctx context.Context) {
	files, err := wailsRuntime.OpenMultipleFilesDialog(ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Videos",
		Filters: []wailsRuntime.FileFilter{
			{
				DisplayName: "Videos",
				Pattern:     "*.mp4;*.mov;*.ogg;*.webm;*.3gp;*.mkv;*.avi;*.flv;*.ts;*.mts;*.m2ts;*.wmv;*.rm;*.rmvb;*.vob;*.mpg;*.mpeg;*.m4v",
			},
		},
	})
	if err != nil || len(files) == 0 {
		return
	}

	items := a.store.InitNewPlaylistItems(files)
	a.log.Info("adding playlist items", "count", len(items))
	wailsRuntime.EventsEmit(ctx, "add-playlist-item", items)
}
