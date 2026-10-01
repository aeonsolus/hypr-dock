package conf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"

	"hypr-dock/pkg/ini"
)

func newTestConfig(t *testing.T, body string) *Config {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "hypr-dock.conf")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	config, err := New(path, filepath.Join(dir, "themes"), hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("conf.New: %v", err)
	}
	return config
}

func TestSetByPathRoundTrip(t *testing.T) {
	config := newTestConfig(t, "[General]\nIconSize = 48\n")

	if err := config.SetByPath("General.IconSize", "56"); err != nil {
		t.Fatalf("SetByPath: %v", err)
	}
	if got, err := config.GetByPath("General.IconSize"); err != nil || got != "56" {
		t.Fatalf("GetByPath = %q, %v", got, err)
	}

	if err := config.Save(config.ConfigPath()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := New(config.ConfigPath(), filepath.Join(filepath.Dir(config.ConfigPath()), "themes"), hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got, _ := reloaded.GetByPath("General.IconSize"); got != "56" {
		t.Fatalf("persisted IconSize = %q, want 56", got)
	}
}

func TestSetByPathRejectsUnknownPathsAndValues(t *testing.T) {
	config := newTestConfig(t, "[General]\nIconSize = 48\n")

	if err := config.SetByPath("General.Nope", "1"); err == nil {
		t.Fatal("unknown key accepted")
	}
	if err := config.SetByPath("Nope.Nope", "1"); err == nil {
		t.Fatal("unknown section accepted")
	}
	if err := config.SetByPath("General.IconSize", "abc"); err == nil {
		t.Fatal("non-number accepted for int key")
	}
	if err := config.SetByPath("General.Exclusive", "maybe"); err == nil {
		t.Fatal("non-boolean accepted for bool key")
	}
}

func TestPathsCoversKeySections(t *testing.T) {
	config := newTestConfig(t, "[General]\nIconSize = 48\n")

	paths := config.Paths()
	want := map[string]bool{
		"General.IconSize":        false,
		"General.ShowRecentApps":  false,
		"Appearance.PanelOpacity": false,
		"Displays.Mode":           false,
		"Theme.preview.Size":      false,
	}
	for _, path := range paths {
		if _, ok := want[path]; ok {
			want[path] = true
		}
	}
	for path, seen := range want {
		if !seen {
			t.Fatalf("Paths() missing %q (got %d paths)", path, len(paths))
		}
	}
}

func TestSavePreservesCommentsAndUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hypr-dock.conf")

	original := `[General]
CurrentTheme = omarchy

# Icon size (px) (default 23)
IconSize = 48

[User]
Custom = yes
`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	config, err := New(path, filepath.Join(dir, "themes"), hclog.NewNullLogger())
	if err != nil {
		t.Fatal(err)
	}
	if err := config.SetByPath("General.IconSize", "40"); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	for _, want := range []string{
		"# Icon size (px) (default 23)",
		"IconSize = 40",
		"[User]",
		"Custom = yes",
		"CurrentTheme = omarchy",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q after save:\n%s", want, got)
		}
	}
}

func TestDefaultsMaterializeWhenFileEmpty(t *testing.T) {
	config := newTestConfig(t, "")

	if config.IconSize != 23 {
		t.Fatalf("default IconSize = %d, want 23", config.IconSize)
	}
	if config.Position != "bottom" || config.Layer != "top" {
		t.Fatalf("bad defaults: %s/%s", config.Position, config.Layer)
	}
	if config.ShowRecentApps {
		t.Fatal("ShowRecentApps must default to false")
	}
	if got, _ := config.GetByPath("General.FollowMouse"); got != "true" {
		t.Fatalf("FollowMouse default = %q", got)
	}
}

func TestValidationEnforcesBoundsAndEnums(t *testing.T) {
	config := newTestConfig(t, "")

	if err := config.SetByPath("General.IconSize", "999"); err == nil {
		t.Fatal("IconSize above max accepted")
	}
	if err := config.SetByPath("General.Layer", "bogus"); err == nil {
		t.Fatal("invalid layer enum accepted")
	}
	if err := config.SetByPath("General.Layer", "overlay"); err != nil {
		t.Fatalf("valid layer rejected: %v", err)
	}
}

// The ini import keeps the package graph exercised end-to-end; guard the
// symbol so removal is noticed.
var _ = ini.Update
