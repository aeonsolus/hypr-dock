package settingsapp

import (
	"github.com/gotk3/gotk3/gtk"
)

// WindowsPage — preview popups and window indicators.

func newWindowsPage(app *App) gtk.IWidget {
	page := verticalPageBox()

	page.PackStart(sectionTitle("Window Indicators"), false, false, 0)
	indGrid := grid()

	row := 0
	indGrid.Attach(switchWidget(app.config.ShowWindowCount, func(v bool) {
		app.config.ShowWindowCount = v
		app.Apply()
	}), 1, row, 1, 1)
	addLabelRow(indGrid, row, "Show running indicator dots")
	row++

	indRow, _ := intScale(float64(app.config.IndicatorSize), 20, 100, 1, func(v int) {
		app.config.IndicatorSize = v
		app.Apply()
	})
	addRow(indGrid, row, "Indicator size", indRow, "% of icon size")
	row++

	page.PackStart(indGrid, false, false, 0)

	page.PackStart(sectionTitle("Window Previews"), false, false, 0)
	pvGrid := grid()

	prow := 0
	pvGrid.Attach(comboWidget([]string{"none", "static", "live"}, app.config.Preview.Mode, func(v string) {
		app.config.Preview.Mode = v
		app.Apply()
	}), 1, prow, 1, 1)
	addLabelRow(pvGrid, prow, "Preview mode")
	prow++

	fpsRow, _ := intScale(float64(app.config.Preview.FPS), 1, 120, 1, func(v int) {
		app.config.Preview.FPS = v
		app.Apply()
	})
	addRow(pvGrid, prow, "Live preview FPS", fpsRow, "")
	prow++

	bufferRow, _ := intScale(float64(app.config.Preview.BufferSize), 1, 20, 1, func(v int) {
		app.config.Preview.BufferSize = v
		app.Apply()
	})
	addRow(pvGrid, prow, "Stream buffer", bufferRow, "frames")
	prow++

	showRow, _ := intScale(float64(app.config.Preview.ShowDelay), 0, 5000, 10, func(v int) {
		app.config.Preview.ShowDelay = v
		app.Apply()
	})
	addRow(pvGrid, prow, "Show delay", showRow, "ms")
	prow++

	hideRow, _ := intScale(float64(app.config.Preview.HideDelay), 0, 5000, 10, func(v int) {
		app.config.Preview.HideDelay = v
		app.Apply()
	})
	addRow(pvGrid, prow, "Hide delay", hideRow, "ms")
	prow++

	moveRow, _ := intScale(float64(app.config.Preview.MoveDelay), 0, 2000, 10, func(v int) {
		app.config.Preview.MoveDelay = v
		app.Apply()
	})
	addRow(pvGrid, prow, "Move delay", moveRow, "ms")
	prow++

	page.PackStart(pvGrid, false, false, 0)

	page.PackStart(hintLabel("static shows the last window frame; live streams window contents (experimental). Window capture happens only while a thumbnail is displayed."), false, false, 0)

	return pageScroll(page)
}
