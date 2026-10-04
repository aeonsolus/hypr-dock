package item

import (
	"log"
	"slices"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/hashicorp/go-hclog"

	"hypr-dock/internal/desktop"
	layerinfo "hypr-dock/internal/layerInfo"

	"hypr-dock/internal/pkg/indicator"
	"hypr-dock/internal/pkg/pinned"
	"hypr-dock/internal/pkg/utils"
	"hypr-dock/internal/settings"
	"hypr-dock/internal/terminal"

	"hypr-dock/pkg/ipc"
)

type Item struct {
	Windows        map[string]*ipc.Client
	App            *desktop.App
	ClassName      string
	Button         *gtk.Button
	ButtonBox      *gtk.Box
	IndicatorImage *gtk.Image

	Settings   *settings.Settings
	List       map[string]*Item
	PinnedList *[]string

	// Active tracks whether one of the app's windows currently has focus.
	Active bool

	// OnPinChange is invoked after pin state changes so the dock can re-derive
	// icon order (pinned section first). Set by the app layer.
	OnPinChange func()

	// MinimizeRestore remembers the originating workspace of windows parked
	// in the special minimize workspace, so a click restores them where they
	// lived instead of leaving the user stranded.
	MinimizeRestore map[string]int

	log hclog.Logger
}

func New(className string, settings *settings.Settings, log hclog.Logger) (*Item, error) {
	return NewWithTitle(className, "", settings, log)
}

func NewWithTitle(className, windowTitle string, settings *settings.Settings, log hclog.Logger) (*Item, error) {
	app, err := desktop.NewWithTitle(className, windowTitle)
	if err != nil {
		log.Debug("Desktop metadata unavailable; using fallback", "className", className, "error", err)
	}
	return NewWithApp(className, app, settings, log)
}

func NewTerminalGroup(settings *settings.Settings, log hclog.Logger) (*Item, error) {
	app := desktop.NewVirtual("Terminal Apps", "terminal")
	return NewWithApp(terminal.GroupClass, app, settings, log)
}

const TrashName = "hypr-dock-trash"

func NewTrash(settings *settings.Settings, log hclog.Logger) (*Item, error) {
	app := desktop.NewVirtualWithExec("Recycle Bin", "user-trash", "gio open trash:///")
	return NewWithApp(TrashName, app, settings, log)
}

func NewWithApp(className string, app *desktop.App, settings *settings.Settings, log hclog.Logger) (*Item, error) {
	orientation := gtk.ORIENTATION_VERTICAL
	switch settings.Position {
	case "left", "right":
		orientation = gtk.ORIENTATION_HORIZONTAL
	}

	item, err := gtk.BoxNew(orientation, 0)
	if err != nil {
		return nil, err
	}

	indicatorImage, err := indicator.New(0, settings)
	if err == nil && settings.ShowWindowCount {
		appendInducator(item, indicatorImage, settings.Position)
	} else if err != nil {
		log.Error("Unable to create windows indicator", "className", className, "error", err)
	}

	button, err := gtk.ButtonNew()
	if err == nil {
		image, err := utils.CreateImage(app.GetIcon(), settings.IconSize)
		if err == nil {
			image.SetSizeRequest(settings.IconSize, settings.IconSize)
			image.SetHAlign(gtk.ALIGN_CENTER)
			image.SetVAlign(gtk.ALIGN_CENTER)
			button.SetImage(image)
		} else {
			log.Error("Unable to create image", "error", err)
		}

		buttonName := className
		tooltip := app.GetName()
		if className == terminal.GroupClass {
			buttonName = "terminal-apps"
			tooltip = "Terminal Apps"
		}
		button.SetName(buttonName)
		button.SetTooltipText(tooltip)
		button.SetSizeRequest(settings.IconSize, settings.IconSize)

		utils.SetCursorPointer(button.ToWidget())

		item.Add(button)
	} else {
		log.Error("Unable to create button", "error", err)
	}

	return &Item{
		Windows:         map[string]*ipc.Client{},
		MinimizeRestore: map[string]int{},
		IndicatorImage:  indicatorImage,
		Button:          button,
		ButtonBox:       item,
		App:             app,
		ClassName:       className,

		Settings:   settings,
		List:       nil,
		PinnedList: nil,

		log: log,
	}, nil
}

// RefreshIndicator rebuilds the running-count dots for the current window
// set. Honors ShowWindowCount.
func (i *Item) RefreshIndicator() {
	if i.IndicatorImage != nil {
		i.IndicatorImage.Destroy()
		i.IndicatorImage = nil
	}

	if !i.Settings.ShowWindowCount {
		return
	}

	instances := len(i.Windows)
	newImage, err := indicator.New(instances, i.Settings)
	if err == nil {
		appendInducator(i.ButtonBox, newImage, i.Settings.Position)
	}
	i.IndicatorImage = newImage
}

func (i *Item) RemoveWindow(windowAddress string) {
	delete(i.Windows, windowAddress)
	i.RefreshIndicator()

	if len(i.Windows) == 0 && i.Settings.Preview.Mode != "none" {
		i.Button.SetTooltipText(i.App.GetName())
	}
}

func (i *Item) AddWindow(ipcClient ipc.Client) {
	i.Windows[ipcClient.Address] = &ipcClient
	i.RefreshIndicator()

	if len(i.Windows) != 0 && i.Settings.Preview.Mode != "none" {
		i.Button.SetTooltipText("")
	}
}

func (i *Item) IsTerminalGroup() bool {
	return i.ClassName == terminal.GroupClass
}

func (i *Item) IsPinned() bool {
	return slices.Contains(*i.PinnedList, i.ClassName)
}

func (i *Item) TogglePin() {
	list := i.PinnedList
	className := i.ClassName

	pin := i.IsPinned()
	running := len(i.Windows) > 0

	if pin {
		*list = slices.DeleteFunc(*list, func(name string) bool {
			return name == className
		})
		i.log.Trace("Remove", className)
	}

	if pin && !running {
		i.Remove()
	}

	if !pin {
		*list = append(*list, className)
		i.log.Trace("Add", className)
	}

	file := i.Settings.PinnedPath
	err := pinned.Save(file, *list)
	if err != nil {
		i.log.Error("Failed to save pinned list", "file", file, "error", err)
		return
	}

	i.log.Trace("File saved successfully!", file, className)

	if i.OnPinChange != nil {
		i.OnPinChange()
	}
}

// PinAt inserts the app into the pinned list at a specific position (drag
// landing spot). Running state is preserved; no duplicate icon is created
// because the item identity is the class.
func (i *Item) PinAt(position int) {
	list := i.PinnedList

	if i.IsPinned() {
		i.TogglePin()
		return
	}

	className := i.ClassName
	if position < 0 {
		position = 0
	}
	if position > len(*list) {
		position = len(*list)
	}

	*list = slices.Insert(*list, position, className)

	file := i.Settings.PinnedPath
	if err := pinned.Save(file, *list); err != nil {
		i.log.Error("Failed to save pinned list", "file", file, "error", err)
		return
	}

	if i.OnPinChange != nil {
		i.OnPinChange()
	}
}

func (i *Item) Remove() {
	i.ButtonBox.Destroy()
	delete(i.List, i.ClassName)
}

type Position struct {
	X, Y       int
	CX, CY     int
	RelX, RelY int

	W, H    int
	Monitor *gdk.Monitor
}

func (i *Item) GetCord() (*Position, error) {
	margin := i.Settings.ContextPos
	pos := i.Settings.Position
	v := i.Button

	result := &Position{
		RelX: v.GetAllocation().GetX(),
		RelY: v.GetAllocation().GetY(),

		W: v.GetAllocatedWidth(),
		H: v.GetAllocatedHeight(),

		X: 0,
		Y: 0,
	}

	// get main layer info
	dock, err := layerinfo.GetDock()
	if err != nil {
		return nil, err
	}

	// Add monitor offset
	hyprMonitor, err := ipc.SearchMonitorByName(dock.Monitor)
	if err != nil {
		i.log.Error("Error getting monitor:", "error", err)
	}

	log.Println(dock.Y, hyprMonitor.Height)

	// get coord with centring
	switch pos {
	case "bottom":
		result.CX = result.X + dock.X + result.RelX + result.W/2
		result.CY = result.Y + margin + dock.H + (hyprMonitor.Height - (dock.Y + dock.H))
	case "top":
		result.CX = result.X + dock.X + result.RelX + result.W/2
		result.CY = result.Y + margin + dock.H + dock.Y
	case "left":
		result.CX = result.X + margin + dock.W + dock.X
		result.CY = result.Y + dock.Y + result.RelY + result.H/2
	case "right":
		result.CX = result.X + margin + dock.W + (hyprMonitor.Width - (dock.X + dock.W))
		result.CY = result.Y + dock.Y + result.RelY + result.H/2
	}

	// get absolute coord
	result.X = result.RelX + dock.X
	result.Y = result.RelY + dock.Y

	// Monitor
	display, err := gdk.DisplayGetDefault()
	if err != nil {
		return result, err
	}

	monitor, err := display.GetMonitorAtPoint(result.X, result.Y)
	if err != nil {
		return result, err
	}

	result.Monitor = monitor

	return result, nil
}

func appendInducator(parent *gtk.Box, child *gtk.Image, pos string) {
	child.SetName("indicator")

	switch pos {
	case "left", "top":
		parent.PackStart(child, false, false, 0)
		parent.ReorderChild(child, 0)
	case "bottom", "right":
		parent.PackEnd(child, false, false, 0)
	}
}
