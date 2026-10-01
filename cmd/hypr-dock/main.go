package main

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/allan-simon/go-singleinstance"
	"github.com/gotk3/gotk3/gtk"

	"hypr-dock/internal/app"
	"hypr-dock/internal/follow"
	"hypr-dock/internal/hypr/hyprEvents"
	"hypr-dock/internal/layering"
	"hypr-dock/internal/pkg/flags"
	"hypr-dock/internal/pkg/signals"
	"hypr-dock/internal/pkg/utils"
	"hypr-dock/internal/settings"
	"hypr-dock/internal/state"
)

func main() {
	signals.Handler()

	lockFilePath := fmt.Sprintf("%s/hypr-dock-%s.lock", utils.TempDir(), os.Getenv("USER"))
	lockFile, err := singleinstance.CreateLockFile(lockFilePath)
	if err != nil {
		file, readErr := utils.LoadTextFile(lockFilePath)
		pidInt := 0
		if readErr == nil && len(file) > 0 {
			pidInt, _ = strconv.Atoi(file[0])
		}

		// A live owner means this invocation is the toggle command.
		if pidInt > 1 && syscall.Kill(pidInt, 0) == nil {
			syscall.Kill(pidInt, syscall.SIGUSR1)
			os.Exit(0)
		}

		// The previous owner is gone; recover from the stale lock instead of
		// silently exiting forever after a crash or forced termination.
		_ = os.Remove(lockFilePath)
		lockFile, err = singleinstance.CreateLockFile(lockFilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "hypr-dock: unable to acquire instance lock: %v\n", err)
			os.Exit(1)
		}
	}
	defer lockFile.Close()

	// flags
	flags := flags.Get()

	logger := utils.СreateLogger(flags.LogLevel)

	// window build
	settings, err := settings.Init(flags, logger)
	if err != nil {
		logger.Error("Settings init error:", "err", err)
	}

	gtk.Init(nil)

	appState := state.New(settings, logger)

	window, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		logger.Error("Unable to create window:", "err", err)
		os.Exit(2)
	}
	appState.SetWindow(window)

	// Without app-paintable GTK draws its theme background on the window
	// before CSS runs, so `window { background-color: transparent }` can't
	// win — the dock reads as an opaque slab no matter the panel alpha.
	// (Upstream issue #41.) Because the CSS provider runs at application
	// priority and GTK3 can't parse !important, also force the window's
	// background RGBA directly — this beats the theme without touching CSS.
	window.SetAppPaintable(true)

	window.SetTitle("hypr-dock")

	layerctl := layering.NewInit(window, settings)
	appState.SetLayerctl(layerctl)

	// With follow enabled the dock starts off-screen and slideIn() animates
	// it in exactly once; without this it would show statically at rest first
	// and then re-slide (~180ms later) — the double appear animation.
	if settings.FollowMouse {
		follow.PreHide(window, layerctl)
	}

	err = utils.AddCssProvider(settings.ThemeStyle)
	if err != nil {
		logger.Warn("CSS file not found, the default GTK theme is running!", "err", err)
	}

	app := app.BuildApp(appState)

	window.Add(app)
	window.Connect("destroy", func() { gtk.MainQuit() })
	window.ShowAll()

	// post
	hyprEvents.Init(appState)
	follow.Init(appState)

	// end
	gtk.Main()
}
