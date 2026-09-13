package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"script-manager/internal/config"
	"script-manager/internal/configmigrate"
	"script-manager/internal/ui"
)

func main() {
	cfgPath := flag.String("config", "", "path to config file (default: auto-detect)")
	flag.Parse()

	if err := ensureConfigFormat(*cfgPath); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	loadConfig := func() (*config.Config, error) {
		if *cfgPath != "" {
			return config.LoadFromWithError(*cfgPath)
		}
		return config.LoadWithError()
	}
	cfg, err := loadConfig()

	p := tea.NewProgram(ui.NewApp(cfg, loadConfig, err), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// ensureConfigFormat asks before converting a pre-env config. It runs before
// the alt screen is entered, so a plain prompt on stdin is enough. Any problem
// other than a decline is left to the load, whose error says more.
func ensureConfigFormat(cfgPath string) error {
	path := cfgPath
	if path == "" {
		resolved, err := config.ResolvePath()
		if err != nil {
			return nil
		}
		path = resolved
	}

	backup, err := configmigrate.Ensure(path, approveConversion)
	if backup != "" {
		fmt.Printf("Converted. Backup: %s\n", backup)
	}
	if err == configmigrate.ErrDeclined {
		return err
	}
	return nil
}

func approveConversion(path, backupPath string) bool {
	fmt.Println(configmigrate.PromptMessage(path, backupPath))
	fmt.Print("[y/N] ")

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
