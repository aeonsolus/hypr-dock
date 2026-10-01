package desktop

import (
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"hypr-dock/pkg/ini"
)

func SearchDesktopFile(className string) string {
	return searchDesktopFile(className, GetAppDirs())
}

func searchDesktopFile(className string, appDirs []string) string {
	browserHost, isBrowserApp := browserHostFromClass(className)

	for _, appDir := range appDirs {
		files, err := os.ReadDir(appDir)
		if err != nil {
			continue
		}

		// The desktop ID is the strongest match.
		exact := filepath.Join(appDir, className+".desktop")
		if _, err := os.Stat(exact); err == nil {
			return exact
		}

		// StartupWMClass is the canonical native-app-to-window mapping.
		for _, file := range files {
			path := filepath.Join(appDir, file.Name())
			general, ok := desktopEntry(path)
			if !ok {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(general["StartupWMClass"]), className) {
				return path
			}
		}

		// Chromium/Chrome web apps expose their hostname in the WM class,
		// while their desktop file normally contains the launch URL.
		if isBrowserApp {
			for _, file := range files {
				path := filepath.Join(appDir, file.Name())
				general, ok := desktopEntry(path)
				if !ok || !desktopExecMatchesHost(general["Exec"], browserHost) {
					continue
				}
				return path
			}
		}

		// Match common reverse-DNS classes such as org.omarchy.btop to
		// btop.desktop without confusing unrelated applications.
		if component := lastClassComponent(className); component != "" {
			for _, file := range files {
				stem := strings.TrimSuffix(strings.ToLower(file.Name()), ".desktop")
				if stem == component {
					return filepath.Join(appDir, file.Name())
				}
			}
		}

		// Existing compatibility fallbacks.
		for _, file := range files {
			if strings.Count(file.Name(), ".") > 1 && strings.Contains(strings.ToLower(file.Name()), strings.ToLower(className)) {
				return filepath.Join(appDir, file.Name())
			}
		}

		for _, file := range files {
			if file.Name() == strings.Split(strings.ToLower(className), " ")[0]+".desktop" {
				return filepath.Join(appDir, file.Name())
			}
		}

		for _, file := range files {
			fileName := strings.ToLower(file.Name())
			classNameLower := strings.ReplaceAll(strings.ToLower(className), " ", "-")
			if fileName == classNameLower+".desktop" {
				return filepath.Join(appDir, file.Name())
			}
		}
	}

	path, exist := GetFiles()[className]
	if exist {
		return path
	}

	return ""
}

func desktopEntry(path string) (map[string]string, bool) {
	if !strings.HasSuffix(strings.ToLower(path), ".desktop") {
		return nil, false
	}
	raw, err := ini.GetMap(path, "Desktop Entry")
	if err != nil {
		return nil, false
	}
	general, ok := raw["Desktop Entry"]
	return general, ok
}

func browserHostFromClass(className string) (string, bool) {
	lower := strings.ToLower(className)
	prefixes := []string{"chrome-", "chromium-", "brave-", "microsoft-edge-"}
	for _, prefix := range prefixes {
		if !strings.HasPrefix(lower, prefix) {
			continue
		}
		host := strings.SplitN(lower[len(prefix):], "__", 2)[0]
		host = strings.Trim(host, "-")
		if host == "" || (net.ParseIP(host) == nil && !strings.Contains(host, ".")) {
			return "", false
		}
		return strings.TrimPrefix(host, "www."), true
	}
	return "", false
}

func desktopExecMatchesHost(execLine, wantedHost string) bool {
	for _, field := range strings.Fields(execLine) {
		start := strings.Index(field, "http://")
		if start < 0 {
			start = strings.Index(field, "https://")
		}
		if start < 0 {
			continue
		}
		raw := strings.Trim(field[start:], "\"'()")
		u, err := url.Parse(raw)
		if err != nil || u.Hostname() == "" {
			continue
		}
		host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
		if host == wantedHost {
			return true
		}
	}
	return false
}

func lastClassComponent(className string) string {
	parts := strings.FieldsFunc(strings.ToLower(className), func(r rune) bool {
		return r == '.' || r == '/' || r == ':'
	})
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

var (
	dirs  []string
	dOnce sync.Once
)

func GetAppDirs() []string {
	dOnce.Do(func() {
		dirs = newAppDirs()
	})
	return dirs
}

func newAppDirs() []string {
	home, _ := os.UserHomeDir()

	return ProcessDirectories(append([]string{
		// dock custom apps
		filepath.Join(home, ".local/share/hypr-dock"),

		// user local apps
		os.Getenv("XDG_DATA_HOME"),
		filepath.Join(home, ".local/share"),

		// flatpak
		filepath.Join(home, ".local/share/flatpak/exports/share"),
		"/var/lib/flatpak/exports/share",

		// system
		"/usr/local/share",
		"/usr/share",

		// xdg
	}, strings.Split(os.Getenv("XDG_DATA_DIRS"), ":")...))
}

func ProcessDirectories(paths []string) []string {
	uniquePaths := make(map[string]bool)
	var result []string

	for _, path := range paths {
		if path == "" {
			continue
		}

		path = filepath.Clean(path)
		path = filepath.Join(path, "applications")

		if !filepath.IsAbs(path) {
			continue
		}

		fileInfo, err := os.Stat(path)
		if err != nil {
			continue
		}

		if !fileInfo.IsDir() {
			continue
		}

		if !uniquePaths[path] {
			uniquePaths[path] = true
			result = append(result, path)
		}
	}

	return result
}
