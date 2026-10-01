package app

import (
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gotk3/gotk3/glib"

	"hypr-dock/internal/appearance"
	"hypr-dock/internal/follow"
	"hypr-dock/internal/hypr/hyprOpt"
	"hypr-dock/internal/omarchy"
	"hypr-dock/internal/pkg/utils"
	"hypr-dock/internal/settings"
	"hypr-dock/internal/state"
)

// Live configuration reload.
//
// ApplySet writes one setting through to disk and re-reads everything from
// disk, then re-applies. Re-reading keeps the file as the single source of
// truth: manual edits, the settings app and IPC all converge on the same
// engine.
//
// Relayout tears down and rebuilds the dock's widget tree (the layer surface
// itself survives), which is what makes icon size, spacing, position,
// orientation, launcher visibility etc. update live.

// ApplySet writes a single dotted key ("General.IconSize") to the config file
// and triggers a live apply. Returns an error for unknown keys or values.
func ApplySet(appState *state.State, key, value string) error {
	s := appState.GetSettings()
	if s == nil {
		return errNoSettings
	}

	if err := s.SetByPath(key, value); err != nil {
		return err
	}

	if err := s.Save(s.ConfigPath); err != nil {
		return err
	}

	ApplyFull(appState)
	return nil
}

type simpleError string

func (e simpleError) Error() string { return string(e) }

const errNoSettings = simpleError("settings not initialised")

// ApplyFull re-reads config + theme + pinned from disk and applies any
// changes live. Must run on the GTK main thread (callers use glib.IdleAdd).
func ApplyFull(appState *state.State) {
	log := appState.GetLogger()

	old := appState.GetSettings()
	if old == nil {
		log.Error("Reload requested before settings were ready")
		return
	}

	newSettings, err := settings.Load(old.ConfigPath, old.ConfigDir, old.LocalDir, log)
	if err != nil {
		log.Error("Reload: settings load failed", "error", err)
		return
	}

	// Diff for logging/categorisation (everything is rebuilt regardless).
	oldMap := old.MarshalMap()
	newMap := newSettings.MarshalMap()
	changed := make(map[string]bool)
	for section, keys := range newMap {
		prev, ok := oldMap[section]
		if !ok {
			changed[section] = true
			continue
		}
		for k, v := range keys {
			if prev[k] != v {
				changed[section] = true
			}
		}
	}

	appState.SetSettings(newSettings)

	// Stop any preview popup early: it holds widget references that are
	// about to be destroyed.
	appState.GetPV().Hide()

	// Layer geometry (anchor/layer/exclusive/smart zone) can all be updated
	// on the live layer surface.
	applyLayering(appState)

	// Rebuild the whole widget tree from the new settings.
	Relayout(appState)

	// Theme + generated overrides.
	ReplaceCss(appState)

	if len(changed) > 0 {
		log.Debug("Reloaded config", "sections", strings.Join(sortedKeys(changed), ","))
	}
}

// Relayout destroys the dock's current widget tree and rebuilds it from the
// active settings. The gtk window (layer surface) itself is preserved, so
// Hyprland keeps the surface's layer/anchors without a restart.
func Relayout(appState *state.State) {
	window := appState.GetWindow()
	if window == nil {
		return
	}

	if old := appState.GetAppBox(); old != nil {
		window.Remove(old)
		old.Destroy()
	}

	appBox := BuildApp(appState)
	appState.SetAppBox(appBox)
	window.Add(appBox)
	window.ShowAll()
}

// applyLayering pushes geometry settings into the live layer surface.
func applyLayering(appState *state.State) {
	layerctl := appState.GetLayerctl()
	if layerctl == nil {
		return
	}

	layerctl.SetSettings(appState.GetSettings())
	layerctl.SetPosition()
	layerctl.SetLayer()
	initMarginSafe(appState)
	follow.Update(appState)
}

// ReplaceCss swaps the theme stylesheet and the generated override provider.
func ReplaceCss(appState *state.State) {
	log := appState.GetLogger()
	s := appState.GetSettings()
	if s == nil {
		return
	}

	// Theme file.
	themeProvider, err := utils.AddCssProvider(s.ThemeStyle)
	if err == nil {
		utils.RemoveCssProvider(appState.GetThemeProvider())
		appState.SetThemeProvider(themeProvider)
	} else {
		log.Warn("CSS file not found, keeping previous theme", "err", err)
	}

	// Generated overrides (appearance settings + omarchy follow), at a
	// priority above the theme file so structured values win without
	// rewriting the user's style.css.
	override := BuildOverrideCss(s)
	utils.RemoveCssProvider(appState.GetOverrideProvider())
	appState.SetOverrideProvider(nil)

	if strings.TrimSpace(override) != "" {
		provider, err := utils.AddCssData(override, overridePriority)
		if err == nil {
			appState.SetOverrideProvider(provider)
		} else {
			log.Error("Failed to register appearance overrides", "error", err)
		}
	}
}

// ScheduleApply coalesces bursts of apply requests (slider drags, watcher
// storms) into one ApplyFull after the storm quiets down.
var (
	scheduleMu    sync.Mutex
	scheduleTimer *time.Timer
)

// ScheduleApply triggers one debounced ApplyFull on the GTK main thread.
func ScheduleApply(appState *state.State) {
	scheduleMu.Lock()
	defer scheduleMu.Unlock()

	if scheduleTimer != nil {
		scheduleTimer.Stop()
	}
	scheduleTimer = time.AfterFunc(80*time.Millisecond, func() {
		glib.IdleAdd(func() { ApplyFull(appState) })
	})
}

// WatchConfigFiles polls the config, theme, and pinned files so manual edits
// are picked up without a restart. Cheap: a stat per file per tick. Also
// follows the Omarchy theme when enabled.
func WatchConfigFiles(appState *state.State) {
	s := appState.GetSettings()
	if s == nil {
		return
	}

	paths := []string{s.ConfigPath, s.ThemeStyle, s.ThemeConf, s.PinnedPath}

	apply := func() {
		ScheduleApply(appState)
	}

	for _, path := range paths {
		if path == "" {
			continue
		}
		omarchy.NewWatcher(path, 2*time.Second, apply).Start()
	}

	// Omarchy follow: watch the active theme's colors unconditionally and
	// let the callback decide whether to act — FollowOmarchy can be turned
	// on at runtime, and re-arming watchers on every apply would stack them.
	if omarchy.Detect() {
		if colors := omarchy.ColorsPath(); colors != "" {
			omarchy.NewWatcher(colors, 3*time.Second, func() {
				glib.IdleAdd(func() {
					s := appState.GetSettings()
					if s == nil || !s.FollowOmarchy || s.CurrentTheme != "omarchy" {
						return
					}
					pal, err := omarchy.ReadPalette()
					if err != nil {
						return
					}
					if err := omarchy.RegenerateTheme(s.ThemesDir+"/omarchy", pal); err == nil {
						ApplyFull(appState)
					}
				})
			}).Start()
		}
	}
}

const overridePriority = 900

// BuildOverrideCss computes the generated CSS for the current settings.
// Exposed for tests.
func BuildOverrideCss(s *settings.Settings) string {
	var pal *appearance.Palette

	if s.FollowOmarchy && s.CurrentTheme == "omarchy" && omarchy.Detect() {
		if p, err := omarchy.ReadPalette(); err == nil {
			pal = &p
		}
	}

	themeCSS := ""
	if data, err := os.ReadFile(s.ThemeStyle); err == nil {
		themeCSS = string(data)
	}

	return appearance.Override(
		themeCSS,
		s.Appearance.PanelOpacity,
		s.Appearance.BorderRadius,
		s.Appearance.BorderWidth,
		s.Appearance.PanelPadding,
		s.Appearance.HoverEffects,
		s.Appearance.ActiveTint,
		s.Appearance.Accent,
		pal,
	)
}

// gapWatchOnce guards the Hyprland gap-change listener so margin reloads
// don't stack duplicate listeners.
var gapWatchOnce sync.Once

// initMarginSafe re-applies the dock's outer margin from settings (system gap
// or fixed), registering the Hyprland gap-change listener exactly once.
func initMarginSafe(appState *state.State) {
	settings := appState.GetSettings()
	app := appState.GetAppBox()
	if app == nil {
		return
	}

	position := settings.Position

	if settings.SystemGapUsed {
		margin, err := hyprOpt.GetGap()
		if err == nil {
			setMargin(app, position, margin...)
		} else {
			appState.GetLogger().Error("Failed to get gaps, indent set from settings", "error", err)
			setMargin(app, position, settings.Margin)
		}

		gapWatchOnce.Do(func() {
			hyprOpt.GapChangeEvent(func(gaps []int) {
				glib.IdleAdd(func() {
					if s := appState.GetSettings(); s != nil && s.SystemGapUsed && appState.GetAppBox() != nil {
						setMargin(appState.GetAppBox(), s.Position, gaps...)
					}
				})
			})
		})
	} else {
		setMargin(app, position, settings.Margin)
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
