package settingsapp

import (
	"os"
	"strings"

	"github.com/gotk3/gotk3/gtk"

	"hypr-dock/internal/diag"
	"hypr-dock/internal/version"
)

// AdvancedPage — blacklist, CSS editor, diagnostics, section resets.

func newAdvancedPage(app *App) gtk.IWidget {
	page := verticalPageBox()
	page.PackStart(newMaintenanceContent(app), false, false, 0)
	return pageScroll(page)
}

func newStylesheetEditor(app *App) gtk.IWidget {
	cssSection, _ := gtk.ExpanderNew("Custom stylesheet")
	cssContent := verticalPageBox()
	cssContent.PackStart(hintLabel("Advanced theme editing. Apply saves the stylesheet to disk."), false, false, 0)

	editing, _ := gtk.TextViewNew()
	buffer, _ := editing.GetBuffer()

	if css, err := os.ReadFile(app.config.ThemeStyle); err == nil {
		buffer.SetText(string(css))
	}
	editing.SetVExpand(true)

	cssButtons, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	applyCss, _ := gtk.ButtonNewWithLabel("Apply CSS")
	applyCss.Connect("clicked", func() {
		start, end := buffer.GetBounds()
		text, _ := buffer.GetText(start, end, true)
		if err := os.WriteFile(app.config.ThemeStyle, []byte(text), 0644); err != nil {
			app.SetStatus("CSS write failed: " + err.Error())
			return
		}
		app.Apply()
	})

	cssButtons.PackStart(applyCss, false, false, 0)
	editing.SetSizeRequest(-1, 180)
	cssContent.PackStart(editing, true, true, 0)
	cssContent.PackStart(cssButtons, false, false, 0)
	cssSection.Add(cssContent)
	return cssSection
}

func newMaintenanceContent(app *App) gtk.IWidget {
	page := verticalPageBox()
	page.PackStart(sectionTitle("HyprDock+ updater"), false, false, 0)
	page.PackStart(hintLabel("Checks your private GitHub main branch. Downloads and verifies the prebuilt binaries for newer versions, preserving your preferences."), false, false, 0)
	updateButton, _ := gtk.ButtonNewWithLabel("Install update")
	app.updateButton = updateButton
	updateCheck, _ := gtk.ButtonNewWithLabel("Check for updates")
	app.updateCheck = updateCheck
	updateStatus, _ := gtk.LabelNew("")
	updateStatus.SetHAlign(gtk.ALIGN_START)
	updateStatus.SetLineWrap(true)
	updateStatus.SetMaxWidthChars(65)
	app.updateStatus = updateStatus
	page.PackStart(updateStatus, false, false, 0)
	updateCheck.Connect("clicked", app.checkUpdates)
	updateButton.Connect("clicked", app.installUpdate)
	updateActions, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	updateActions.PackStart(updateCheck, false, false, 0)
	updateActions.PackStart(updateButton, false, false, 0)
	page.PackStart(updateActions, false, false, 0)
	app.refreshUpdater()

	diagnostics, _ := gtk.ExpanderNew("Diagnostics")
	diagnosticContent := verticalPageBox()

	diagLabel, _ := gtk.LabelNew("")
	diagLabel.SetHAlign(gtk.ALIGN_START)
	diagLabel.SetMarkup("<small>" + strings.TrimSpace(diagSummary()) + "</small>")
	diagLabel.SetHExpand(true)
	diagLabel.SetLineWrap(true)
	diagLabel.SetMaxWidthChars(70)

	diagButtons, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	diagRefresh, _ := gtk.ButtonNewWithLabel("Refresh diagnostics")
	diagRefresh.Connect("clicked", func() {
		diagLabel.SetMarkup("<small>" + strings.TrimSpace(diagSummary()) + "</small>")
	})
	diagButtons.PackStart(diagRefresh, false, false, 0)

	diagnosticContent.PackStart(diagLabel, false, false, 0)
	diagnosticContent.PackStart(diagButtons, false, false, 0)
	diagnostics.Add(diagnosticContent)
	page.PackStart(diagnostics, false, false, 0)

	page.PackStart(sectionTitle("Reset"), false, false, 0)
	resetGrid := grid()

	rrow := 0
	resetButton := func(label, section string) {
		button, _ := gtk.ButtonNewWithLabel(label)
		button.Connect("clicked", func() {
			app.config.ResetSection(section)
			app.Apply()
			app.rebuild()
			app.SetStatus("Reset " + section)
		})
		resetGrid.Attach(button, rrow%2, rrow/2, 1, 1)
		rrow++
	}

	resetButton("Reset Appearance", "Appearance")
	resetButton("Reset Behavior", "General")
	resetButton("Reset Preview", "General.preview")
	resetButton("Reset Displays", "Displays")
	resetButton("Reset Theme", "Theme")
	resetButton("Reset Theme.preview", "Theme.preview")

	page.PackStart(resetGrid, false, false, 0)

	return page
}

// diagSummary runs the doctor checks for the diagnostics label.
func diagSummary() string {
	return diag.Report(diagVersion)
}

// diagVersion mirrors internal/version to avoid another import here.
var diagVersion = version.Version
