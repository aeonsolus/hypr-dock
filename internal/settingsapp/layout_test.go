package settingsapp

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/hashicorp/go-hclog"
	"hypr-dock/internal/pkg/conf"
	"hypr-dock/internal/settings"
)

// Run explicitly on a workstation to verify real GTK size negotiation without
// mapping a window or modifying the user's preferences.
func TestPreferencesFit(t *testing.T) {
	if os.Getenv("HYPR_DOCK_GTK_TESTS") != "1" {
		t.Skip("opt-in GTK layout check")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	gtk.Init(nil)
	if err := installPreferencesStyle(); err != nil {
		t.Fatal(err)
	}
	window, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		t.Fatal(err)
	}
	defer window.Destroy()
	dir := t.TempDir()
	a := &App{window: window, log: hclog.NewNullLogger(), config: &settings.Settings{Config: conf.Defaults(), ConfigPath: filepath.Join(dir, "dock.conf"), OrderPath: filepath.Join(dir, "order"), PinnedPath: filepath.Join(dir, "pins"), ThemesDir: dir}}
	a.build()
	for _, source := range []gtk.IWidget{newBehaviorPage(a), newWindowsPage(a), newDockLayoutPage(a)} {
		box := source.(*gtk.Box)
		for l := box.GetChildren(); l != nil && l.Data() != nil; l = l.Next() {
			w := l.Data().(gtk.IWidget)
			name, _ := w.ToWidget().GetName()
			if name != "settings-group" {
				continue
			}
			container := &gtk.Container{Widget: *w.ToWidget()}
			cells := map[[2]int]bool{}
			for c := container.GetChildren(); c != nil && c.Data() != nil; c = c.Next() {
				child := c.Data().(gtk.IWidget)
				x, err := container.ChildGetProperty(child, "left-attach", glib.TYPE_INT)
				if err != nil {
					t.Fatal(err)
				}
				y, err := container.ChildGetProperty(child, "top-attach", glib.TYPE_INT)
				if err != nil {
					t.Fatal(err)
				}
				cell := [2]int{x.(int), y.(int)}
				if cells[cell] {
					t.Errorf("overlapping controls at grid cell %v", cell)
				}
				cells[cell] = true
			}
		}
		box.Destroy()
	}
	child, _ := window.GetChild()
	child.ToWidget().ShowAll()
	for _, dark := range []bool{false, true} {
		a.dark = dark
		a.setTheme()
		for _, name := range []string{"Appearance", "Behavior", "Applications", "Maintenance"} {
			a.stack.SetVisibleChildName(name)
			page, _ := a.stack.GetVisibleChild()
			width, _ := page.ToWidget().GetPreferredWidth()
			height, _ := page.ToWidget().GetPreferredHeightForWidth(948)
			t.Logf("%s dark=%t minimum=%dx%d", name, dark, width, height)
			if width > 948 || height > 610 {
				t.Errorf("%s exceeds compact content area: %dx%d", name, width, height)
			}
			rootHeight, _ := child.ToWidget().GetPreferredHeightForWidth(980)
			t.Logf("%s complete content height=%d", name, rootHeight)
			if rootHeight > 710 {
				t.Errorf("%s complete window content too tall: %d", name, rootHeight)
			}
		}
	}
}
