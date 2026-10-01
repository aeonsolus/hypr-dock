package desktop

import (
	"hypr-dock/pkg/ini"
	"os/exec"
)

func iniGetMap(path string) (map[string]map[string]string, error) {
	return ini.GetMap(path, "Desktop Entry")
}

func lookPath(name string) (string, error) {
	return exec.LookPath(name)
}
