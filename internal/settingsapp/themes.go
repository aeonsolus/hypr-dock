package settingsapp

import (
	"os"
	"path/filepath"

	"github.com/gotk3/gotk3/gtk"

	"hypr-dock/internal/omarchy"
	"hypr-dock/internal/pkg/utils"
)

// ThemePage — theme discovery, instant switching, follow-Omarchy, and the
// theme's own structured settings (spacing, preview geometry).

type ThemePage struct {
	app   *App
	cards *gtk.Box
}

func newThemePage(app *App) (gtk.IWidget, *ThemePage) {
	page := &ThemePage{app: app}

	content := verticalPageBox()
	content.PackStart(sectionTitle("Themes"), false, false, 0)

	cards, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 6)
	page.cards = cards
	content.PackStart(cards, false, false, 0)

	if omarchy.Detect() {
		content.PackStart(sectionTitle("Omarchy"), false, false, 0)
		content.PackStart(switchWidget(app.config.FollowOmarchy, func(v bool) {
			app.config.FollowOmarchy = v
			if v {
				app.config.CurrentTheme = "omarchy"
			}
			app.Apply()
			page.refresh()
		}), false, false, 0)
	}

	page.refresh()

	return pageScroll(content), page
}

func (p *ThemePage) widget() gtk.IWidget {
	return p.cards
}

func (p *ThemePage) refresh() {
	if p.cards == nil {
		return
	}

	children := p.cards.GetChildren()
	for l := children; l != nil && l.Data() != nil; l = l.Next() {
		if w, ok := l.Data().(gtk.IWidget); ok {
			p.cards.Remove(w)
		}
	}

	themes := p.discover()
	for _, theme := range themes {
		card := p.buildCard(theme)
		p.cards.PackStart(card, false, false, 0)
	}

	p.cards.ShowAll()
}

type themeInfo struct {
	name string
	dir  string
}

func (p *ThemePage) discover() []themeInfo {
	var out []themeInfo

	entries, err := os.ReadDir(p.app.config.ThemesDir)
	if err != nil {
		return out
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(p.app.config.ThemesDir, entry.Name())
		if !utils.FileExists(filepath.Join(dir, "style.css")) {
			continue
		}
		out = append(out, themeInfo{name: entry.Name(), dir: dir})
	}

	return out
}

func (p *ThemePage) buildCard(theme themeInfo) gtk.IWidget {
	card, _ := gtk.ButtonNew()
	card.SetName("theme-card")

	inner, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 12)
	inner.SetMarginTop(6)
	inner.SetMarginBottom(6)

	swatch := themePreviewSwatch(theme.dir)
	inner.PackStart(swatch, false, false, 0)

	labelBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	nameLabel, _ := gtk.LabelNew("")
	nameLabel.SetMarkup("<b>" + theme.name + "</b>")
	nameLabel.SetHAlign(gtk.ALIGN_START)

	active := p.app.config.CurrentTheme == theme.name
	sub := ""
	if active {
		sub = "Active"
	} else {
		sub = "Click to apply"
	}
	subLabel, _ := gtk.LabelNew("")
	subLabel.SetMarkup("<small>" + sub + "</small>")
	subLabel.SetHAlign(gtk.ALIGN_START)

	labelBox.PackStart(nameLabel, false, false, 0)
	labelBox.PackStart(subLabel, false, false, 0)
	inner.PackStart(labelBox, true, true, 0)

	card.Add(inner)

	card.Connect("clicked", func() {
		p.app.config.CurrentTheme = theme.name
		p.app.Apply()
		p.refresh()
	})

	return card
}
