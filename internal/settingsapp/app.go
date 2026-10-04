package settingsapp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/hashicorp/go-hclog"

	"hypr-dock/internal/appearance"
	"hypr-dock/internal/ctl"
	"hypr-dock/internal/omarchy"
	"hypr-dock/internal/pkg/pinned"
	"hypr-dock/internal/pkg/utils"
	"hypr-dock/internal/settings"
	"hypr-dock/internal/updater"
)

// Settings application for hypr-dock.
//
// Every control edits the typed config and pushes changes through the same
// path as manual edits: config file → ctl apply → dock rebuild. Unknown keys
// and user comments in the ini files are preserved by the writer, and the
// dock applies geometry/style changes live over the existing layer surface.

type App struct {
	window *gtk.Window
	stack  *gtk.Stack
	log    hclog.Logger

	config *settings.Settings
	pins   []string

	status           *gtk.Label
	selectedPage     string
	applyTimer       glib.SourceHandle
	dark             bool
	updating         bool
	updateButton     *gtk.Button
	updateStatus     *gtk.Label
	updateCheck      *gtk.Button
	checkingUpdate   bool
	installedVersion string
	updateMessage    string
	updateRelease    *updater.Release
	updatePoll       glib.SourceHandle

	// pages keep references so they can refresh (e.g. theme switch)
	themePage *ThemePage
	appPage   *ApplicationsPage
}

func New(window *gtk.Window, log hclog.Logger) *App {
	self := &App{window: window, log: log}
	self.load()
	home, _ := os.UserHomeDir()
	mode, _ := os.ReadFile(filepath.Join(home, ".config", "hypr-dock", "preferences-theme"))
	self.dark = string(mode) == "dark"
	if err := installPreferencesStyle(); err != nil {
		log.Error("Preferences stylesheet", "error", err)
	}
	self.build()
	self.checkUpdates()
	self.updatePoll = glib.TimeoutAdd(10*60*1000, func() bool { self.checkUpdates(); return true })
	window.Connect("destroy", func() {
		if self.updatePoll != 0 {
			glib.SourceRemove(self.updatePoll)
			self.updatePoll = 0
		}
	})
	window.Connect("delete-event", func() bool {
		if self.applyTimer != 0 {
			glib.SourceRemove(self.applyTimer)
			self.applyTimer = 0
			self.applyNow()
		}
		return false
	})
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
	if a.applyTimer != 0 {
		glib.SourceRemove(a.applyTimer)
	}
	a.SetStatus("Saving changes…")
	a.applyTimer = glib.TimeoutAdd(250, func() bool {
		a.applyTimer = 0
		a.applyNow()
		return false
	})
}

func (a *App) applyNow() {
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

func (a *App) setTheme() {
	context, err := a.window.GetStyleContext()
	if err != nil {
		return
	}
	if a.dark {
		context.AddClass("dark")
	} else {
		context.RemoveClass("dark")
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
	a.window.SetTitle("HyprDock+ Settings")
	a.window.SetName("preferences")
	a.window.SetTypeHint(gdk.WINDOW_TYPE_HINT_DIALOG)
	a.window.SetDefaultSize(900, 740)
	header, _ := gtk.HeaderBarNew()
	header.SetTitle("HyprDock+")
	header.SetSubtitle("Preferences")
	header.SetShowCloseButton(true)
	a.window.SetTitlebar(header)
	mode := "Light"
	if a.dark {
		mode = "Dark"
	}
	themeControl := comboWidget([]string{"Light", "Dark"}, mode, func(v string) {
		a.dark = v == "Dark"
		a.setTheme()
		home, _ := os.UserHomeDir()
		if err := os.WriteFile(filepath.Join(home, ".config", "hypr-dock", "preferences-theme"), []byte(strings.ToLower(v)), 0644); err != nil {
			a.SetStatus("Could not save preferences theme: " + err.Error())
		}
	})
	header.PackEnd(themeControl)
	a.setTheme()
	a.window.SetResizable(true)
	a.window.SetPosition(gtk.WIN_POS_CENTER)

	outer, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	outer.SetHExpand(true)
	outer.SetVExpand(true)
	a.window.Add(outer)

	sidebar, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	sidebar.SetName("navigation")
	sidebar.SetHAlign(gtk.ALIGN_CENTER)
	sidebar.SetMarginTop(16)
	sidebar.SetMarginBottom(8)

	stack, _ := gtk.StackNew()
	a.stack = stack
	stack.SetHomogeneous(false)
	stack.SetHExpand(true)
	stack.SetVExpand(true)
	stack.SetHAlign(gtk.ALIGN_FILL)
	stack.SetVAlign(gtk.ALIGN_FILL)
	stack.SetTransitionType(gtk.STACK_TRANSITION_TYPE_SLIDE_LEFT_RIGHT)
	stack.SetMarginTop(8)
	stack.SetMarginBottom(8)
	stack.SetMarginEnd(16)
	stack.SetMarginStart(16)
	pageTitle, _ := gtk.LabelNew("Appearance")
	pageTitle.SetName("page-title")
	pageTitle.SetHAlign(gtk.ALIGN_START)
	pageTitle.SetMarginStart(28)
	pageTitle.SetMarginTop(8)
	var nav []*gtk.ToggleButton
	selecting := false

	addPage := func(name string, page gtk.IWidget) {
		button, _ := gtk.ToggleButtonNew()
		button.SetName("nav-item")
		button.SetLabel(name)
		button.SetHAlign(gtk.ALIGN_FILL)
		button.SetSizeRequest(130, -1)
		button.Connect("clicked", func() {
			if selecting {
				return
			}
			selecting = true
			for _, other := range nav {
				other.SetActive(other == button)
			}
			selecting = false
			a.selectedPage = name
			pageTitle.SetText(name)
			stack.SetVisibleChild(page)
			// Like native preferences dialogs, fit the active pane rather than
			// reserving a large blank area for the largest hidden pane.
			glib.IdleAdd(func() {
				child, err := a.window.GetChild()
				if err != nil || child == nil {
					return
				}
				width, _ := a.window.GetSize()
				if width < 650 {
					width = 900
				}
				height, _ := child.ToWidget().GetPreferredHeightForWidth(width)
				headerHeight, _ := header.GetPreferredHeightForWidth(width)
				height += headerHeight + 12
				if height < 360 {
					height = 360
				}
				a.window.Resize(width, height)
			})
		})
		nav = append(nav, button)
		sidebar.PackStart(button, false, false, 0)
		stack.AddNamed(page, name)
		if a.selectedPage == name {
			button.SetActive(true)
			stack.SetVisibleChild(page)
			pageTitle.SetText(name)
		}
	}

	applicationsWidget, applicationsPage := newApplicationsPage(a)
	a.appPage = applicationsPage

	themePage := &ThemePage{app: a}
	a.themePage = themePage

	if a.selectedPage == "" {
		a.selectedPage = "Appearance"
	}
	behaviorContent := compactColumns(newBehaviorPage(a), newWindowsPage(a), newDisplaysPage(a))
	lookContent := verticalPageBox()
	themeNames := []string{}
	for _, theme := range themePage.discover() {
		themeNames = append(themeNames, theme.name)
	}
	themeRow, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	themeLabel, _ := gtk.LabelNew("Dock theme")
	themeRow.PackStart(themeLabel, false, false, 0)
	themeCombo := comboWidget(themeNames, a.config.CurrentTheme, func(v string) { a.config.CurrentTheme = v; a.Apply() })
	themeRow.PackStart(themeCombo, true, true, 0)
	if omarchy.Detect() {
		follow, _ := gtk.CheckButtonNewWithLabel("Follow desktop colors")
		follow.SetActive(a.config.FollowOmarchy)
		follow.Connect("toggled", func() {
			a.config.FollowOmarchy = follow.GetActive()
			if follow.GetActive() {
				a.config.CurrentTheme = "omarchy"
				for i, name := range themeNames {
					if name == "omarchy" {
						themeCombo.SetActive(i)
						break
					}
				}
			}
			a.Apply()
		})
		themeRow.PackStart(follow, false, false, 0)
	}
	lookContent.PackStart(themeRow, false, false, 0)
	lookContent.PackStart(compactColumns(newDockLayoutPage(a), newAppearancePage(a), newIndicatorsPage(a)), false, false, 0)
	lookContent.PackStart(newStylesheetEditor(a), false, false, 0)
	addPage("Appearance", lookContent)
	addPage("Behavior", behaviorContent)
	addPage("Applications", applicationsWidget)
	addPage("Maintenance", newAdvancedPage(a))

	outer.PackStart(sidebar, false, false, 0)

	separator, _ := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	outer.PackStart(separator, false, false, 0)

	// Right side: pages + status bar.
	right, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	// The selected tab already names the page; avoid a duplicate giant heading.
	right.PackStart(stack, true, true, 0)
	right.PackStart(gtkSeparatorHorizontal(), false, false, 0)

	statusBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 6)
	statusBox.SetName("status-bar")
	statusBox.SetMarginTop(4)
	statusBox.SetMarginBottom(6)
	statusBox.SetMarginStart(12)
	statusBox.SetMarginEnd(12)

	status, _ := gtk.LabelNew("Changes are saved automatically")
	status.SetEllipsize(3)
	status.SetHAlign(gtk.ALIGN_START)
	status.SetHExpand(true)
	a.status = status

	applyNow, _ := gtk.ButtonNewWithLabel("Apply now")
	applyNow.Connect("clicked", func() {
		if a.applyTimer != 0 {
			glib.SourceRemove(a.applyTimer)
			a.applyTimer = 0
		}
		a.applyNow()
	})

	refresh, _ := gtk.ButtonNewWithLabel("Reload from disk")
	refresh.Connect("clicked", func() {
		if a.applyTimer != 0 {
			glib.SourceRemove(a.applyTimer)
			a.applyTimer = 0
			a.applyNow()
		}
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
	area.SetName("swatch")
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
