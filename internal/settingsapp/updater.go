package settingsapp

import (
	"context"
	"time"

	"github.com/gotk3/gotk3/glib"
	"hypr-dock/internal/updater"
	"hypr-dock/internal/version"
)

func (a *App) refreshUpdater() {
	if a.updateStatus == nil {
		return
	}
	if a.installedVersion == "" {
		a.installedVersion = version.Version
	}
	message := a.updateMessage
	if message == "" {
		message = "Updates are checked automatically when preferences opens."
	}
	a.updateStatus.SetText("Installed: " + a.installedVersion + "\n" + message)
	a.updateCheck.SetSensitive(!a.checkingUpdate && !a.updating)
	available := a.updateRelease != nil && a.updateRelease.Available
	a.updateButton.SetSensitive(available && !a.checkingUpdate && !a.updating)
	if available {
		a.updateButton.SetLabel("Install " + a.updateRelease.Version)
	} else {
		a.updateButton.SetLabel("Install update")
	}
}

func (a *App) checkUpdates() {
	if a.checkingUpdate || a.updating {
		return
	}
	if a.installedVersion == "" {
		a.installedVersion = version.Version
	}
	installed := a.installedVersion
	a.checkingUpdate = true
	a.updateMessage = "Checking GitHub main…"
	a.refreshUpdater()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		release, err := updater.Check(ctx, installed)
		glib.IdleAdd(func() {
			a.checkingUpdate = false
			a.updateRelease = release
			if err != nil {
				a.updateMessage = "Could not check for updates. Verify gh authentication and connectivity.\n" + err.Error()
			} else if release.Available {
				a.updateMessage = "New version available: " + release.Version
			} else {
				a.updateMessage = "Up to date. GitHub main: " + release.Version
			}
			a.refreshUpdater()
		})
	}()
}

func (a *App) installUpdate() {
	if a.updating || a.checkingUpdate || a.updateRelease == nil || !a.updateRelease.Available {
		return
	}
	release := *a.updateRelease
	installed := a.installedVersion
	a.updating = true
	a.updateMessage = "Downloading and verifying binaries for " + release.Version + "…"
	a.refreshUpdater()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		// Recheck at install time so a release published after the last poll
		// is downloaded instead of an older notification's snapshot.
		latest, err := updater.Check(ctx, installed)
		if err == nil {
			release = *latest
			if release.Available {
				err = updater.Install(ctx, release)
			}
		}
		glib.IdleAdd(func() {
			a.updating = false
			if err != nil {
				a.updateMessage = "Update failed.\n" + err.Error()
				a.SetStatus("Update failed")
			} else if !release.Available {
				a.updateRelease = nil
				a.updateMessage = "No newer version is available. GitHub main: " + release.Version
			} else {
				a.installedVersion = release.Version
				a.updateRelease = nil
				a.updateMessage = "Installed successfully. Restart the dock and reopen preferences to use the new version."
				a.SetStatus("Update installed")
			}
			a.refreshUpdater()
		})
	}()
}
