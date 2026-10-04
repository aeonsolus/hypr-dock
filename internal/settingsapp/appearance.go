package settingsapp

import (
	"github.com/gotk3/gotk3/gtk"
)

// AppearancePage — dock geometry (icon size, position, margins, layer) and
// structured panel styling. Geometry changes re-layout the dock live over the
// existing layer surface; styling goes through the generated override CSS so
// the user's style.css is never rewritten.

func newDockLayoutPage(app *App) gtk.IWidget {
	page := verticalPageBox()

	page.PackStart(sectionTitle("Dock"), false, false, 0)
	dockGrid := grid()

	row := 0
	sizeRow, sizeScale := intScale(float64(app.config.IconSize), 12, 128, 1, func(v int) {
		app.config.IconSize = v
		app.Apply()
	})
	addRow(dockGrid, row, "Icon size", sizeRow, "px (12–128)")
	_ = sizeScale
	row++

	posCombo := comboWidget([]string{"top", "bottom", "left", "right"}, app.config.Position, func(v string) {
		app.config.Position = v
		app.Apply()
	})
	addRow(dockGrid, row, "Position", posCombo, "")
	row++

	layerCombo := comboWidget([]string{"background", "bottom", "top", "overlay"}, app.config.Layer, func(v string) {
		app.config.Layer = v
		app.Apply()
	})
	addRow(dockGrid, row, "Layer", layerCombo, "")
	row++

	marginRow, _ := intScale(float64(app.config.Margin), 0, 64, 1, func(v int) {
		app.config.Margin = v
		app.Apply()
	})
	addRow(dockGrid, row, "Margin from screen edge", marginRow, "used when system gap is off")
	row++

	spacingRow, _ := intScale(float64(app.config.Spacing), 0, 64, 1, func(v int) {
		app.config.Spacing = v
		app.config.SaveTheme()
		app.Apply()
	})
	addRow(dockGrid, row, "Icon spacing", spacingRow, "from the active theme")
	row++

	exclusiveSwitch := switchWidget(app.config.Exclusive, func(v bool) {
		app.config.Exclusive = v
		app.Apply()
	})
	addRow(dockGrid, row, "Reserve space (exclusive zone)", exclusiveSwitch, "windows avoid the dock")
	row++

	gapSwitch := switchWidget(app.config.SystemGapUsed, func(v bool) {
		app.config.SystemGapUsed = v
		app.Apply()
	})
	addRow(dockGrid, row, "Use Hyprland system gap", gapSwitch, "margin follows gaps_out")
	row++

	page.PackStart(dockGrid, false, false, 0)
	return page
}

func newAppearancePage(app *App) gtk.IWidget {
	page := verticalPageBox()
	page.PackStart(sectionTitle("Panel"), false, false, 0)
	panelGrid := grid()

	prow := 0
	opacityRow, _ := intScale(float64(app.config.Appearance.PanelOpacity), 0, 100, 1, func(v int) {
		app.config.Appearance.PanelOpacity = v
		app.Apply()
	})
	addRow(panelGrid, prow, "Background opacity", opacityRow, "0 = keep theme")
	prow++

	radiusRow, _ := intScale(float64(app.config.Appearance.BorderRadius), 0, 128, 1, func(v int) {
		app.config.Appearance.BorderRadius = v
		app.Apply()
	})
	addRow(panelGrid, prow, "Border radius", radiusRow, "0 = keep theme")
	prow++

	borderRow, _ := intScale(float64(app.config.Appearance.BorderWidth), 0, 16, 1, func(v int) {
		app.config.Appearance.BorderWidth = v
		app.Apply()
	})
	addRow(panelGrid, prow, "Border width", borderRow, "0 = keep theme")
	prow++

	paddingRow, _ := intScale(float64(app.config.Appearance.PanelPadding), 0, 64, 1, func(v int) {
		app.config.Appearance.PanelPadding = v
		app.Apply()
	})
	addRow(panelGrid, prow, "Padding", paddingRow, "0 = keep theme")
	prow++

	hoverSwitch := switchWidget(app.config.Appearance.HoverEffects, func(v bool) {
		app.config.Appearance.HoverEffects = v
		app.Apply()
	})
	addRow(panelGrid, prow, "Hover effects", hoverSwitch, "")
	prow++

	activeSwitch := switchWidget(app.config.Appearance.ActiveTint, func(v bool) {
		app.config.Appearance.ActiveTint = v
		app.Apply()
	})
	addRow(panelGrid, prow, "Highlight focused app", activeSwitch, "")
	prow++

	accentEntry := entryWidget(app.config.Appearance.Accent, func(v string) {
		app.config.Appearance.Accent = v
		app.Apply()
	})
	addRow(panelGrid, prow, "Accent color (hex)", accentEntry, "e.g. #8ba4b0 — empty = theme")
	prow++

	page.PackStart(panelGrid, false, false, 0)

	page.PackStart(hintLabel("Panel adjustments preserve your theme stylesheet. For advanced changes, expand Custom stylesheet below."), false, false, 0)

	return page
}
