package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"hypr-dock/internal/ctl"
	"hypr-dock/internal/diag"
	"hypr-dock/internal/pkg/conf"
	"hypr-dock/internal/pkg/pinned"
	"hypr-dock/internal/pkg/utils"
	"hypr-dock/internal/settings"
	"hypr-dock/internal/version"
)

// hypr-dockctl — control plane CLI for a running hypr-dock instance:
//
//	hypr-dockctl reload                       apply config/theme/pinned from disk
//	hypr-dockctl set General.IconSize 48      write one key, apply live
//	hypr-dockctl get General.IconSize         read one key
//	hypr-dockctl status                       dock summary
//	hypr-dockctl settings                     open the settings app
//	hypr-dockctl pin <class> [position]       append/insert into pinned list
//	hypr-dockctl unpin <class>                remove from pinned list
//	hypr-dockctl quit                         stop the dock gracefully
//	hypr-dockctl doctor                       diagnostics
//	hypr-dockctl version                      print version

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	command := args[0]
	rest := args[1:]

	switch command {
	case "ping":
		respond(ctl.Send(ctl.Request{Cmd: ctl.CmdPing}))

	case "reload":
		respond(ctl.Send(ctl.Request{Cmd: ctl.CmdApply}))

	case "set":
		if len(rest) < 2 {
			fatal("usage: hypr-dockctl set <Section.Key> <value>")
		}
		respond(ctl.Send(ctl.Request{Cmd: ctl.CmdSet, Key: rest[0], Value: strings.Join(rest[1:], " ")}))

	case "get":
		if len(rest) < 1 {
			fatal("usage: hypr-dockctl get <Section.Key>")
		}
		config := loadConfig()
		value, err := config.GetByPath(rest[0])
		if err != nil {
			fatal(err.Error())
		}
		fmt.Println(value)

	case "status":
		resp, err := ctl.Send(ctl.Request{Cmd: ctl.CmdStatus})
		if err != nil {
			fatal(err.Error())
		}
		for _, key := range sortedDataKeys(resp.Data) {
			fmt.Printf("%-10s %s\n", key, resp.Data[key])
		}

	case "settings":
		if err := startSettings(); err != nil {
			fatal(err.Error())
		}

	case "pin":
		if len(rest) < 1 {
			fatal("usage: hypr-dockctl pin <class> [position]")
		}
		position := -1
		if len(rest) > 1 {
			if n, err := strconv.Atoi(rest[1]); err == nil {
				position = n
			}
		}
		s := loadSettings()
		pins, err := pinned.Open(s.PinnedPath)
		if err != nil {
			fatal(err.Error())
		}
		for _, pin := range pins {
			if pin == rest[0] {
				fmt.Println("already pinned:", rest[0])
				return
			}
		}
		if position < 0 || position > len(pins) {
			position = len(pins)
		}
		pins = append(pins[:position], append([]string{rest[0]}, pins[position:]...)...)
		if err := pinned.Save(s.PinnedPath, pins); err != nil {
			fatal(err.Error())
		}
		respond(ctl.Send(ctl.Request{Cmd: ctl.CmdApply}))

	case "unpin":
		if len(rest) < 1 {
			fatal("usage: hypr-dockctl unpin <class>")
		}
		s := loadSettings()
		pins, err := pinned.Open(s.PinnedPath)
		if err != nil {
			fatal(err.Error())
		}
		var out []string
		for _, pin := range pins {
			if pin != rest[0] {
				out = append(out, pin)
			}
		}
		if err := pinned.Save(s.PinnedPath, out); err != nil {
			fatal(err.Error())
		}
		respond(ctl.Send(ctl.Request{Cmd: ctl.CmdApply}))

	case "quit":
		respond(ctl.Send(ctl.Request{Cmd: ctl.CmdQuit}))

	case "doctor":
		fmt.Print(diag.Report(version.Version))

	case "version":
		fmt.Println("hypr-dock", version.Version)

	default:
		usage()
		os.Exit(2)
	}
}

// startSettings launches the settings app: prefer the sibling binary next to
// this CLI, fall back to PATH.
func startSettings() error {
	candidates := []string{}

	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "hypr-dock-settings"))
	}
	candidates = append(candidates, "hypr-dock-settings")

	var lastErr error
	for _, candidate := range candidates {
		cmd := exec.Command(candidate)
		if err := cmd.Start(); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return fmt.Errorf("unable to start hypr-dock-settings: %w", lastErr)
}

// paths computes the dock's standard config layout without starting GTK.
func paths() (configDir, localDir, configPath string) {
	home, _ := os.UserHomeDir()
	configDir = filepath.Join(home, ".config", "hypr-dock")
	localDir = filepath.Join(home, ".local", "share", "hypr-dock")
	configPath = filepath.Join(configDir, "hypr-dock.conf")
	return configDir, localDir, configPath
}

func loadConfig() *conf.Config {
	_, _, configPath := paths()
	home, _ := os.UserHomeDir()
	themesDir := filepath.Join(home, ".config", "hypr-dock", "themes")

	config, err := conf.New(configPath, themesDir, nil)
	if err != nil {
		return conf.Defaults()
	}
	return config
}

func loadSettings() *settings.Settings {
	configDir, localDir, configPath := paths()

	log := utils.СreateLogger("error")
	s, err := settings.Load(configPath, configDir, localDir, log)
	if err != nil {
		fatal(err.Error())
	}
	return s
}

func respond(resp ctl.Response, err error) {
	if err != nil {
		fatal(err.Error())
	}
	if !resp.Ok {
		fatal(resp.Error)
	}
	fmt.Println("ok")
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, "hypr-dockctl:", message)
	os.Exit(1)
}

func sortedDataKeys(data map[string]string) []string {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

func usage() {
	fmt.Print(`hypr-dockctl — control a running hypr-dock

  reload                       apply config/theme/pinned from disk
  set <Section.Key> <value>    write one key, apply live
  get <Section.Key>            read one key
  status                       dock summary
  settings                     open the settings application
  pin <class> [position]       pin an application (insert at position)
  unpin <class>                unpin an application
  quit                         stop the dock
  doctor                       diagnostics
  version                      print version
`)
}
