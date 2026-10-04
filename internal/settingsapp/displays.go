package settingsapp

import (
	"github.com/gotk3/gotk3/gtk"
	"hypr-dock/pkg/ipc"
)

// DisplaysPage — monitor targeting. The current architecture hosts one layer
// surface, so per-monitor layouts are a future extension; "all" stays
// disabled with an explanatory note.

func newDisplaysPage(app *App) gtk.IWidget {
	page := verticalPageBox()

	page.PackStart(sectionTitle("Display"), false, false, 0)

	modeCombo := comboWidget([]string{"focused", "primary", "specific", "all"}, app.config.Displays.Mode, func(v string) {
		app.config.Displays.Mode = v
		app.Apply()
	})

	gridBox := grid()
	addRow(gridBox, 0, "Show the dock on", modeCombo, "")

	// Monitor list for the "specific" mode.
	monitorNames := []string{""}
	if monitors, err := ipc.GetMonitors(); err == nil {
		for _, mon := range monitors {
			monitorNames = append(monitorNames, mon.Name)
		}
	}

	monitorCombo := comboWidget(monitorNames, app.config.Displays.MonitorName, func(v string) {
		app.config.Displays.MonitorName = v
		app.config.Displays.Mode = "specific"
		app.Apply()
	})
	addRow(gridBox, 1, "Specific monitor", monitorCombo, "applies when mode is “specific”")

	page.PackStart(gridBox, false, false, 0)

	page.PackStart(hintLabel("“all displays” needs multiple dock instances and is reserved for a future release. With FollowMouse enabled the dock tracks the cursor and overrides this selection."), false, false, 0)

	return page
}
