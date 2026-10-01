package item

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/hashicorp/go-hclog"

	"hypr-dock/internal/desktop"
	"hypr-dock/internal/pkg/utils"
	"hypr-dock/pkg/ipc"
)

func (i *Item) WindowsMenu() (*gtk.Menu, error) {
	menu, err := gtk.MenuNew()
	if err != nil {
		return nil, err
	}

	AddWindowsItemToMenu(menu, i.Windows, i.App, i.log)

	menu.SetName("windows-menu")
	menu.ShowAll()

	return menu, nil
}

func (i *Item) ContextMenu() (*gtk.Menu, error) {
	menu, err := gtk.MenuNew()
	if err != nil {
		return nil, err
	}

	app := i.App
	actions := app.GetActions()
	running := len(i.Windows) != 0

	// Launch — closed apps and multi-instance apps get "New Window".
	if !i.IsTerminalGroup() {
		launchMenuItem, err := BuildLaunchMenuItem(i)
		if err == nil {
			menu.Append(launchMenuItem)
		} else {
			i.log.Error("Unable to create launch menu item", "error", err)
		}
	}

	// Windows — individual windows live in a submenu so multi-window apps
	// stay a single dock icon.
	if running && len(i.Windows) > 1 {
		windowsItem, err := BuildWindowsSubmenu(i)
		if err == nil {
			menu.Append(windowsItem)
		} else {
			i.log.Error("Unable to create windows submenu", "error", err)
		}
	} else if running {
		// Single window: keep it directly in the menu (fast focus path).
		AddWindowsItemToMenu(menu, i.Windows, app, i.log)
	}

	if !i.IsTerminalGroup() {
		pinMenuItem, err := BuildPinMenuItem(i)
		if err == nil {
			menu.Append(pinMenuItem)
		} else {
			i.log.Error("Unable to create pin menu item", "error", err)
		}
	}

	if running {
		if len(i.Windows) == 1 {
			client, ok := utils.GetSingleValue(i.Windows)
			if ok {
				closeMenuItem, err := BuildContextItem("Close", func() {
					ipc.CloseWindow(client.Address)
				}, "close-symbolic")
				if err == nil {
					menu.Append(closeMenuItem)
				} else {
					i.log.Error("Unable to create close menu item", "error", err)
				}
			}
		} else {
			closeAllMenuItem, err := BuildContextItem("Close All", func() {
				for _, client := range i.Windows {
					go ipc.CloseWindow(client.Address)
				}
			}, "close-symbolic")
			if err == nil {
				menu.Append(closeAllMenuItem)
			} else {
				i.log.Error("Unable to create close-all menu item", "error", err)
			}
		}
	}

	if len(actions) > 0 {
		separator, err := gtk.SeparatorMenuItemNew()
		if err == nil {
			menu.Append(separator)
		} else {
			i.log.Error("Unable to create gtk separator", "error", err)
		}

		for _, action := range actions {
			exec := func() {
				action.Run()
			}

			var actionMenuItem *gtk.MenuItem
			var err error

			if action.GetIcon() == "" {
				actionMenuItem, err = BuildContextItem(action.GetName(), exec)
			} else {
				actionMenuItem, err = BuildContextItem(action.GetName(), exec, action.GetIcon())
			}

			if err == nil {
				menu.Append(actionMenuItem)
			} else {
				i.log.Error("Unable to create context item", "error", err)
			}
		}
	}

	if running {
		separator, err := gtk.SeparatorMenuItemNew()
		if err == nil {
			menu.Append(separator)
		}

		terminateMenuItem, err := BuildContextItem("Terminate Process", func() {
			i.TerminateProcesses()
		}, "process-stop-symbolic")
		if err == nil {
			menu.Append(terminateMenuItem)
		} else {
			i.log.Error("Unable to create terminate process menu item", "error", err)
		}
	}

	// Always reachable, independent of app state.
	separator, err := gtk.SeparatorMenuItemNew()
	if err == nil {
		menu.Append(separator)
	}

	settingsMenuItem, err := BuildContextItem("Dock Settings…", func() {
		go OpenDockSettings(i.log)
	}, "preferences-desktop-symbolic")
	if err == nil {
		menu.Append(settingsMenuItem)
	} else {
		i.log.Error("Unable to create settings menu item", "error", err)
	}

	menu.SetName("context-menu")
	menu.ShowAll()

	return menu, nil
}

// OpenDockSettings launches the settings application. The binary is searched
// next to the dock executable first (same install), then on PATH.
func OpenDockSettings(log hclog.Logger) {
	candidates := []string{"hypr-dock-settings"}

	if exe, err := os.Executable(); err == nil {
		candidates = append([]string{filepath.Join(filepath.Dir(exe), "hypr-dock-settings")}, candidates...)
	}

	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate); err == nil {
			if err := exec.Command(candidate).Start(); err != nil {
				log.Error("Unable to start settings", "binary", candidate, "error", err)
			}
			return
		}
	}

	log.Error("hypr-dock-settings binary not found")
}

// BuildWindowsSubmenu nests the window list under a "Windows" item.
func BuildWindowsSubmenu(i *Item) (*gtk.MenuItem, error) {
	item, err := gtk.MenuItemNew()
	if err != nil {
		return nil, err
	}

	item.SetName("menu-item")
	item.SetLabel("Windows")

	submenu, err := gtk.MenuNew()
	if err != nil {
		return nil, err
	}

	AddWindowsItemToMenu(submenu, i.Windows, i.App, i.log)
	item.SetSubmenu(submenu)

	return item, nil
}

func (i *Item) TerminateProcesses() {
	clients, err := ipc.GetClients()
	if err != nil {
		i.log.Error("Unable to refresh windows before terminating process", "error", err)
		return
	}

	currentByAddress := make(map[string]ipc.Client, len(clients))
	for _, client := range clients {
		currentByAddress[client.Address] = client
	}

	pids := make(map[int]struct{})
	for address := range i.Windows {
		client, ok := currentByAddress[address]
		if !ok || client.Pid <= 1 || client.Pid == os.Getpid() {
			continue
		}
		pids[client.Pid] = struct{}{}
	}

	for pid := range pids {
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			i.log.Error("Unable to terminate process", "pid", pid, "error", err)
			continue
		}
		i.log.Info("Terminate process requested", "pid", pid, "class", i.ClassName)
	}
}

func AddWindowsItemToMenu(menu *gtk.Menu, windows map[string]*ipc.Client, app *desktop.App, log hclog.Logger) {
	for _, window := range windows {
		menuItem, err := BuildContextItem(window.Title, func() {
			go ipc.FocusWindow(window.Address)
		}, app.GetIcon())

		if err != nil {
			log.Error("Unable to create launch menu item", "error", err)
			continue
		}

		menu.Append(menuItem)
	}
}

func BuildLaunchMenuItem(item *Item) (*gtk.MenuItem, error) {
	app := item.App

	instances := len(item.Windows)

	if instances != 0 && app.GetSingleWindow() {
		return nil, errors.New("")
	}

	labelText := app.GetName()
	if instances != 0 {
		labelText = "New Window - " + labelText
	}

	launchMenuItem, err := BuildContextItem(labelText, func() {
		app.Run()
	}, app.GetIcon())

	if err != nil {
		return nil, err
	}

	return launchMenuItem, nil
}

func BuildPinMenuItem(item *Item) (*gtk.MenuItem, error) {
	labelText := "Pin"
	if item.IsPinned() {
		labelText = "Unpin"
	}

	menuItem, err := BuildContextItem(labelText, func() {
		item.TogglePin()
	})

	if err != nil {
		return nil, err
	}

	return menuItem, nil
}

func BuildContextItem(labelText string, connectFunc func(), iconName ...string) (*gtk.MenuItem, error) {
	size := 16
	spacing := 6

	menuItem, err := gtk.MenuItemNew()
	if err != nil {
		return nil, err
	}

	menuItem.SetName("menu-item")

	hbox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, spacing)
	if err != nil {
		return nil, err
	}

	hbox.SetName("hbox")
	/* Hack (HELP ME)*/
	/* stackoverflow.com/questions/48452717/how-to-replace-the-deprecated-gtk3-gtkimagemenuitem */
	utils.AddStyle(hbox, fmt.Sprintf("#hbox {margin-left: %dpx;}", 0-(size+spacing)))

	label, err := gtk.LabelNew(labelText)
	if err != nil {
		return nil, err
	}

	label.SetEllipsize(pango.ELLIPSIZE_END)
	label.SetMaxWidthChars(30)

	if len(iconName) > 0 {
		icon, err := utils.CreateImage(iconName[0], size)
		if err == nil {
			hbox.Add(icon)
		}
	} else {
		label.SetMarginStart(size + spacing)
	}

	if connectFunc != nil {
		menuItem.Connect("activate", func() {
			connectFunc()
		})
	}

	hbox.Add(label)
	menuItem.SetReserveIndicator(false)
	menuItem.Add(hbox)

	return menuItem, nil
}
