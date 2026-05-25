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
	"strings"

	"graftik-wails/internal"
	"graftik-wails/internal/data"
	"graftik-wails/internal/hls"
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
}

func NewApp() *App {
	userDataDir, err := os.UserConfigDir()
	appDataDir := ""
	if err == nil {
		appDataDir = filepath.Join(userDataDir, "graftik-video-player")
	}

	pluginsDir := filepath.Join(appDataDir, "plugins")

	return &App{
		Service:       internal.NewPlayerService(nil, nil),
		pluginManager: plugin.NewManager(pluginsDir),
		pluginsDir:    pluginsDir,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	userDataDir, err := os.UserConfigDir()
	if err != nil {
		println("Error getting user config dir:", err.Error())
		return
	}
	appDataDir := filepath.Join(userDataDir, "graftik-video-player")
	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		println("Error creating app data dir:", err.Error())
		return
	}

	dbPath := filepath.Join(appDataDir, "player.db")

	a.store, err = data.NewPlayerDataStore(appDataDir, dbPath)
	if err != nil {
		println("Error initializing data store:", err.Error())
		return
	}

	if err := a.store.Initialize(); err != nil {
		println("Error running migrations:", err.Error())
		return
	}

	if err := a.store.AddDefaultPlaylist(); err != nil {
		println("Error creating default playlist:", err.Error())
		return
	}

	a.thumbnailStore = data.NewThumbnailDataStore(appDataDir)

	// Start dedicated video file server with mux
	a.videoServer, err = NewVideoServer()
	if err != nil {
		println("Error starting video server:", err.Error())
	}

	// Locate ffmpeg/ffprobe - first check bundled, then PATH
	a.ffmpegDir = a.findFFmpegDir()
	ffmpegPath := filepath.Join(a.ffmpegDir, "ffmpeg.exe")
	ffprobePath := filepath.Join(a.ffmpegDir, "ffprobe.exe")

	// Init HLS engine
	hlsDir := filepath.Join(os.TempDir(), "graftik-hls")
	a.hlsEngine = hls.NewEngine(ffmpegPath, hlsDir)

	a.Service.SetStore(a.store)
	a.Service.SetThumbnailStore(a.thumbnailStore)
	a.Service.SetContext(ctx)
	a.Service.SetFFmpegPaths(ffmpegPath, ffprobePath)
	a.Service.SetHlsEngine(a.hlsEngine)

	// Register HLS routes on video server
	if a.videoServer != nil {
		a.videoServer.RegisterHLS(a.hlsEngine.BaseDir())
	}

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
			println("plugin action error:", err.Error())
		}
	})

	// Discover Lua plugins
	if err := a.pluginManager.Discover(ctx); err != nil {
		println("Plugin discovery error:", err.Error())
	}

	// Restore window size from prefs
	if a.store != nil {
		prefs := a.store.GetPreferences()
		if w, h := prefs.WindowWidth, prefs.WindowHeight; w > 0 && h > 0 {
			wailsRuntime.WindowSetSize(ctx, w, h)
		}
	}
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
	if a.store == nil || a.hlsEngine == nil {
		return nil
	}

	item := a.store.GetPlaylistItem(playlistItemID)
	if item == nil {
		return nil
	}

	// Stop previous HLS stream if any
	if a.currentStreamID != "" {
		a.hlsEngine.StopStream(a.currentStreamID)
		a.currentStreamID = ""
	}

	// For native extensions - skip probe and go directly
	if media.IsNativeExtension(item.Path) {
		return &data.StreamURLResult{
			URL: fmt.Sprintf("http://127.0.0.1:%d/api/video?path=%s",
				a.videoServer.Port(), url.QueryEscape(item.Path)),
		}
	}

	// Probe to determine remux vs transcode
	info, err := media.Probe(a.Service.FFprobePath(), item.Path)
	if err != nil {
		println("probe error:", err.Error())
		return nil
	}

	// If natively playable, serve directly
	if info.Action == "native" {
		return &data.StreamURLResult{
			URL: fmt.Sprintf("http://127.0.0.1:%d/api/video?path=%s",
				a.videoServer.Port(), url.QueryEscape(item.Path)),
		}
	}

	// Detect HW encoder for transcodes
	if info.Action == "sw_transcode" {
		hw := media.DetectHWEncoder(a.Service.FFmpegPath())
		if hw != "" {
			info.Action = "hw_transcode"
			info.HWEncoder = hwEncoderShortLabel(hw)
			info.ActionLabel = fmt.Sprintf("HW Transcode (%s)", info.HWEncoder)
		}
	}

	// Start HLS stream
	streamID, err := a.hlsEngine.StartStream(item.Path, info)
	if err != nil {
		println("hls start error:", err.Error())
		return nil
	}

	a.currentStreamID = streamID

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

	fmt.Printf("plugin: installed from url %s -> %s v%s\n", url, info.Name, info.Version)
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
	fmt.Printf("Adding %d items\n", len(items))
	wailsRuntime.EventsEmit(ctx, "add-playlist-item", items)
}
