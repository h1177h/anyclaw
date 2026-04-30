package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewDesktopApp()

	err := wails.Run(&options.App{
		Title:            "AnyClaw Desktop",
		Width:            desktopWindowDefaultWidth,
		Height:           desktopWindowDefaultHeight,
		MinWidth:         desktopWindowMinWidth,
		MinHeight:        desktopWindowMinHeight,
		Frameless:        true,
		BackgroundColour: options.NewRGBA(0, 0, 0, 0),
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Windows: &windows.Options{
			DisableWindowIcon:                 false,
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
			DisableFramelessWindowDecorations: true,
		},
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
