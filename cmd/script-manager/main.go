package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"script-manager/internal/config"
	"script-manager/internal/ui"
)

func main() {
	cfgPath := flag.String("config", "", "path to config file (default: auto-detect)")
	flag.Parse()

	loadConfig := func() (*config.Config, error) {
		if *cfgPath != "" {
			return config.LoadFromWithError(*cfgPath)
		}
		return config.LoadWithError()
	}
	cfg, _ := loadConfig()

	p := tea.NewProgram(ui.NewApp(cfg, loadConfig), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
