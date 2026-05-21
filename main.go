package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

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
		Bind: []interface{}{
			app.Service,
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
		},
		Linux: &linux.Options{},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
