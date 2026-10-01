package settingsapp

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gotk3/gotk3/gtk"
	"github.com/hashicorp/go-hclog"

	"hypr-dock/internal/appearance"
	"hypr-dock/internal/ctl"
	"hypr-dock/internal/pkg/pinned"
	"hypr-dock/internal/pkg/utils"
	"hypr-dock/internal/settings"
)

// Settings application for hypr-dock.
//
// Every control edits the typed config and pushes changes through the same
// path as manual edits: config file → ctl apply → dock rebuild. Unknown keys
// and user comments in the ini files are preserved by the writer, and the
// dock applies geometry/style changes live over the existing layer surface.

type App struct {
	window *gtk.Window
	log    hclog.Logger

	config *settings.Settings
	pins   []string

	status *gtk.Label

	// pages keep references so they can refresh (e.g. theme switch)
	themePage *ThemePage
	appPage   *ApplicationsPage
}

func New(window *gtk.Window, log hclog.Logger) *App {
	self := &App{window: window, log: log}
	self.load()
	self.build()
	return self
}

// load reads config + pinned from the dock's standard locations.
func (a *App) load() {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "hypr-dock")
	localDir := filepath.Join(home, ".local", "share", "hypr-dock")
	configPath := filepath.Join(configDir, "hypr-dock.conf")

	s, err := settings.Load(configPath, configDir, localDir, a.log)
	if err != nil || s == nil {
		a.log.Error("Settings load failed", "error", err)
		a.SetStatus("Config load failed — check the config file")
		return
	}

	pins, err := pinned.Open(s.PinnedPath)
	if err != nil {
		a.log.Error("Pinned load failed", "error", err)
		pins = nil
	}

	a.config = s
	a.pins = pins
}

// Apply persists the typed config and tells the running dock to re-read.
func (a *App) Apply() {
	if err := a.config.Save(a.config.ConfigPath); err != nil {
		a.SetStatus("Save failed: " + err.Error())
		a.log.Error("Save failed", "error", err)
		return
	}

	resp, err := ctl.Send(ctl.Request{Cmd: ctl.CmdApply})
	if err != nil {
		a.SetStatus("Saved (dock not running: " + ctlSocketHint() + ")")
		return
	}
	if !resp.Ok {
		a.SetStatus("Dock rejected apply: " + resp.Error)
		return
	}
	a.SetStatus("Applied " + time.Now().Format("15:04:05"))
}

// SavePins writes the pinned list and tells the dock.
func (a *App) SavePins() {
	if err := pinned.Save(a.config.PinnedPath, a.pins); err != nil {
		a.SetStatus("Pinned save failed: " + err.Error())
		return
	}
	a.Apply()
}

func (a *App) SetStatus(text string) {
	if a.status != nil {
		a.status.SetText(text)
	}
}

func ctlSocketHint() string {
	if _, err := os.Stat(ctl.SocketPath()); err != nil {
		return "no control socket"
	}
	return "socket did not respond"
}

// build lays out sidebar + pages.
func (a *App) build() {
	a.window.SetTitle("Hypr-Dock Settings")
	a.window.SetDefaultSize(880, 560)
	a.window.SetPosition(gtk.WIN_POS_CENTER)

	outer, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	a.window.Add(outer)

	sidebar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	sidebar.SetName("sidebar")
	sidebar.SetMarginTop(12)
	sidebar.SetMarginBottom(12)
	sidebar.SetMarginStart(10)
	sidebar.SetMarginEnd(6)

	title, _ := gtk.LabelNew("")
	title.SetMarkup("<b>Hypr-Dock</b>\n<small>Settings</small>")
	title.SetHAlign(gtk.ALIGN_START)
	title.SetMarginBottom(10)
	sidebar.PackStart(title, false, false, 0)

	stack, _ := gtk.StackNew()
	stack.SetHExpand(true)
	stack.SetVExpand(true)
	stack.SetTransitionType(gtk.STACK_TRANSITION_TYPE_SLIDE_LEFT_RIGHT)
	stack.SetMarginTop(12)
	stack.SetMarginBottom(8)
	stack.SetMarginEnd(14)
	stack.SetMarginStart(6)

	addPage := func(name string, page gtk.IWidget) {
		button, _ := gtk.ButtonNew()
		button.SetName("nav-item")
		button.SetLabel(name)
		button.SetHAlign(gtk.ALIGN_FILL)
		button.Connect("clicked", func() {
			stack.SetVisibleChild(page)
		})
		sidebar.PackStart(button, false, false, 0)
		stack.AddNamed(page, name)
	}

	applicationsWidget, applicationsPage := newApplicationsPage(a)
	a.appPage = applicationsPage

	themeWidget, themePage := newThemePage(a)
	a.themePage = themePage

	addPage("Appearance", newAppearancePage(a))
	addPage("Behavior", newBehaviorPage(a))
	addPage("Applications", applicationsWidget)
	addPage("Windows", newWindowsPage(a))
	addPage("Displays", newDisplaysPage(a))
	addPage("Themes", themeWidget)
	addPage("Advanced", newAdvancedPage(a))

	outer.PackStart(sidebar, false, false, 0)

	separator, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	outer.PackStart(separator, false, false, 0)

	// Right side: pages + status bar.
	right, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	right.PackStart(stack, true, true, 0)
	right.PackStart(gtkSeparatorHorizontal(), false, false, 0)

	statusBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 6)
	statusBox.SetMarginTop(4)
	statusBox.SetMarginBottom(6)
	statusBox.SetMarginStart(12)
	statusBox.SetMarginEnd(12)

	status, _ := gtk.LabelNew("Ready")
	status.SetHAlign(gtk.ALIGN_START)
	status.SetHExpand(true)
	a.status = status

	applyNow, _ := gtk.ButtonNewWithLabel("Apply now")
	applyNow.Connect("clicked", func() {
		a.Apply()
	})

	refresh, _ := gtk.ButtonNewWithLabel("Reload from disk")
	refresh.Connect("clicked", func() {
		a.load()
		a.rebuild()
		a.SetStatus("Reloaded from disk")
	})

	statusBox.PackStart(status, true, true, 0)
	statusBox.PackStart(applyNow, false, false, 0)
	statusBox.PackStart(refresh, false, false, 0)

	right.PackStart(statusBox, false, false, 0)
	outer.PackStart(right, true, true, 0)
}

func (a *App) rebuild() {
	// Replace stack content wholesale: easiest correct refresh path.
	if child, err := a.window.GetChild(); err == nil && child != nil {
		if widget, isWidget := child.(gtk.IWidget); isWidget {
			a.window.Remove(widget)
		}
		if container, isContainer := child.(interface{ Destroy() }); isContainer {
			container.Destroy()
		}
	}
	a.build()
	a.window.ShowAll()
}

func gtkSeparatorHorizontal() gtk.IWidget {
	sep, _ := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	return sep
}

// ThemePreviewSwatch renders a small color preview for a theme directory —
// parses the theme's #app background from style.css.
func themePreviewSwatch(themeDir string) gtk.IWidget {
	area, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	area.SetSizeRequest(64, 40)

	css := ""
	if data, err := os.ReadFile(filepath.Join(themeDir, "style.css")); err == nil {
		css = string(data)
	}

	block := appearance.ParseAppBlock(css)

	bg := "rgba(0,0,0,0)"
	if block.HasBg {
		bg = fmt.Sprintf("#%02x%02x%02x", int(block.BgR), int(block.BgG), int(block.BgB))
	}

	style := fmt.Sprintf(
		"#swatch { background-color: %s; border-radius: 6px; border: 1px solid rgba(255,255,255,0.2); }",
		bg)
	utils.AddStyle(area, style)

	return area
}
