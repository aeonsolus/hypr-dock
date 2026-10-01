package diag

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"hypr-dock/internal/ctl"
	"hypr-dock/internal/desktop"
	"hypr-dock/internal/omarchy"
	"hypr-dock/internal/pkg/conf"
	"hypr-dock/internal/pkg/pinned"
	"hypr-dock/pkg/ipc"
)

// Check is one doctor finding.
type Check struct {
	Name   string
	Status string // ok / warn / fail
	Detail string
}

// Report runs the diagnostics and renders them as plain text.
func Report(version string) string {
	checks := Run(version)

	var b strings.Builder
	for _, check := range checks {
		marker := "•"
		switch check.Status {
		case "ok":
			marker = "✔"
		case "warn":
			marker = "!"
		case "fail":
			marker = "✘"
		}
		fmt.Fprintf(&b, "%s %-28s %s\n", marker, check.Name, check.Detail)
	}
	return b.String()
}

func ok(name, detail string) Check   { return Check{name, "ok", detail} }
func warn(name, detail string) Check { return Check{name, "warn", detail} }
func fail(name, detail string) Check { return Check{name, "fail", detail} }

// Run executes all diagnostics. No GTK is involved: safe from scripts and
// cron jobs.
func Run(version string) []Check {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "hypr-dock")
	configPath := filepath.Join(configDir, "hypr-dock.conf")
	localDir := filepath.Join(home, ".local", "share", "hypr-dock")
	themesDir := filepath.Join(configDir, "themes")
	pinnedPath := filepath.Join(localDir, "pinned")

	checks := []Check{ok("version", version)}

	// Hyprland presence.
	sig := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")
	if sig == "" {
		checks = append(checks, warn("Hyprland", "HYPRLAND_INSTANCE_SIGNATURE not set (run inside a Hyprland session)"))
	} else {
		sock := filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "hypr", sig, ".socket.sock")
		if _, err := os.Stat(sock); err == nil {
			checks = append(checks, ok("Hyprland", "socket "+sock))
		} else {
			checks = append(checks, fail("Hyprland", "instance socket missing: "+sock))
		}
	}

	// gtk-layer-shell: build-time dependency.
	checks = append(checks, ok("gtk-layer-shell", "built in (gotk3-layershell)"))

	// Config.
	config, err := conf.New(configPath, themesDir, nil)
	if err != nil {
		checks = append(checks, fail("config", configPath+" — "+err.Error()))
	} else {
		detail := fmt.Sprintf("%s (theme: %s)", configPath, config.CurrentTheme)
		checks = append(checks, ok("config", detail))

		themeDir := config.ThemeDir
		if _, err := os.Stat(themeDir); err != nil {
			checks = append(checks, fail("theme", "dir missing: "+themeDir))
		} else {
			style := filepath.Join(themeDir, "style.css")
			if _, err := os.Stat(style); err != nil {
				checks = append(checks, warn("theme", "style.css missing: "+style))
			} else {
				checks = append(checks, ok("theme", themeDir))
			}
		}

		if problems := config.Validate(); len(problems) > 0 {
			checks = append(checks, warn("validation", strings.Join(problems, "; ")))
		} else {
			checks = append(checks, ok("validation", "no problems"))
		}
	}

	// Pinned.
	pins, err := pinned.Open(pinnedPath)
	if err != nil {
		checks = append(checks, fail("pinned", pinnedPath+" — "+err.Error()))
	} else {
		checks = append(checks, ok("pinned", fmt.Sprintf("%s (%d apps: %s)", pinnedPath, len(pins), strings.Join(pins, ", "))))
	}

	// Monitors.
	if monitors, err := ipc.GetMonitors(); err != nil {
		checks = append(checks, fail("monitors", err.Error()))
	} else {
		names := make([]string, 0, len(monitors))
		for _, mon := range monitors {
			names = append(names, fmt.Sprintf("%s (%dx%d)", mon.Name, mon.Width, mon.Height))
		}
		checks = append(checks, ok("monitors", strings.Join(names, ", ")))
	}

	// Desktop entries.
	checks = append(checks, ok("desktop entries", strconv.Itoa(len(desktop.GetFiles()))+" indexed via StartupWMClass"))

	// Omarchy.
	if omarchy.Detect() {
		path := omarchy.ColorsPath()
		if path != "" {
			checks = append(checks, ok("omarchy", "detected; colors: "+path))
		} else {
			checks = append(checks, warn("omarchy", "detected but no colors.toml found"))
		}
	} else {
		checks = append(checks, ok("omarchy", "not present"))
	}

	// Control IPC + running dock.
	socket := ctl.SocketPath()
	if _, err := os.Stat(socket); err != nil {
		checks = append(checks, warn("control ipc", "socket not found: "+socket+" (dock not running?)"))
	} else if resp, err := ctl.Send(ctl.Request{Cmd: ctl.CmdStatus}); err == nil && resp.Ok {
		detail := fmt.Sprintf("dock pid %s", resp.Data["pid"])
		if theme, ok := resp.Data["theme"]; ok {
			detail += ", theme " + theme
		}
		checks = append(checks, ok("control ipc", detail))
	} else {
		checks = append(checks, warn("control ipc", "socket exists but no response: "+err.Error()))
	}

	// Lock file owner.
	lockPath := filepath.Join(os.TempDir(), fmt.Sprintf("hypr-dock-%s.lock", os.Getenv("USER")))
	if data, err := os.ReadFile(lockPath); err == nil && len(data) > 0 {
		pid, err := strconv.Atoi(strings.TrimSpace(strings.Split(string(data), "\n")[0]))
		if err == nil && pid > 1 {
			if _, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid))); err == nil {
				checks = append(checks, ok("dock process", "running, pid "+strconv.Itoa(pid)))
			} else {
				checks = append(checks, warn("dock process", "stale lock, pid "+strconv.Itoa(pid)+" dead"))
			}
		}
	} else {
		checks = append(checks, warn("dock process", "not running"))
	}

	// Uptime of the running dock via socket status is unknowable; keep the
	// report time for correlation.
	_ = time.Now()

	return checks
}
