package ini

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
)

func TestUpdateValuePreservesCommentsAndUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hypr-dock.conf")

	original := `[General]
CurrentTheme = omarchy

# Icon size (px) (default 23)
IconSize = 48

# Custom user section
[User]
SecretKey = keepme
`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Update(path, "General", map[string]map[string]string{"General": {"IconSize": "64"}}); err != nil {
		t.Fatalf("UpdateValue: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	wantParts := []string{
		"CurrentTheme = omarchy",
		"# Icon size (px) (default 23)",
		"IconSize = 64",
		"# Custom user section",
		"[User]",
		"SecretKey = keepme",
	}
	for _, want := range wantParts {
		if !strings.Contains(got, want) {
			t.Fatalf("updated file missing %q:\n%s", want, got)
		}
	}
}

func TestUpdateValueCreatesMissingKeyInExistingSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conf.ini")

	if err := os.WriteFile(path, []byte("[General]\nTheme = a\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Update(path, "General", map[string]map[string]string{"General": {"IconSize": "42"}}); err != nil {
		t.Fatalf("UpdateValue: %v", err)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	if !strings.Contains(got, "IconSize = 42") {
		t.Fatalf("key not appended:\n%s", got)
	}
	if !strings.Contains(got, "[General]") {
		t.Fatalf("section header lost:\n%s", got)
	}
}

func TestUpdateValueAtomicity(t *testing.T) {
	// A failing update must not truncate or corrupt the target file.
	dir := t.TempDir()
	path := filepath.Join(dir, "conf.ini")

	original := "[General]\nIconSize = 48\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	// Unknown section still writes the key under a new section; the file must
	// remain readable and hold the original content.
	_ = Update(path, "General", map[string]map[string]string{"Nope": {"Key": "1"}})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("file was emptied by a failed update")
	}
	if !strings.Contains(string(data), "IconSize = 48") {
		t.Fatalf("original key lost:\n%s", string(data))
	}
}

func TestGetMapReadsSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.ini")

	if err := os.WriteFile(path, []byte("[Desktop Entry]\nName=Test\nHidden=false\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := GetMap(path, "Desktop Entry")
	if err != nil {
		t.Fatalf("GetMap: %v", err)
	}

	section, ok := m["Desktop Entry"]
	if !ok {
		t.Fatalf("missing section: %v", m)
	}
	if section["Name"] != "Test" || section["Hidden"] != "false" {
		t.Fatalf("bad values: %v", section)
	}
}

type testCfg struct {
	General testGeneral `section:"General"`
}

type testGeneral struct {
	IconSize  int `ini:"IconSize" def:"48"`
	Missing   int `ini:"Nope" def:"5"`
	Defaulted int `def:"9"`
}

func TestUnmarshalRespectsDefaultsAndFileOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.ini")

	if err := os.WriteFile(path, []byte("[General]\nIconSize = 77\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := hclog.NewNullLogger()
	ini := New(path, logger)

	var cfg testCfg
	if err := ini.Unmarshal(&cfg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if cfg.General.IconSize != 77 {
		t.Fatalf("IconSize = %d, want 77 (from file)", cfg.General.IconSize)
	}
	if cfg.General.Missing != 5 || cfg.General.Defaulted != 9 {
		t.Fatalf("defaults not applied: %+v", cfg.General)
	}
}
