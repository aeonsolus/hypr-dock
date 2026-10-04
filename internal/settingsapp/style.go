package settingsapp

import (
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

// Scoped to the preferences window: the dock and desktop retain their styles.
const preferencesCSS = `
window#preferences { background: #f5f5f7; color: #242426; }
#preferences * { font-family: "Inter", "Helvetica Neue", sans-serif; font-size: 13px; text-shadow: none; }
#preferences headerbar { background: #ececee; color: #242426; border-bottom: 1px solid #d5d5d9; box-shadow: none; }
#preferences #sidebar { background: #e9e9ed; padding: 20px 12px; border-right: 1px solid #d5d5da; }
#preferences #brand { font-size: 20px; font-weight: 700; }
#preferences #nav-item { background: transparent; background-image: none; border: none; box-shadow: none; color: #39393e; border-radius: 8px; padding: 10px 12px; margin: 2px 0; }
#preferences #nav-item:hover { background: #dddde3; }
#preferences #nav-item:checked { background: #007aff; color: white; }
#preferences #navigation { background: #e1e1e7; border: 1px solid #d1d1d7; border-radius: 9px; padding: 3px; }
#preferences #page-title { font-size: 24px; font-weight: 700; }
#preferences #section-title { font-size: 13px; font-weight: 600; color: #4c4c52; margin: 0; }
#preferences #hint { color: #77777f; font-size: 12px; }
#preferences #settings-group, #preferences #pin-row { background: white; border: 1px solid #dfdfe4; border-radius: 8px; padding: 8px; }
#preferences button { background: white; background-image: none; color: #29292f; border: 1px solid #cfcfd6; border-radius: 6px; padding: 4px 8px; box-shadow: 0 1px 2px rgba(0,0,0,0.06); }
#preferences button:hover { background: #f0f0f5; }
#preferences button:disabled { color: #92929a; }
#preferences entry, #preferences textview text { background: white; color: #29292f; caret-color: #007aff; }
#preferences entry { border: 1px solid #d1d1d8; border-radius: 6px; padding: 4px; box-shadow: none; }
#preferences scale trough { background: #d8d8df; min-height: 4px; border: none; border-radius: 4px; }
#preferences scale highlight { background: #007aff; border: none; }
#preferences scale slider { background: white; border: 1px solid #c7c7cf; min-width: 16px; min-height: 16px; box-shadow: 0 1px 3px rgba(0,0,0,0.15); }
#preferences switch { background: #d1d1d7; border: none; border-radius: 12px; }
#preferences switch:checked { background: #34c759; }
#preferences switch slider { background: white; border: none; }
#preferences #status-bar { background: #eeeef1; padding: 10px 18px; border-top: 1px solid #dddde3; }
#preferences separator { background: #dddde3; min-height: 1px; }
#preferences scrolledwindow, #preferences viewport { background: transparent; }
#preferences.dark { background: #242426; color: #f3f3f5; }
#preferences.dark headerbar { background: #303033; color: #f3f3f5; border-color: #454549; }
#preferences.dark #navigation { background: #38383c; border-color: #48484e; }
#preferences.dark #nav-item { color: #eeeef0; }
#preferences.dark #nav-item:hover { background: #48484e; }
#preferences.dark #nav-item:checked { background: #0a84ff; color: white; }
#preferences.dark #section-title { color: #c7c7cc; }
#preferences.dark #hint { color: #a0a0a8; }
#preferences.dark #settings-group, #preferences.dark #pin-row { background: #303033; border-color: #454549; }
#preferences.dark button { background: #414145; color: #f3f3f5; border-color: #55555c; }
#preferences.dark button:hover { background: #505057; }
#preferences.dark entry, #preferences.dark textview text { background: #1d1d20; color: #f3f3f5; border-color: #505057; }
#preferences.dark #status-bar { background: #303033; border-color: #454549; }
#preferences.dark separator { background: #454549; }
#preferences.dark scale trough { background: #55555c; }
`

func installPreferencesStyle() error {
	provider, err := gtk.CssProviderNew()
	if err != nil {
		return err
	}
	if err = provider.LoadFromData(preferencesCSS); err != nil {
		return err
	}
	screen, err := gdk.ScreenGetDefault()
	if err != nil {
		return err
	}
	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_USER)
	return nil
}
