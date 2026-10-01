package settingsapp

import (
	"fmt"
	"strconv"

	"github.com/gotk3/gotk3/gtk"
)

// Shared control builders. Every widget reports through a setter that edits
// the typed config; pages debounce persistence themselves.

// sectionTitle renders a page section header.
func sectionTitle(text string) gtk.IWidget {
	label, _ := gtk.LabelNew("")
	label.SetMarkup("<b>" + text + "</b>")
	label.SetHAlign(gtk.ALIGN_START)
	label.SetMarginTop(10)
	label.SetMarginBottom(4)
	return label
}

// hintLabel renders muted helper text.
func hintLabel(text string) gtk.IWidget {
	label, _ := gtk.LabelNew("")
	label.SetMarkup("<small><i>" + text + "</i></small>")
	label.SetHAlign(gtk.ALIGN_START)
	label.SetMarginBottom(6)
	return label
}

func grid() *gtk.Grid {
	grid, _ := gtk.GridNew()
	grid.SetRowSpacing(8)
	grid.SetColumnSpacing(12)
	grid.SetMarginTop(4)
	grid.SetMarginBottom(10)
	return grid
}

// addRow places label + control + optional hint in the grid.
func addRow(grid *gtk.Grid, row int, label string, control gtk.IWidget, hint string) {
	labelWidget, _ := gtk.LabelNew(label)
	labelWidget.SetHAlign(gtk.ALIGN_START)
	grid.Attach(labelWidget, 0, row, 1, 1)
	grid.Attach(control, 1, row, 1, 1)

	if hint != "" {
		hintWidget, _ := gtk.LabelNew("")
		hintWidget.SetMarkup("<small>" + hint + "</small>")
		hintWidget.SetHAlign(gtk.ALIGN_START)
		grid.Attach(hintWidget, 2, row, 1, 1)
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
