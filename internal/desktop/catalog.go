package desktop

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// CatalogEntry describes one installed application for the settings app.
type CatalogEntry struct {
	ID   string // desktop file id, e.g. "org.gnome.Nautilus"
	Name string // localized display name
	Icon string
}

var (
	catalog     []CatalogEntry
	catalogOnce sync.Once
)

// Catalog lists every launchable application found in the standard desktop
// entry directories, deduplicated by desktop id (earlier dirs win, matching
// XDG precedence).
func Catalog() []CatalogEntry {
	catalogOnce.Do(func() {
		seen := make(map[string]bool)

		for _, dir := range GetAppDirs() {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				if !strings.HasSuffix(strings.ToLower(name), ".desktop") {
					continue
				}

				id := strings.TrimSuffix(name, ".desktop")
				if seen[id] {
					continue
				}

				raw, err := iniGetMap(filepath.Join(dir, name))
				if err != nil {
					continue
				}
				general, ok := raw["Desktop Entry"]
				if !ok {
					continue
				}

				// Skip entries users never want to pin.
				if general["Hidden"] == "true" || general["NoDisplay"] == "true" {
					continue
				}
				if t := strings.TrimSpace(general["Type"]); t != "" && t != "Application" {
					continue
				}
				if strings.TrimSpace(general["TryExec"]) != "" {
					if _, err := lookPath(strings.TrimSpace(general["TryExec"])); err != nil {
						continue
					}
				}

				displayName := GetLocalizedValue(general, "")
				if displayName == "" {
					displayName = id
				}

				seen[id] = true
				catalog = append(catalog, CatalogEntry{ID: id, Name: displayName, Icon: general["Icon"]})
			}
		}

		sort.Slice(catalog, func(i, j int) bool {
			return strings.ToLower(catalog[i].Name) < strings.ToLower(catalog[j].Name)
		})
	})

	return catalog
}
