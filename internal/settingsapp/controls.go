package settingsapp

import (
	"fmt"
	"strconv"

	"github.com/gotk3/gotk3/gtk"

	"hypr-dock/internal/pkg/utils"
)

// Shared control builders. Every widget reports through a setter that edits
// the typed config; pages debounce persistence themselves.

// sectionTitle renders a page section header.
func sectionTitle(text string) gtk.IWidget {
	label, _ := gtk.LabelNew("")
	label.SetMarkup("<b>" + text + "</b>")
	label.SetName("section-title")
	label.SetHAlign(gtk.ALIGN_START)
	label.SetMarginTop(6)
	label.SetMarginBottom(2)
	return label
}

// hintLabel renders muted helper text.
func hintLabel(text string) gtk.IWidget {
	label, _ := gtk.LabelNew("")
	label.SetText(text)
	label.SetName("hint")
	label.SetLineWrap(true)
	label.SetMaxWidthChars(70)
	label.SetHAlign(gtk.ALIGN_START)
	label.SetMarginBottom(6)
	return label
}

func grid() *gtk.Grid {
	grid, _ := gtk.GridNew()
	grid.SetName("settings-group")
	grid.SetRowSpacing(6)
	grid.SetColumnSpacing(12)
	grid.SetMarginTop(4)
	grid.SetMarginBottom(4)
	return grid
}

// addRow places label + control + optional hint in the grid.
func addRow(grid *gtk.Grid, row int, label string, control gtk.IWidget, hint string) {
	labelWidget, _ := gtk.LabelNew(label)
	labelWidget.SetHAlign(gtk.ALIGN_START)
	labelWidget.SetXAlign(0)
	labelWidget.SetWidthChars(18)
	labelWidget.SetMaxWidthChars(22)
	labelWidget.SetLineWrap(true)
	textBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	textBox.SetVAlign(gtk.ALIGN_CENTER)
	textBox.PackStart(labelWidget, false, false, 0)
	grid.Attach(textBox, 0, row, 1, 1)
	grid.SetColumnHomogeneous(false)
	grid.SetColumnSpacing(12)
	grid.Attach(control, 1, row, 1, 1)
	control.ToWidget().SetHExpand(true)
	control.ToWidget().SetHAlign(gtk.ALIGN_FILL)
	if _, ok := control.(*gtk.Switch); ok {
		control.ToWidget().SetHExpand(true)
		control.ToWidget().SetHAlign(gtk.ALIGN_END)
	}

	if hint != "" {
		labelWidget.SetTooltipText(hint)
		control.ToWidget().SetTooltipText(hint)
	}
}

// intScale builds a bounded slider with a live value readout.
func intScale(value, min, max, step float64, onChange func(int)) (*gtk.Box, *gtk.Scale) {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)

	scale, _ := gtk.ScaleNewWithRange(gtk.ORIENTATION_HORIZONTAL, min, max, step)
	scale.SetValue(value)
	scale.SetHExpand(true)
	scale.SetDrawValue(false)
	scale.SetRoundDigits(0)

	readout, _ := gtk.LabelNew(strconv.Itoa(int(value)))
	readout.SetWidthChars(4)

	scale.Connect("value-changed", func() {
		current := int(scale.GetValue())
		readout.SetText(strconv.Itoa(current))
		onChange(current)
	})

	box.PackStart(scale, true, true, 0)
	box.PackStart(readout, false, false, 0)

	return box, scale
}

// switchWidget builds a boolean switch.
func switchWidget(value bool, onChange func(bool)) *gtk.Switch {
	s, _ := gtk.SwitchNew()
	s.SetName("compact-switch")
	s.SetHAlign(gtk.ALIGN_END)
	s.SetVAlign(gtk.ALIGN_CENTER)
	// Override theme defaults: some GTK themes make switches excessively wide.
	utils.AddStyle(s, `
switch#compact-switch {
  min-width: 34px;
  min-height: 18px;
  padding: 0;
  border-radius: 9px;
}
switch#compact-switch slider {
  min-width: 14px;
  min-height: 14px;
  margin: 2px;
  border-radius: 7px;
}
switch#compact-switch:checked {
  background-color: rgba(120, 170, 255, 0.85);
}
switch#compact-switch:not(:checked) {
  background-color: rgba(120, 130, 145, 0.35);
}
`)
	s.SetActive(value)
	s.Connect("notify::active", func() {
		onChange(s.GetActive())
	})
	return s
}

// comboWidget builds an enum selector.
func comboWidget(options []string, current string, onChange func(string)) *gtk.ComboBoxText {
	combo, _ := gtk.ComboBoxTextNew()
	for _, option := range options {
		combo.AppendText(option)
	}

	selected := -1
	for i, option := range options {
		if option == current {
			selected = i
		}
	}
	if selected >= 0 {
		combo.SetActive(selected)
	}

	combo.Connect("changed", func() {
		text := combo.GetActiveText()
		onChange(text)
	})

	return combo
}

// entryWidget builds a text field.
func entryWidget(value string, onChange func(string)) *gtk.Entry {
	entry, _ := gtk.EntryNew()
	entry.SetText(value)
	entry.SetHExpand(true)
	entry.Connect("changed", func() {
		text, _ := entry.GetText()
		onChange(text)
	})
	return entry
}

func pageScroll(content gtk.IWidget) gtk.IWidget {
	scroll, _ := gtk.ScrolledWindowNew(nil, nil)
	scroll.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	scroll.SetVExpand(true)
	scroll.Add(content)
	return scroll
}

func verticalPageBox() *gtk.Box {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 4)
	box.SetMarginTop(6)
	return box
}

var _ = fmt.Sprintf

// Place complete related groups into balanced columns, rather than one long
// scrolling form. Widgets keep their original setters and ownership.
func compactColumns(pages ...gtk.IWidget) gtk.IWidget {
	root, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 14)
	root.SetHomogeneous(true)
	columns := make([]*gtk.Box, 2)
	loads := make([]int, 2)
	for n := range columns {
		columns[n], _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
		root.PackStart(columns[n], true, true, 0)
	}
	for _, page := range pages {
		box := page.(*gtk.Box)
		var group *gtk.Box
		col := 0
		for l := box.GetChildren(); l != nil && l.Data() != nil; l = l.Next() {
			w, ok := l.Data().(gtk.IWidget)
			if !ok {
				continue
			}
			name, _ := w.ToWidget().GetName()
			// Long explanatory paragraphs belong in tooltips in compact mode.
			if name == "hint" {
				label := &gtk.Label{Widget: *w.ToWidget()}
				text, _ := label.GetText()
				root.SetTooltipText(text)
				continue
			}
			if group == nil || name == "section-title" {
				col = 0
				for n := 1; n < len(columns); n++ {
					if loads[n] < loads[col] {
						col = n
					}
				}
				group, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
				columns[col].PackStart(group, false, false, 0)
				loads[col] += 2
			}
			w.ToWidget().Ref()
			box.Remove(w)
			group.PackStart(w, false, false, 0)
			w.ToWidget().Unref()
			if name == "settings-group" {
				container := &gtk.Container{Widget: *w.ToWidget()}
				loads[col] += int(container.GetChildren().Length()) / 2
			}
		}
	}
	return root
}
