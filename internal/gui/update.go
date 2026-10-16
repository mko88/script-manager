package gui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"script-manager/internal/applog"
	"script-manager/internal/uiprefs"
	"script-manager/internal/update"
	"script-manager/internal/version"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type UpdateInfoDTO struct {
	Available bool   `json:"available"`
	Current   string `json:"current"`
	Latest    string `json:"latest"`
}

func (a *App) updateClient() *update.Client {
	return update.NewClient(version.Version, runtime.GOOS, runtime.GOARCH)
}

func (a *App) check() (update.Result, error) {
	res, err := a.updateClient().Check(context.Background())
	if err == nil {
		a.updateMu.Lock()
		a.updateResult = res
		a.updateMu.Unlock()
	}
	return res, err
}

// CheckForUpdateOnStartup returns the newer release, or nil when there is
// none, the check is off, or it failed. Failures are only logged.
func (a *App) CheckForUpdateOnStartup() *UpdateInfoDTO {
	if !uiprefs.Load(a.appDataDir).CheckForUpdates || !update.Checkable(version.Version) {
		return nil
	}
	res, err := a.check()
	if err != nil {
		applog.Printf("update check: %v", err)
		return nil
	}
	if !res.Available {
		return nil
	}
	return &UpdateInfoDTO{Available: true, Current: res.Current, Latest: res.Latest}
}

func (a *App) CheckForUpdate() (UpdateInfoDTO, error) {
	res, err := a.check()
	if err != nil {
		applog.Printf("update check: %v", err)
		return UpdateInfoDTO{}, err
	}
	return UpdateInfoDTO{Available: res.Available, Current: res.Current, Latest: res.Latest}, nil
}

// InstallUpdate installs the release found by the last check, then starts
// the new build with this one's arguments and quits.
func (a *App) InstallUpdate() error {
	if !a.installMu.TryLock() {
		return errors.New("an update is already being installed")
	}
	defer a.installMu.Unlock()

	if a.exePath == "" {
		return errors.New("can't locate the running executable")
	}
	a.updateMu.Lock()
	res := a.updateResult
	a.updateMu.Unlock()
	if !res.Available {
		var err error
		if res, err = a.check(); err != nil {
			return err
		}
	}

	if err := a.updateClient().Install(context.Background(), res, a.exeDir); err != nil {
		applog.Printf("update install: %v", err)
		return err
	}
	applog.Printf("update installed: %s", res.Latest)
	if err := update.Start(filepath.Join(a.exeDir, filepath.Base(a.exePath)), os.Args[1:]); err != nil {
		applog.Printf("update restart: %v", err)
		return errors.New("the update was installed, but the app couldn't restart; start it again yourself")
	}
	wailsruntime.Quit(a.ctx)
	return nil
}

func (a *App) SetCheckForUpdates(on bool) uiprefs.Prefs {
	p := uiprefs.Load(a.appDataDir)
	p.CheckForUpdates = on
	saved, _ := uiprefs.Save(a.appDataDir, p)
	return saved
}
