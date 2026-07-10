package main

import (
	"embed"
	"flag"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"script-manager/internal/config"
	"script-manager/internal/gui"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed frontend/src/messages.json
var defaultMessagesJSON []byte

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
	app.SetDefaultMessages(defaultMessagesJSON)

	err := wails.Run(&options.App{
		Title:  "Script Manager",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
