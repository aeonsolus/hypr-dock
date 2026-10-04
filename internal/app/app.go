package app

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	"hypr-dock/internal/btnctl"
	"hypr-dock/internal/item"
	"hypr-dock/internal/pkg/utils"
	"hypr-dock/internal/reorder"
	"hypr-dock/internal/state"
	"hypr-dock/internal/terminal"
	"hypr-dock/pkg/ipc"
)

func BuildApp(appState *state.State) *gtk.Box {
	settings := appState.GetSettings()
	orientation := appState.GetLayerctl().GetOrientation()
	log := appState.GetLogger()

	app, err := gtk.BoxNew(orientation, 0)
	if err != nil {
		log.Error("Unable to create gtk box:", "package", "app", "err", err)
		os.Exit(2)
	}

	app.SetName("app")
	appState.SetAppBox(app)

	initMarginSafe(appState)

	itemsBox, _ := gtk.BoxNew(orientation, settings.Spacing)
	itemsBox.SetName("items-box")

	switch orientation {
	case gtk.ORIENTATION_HORIZONTAL:
		itemsBox.SetMarginEnd(int(float64(settings.Spacing) * 0.8))
		itemsBox.SetMarginStart(int(float64(settings.Spacing) * 0.8))
	case gtk.ORIENTATION_VERTICAL:
		itemsBox.SetMarginBottom(int(float64(settings.Spacing) * 0.8))
		itemsBox.SetMarginTop(int(float64(settings.Spacing) * 0.8))
	}

	appState.SetItemsBox(itemsBox)
	item.InitDrag(itemsBox)
	buildLauncher(appState)
	buildTrash(appState)
	renderItems(appState)
	app.Add(itemsBox)

	return app
}

func buildTrash(appState *state.State) {
	list := appState.GetList()
	if list.Get(item.TrashName) != nil {
		return
	}
	trash, err := item.NewTrash(appState.GetSettings(), appState.GetLogger())
	if err != nil {
		appState.GetLogger().Error("Unable to create recycle bin", "error", err)
		return
	}
	trash.List = list.GetMap()
	trash.PinnedList = appState.GetPinned()
	btnctl.Dispatch(trash, appState)
	trash.AttachDrag()
	list.Add(item.TrashName, trash)
	appState.GetItemsBox().Add(trash.ButtonBox)
	go watchTrash(trash)
}

func watchTrash(trash *item.Item) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	filesDir := filepath.Join(home, ".local", "share", "Trash", "files")
	lastFull := false
	for {
		entries, _ := os.ReadDir(filesDir)
		full := len(entries) > 0
		if full != lastFull {
			lastFull = full
			icon := "user-trash"
			if full {
				icon = "user-trash-full"
			}
			glib.IdleAdd(func() { trash.SetIcon(icon) })
		}
		time.Sleep(time.Second)
	}
}

func InitTerminalGroup(appState *state.State) {
	list := appState.GetList()
	if list.Get(terminal.GroupClass) != nil {
		return
	}

	terminalItem, err := item.NewTerminalGroup(appState.GetSettings(), appState.GetLogger())
	if err != nil {
		appState.GetLogger().Error("Unable to create terminal group", "error", err)
		return
	}

	terminalItem.List = list.GetMap()
	terminalItem.PinnedList = appState.GetPinned()
	btnctl.Dispatch(terminalItem, appState)
	terminalItem.AttachDrag()
	list.Add(terminal.GroupClass, terminalItem)
	appState.GetItemsBox().Add(terminalItem.ButtonBox)
}

func renderItems(appState *state.State) {
	settings := appState.GetSettings()
	clients, _ := ipc.GetClients()

	InitTerminalGroup(appState)

	if settings.ShowPinnedApps {
		for _, className := range *appState.GetPinned() {
			// The recycle bin is a synthetic item, never a desktop-app pin.
			if className == item.TrashName {
				continue
			}
			// All terminal pins collapse into the synthetic Terminal Apps item.
			if terminal.IsTerminalClass(className) {
				continue
			}
			if settings.IsHidden(className) {
				continue
			}
			InitNewItemInClass(className, appState)
		}
	}

	if settings.ShowRunningApps {
		for _, ipcClient := range clients {
			InitNewItemInIPC(ipcClient, appState)
		}
	}

	reorder.All(appState)

	ipc.DispatchEvent("hd>>dock-render-finish")
}

func InitNewItemInIPC(ipcClient ipc.Client, appState *state.State) {
	list := appState.GetList()

	if terminal.IsTerminalClient(ipcClient) {
		InitTerminalGroup(appState)
		if group := list.Get(terminal.GroupClass); group != nil {
			group.AddWindow(ipcClient)
			appState.GetWindow().ShowAll()
		}
		return
	}

	className := ipcClient.Class

	if className == "" {
		className = utils.NormaliseTitle(ipcClient.InitialTitle)
	}
	if className == item.TrashName {
		return
	}

	// Blacklisted classes (tray/background apps) never get a dock icon.
	if appState.GetSettings().IsHidden(className) {
		return
	}

	pin := slices.Contains(*appState.GetPinned(), className)
	added := list.Get(className) != nil

	title := strings.TrimSpace(ipcClient.Title)
	if title == "" {
		title = strings.TrimSpace(ipcClient.InitialTitle)
	}

	if !pin && !added {
		InitNewItemInClassWithTitle(className, title, appState)
	}

	list.Get(className).AddWindow(ipcClient)
	appState.GetWindow().ShowAll()
}

func InitNewItemInClass(className string, appState *state.State) {
	InitNewItemInClassWithTitle(className, "", appState)
}

func InitNewItemInClassWithTitle(className, windowTitle string, appState *state.State) {
	log := appState.GetLogger()

	list := appState.GetList()
	item, err := item.NewWithTitle(className, windowTitle, appState.GetSettings(), appState.GetLogger())
	if err != nil {
		log.Error("Unable to creat app item", "err", err)
		return
	}

	btnctl.Dispatch(item, appState)

	item.AttachDrag()

	item.List = list.GetMap()
	item.PinnedList = appState.GetPinned()
	list.Add(className, item)

	appState.GetItemsBox().Add(item.ButtonBox)
	appState.GetWindow().ShowAll()
	// New running items are appended by GTK; restore the persistent pinned
	// prefix immediately so unpinned apps cannot displace pinned slots.
	reorder.All(appState)
}

func RemoveApp(address string, appState *state.State) {
	item, _, err := appState.GetList().SearchWindow(address)
	if err != nil {
		return
	}

	if item.IsTerminalGroup() {
		item.RemoveWindow(address)
		appState.GetWindow().ShowAll()
		return
	}

	lastWindow := len(item.Windows) == 1
	pin := slices.Contains(*appState.GetPinned(), item.ClassName)

	if lastWindow && !pin {
		item.Remove()
		reorder.All(appState)
		return
	}

	item.RemoveWindow(address)

	appState.GetWindow().ShowAll()
	reorder.All(appState)
}

func ChangeWindowTitle(address string, title string, appState *state.State) {
	_, client, err := appState.GetList().SearchWindow(address)
	if err != nil {
		return
	}

	client.Title = title
}

func setMargin(app *gtk.Box, position string, margin ...int) {
	if len(margin) == 1 {
		switch position {
		case "bottom":
			app.SetMarginBottom(margin[0])
		case "left":
			app.SetMarginStart(margin[0])
		case "right":
			app.SetMarginEnd(margin[0])
		case "top":
			app.SetMarginTop(margin[0])
		}
	}

	if len(margin) == 4 {
		switch position {
		case "top":
			app.SetMarginTop(margin[0])
		case "right":
			app.SetMarginEnd(margin[1])
		case "bottom":
			app.SetMarginBottom(margin[2])
		case "left":
			app.SetMarginStart(margin[3])
		}
	}
}
