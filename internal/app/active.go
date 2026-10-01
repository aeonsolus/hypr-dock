package app

import (
	"strings"

	"hypr-dock/internal/item"
	"hypr-dock/internal/pkg/utils"
	"hypr-dock/internal/state"
	"hypr-dock/pkg/ipc"
)

// UpdateActive marks the item owning `address` as focused and reflects it in
// the widget style ("active" class), enabling the macOS-like active-tint.
func UpdateActive(appState *state.State, address string) {
	appState.SetActiveAddress(address)

	list := appState.GetList().GetMap()
	for _, it := range list {
		active := false
		if address != "" {
			if _, ok := it.Windows[address]; ok {
				active = true
			}
		}

		if it.Active != active {
			it.Active = active
			applyActiveClass(it)
		}
	}
}

func applyActiveClass(it *item.Item) {
	if it == nil || it.Button == nil {
		return
	}

	context, err := it.Button.GetStyleContext()
	if err != nil {
		return
	}

	if it.Active {
		context.AddClass("active")
	} else {
		context.RemoveClass("active")
	}
}

// FocusFromClick focuses the app owning the given window address and restores
// any minimized windows of that app first. Used by click actions that received
// a direct address (e.g. preview click).
func FocusFromClick(appState *state.State, address string) {
	if address == "" {
		return
	}

	found, _, err := appState.GetList().SearchWindow(address)
	if err != nil {
		return
	}

	for restoreAddress, workspace := range found.MinimizeRestore {
		ipc.MoveWindowToWorkspace(restoreAddress, workspace)
		delete(found.MinimizeRestore, restoreAddress)
	}

	ipc.FocusWindow(address)
}

// RefreshFromClients resynchronizes every item's window set with Hyprland's
// current client list. Used after config reloads and focus/monitor events so
// the model can't drift from reality.
func RefreshFromClients(appState *state.State) {
	clients, err := ipc.GetClients()
	if err != nil {
		return
	}

	list := appState.GetList().GetMap()

	// Current addresses per item.
	current := make(map[string]map[string]*ipc.Client, len(list))
	for className := range list {
		current[className] = map[string]*ipc.Client{}
	}

	for _, client := range clients {
		item, _ := clientToItem(client, appState)
		if item == nil {
			continue
		}
		c := client
		current[item.ClassName][client.Address] = &c
	}

	for className, it := range list {
		windows := current[className]

		for address := range it.Windows {
			if _, ok := windows[address]; !ok {
				delete(it.Windows, address)
			}
		}
		for address, client := range windows {
			if _, ok := it.Windows[address]; !ok {
				it.Windows[address] = client
			}
		}

		it.RefreshIndicator()

		// Drop the item when it is unpinned and no longer running.
		if len(it.Windows) == 0 && !it.IsPinned() && !it.IsTerminalGroup() {
			it.Remove()
		}
	}

	UpdateActive(appState, appState.GetActiveAddress())
}

// clientToItem maps a Hyprland client to its dock item, creating nothing.
// Class keys dominate; title-normalized classes are the last resort for
// classless windows.
func clientToItem(client ipc.Client, appState *state.State) (*item.Item, bool) {
	list := appState.GetList()

	candidates := []string{client.Class, client.InitialClass}
	if normalized := utils.NormaliseTitle(client.InitialTitle); normalized != "" {
		candidates = append(candidates, normalized)
	}

	for _, className := range candidates {
		className = strings.TrimSpace(className)
		if className == "" {
			continue
		}
		if it := list.Get(className); it != nil {
			return it, true
		}
	}

	return nil, false
}
