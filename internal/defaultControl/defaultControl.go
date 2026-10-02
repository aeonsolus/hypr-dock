package defaultcontrol

import (
	"hypr-dock/internal/item"
	"hypr-dock/internal/pkg/utils"
	"hypr-dock/internal/settings"
	"hypr-dock/pkg/ipc"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/hashicorp/go-hclog"
)

type Control struct {
	item     *item.Item
	settings *settings.Settings
	log      hclog.Logger

	// previewMode reroutes running-app clicks to the preview wiring (single
	// → focus, multi → popup preview) instead of the plain click actions.
	previewMode bool

	zeroHandler   func()
	singleHandler func()
	multiHandler  func(onContextClose func())

	onContextOpen  func()
	onContextClose func()
}

func New(item *item.Item, settings *settings.Settings, log hclog.Logger) *Control {
	zeroHandler := func() {
		item.App.Run()
	}

	singleHandler := func() {
		client, ok := utils.GetFocusedValue(item.Windows)
		if ok {
			ipc.FocusWindow(client.Address)
		}
	}

	multiHandler := func(onContextClose func()) {
		menu, err := item.WindowsMenu()
		if err != nil {
			log.Error("Unable to create windows menu", "error", err)
			return
		}

		win, zone, err := getActivateZone(item.Button, settings.ContextPos, settings.Position)
		if err != nil {
			log.Error("Failed to get activate zone", "error", err)
			return
		}

		firstg, secondg := getGravity(settings.Position)
		menu.PopupAtRect(win, zone, firstg, secondg, nil)
		menu.Connect("deactivate", func() {
			item.Button.SetStateFlags(gtk.STATE_FLAG_NORMAL, true)
			if onContextClose != nil {
				onContextClose()
			}
		})
	}

	return &Control{
		item:     item,
		settings: settings,
		log:      log,

		zeroHandler:   zeroHandler,
		singleHandler: singleHandler,
		multiHandler:  multiHandler,
	}
}

// showWindows pops the per-window menu (used by "show windows" actions).
func (c *Control) showWindows() {
	c.multiHandler(c.onContextClose)
}

// focusItem restores minimized windows into their original workspaces and
// focuses the most recently focused one.
func (c *Control) focusItem() {
	i := c.item

	for address, workspace := range i.MinimizeRestore {
		ipc.MoveWindowToWorkspace(address, workspace)
		delete(i.MinimizeRestore, address)
	}

	client, ok := utils.GetFocusedValue(i.Windows)
	if !ok {
		return
	}
	ipc.FocusWindow(client.Address)
}

// minimizeItem parks every window of the app in the special minimize
// workspace, remembering where each window came from for restoration.
func (c *Control) minimizeItem() {
	i := c.item

	for address, client := range i.Windows {
		if client.Workspace.Id < 1 {
			continue
		}
		if err := ipc.MinimizeWindow(address); err != nil {
			c.log.Warn("Unable to minimize window", "address", address, "error", err)
			continue
		}
		i.MinimizeRestore[address] = client.Workspace.Id
	}
	_ = ipc.FocusCurrentOrLast()
}

func (c *Control) actuallyFocused() bool {
	address, err := ipc.ActiveAddress()
	if err != nil || address == "" {
		return false
	}
	_, ok := c.item.Windows[address]
	return ok
}

// applyRunning handles clicks on running-but-unfocused applications.
func (c *Control) applyRunning() {
	switch c.settings.ClickAction {
	case "launch":
		c.item.App.Run()
	case "minimize", "focus":
		c.focusItem()
	case "show":
		c.showWindows()
	case "cycle":
		ipc.CycleWindows()
	case "none":
	}
}

// applyFocused handles clicks on the focused application.
func (c *Control) applyFocused() {
	switch c.settings.ClickActionFocused {
	case "minimize":
		c.minimizeItem()
	case "show":
		c.showWindows()
	case "cycle":
		ipc.CycleWindows()
	case "none":
	}
}

func (c *Control) Init() {
	c.connectContextMenu()

	leftClick(c.item.Button, func(e *gdk.Event) {
		if item.DragActive() {
			return
		}

		if c.item.IsTerminalGroup() {
			c.multiHandler(c.onContextClose)
			return
		}

		instances := len(c.item.Windows)

		if instances == 0 {
			c.zeroHandler()
			return
		}

		if c.previewMode {
			if instances == 1 {
				if c.singleHandler != nil {
					c.singleHandler()
				} else {
					c.focusItem()
				}
				return
			}
			c.multiHandler(c.onContextClose)
			return
		}

		if c.actuallyFocused() {
			c.applyFocused()
			return
		}

		c.applyRunning()
	})

	c.connectMiddleClick()
}

// SetPreviewMode routes running-app clicks through the preview handlers.
func (c *Control) SetPreviewMode(enabled bool) {
	c.previewMode = enabled
}

// connectMiddleClick wires the configurable middle-click action
// (default: launch a new instance).
func (c *Control) connectMiddleClick() {
	c.item.Button.Connect("button-release-event", func(_ *gtk.Button, e *gdk.Event) {
		event := gdk.EventButtonNewFromEvent(e)
		if event.Button() != 2 {
			return
		}

		if item.DragActive() {
			return
		}

		if c.item.IsTerminalGroup() {
			return
		}

		switch c.settings.MiddleClickAction {
		case "launch":
			if len(c.item.Windows) != 0 && c.item.App.GetSingleWindow() {
				// Single-window apps refuse extra instances.
				return
			}
			c.item.App.Run()
		}
	})
}

func (c *Control) ResetZero(newHandler func()) {
	c.zeroHandler = newHandler
}

func (c *Control) ResetSingle(newHandler func()) {
	c.singleHandler = newHandler
}

func (c *Control) ResetMulti(newHandler func()) {
	c.multiHandler = func(onContextClose func()) {
		newHandler()
	}
}

func (c *Control) OnContextOpen(handler func()) {
	c.onContextOpen = handler
}

func (c *Control) OnContextClose(handler func()) {
	c.onContextClose = handler
}

func (c *Control) connectContextMenu() {
	c.item.Button.Connect("button-release-event", func(button *gtk.Button, e *gdk.Event) {
		event := gdk.EventButtonNewFromEvent(e)
		if event.Button() == 3 {
			menu, err := c.item.ContextMenu()
			if err != nil {
				c.log.Error("Unable to create context menu", "error", err)
				return
			}

			win, zone, err := getActivateZone(c.item.Button, c.settings.ContextPos, c.settings.Position)
			if err != nil {
				c.log.Error("Failed to get activate zone", "error", err)
				return
			}

			firstg, secondg := getGravity(c.settings.Position)
			menu.PopupAtRect(win, zone, firstg, secondg, nil)

			if c.onContextOpen != nil {
				c.onContextOpen()
			}

			menu.Connect("deactivate", func() {
				c.item.Button.SetStateFlags(gtk.STATE_FLAG_NORMAL, true)
				if c.onContextClose != nil {
					c.onContextClose()
				}
			})

			return
		}
	})
}
