package settingsapp

import (
	"os"
	"os/exec"
	"strings"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	"hypr-dock/internal/diag"
)

// AdvancedPage — blacklist, CSS editor, diagnostics, section resets.

func newAdvancedPage(app *App) gtk.IWidget {
	page := verticalPageBox()

	page.PackStart(sectionTitle("Hidden applications"), false, false, 0)
	page.PackStart(hintLabel("Window classes that never get a dock icon. Comma-separated; * wildcards allowed (e.g. scratch-*)."), false, false, 0)
	page.PackStart(entryWidget(app.config.HiddenApps, func(v string) {
		app.config.HiddenApps = v
		app.Apply()
	}), false, false, 0)

	page.PackStart(sectionTitle("Theme CSS"), false, false, 0)
	page.PackStart(hintLabel("Edits "+app.config.ThemeStyle+" directly. Applied live; reverts on Reload from disk until saved."), false, false, 0)

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
	page.PackStart(editing, true, true, 0)
	page.PackStart(cssButtons, false, false, 0)

	page.PackStart(sectionTitle("HyprDock+ updater"), false, false, 0)
	page.PackStart(hintLabel("Pulls aeonsolus/hypr-dock using your authenticated gh account, rebuilds the four binaries, and installs them."), false, false, 0)
	updateButton, _ := gtk.ButtonNewWithLabel("Check for updates and install")
	updateButton.Connect("clicked", func() {
		updateButton.SetSensitive(false)
		app.SetStatus("Updating HyprDock+…")
		go func() {
			cmd := exec.Command("sh", "-c", `
set -eu
repo=/tmp/hyprdock-update
if [ -d "$repo/.git" ]; then
  git -C "$repo" fetch origin main
  git -C "$repo" reset --hard origin/main
else
  rm -rf "$repo"
  gh repo clone aeonsolus/hypr-dock "$repo" -- --depth=1
fi
cd "$repo"
make build
pkexec cp bin/hypr-dock bin/hypr-dock-settings bin/hypr-dockctl /usr/bin/
`)
			err := cmd.Run()
			glib.IdleAdd(func() {
				updateButton.SetSensitive(true)
				if err != nil {
					app.SetStatus("Update failed: " + err.Error())
					return
				}
				app.SetStatus("Updated. Restart HyprDock+ to apply the new binary.")
			})
		}()
	})
	page.PackStart(updateButton, false, false, 0)

	page.PackStart(sectionTitle("Diagnostics"), false, false, 0)

	diagLabel, _ := gtk.LabelNew("")
	diagLabel.SetHAlign(gtk.ALIGN_START)
	diagLabel.SetMarkup("<small>" + strings.TrimSpace(diagSummary()) + "</small>")
	diagLabel.SetHExpand(true)

	diagButtons, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	diagRefresh, _ := gtk.ButtonNewWithLabel("Refresh diagnostics")
	diagRefresh.Connect("clicked", func() {
		diagLabel.SetMarkup("<small>" + strings.TrimSpace(diagSummary()) + "</small>")
	})
	diagButtons.PackStart(diagRefresh, false, false, 0)

	page.PackStart(diagLabel, false, false, 0)
	page.PackStart(diagButtons, false, false, 0)

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
		resetGrid.Attach(button, rrow%4, rrow/4, 1, 1)
		rrow++
	}

	resetButton("Reset Appearance", "Appearance")
	resetButton("Reset Behavior", "General")
	resetButton("Reset Preview", "General.preview")
	resetButton("Reset Displays", "Displays")
	resetButton("Reset Theme", "Theme")
	resetButton("Reset Theme.preview", "Theme.preview")

	page.PackStart(resetGrid, false, false, 0)

	return pageScroll(page)
}

// diagSummary runs the doctor checks for the diagnostics label.
func diagSummary() string {
	return diag.Report(diagVersion)
}

// diagVersion mirrors internal/version to avoid another import here.
const diagVersion = "1.3.0-custom"
