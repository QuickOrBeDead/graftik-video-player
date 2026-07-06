package main

import (
	"context"
	"embed"
	"os"
	"path/filepath"

	"graftik-wails/internal/logger"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

var readyToClose = make(chan bool)

func (app *App) SetReadyToClose() {
	app.log.Debug("App: SetReadyToClose is called")
	readyToClose <- true
}

func main() {
	logDir := ""
	if userDataDir, err := os.UserConfigDir(); err == nil {
		logDir = filepath.Join(userDataDir, "graftik-video-player", "logs")
	}

	log := logger.New(logger.LogConfig{
		Level:     logger.LevelDebug,
		LogToFile: false,
		LogDir:    logDir,
	})

	app := NewApp(log)

	err := wails.Run(&options.App{
		Title:     "Graftik Video Player",
		Width:     1000,
		Height:    670,
		MinWidth:  800,
		MinHeight: 480,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 15, B: 15, A: 255},
		Menu:             app.CreateAppMenu(),
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []any{
			app.Service,
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
		},
		Linux: &linux.Options{
			WindowIsTranslucent: false,
			WebviewGpuPolicy:    linux.WebviewGpuPolicyAlways,
			ProgramName:         "graftik-video-player",
			Icon:                appIcon,
		},
		OnBeforeClose: func(ctx context.Context) bool {
			log.Debug("Main: Wails app on before close: start emit before-app-close event")
			wailsRuntime.EventsEmit(ctx, "before-app-close")
			log.Debug("Main: Wails app on before close: emit before-app-close event finished")
			<-readyToClose
			log.Debug("Main: Wails app on before close: readyToClose signal received. Closing app.")
			return false
		},
	})

	if err != nil {
		log.Error("application failed to start", "error", err)
	}
}
