package terminal

import (
	"os"
	"path/filepath"
	"strings"

	"hypr-dock/pkg/ipc"
)

// GroupClass is the synthetic dock item that owns all terminal-backed windows.
const GroupClass = "__hypr_dock_terminal_apps__"

var terminalExecutables = map[string]struct{}{
	"alacritty":      {},
	"foot":           {},
	"gnome-terminal": {},
	"kitty":          {},
	"konsole":        {},
	"mate-terminal":  {},
	"st":             {},
	"terminator":     {},
	"tilix":          {},
	"wezterm":        {},
	"xfce4-terminal": {},
	"xterm":          {},
}

var terminalClasses = map[string]struct{}{
	"alacritty":      {},
	"foot":           {},
	"gnome-terminal": {},
	"kitty":          {},
	"konsole":        {},
	"mate-terminal":  {},
	"st":             {},
	"terminator":     {},
	"tilix":          {},
	"wezterm":        {},
	"xfce4-terminal": {},
	"xterm":          {},
}

// IsTerminalClient identifies terminal-backed windows without relying only on
// the visible title. The Hyprland client PID is normally the terminal emulator
// itself, including terminals launched with a custom app-id/class for btop,
// nvtop, shells, editors, and other TUI programs.
func IsTerminalClient(client ipc.Client) bool {
	for _, className := range []string{client.Class, client.InitialClass} {
		className = strings.ToLower(strings.TrimSpace(className))
		if _, ok := terminalClasses[className]; ok {
			return true
		}
	}

	if client.Pid <= 1 {
		return false
	}
	data, err := os.ReadFile(filepath.Join("/proc", itoa(client.Pid), "cmdline"))
	if err != nil {
		return false
	}
	commandLine := strings.ReplaceAll(string(data), "\x00", " ")
	fields := strings.Fields(commandLine)
	if len(fields) == 0 {
		return false
	}
	_, ok := terminalExecutables[strings.ToLower(filepath.Base(fields[0]))]
	return ok
}

// itoa is kept local to avoid pulling formatting into the hot event path.
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var digits [20]byte
	pos := len(digits)
	for value > 0 {
		pos--
		digits[pos] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		pos--
		digits[pos] = '-'
	}
	return string(digits[pos:])
}
