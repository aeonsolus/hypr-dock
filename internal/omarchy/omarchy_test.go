package omarchy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hypr-dock/internal/appearance"
)

func TestParsePaletteReadsTomlHexes(t *testing.T) {
	toml := `
background = "#1a1b26"
foreground = "#c0caf5"
accent = "#7aa2f7"
unused = "not-a-color"
`
	pal := appearance.ParsePalette(toml)

	if pal.BG != "#1a1b26" || pal.FG != "#c0caf5" {
		t.Fatalf("bg/fg = %q/%q", pal.BG, pal.FG)
	}
	if pal.Accent != "#7aa2f7" {
		t.Fatalf("accent = %q", pal.Accent)
	}
}

func TestRegenerateThemeWritesStyleAndDots(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "omarchy")

	pal := appearance.Palette{BG: "#101010", FG: "#e0e0e0", Accent: "#00aaff"}
	if err := RegenerateTheme(dir, pal); err != nil {
		t.Fatalf("RegenerateTheme: %v", err)
	}

	css, err := os.ReadFile(filepath.Join(dir, "style.css"))
	if err != nil {
		t.Fatalf("style.css missing: %v", err)
	}
	if !strings.Contains(string(css), "rgba(16, 16, 16") {
		t.Fatalf("palette bg not in style.css:\n%s", string(css))
	}

	dot, err := os.ReadFile(filepath.Join(dir, "point", "3.svg"))
	if err != nil {
		t.Fatalf("3.svg missing: %v", err)
	}
	if !strings.Contains(string(dot), "#00aaff") {
		t.Fatalf("accent not in dot svg:\n%s", string(dot))
	}

	// theme.conf is only written when absent.
	confPath := filepath.Join(dir, "theme.conf")
	if _, err := os.Stat(confPath); err != nil {
		t.Fatal("theme.conf not created")
	}
}

func TestEnsureCurrentThemeUpdatesConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hypr-dock.conf")

	if err := os.WriteFile(path, []byte("[General]\nCurrentTheme = lotos\nIconSize = 48\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureCurrentTheme(path); err != nil {
		t.Fatalf("EnsureCurrentTheme: %v", err)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	if !strings.Contains(got, "CurrentTheme = omarchy") {
		t.Fatalf("theme not switched:\n%s", got)
	}
	if !strings.Contains(got, "IconSize = 48") {
		t.Fatalf("other keys lost:\n%s", got)
	}
}
