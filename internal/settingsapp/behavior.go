package settingsapp

import (
	"github.com/gotk3/gotk3/gtk"
)

// BehaviorPage — visibility, click semantics, launcher.

func newBehaviorPage(app *App) gtk.IWidget {
	page := verticalPageBox()

	page.PackStart(sectionTitle("Visibility"), false, false, 0)
	visGrid := grid()

	row := 0
	visGrid.Attach(switchWidget(app.config.ShowPinnedApps, func(v bool) {
		app.config.ShowPinnedApps = v
		app.Apply()
	}), 1, row, 1, 1)
	addLabelRow(visGrid, row, "Show pinned applications")
	row++

	visGrid.Attach(switchWidget(app.config.ShowRunningApps, func(v bool) {
		app.config.ShowRunningApps = v
		app.Apply()
	}), 1, row, 1, 1)
	addLabelRow(visGrid, row, "Show running (unpinned) applications")
	row++

	visGrid.Attach(switchWidget(app.config.ShowRecentApps, func(v bool) {
		app.config.ShowRecentApps = v
		app.Apply()
	}), 1, row, 1, 1)
	addLabelRow(visGrid, row, "Show recent applications (reserved; default off)")
	row++

	page.PackStart(visGrid, false, false, 0)

	page.PackStart(sectionTitle("SmartView"), false, false, 0)
	smartGrid := grid()

	srow := 0
	smartGrid.Attach(switchWidget(app.config.SmartView, func(v bool) {
		app.config.SmartView = v
		app.Apply()
	}), 1, srow, 1, 1)
	addLabelRow(smartGrid, srow, "Hide under windows until the cursor touches the edge")
	srow++

	hideRow, _ := intScale(float64(app.config.AutoHideDelay), 0, 5000, 10, func(v int) {
		app.config.AutoHideDelay = v
		app.Apply()
	})
	addRow(smartGrid, srow, "Hide delay", hideRow, "ms")
	srow++

	followGrid := smartGrid

	followGrid.Attach(switchWidget(app.config.FollowMouse, func(v bool) {
		app.config.FollowMouse = v
		app.Apply()
	}), 1, srow, 1, 1)
	addLabelRow(followGrid, srow, "Follow the cursor across monitors")
	srow++

	followAnimRow, _ := intScale(float64(app.config.FollowAnimMs), 0, 2000, 5, func(v int) {
		app.config.FollowAnimMs = v
		app.Apply()
	})
	addRow(followGrid, srow, "Monitor-move animation", followAnimRow, "ms")
	srow++

	page.PackStart(smartGrid, false, false, 0)

	page.PackStart(sectionTitle("Click behavior"), false, false, 0)
	clickGrid := grid()

	crow := 0
	clickGrid.Attach(comboWidget(
		[]string{"launch", "focus", "minimize", "show", "cycle", "none"},
		app.config.ClickAction,
		func(v string) {
			app.config.ClickAction = v
			app.Apply()
		}), 1, crow, 1, 1)
	addLabelRow(clickGrid, crow, "Click on running, unfocused app")
	crow++

	clickGrid.Attach(comboWidget(
		[]string{"minimize", "show", "cycle", "none"},
		app.config.ClickActionFocused,
		func(v string) {
			app.config.ClickActionFocused = v
			app.Apply()
		}), 1, crow, 1, 1)
	addLabelRow(clickGrid, crow, "Click on focused app")
	crow++

	clickGrid.Attach(comboWidget(
		[]string{"launch", "none"},
		app.config.MiddleClickAction,
		func(v string) {
			app.config.MiddleClickAction = v
			app.Apply()
		}), 1, crow, 1, 1)
	addLabelRow(clickGrid, crow, "Middle click")
	crow++

	page.PackStart(clickGrid, false, false, 0)
	return page
}

func newLauncherPage(app *App) gtk.IWidget {
	page := verticalPageBox()
	page.PackStart(sectionTitle("Launcher"), false, false, 0)
	launcherGrid := grid()

	lrow := 0
	launcherGrid.Attach(entryWidget(app.config.LauncherCommand, func(v string) {
		app.config.LauncherCommand = v
		app.Apply()
	}), 1, lrow, 1, 1)
	addLabelRow(launcherGrid, lrow, "Launcher command")
	lrow++

	launcherGrid.Attach(comboWidget([]string{"start", "end"}, app.config.LauncherPosition, func(v string) {
		app.config.LauncherPosition = v
		app.Apply()
	}), 1, lrow, 1, 1)
	addLabelRow(launcherGrid, lrow, "Launcher position")
	lrow++

	page.PackStart(launcherGrid, false, false, 0)

	return page
}

// addLabelRow puts a right-aligned description label in a plain grid row.
func addLabelRow(grid *gtk.Grid, row int, text string) {
	label, _ := gtk.LabelNew(text)
	label.SetHAlign(gtk.ALIGN_START)
	label.SetXAlign(0)
	label.SetLineWrap(true)
	label.SetWidthChars(18)
	label.SetMaxWidthChars(22)
	grid.Attach(label, 0, row, 1, 1)
	if control, err := grid.GetChildAt(1, row); err == nil && control != nil {
		control.ToWidget().SetHExpand(true)
	}
}
