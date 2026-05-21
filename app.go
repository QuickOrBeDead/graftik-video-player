package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"graftik-wails/internal"
	"graftik-wails/internal/data"

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
	videoServerPort int
}

func NewApp() *App {
	return &App{
		Service: internal.NewPlayerService(nil, nil),
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

	// Start dedicated video file server
	a.videoServerPort, err = startVideoServer()
	if err != nil {
		println("Error starting video server:", err.Error())
	}

	// Locate ffmpeg/ffprobe - first check bundled, then PATH
	a.ffmpegDir = a.findFFmpegDir()
	ffmpegPath := filepath.Join(a.ffmpegDir, "ffmpeg.exe")
	ffprobePath := filepath.Join(a.ffmpegDir, "ffprobe.exe")

	a.Service.SetStore(a.store)
	a.Service.SetThumbnailStore(a.thumbnailStore)
	a.Service.SetContext(ctx)
	a.Service.SetFFmpegPaths(ffmpegPath, ffprobePath)
}

func (a *App) shutdown(ctx context.Context) {
	if a.store != nil {
		a.store.Close()
	}
}

func (a *App) GetVideoServerPort() int {
	return a.videoServerPort
}

func (a *App) findFFmpegDir() string {
	exeDir, err := os.Executable()
	if err == nil {
		appDir := filepath.Dir(exeDir)

		// Check bundled bin/ directory next to executable
		// Works for: Windows NSIS ($INSTDIR\bin\), macOS .app (Contents/MacOS/bin/), Linux (next to binary)
		binDir := filepath.Join(appDir, "bin")
		if platformBin("ffmpeg", binDir) != "" {
			return binDir
		}

		// macOS: also check Contents/Resources/bin/ relative to MacOS/
		resBin := filepath.Join(appDir, "..", "Resources", "bin")
		if resolved, _ := filepath.Abs(resBin); resolved != "" {
			if platformBin("ffmpeg", resolved) != "" {
				return resolved
			}
		}
	}

	// Check PATH
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		return filepath.Dir(path)
	}

	return "."
}

// platformBin returns the path to the binary if it exists, handling per-platform extensions
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
				Pattern:     "*.mp4;*.mov;*.ogg;*.webm;*.3gp",
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
