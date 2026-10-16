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
	// Installable is false when the release has no checksums; it can only
	// be installed by hand from ReleaseURL.
	Installable bool   `json:"installable"`
	ReleaseURL  string `json:"releaseUrl"`
}

func toDTO(res update.Result) UpdateInfoDTO {
	return UpdateInfoDTO{
		Available:   res.Available,
		Current:     res.Current,
		Latest:      res.Latest,
		Installable: res.Installable,
		ReleaseURL:  res.ReleaseURL,
	}
}

func (a *App) updateClient() *update.Client {
	c := update.NewClient(version.Version, runtime.GOOS, runtime.GOARCH)
	if o := c.Override(); o != "" {
		applog.Printf("update source overridden by %s: %s", update.EnvBaseURL, o)
	}
	return c
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

// CanCheckForUpdates is false for builds without a release version.
func (a *App) CanCheckForUpdates() bool {
	return update.Checkable(version.Version)
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
	dto := toDTO(res)
	return &dto
}

func (a *App) CheckForUpdate() (UpdateInfoDTO, error) {
	res, err := a.check()
	if err != nil {
		applog.Printf("update check: %v", err)
		return UpdateInfoDTO{}, err
	}
	return toDTO(res), nil
}

// InstallUpdate installs the release found by the last check, then starts
// the new build with this one's arguments and quits.
func (a *App) InstallUpdate() error {
	if !a.installMu.TryLock() {
		return errors.New("an update is already being installed")
	}
	defer a.installMu.Unlock()

	if a.exePath == "" || a.exeDir == "" {
		return errors.New("can't locate the running executable")
	}
	exe := filepath.Join(a.exeDir, filepath.Base(a.exePath))
	a.updateMu.Lock()
	res := a.updateResult
	a.updateMu.Unlock()
	if !res.Available {
		var err error
		if res, err = a.check(); err != nil {
			return err
		}
	}

	if err := a.updateClient().Install(context.Background(), res, exe); err != nil {
		applog.Printf("update install: %v", err)
		return err
	}
	applog.Printf("update installed: %s", res.Latest)
	if err := update.Start(exe, os.Args[1:]); err != nil {
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
