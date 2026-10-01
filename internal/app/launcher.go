package app

import (
	"hypr-dock/internal/item"
	"hypr-dock/internal/state"
)

// buildLauncher adds the launcher button when enabled. The button lives at
// the start (or end) of the items box, before pinned apps; it is not part of
// the application list and is skipped by drag persistence.
func buildLauncher(appState *state.State) {
	settings := appState.GetSettings()
	if !settings.ShowLauncherButton {
		return
	}

	launcher, err := item.NewLauncher(settings, appState.GetLogger())
	if err != nil {
		appState.GetLogger().Error("Unable to create launcher item", "error", err)
		return
	}

	itemsBox := appState.GetItemsBox()
	if settings.LauncherPosition == "end" {
		itemsBox.PackEnd(launcher.ButtonBox, false, false, 0)
	} else {
		itemsBox.PackStart(launcher.ButtonBox, false, false, 0)
	}
}
