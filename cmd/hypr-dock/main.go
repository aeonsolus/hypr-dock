package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/allan-simon/go-singleinstance"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	"hypr-dock/internal/app"
	"hypr-dock/internal/ctl"
	"hypr-dock/internal/diag"
	"hypr-dock/internal/follow"
	"hypr-dock/internal/hypr/hyprEvents"
	"hypr-dock/internal/layering"
	"hypr-dock/internal/pkg/flags"
	"hypr-dock/internal/pkg/signals"
	"hypr-dock/internal/pkg/utils"
	"hypr-dock/internal/settings"
	"hypr-dock/internal/state"
	"hypr-dock/internal/version"
)

func main() {
	signals.Handler()

	// Subcommands that do not start a dock instance.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "doctor":
			diagRun()
			return
		case "settings", "--settings":
			if err := spawnSettings(); err != nil {
				fmt.Fprintf(os.Stderr, "hypr-dock: %v\n", err)
				os.Exit(1)
			}
			return
		case "version", "--version":
			fmt.Println("hypr-dock", version.Version)
			return
		}
	}

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
	parsedFlags := flags.Get()

	logger := utils.СreateLogger(parsedFlags.LogLevel)

	// window build
	settings, err := settings.Init(parsedFlags, logger)
	if err != nil {
		logger.Error("Settings init error:", "err", err)
	}

	gtk.Init(nil)

	appState := state.New(settings, logger)
	// Start the control plane as soon as state exists, so diagnostics and
	// settings remain available even if later GTK initialization is slow.
	go controlServer(appState)

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

	themeProvider, err := utils.AddCssProvider(settings.ThemeStyle)
	if err != nil {
		logger.Warn("CSS file not found, the default GTK theme is running!", "err", err)
	}
	appState.SetThemeProvider(themeProvider)
	app.ReplaceCss(appState)

	appBox := app.BuildApp(appState)
	appState.SetAppBox(appBox)

	window.Add(appBox)
	window.Connect("destroy", func() { gtk.MainQuit() })
	window.ShowAll()

	// post
	hyprEvents.Init(appState)
	follow.Init(appState)
	app.WatchConfigFiles(appState)

	// end
	gtk.Main()
}

// controlServer serves the control socket for hypr-dockctl and the settings
// application.
func controlServer(appState *state.State) {
	logger := utils.СreateLogger("debug")
	logger.Info("Control server goroutine started")

	handler := func(req ctl.Request) ctl.Response {
		switch req.Cmd {
		case ctl.CmdPing:
			return ctl.Response{Ok: true}

		case ctl.CmdApply:
			glib.IdleAdd(func() { app.ApplyFull(appState) })
			return ctl.Response{Ok: true}

		case ctl.CmdSet:
			s := appState.GetSettings()
			if s == nil {
				return ctl.Response{Ok: false, Error: "settings not ready"}
			}
			if err := s.SetByPath(req.Key, req.Value); err != nil {
				return ctl.Response{Ok: false, Error: err.Error()}
			}
			if err := s.Save(s.ConfigPath); err != nil {
				return ctl.Response{Ok: false, Error: err.Error()}
			}
			app.ScheduleApply(appState)
			return ctl.Response{Ok: true}

		case ctl.CmdStatus:
			data := map[string]string{
				"pid":     strconv.Itoa(os.Getpid()),
				"version": version.Version,
			}
			if s := appState.GetSettings(); s != nil {
				data["theme"] = s.CurrentTheme
				data["pinned"] = strconv.Itoa(len(s.PinnedApps))
				data["config"] = s.ConfigPath
			}
			return ctl.Response{Ok: true, Data: data}

		case ctl.CmdQuit:
			glib.IdleAdd(func() { gtk.MainQuit() })
			return ctl.Response{Ok: true}

		default:
			return ctl.Response{Ok: false, Error: "unknown command: " + req.Cmd}
		}
	}

	stop, err := ctl.Serve(handler)
	if err != nil {
		logger.Error("Control socket unavailable", "error", err)
		return
	}
	defer stop()

	// Keep this goroutine alive for the process lifetime: controlServer must
	// not return, because returning runs the deferred teardown above and
	// would immediately close the socket it just opened.
	for {
		time.Sleep(time.Hour)
	}
}

// spawnSettings launches the settings application detached from this process.
func spawnSettings() error {
	candidates := []string{}

	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates,
			filepath.Join(filepath.Dir(exe), "hypr-dock-settings"))
	}
	candidates = append(candidates, "hypr-dock-settings")

	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate); err == nil {
			cmd := exec.Command(candidate)
			cmd.Stdout = nil
			cmd.Stderr = nil
			if err := cmd.Start(); err != nil {
				return fmt.Errorf("unable to start %s: %w", candidate, err)
			}
			return nil
		}
	}

	return fmt.Errorf("hypr-dock-settings binary not found")
}

// diagRun executes the doctor diagnostics.
func diagRun() {
	fmt.Println(diag.Report(version.Version))
}
