package ctl

import (
	"os"
	"path/filepath"
)

// Control-plane protocol between hypr-dock, hypr-dockctl and
// hypr-dock-settings. Newline-delimited JSON over a unix socket in the user
// runtime dir.

type Request struct {
	Cmd   string `json:"cmd"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type Response struct {
	Ok    bool              `json:"ok"`
	Error string            `json:"error,omitempty"`
	Data  map[string]string `json:"data,omitempty"`
}

// Well-known commands.
const (
	CmdPing    = "ping"
	CmdApply   = "apply"  // re-read config + theme + pinned from disk, apply live
	CmdSet     = "set"    // write one key to disk (Section.Key path), then apply
	CmdStatus  = "status" // dock summary for doctor
	CmdRestart = "restart"
	CmdQuit    = "quit"
)

// SocketPath returns the control socket location, mirroring how the dock
// itself derives it so every component agrees without extra config.
func SocketPath() string {
	dir := RuntimeDir()
	return filepath.Join(dir, "ctl.sock")
}

// RuntimeDir returns the dock's runtime directory (created on demand by the
// server).
func RuntimeDir() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "hypr-dock")
	}
	return filepath.Join(os.TempDir(), "hypr-dock-"+os.Getenv("USER"))
}
