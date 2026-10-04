package settingsapp

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/hashicorp/go-hclog"

	"hypr-dock/internal/desktop"
	"hypr-dock/internal/pkg/pinned"
	"hypr-dock/internal/pkg/utils"
)

// ApplicationsPage — graphical pinned application manager.

type ApplicationsPage struct {
	app    *App
	rows   *gtk.Box
	hints  *gtk.Label
	picker *gtk.Box
}

func newApplicationsPage(app *App) (gtk.IWidget, *ApplicationsPage) {
	page := &ApplicationsPage{app: app}

	content := verticalPageBox()
	outer, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 16)
	outer.SetHomogeneous(true)
	outer.PackStart(content, true, true, 0)
	content.PackStart(sectionTitle("System apps"), false, false, 0)
	systemGrid := grid()
	addRow(systemGrid, 0, "Recycle Bin", switchWidget(app.config.ShowTrash, func(v bool) { app.config.ShowTrash = v; app.Apply() }), "Dock-owned; always the last item")
	addRow(systemGrid, 1, "Application launcher", switchWidget(app.config.ShowLauncherButton, func(v bool) { app.config.ShowLauncherButton = v; app.Apply() }), "System launcher icon")
	addRow(systemGrid, 2, "Home folder", switchWidget(app.config.ShowHomeFolder, func(v bool) { app.config.ShowHomeFolder = v; app.Apply() }), "Open your home folder")
	addRow(systemGrid, 3, "Dock preferences", switchWidget(app.config.ShowSettingsIcon, func(v bool) { app.config.ShowSettingsIcon = v; app.Apply() }), "Open this configurator")
	content.PackStart(systemGrid, false, false, 0)
	content.PackStart(newLauncherPage(app), false, false, 0)
	content = verticalPageBox()
	outer.PackStart(content, true, true, 0)
	content.PackStart(sectionTitle("Pinned Applications"), false, false, 0)
	content.PackStart(hintLabel("Drag on the dock to reorder. Order changes here apply immediately to the running dock."), false, false, 0)

	rows, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	page.rows = rows
	pinScroll, _ := gtk.ScrolledWindowNew(nil, nil)
	pinScroll.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	pinScroll.SetSizeRequest(-1, 150)
	pinScroll.Add(rows)
	content.PackStart(pinScroll, false, false, 0)

	buttonsBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	buttonsBox.SetMarginTop(10)

	add, _ := gtk.ButtonNewWithLabel("Add Application")
	add.Connect("clicked", func() {
		page.showAddDialog()
	})

	removeAll, _ := gtk.ButtonNewWithLabel("Remove All")
	removeAll.Connect("clicked", func() {
		app.pins = []string{}
		app.SavePins()
		page.refresh()
	})

	buttonsBox.PackStart(add, false, false, 0)
	buttonsBox.PackStart(removeAll, false, false, 0)
	content.PackStart(buttonsBox, false, false, 0)
	page.picker, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	page.picker.SetNoShowAll(true)
	content.PackStart(page.picker, false, false, 0)

	page.refresh()
	content.PackStart(sectionTitle("Hidden applications"), false, false, 0)
	content.PackStart(hintLabel("Hide these window classes from the dock. Separate with commas; wildcards such as scratch-* are supported."), false, false, 0)
	content.PackStart(entryWidget(app.config.HiddenApps, func(v string) { app.config.HiddenApps = v; app.Apply() }), false, false, 0)

	return outer, page
}

func (p *ApplicationsPage) refresh() {
	if p.rows == nil {
		return
	}

	children := p.rows.GetChildren()
	for l := children; l != nil && l.Data() != nil; l = l.Next() {
		if w, ok := l.Data().(gtk.IWidget); ok {
			p.rows.Remove(w)
		}
	}

	for i, className := range p.app.pins {
		row := p.buildRow(className, i)
		p.rows.PackStart(row, false, false, 0)
	}

	p.rows.ShowAll()
}

func (p *ApplicationsPage) buildRow(className string, index int) gtk.IWidget {
	rowBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	rowBox.SetName("pin-row")
	rowBox.SetMarginTop(2)
	rowBox.SetMarginBottom(2)

	eventBox, _ := gtk.EventBoxNew()
	eventBox.Add(rowBox)

	appEntry, _ := desktop.New(className)
	name := appEntry.GetName()
	if name == className {
		name = className
	}

	icon, err := utils.CreateImage(appEntry.GetIcon(), 24)
	if err != nil || appEntry.GetIcon() == "" {
		icon, _ = utils.CreateImage("application-x-executable", 24)
	}

	orderLabel, _ := gtk.LabelNew(fmt.Sprintf("%d.", index+1))
	orderLabel.SetWidthChars(3)
	orderLabel.SetHAlign(gtk.ALIGN_START)

	nameLabel, _ := gtk.LabelNew(name + "  (" + className + ")")
	nameLabel.SetHAlign(gtk.ALIGN_START)
	nameLabel.SetHExpand(true)
	nameLabel.SetMaxWidthChars(35)
	nameLabel.SetEllipsize(3)

	up, _ := gtk.ButtonNewFromIconName("go-up-symbolic", gtk.ICON_SIZE_BUTTON)
	up.SetRelief(gtk.RELIEF_NONE)
	up.Connect("clicked", func() {
		if index <= 0 {
			return
		}
		p.swapPins(index, index-1)
		p.app.SavePins()
		p.refresh()
	})

	down, _ := gtk.ButtonNewFromIconName("go-down-symbolic", gtk.ICON_SIZE_BUTTON)
	down.SetRelief(gtk.RELIEF_NONE)
	down.Connect("clicked", func() {
		if index >= len(p.app.pins)-1 {
			return
		}
		p.swapPins(index, index+1)
		p.app.SavePins()
		p.refresh()
	})

	remove, _ := gtk.ButtonNewFromIconName("list-remove-symbolic", gtk.ICON_SIZE_BUTTON)
	remove.SetRelief(gtk.RELIEF_NONE)
	remove.SetTooltipText("Unpin")
	remove.Connect("clicked", func() {
		p.app.pins = append(p.app.pins[:index], p.app.pins[index+1:]...)
		p.app.SavePins()
		p.refresh()
	})

	if icon != nil {
		rowBox.PackStart(icon, false, false, 0)
	}
	rowBox.PackStart(orderLabel, false, false, 0)
	rowBox.PackStart(nameLabel, true, true, 0)
	rowBox.PackStart(up, false, false, 0)
	rowBox.PackStart(down, false, false, 0)
	rowBox.PackStart(remove, false, false, 0)

	return eventBox
}

// Respect absolute dock slots, including synthetic items, when moving pins.
func (p *ApplicationsPage) swapPins(a, b int) {
	order, err := pinned.Open(p.app.config.OrderPath)
	if err == nil {
		for _, class := range p.app.pins {
			if !slices.Contains(order, class) {
				order = append(order, class)
			}
		}
		x, y := slices.Index(order, p.app.pins[a]), slices.Index(order, p.app.pins[b])
		order[x], order[y] = order[y], order[x]
		if err := pinned.Save(p.app.config.OrderPath, order); err != nil {
			p.app.SetStatus("Could not save dock order: " + err.Error())
			return
		}
	}
	p.app.pins[a], p.app.pins[b] = p.app.pins[b], p.app.pins[a]
}

// showAddDialog lists all installed applications with a search filter.
func (p *ApplicationsPage) showAddDialog() {
	if p.picker.GetVisible() {
		p.picker.Hide()
		return
	}
	children := p.picker.GetChildren()
	for l := children; l != nil && l.Data() != nil; l = l.Next() {
		if w, ok := l.Data().(gtk.IWidget); ok {
			w.ToWidget().Destroy()
		}
	}
	content := p.picker
	content.SetMarginTop(10)
	content.SetMarginBottom(10)
	content.SetMarginStart(10)
	content.SetMarginEnd(10)

	search, _ := gtk.SearchEntryNew()
	content.PackStart(search, false, false, 4)

	list, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	scroll, _ := gtk.ScrolledWindowNew(nil, nil)
	scroll.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	scroll.SetVExpand(true)
	scroll.SetSizeRequest(-1, 150)
	scroll.Add(list)
	content.PackStart(scroll, true, true, 0)

	apps := desktop.Catalog()
	for _, entry := range apps {
		catalogEntry := entry
		row, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
		row.SetMarginTop(2)
		row.SetMarginBottom(2)

		button, _ := gtk.ButtonNew()
		button.SetRelief(gtk.RELIEF_NONE)

		inner, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
		if icon, err := utils.CreateImage(catalogEntry.Icon, 22); err == nil {
			inner.PackStart(icon, false, false, 0)
		}
		label, _ := gtk.LabelNew(catalogEntry.Name + "  (" + catalogEntry.ID + ")")
		label.SetHAlign(gtk.ALIGN_START)
		inner.PackStart(label, true, true, 0)
		button.Add(inner)

		button.Connect("clicked", func() {
			if slices.Contains(p.app.pins, catalogEntry.ID) {
				return
			}
			p.app.pins = append(p.app.pins, catalogEntry.ID)
			p.app.SavePins()
			p.refresh()
			button.SetSensitive(false)
		})
		button.SetSensitive(!slices.Contains(p.app.pins, catalogEntry.ID))

		row.PackStart(button, false, false, 0)
		list.Add(row)

		// Search filter.
		match := strings.ToLower(catalogEntry.Name + " " + catalogEntry.ID)
		search.Connect("search-changed", func() {
			query, _ := search.GetText()
			visible := strings.Contains(match, strings.ToLower(query))
			row.SetVisible(visible)
		})
	}

	list.ShowAll()
	content.SetNoShowAll(false)
	content.ShowAll()
	content.SetNoShowAll(true)
}

// CatalogEntry is one installable/pinnable application.
type CatalogEntry = desktop.CatalogEntry

var _ = sort.Strings
var _ = glib.IdleAdd
var _ hclog.Logger = nil
