package reorder

import (
	"github.com/gotk3/gotk3/gtk"

	"hypr-dock/internal/item"
	"hypr-dock/internal/state"
	"hypr-dock/internal/terminal"
)

// All re-derives the dock's icon order from the unified model: launcher (if
// start-anchored), pinned apps in pinned order, then running unpinned apps in
// their current relative order. Called after pin/unpin so a pinned running
// app slides into the pinned section without duplication.
func All(appState *state.State) {
	box := appState.GetItemsBox()
	list := appState.GetList().GetMap()
	if box == nil {
		return
	}

	// Snapshot current children, keyed by class where known.
	type entry struct {
		widget *gtk.Widget
		class  string
	}

	children := make([]entry, 0, len(list)+1)
	for l := box.GetChildren(); l != nil && l.Data() != nil; l = l.Next() {
		if w, ok := l.Data().(gtk.IWidget); ok {
			native := w.ToWidget().Native()
			class := ""
			for className, it := range list {
				if it.ButtonBox != nil && it.ButtonBox.ToWidget().Native() == native {
					class = className
					break
				}
			}
			children = append(children, entry{widget: w.ToWidget(), class: class})
		}
	}

	settings := appState.GetSettings()

	// Relative order of unpinned items, from the snapshot.
	unpinnedOrder := make([]string, 0, len(list))
	for _, child := range children {
		if child.class != "" && !isPinned(appState, child.class) {
			unpinnedOrder = append(unpinnedOrder, child.class)
		}
	}

	// Final order: launcher (start position) + pinned + running unpinned.
	final := make([]string, 0, len(list)+1)

	launcherStart := settings.LauncherPosition == "start" && settings.ShowLauncherButton
	if launcherStart {
		final = append(final, item.LauncherName)
	}

	for _, className := range *appState.GetPinned() {
		if it := list[className]; it != nil && !settings.IsHidden(className) {
			final = append(final, className)
		}
	}

	// Terminal emulator pins are represented by one stable synthetic item.
	if list[terminal.GroupClass] != nil {
		final = append(final, terminal.GroupClass)
	}

	final = append(final, unpinnedOrder...)

	if !launcherStart && settings.ShowLauncherButton {
		final = append(final, item.LauncherName)
	}

	// Apply the order. Launcher children are the ones with no class match.
	target := 0
	for _, className := range final {
		var widget *gtk.Widget

		if className == item.LauncherName {
			for _, child := range children {
				if child.class == "" {
					widget = child.widget
					break
				}
			}
		} else if it := list[className]; it != nil && it.ButtonBox != nil {
			widget = it.ButtonBox.ToWidget()
		}

		if widget == nil {
			continue
		}

		box.ReorderChild(widget, target)
		target++
	}
}

func isPinned(appState *state.State, className string) bool {
	for _, pinned := range *appState.GetPinned() {
		if pinned == className {
			return true
		}
	}
	return false
}
