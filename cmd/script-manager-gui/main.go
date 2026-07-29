package main

import (
	"embed"
	"flag"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"script-manager/internal/config"
	"script-manager/internal/gui"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	cfgPath := flag.String("config", "", "path to config file (default: auto-detect)")
	flag.Parse()

	loadConfig := func() (*config.Config, error) {
		if *cfgPath != "" {
			return config.LoadFromWithError(*cfgPath)
		}
		return config.LoadWithError()
	}

	app := gui.NewApp(loadConfig)

	err := wails.Run(&options.App{
		Title:  "Script Manager",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		// No native title bar — App.svelte's own toolbar is the drag
		// handle (via the --wails-draggable CSS property) and carries its
		// own minimize/maximize/close buttons in place of the native ones.
		Frameless: true,
		Bind: []interface{}{
			app,
		},
		// WindowClassName lets internal/gui's window-transparency toggle
		// (Windows-only) find this window unambiguously, even with
		// sm-config-edit — a second Wails app that defaults to the same
		// window class — running alongside it.
		Windows: &windows.Options{
			WindowClassName: gui.WindowClassName,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
