package settingsapp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/hashicorp/go-hclog"

	"hypr-dock/internal/desktop"
	"hypr-dock/internal/pkg/utils"
)

// ApplicationsPage — graphical pinned application manager.

type ApplicationsPage struct {
	app   *App
	rows  *gtk.Box
	hints *gtk.Label
}

func newApplicationsPage(app *App) (gtk.IWidget, *ApplicationsPage) {
	page := &ApplicationsPage{app: app}

	content := verticalPageBox()
	content.PackStart(sectionTitle("Pinned Applications"), false, false, 0)
	content.PackStart(hintLabel("Drag on the dock to reorder. Order changes here apply immediately to the running dock."), false, false, 0)

	rows, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	page.rows = rows
	content.PackStart(rows, false, false, 0)

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

	page.refresh()

	return content, page
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

	up, _ := gtk.ButtonNewFromIconName("go-up-symbolic", gtk.ICON_SIZE_BUTTON)
	up.SetRelief(gtk.RELIEF_NONE)
	up.Connect("clicked", func() {
		if index <= 0 {
			return
		}
		p.app.pins[index-1], p.app.pins[index] = p.app.pins[index], p.app.pins[index-1]
		p.app.SavePins()
		p.refresh()
	})

	down, _ := gtk.ButtonNewFromIconName("go-down-symbolic", gtk.ICON_SIZE_BUTTON)
	down.SetRelief(gtk.RELIEF_NONE)
	down.Connect("clicked", func() {
		if index >= len(p.app.pins)-1 {
			return
		}
		p.app.pins[index+1], p.app.pins[index] = p.app.pins[index], p.app.pins[index+1]
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

// showAddDialog lists all installed applications with a search filter.
func (p *ApplicationsPage) showAddDialog() {
	dialog, _ := gtk.DialogNewWithButtons(
		"Add Application",
		p.app.window,
		gtk.DIALOG_DESTROY_WITH_PARENT,
	)
	dialog.SetDefaultSize(520, 480)

	content, _ := dialog.GetContentArea()
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
			p.app.pins = append(p.app.pins, catalogEntry.ID)
			p.app.SavePins()
			p.refresh()
		})

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
	dialog.ShowAll()

	dialog.Connect("response", func() {
		dialog.Destroy()
	})
}

// CatalogEntry is one installable/pinnable application.
type CatalogEntry = desktop.CatalogEntry

var _ = sort.Strings
var _ = glib.IdleAdd
var _ hclog.Logger = nil
