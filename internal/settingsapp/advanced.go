package settingsapp

import (
	"os"
	"os/exec"
	"strings"

	"github.com/gotk3/gotk3/glib"
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
	page.PackStart(hintLabel("Pulls aeonsolus/hypr-dock using your authenticated gh account, rebuilds the four binaries, and installs them."), false, false, 0)
	updateButton, _ := gtk.ButtonNewWithLabel("Check for updates and install")
	app.updateButton = updateButton
	updateButton.SetSensitive(!app.updating)
	updateStatus, _ := gtk.LabelNew("Installed version: " + version.Version)
	updateStatus.SetHAlign(gtk.ALIGN_START)
	updateStatus.SetLineWrap(true)
	updateStatus.SetMaxWidthChars(65)
	app.updateStatus = updateStatus
	if app.updating {
		updateStatus.SetText("Downloading and building the latest version…")
	}
	page.PackStart(updateStatus, false, false, 0)
	updateButton.Connect("clicked", func() {
		if app.updating {
			return
		}
		app.updating = true
		updateButton.SetSensitive(false)
		app.SetStatus("Updating HyprDock+…")
		updateStatus.SetText("Downloading and building the latest version…")
		go func() {
			cmd := exec.Command("sh", "-c", `
set -eu
repo=$(mktemp -d)
trap 'rm -rf "$repo"' EXIT
gh repo clone aeonsolus/hypr-dock "$repo" -- --depth=1
cd "$repo"
make build
pkexec install -m 755 bin/hypr-dock bin/hypr-dock-settings bin/hypr-dockctl bin/hypr-alttab /usr/bin/
`)
			output, err := cmd.CombinedOutput()
			glib.IdleAdd(func() {
				app.updating = false
				app.updateButton.SetSensitive(true)
				if err != nil {
					app.SetStatus("Update failed: " + err.Error())
					app.updateStatus.SetText("Update failed: " + err.Error() + "\n" + string(output))
					return
				}
				app.SetStatus("Updated. Restart HyprDock+ to apply the new binary.")
				app.updateStatus.SetText("Installation complete. Restart the dock and preferences to use the new version.")
			})
		}()
	})
	page.PackStart(updateButton, false, false, 0)

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
