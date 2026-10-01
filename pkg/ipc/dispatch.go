package ipc

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Hyprland with a Lua config (this Omarchy build) rewrites `dispatch <name>
// <args>` into a Lua expression, which breaks classic dispatch syntax. The
// dock therefore probes `eval` support once and switches its window dispatches
// to Lua form (`hl.dsp.*`) when available; classic Hyprland keeps the raw
// protocol.

var (
	luaCompatOnce sync.Once
	luaCompat     bool
	luaProbed     bool
)

// LuaCompat reports whether the running Hyprland accepts the `eval` command
// (Lua config builds). Probe result is cached.
func LuaCompat() bool {
	luaCompatOnce.Do(func() {
		luaProbed = true

		response, err := Hyprctl(`eval "return 0"`)
		luaCompat = err == nil && strings.TrimSpace(string(response)) != "" &&
			!strings.HasPrefix(strings.TrimSpace(string(response)), "error")

		// Unmarshal-style failures mean classic Hyprland.
		if !luaCompat && err == nil {
			luaCompat = false
		}
	})
	_ = luaProbed
	return luaCompat
}

// dispatch sends `dispatch <lua-or-raw>` using the right protocol form.
func dispatch(lua, raw string) ([]byte, error) {
	if LuaCompat() {
		return Hyprctl("dispatch " + lua)
	}
	return Hyprctl("dispatch " + raw)
}

// FocusWindow focuses one window by address.
func FocusWindow(address string) error {
	lua := fmt.Sprintf(`hl.dsp.focus({ window = "address:%s" })`, address)
	raw := "focuswindow address:" + address
	_, err := dispatch(lua, raw)
	return err
}

// CloseWindow closes one window by address.
func CloseWindow(address string) error {
	lua := fmt.Sprintf(`hl.dsp.window.close({ window = "address:%s" })`, address)
	raw := "closewindow address:" + address
	_, err := dispatch(lua, raw)
	return err
}

// MinimizeWorkspace is the special workspace windows are parked in when
// "minimized" (this Hyprland build has no minimize dispatcher).
const MinimizeWorkspace = "minimized"

// MinimizeWindow hides a window by moving it to the special workspace.
func MinimizeWindow(address string) error {
	lua := fmt.Sprintf(`hl.dsp.window.move({ window = "address:%s", workspace = "special:%s" })`, address, MinimizeWorkspace)
	raw := "movetospecialworkspace " + MinimizeWorkspace + " " + address
	_, err := dispatch(lua, raw)
	return err
}

// ShowMinimized switches the view to the minimized workspace.
func ShowMinimized() error {
	lua := fmt.Sprintf(`hl.dsp.focus({ workspace = "special:%s" })`, MinimizeWorkspace)
	raw := "togglespecialworkspace " + MinimizeWorkspace
	_, err := dispatch(lua, raw)
	return err
}

// MoveWindowToWorkspace moves a window to a numeric workspace.
func MoveWindowToWorkspace(address string, workspaceId int) error {
	ws := strconv.Itoa(workspaceId)
	lua := fmt.Sprintf(`hl.dsp.window.move({ window = "address:%s", workspace = "%s" })`, address, ws)
	raw := "movetoworkspace " + ws + "," + address
	_, err := dispatch(lua, raw)
	return err
}

// CycleWindows cycles focus through windows (the cyclenext dispatcher).
func CycleWindows() error {
	lua := `hl.dsp.window.cycle_next()`
	raw := "cyclenext"
	_, err := dispatch(lua, raw)
	return err
}

// ActiveAddress returns the address of the currently focused window.
func ActiveAddress() (string, error) {
	client, err := GetActiveWindow()
	if err != nil {
		return "", err
	}
	if client == nil || client.Address == "" {
		return "", nil
	}
	return client.Address, nil
}

// ActiveClass returns the class of the currently focused window.
func ActiveClass() (string, error) {
	client, err := GetActiveWindow()
	if err != nil {
		return "", err
	}
	if client == nil {
		return "", nil
	}
	return client.Class, nil
}
