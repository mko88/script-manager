package main

import (
	"embed"
	"flag"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"script-manager/internal/gui"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	cfgPath := flag.String("config", "", "path to config file (default: auto-detect)")
	flag.Parse()

	app := gui.NewApp(*cfgPath)

	err := wails.Run(&options.App{
		Title:  "Script Manager",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		// Only takes effect in a build made with devtools enabled
		// (build.sh --devtools); a normal build ignores it.
		Debug: options.Debug{
			OpenInspectorOnStartup: true,
		},
		Frameless: true,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WindowClassName: gui.WindowClassName,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
