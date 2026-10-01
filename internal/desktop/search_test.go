package desktop

import (
	"os"
	"path/filepath"
	"testing"
)

func writeDesktopTestFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSearchDesktopFileMatchesWebAppByLaunchHost(t *testing.T) {
	dir := t.TempDir()
	want := writeDesktopTestFile(t, dir, "WhatsApp.desktop", `[Desktop Entry]
Name=WhatsApp
Exec=omarchy-launch-webapp https://web.whatsapp.com/
Icon=whatsapp
Type=Application
`)
	writeDesktopTestFile(t, dir, "Web Browser.desktop", `[Desktop Entry]
Name=Web Browser
Exec=google-chrome %U
Icon=google-chrome
Type=Application
`)

	got := searchDesktopFile("chrome-web.whatsapp.com__-Default", []string{dir})
	if got != want {
		t.Fatalf("web app resolution = %q, want %q", got, want)
	}
}

func TestSearchDesktopFileMatchesStartupWMClass(t *testing.T) {
	dir := t.TempDir()
	want := writeDesktopTestFile(t, dir, "Chat.desktop", `[Desktop Entry]
Name=Chat
StartupWMClass=chat-window
Exec=chat
Icon=chat
Type=Application
`)

	got := searchDesktopFile("chat-window", []string{dir})
	if got != want {
		t.Fatalf("StartupWMClass resolution = %q, want %q", got, want)
	}
}

func TestSearchDesktopFileMatchesReverseDNSFinalComponent(t *testing.T) {
	dir := t.TempDir()
	want := writeDesktopTestFile(t, dir, "btop.desktop", `[Desktop Entry]
Name=btop++
Exec=btop
Icon=btop
Type=Application
`)

	got := searchDesktopFile("org.omarchy.btop", []string{dir})
	if got != want {
		t.Fatalf("reverse-DNS resolution = %q, want %q", got, want)
	}
}

func TestNewWithTitleUsesSafeFallbackForUnknownWindow(t *testing.T) {
	app, err := NewWithTitle("unknown-window-class", "My Application")
	if err == nil {
		t.Fatal("unknown window should report missing desktop metadata")
	}
	if app.GetName() != "My Application" {
		t.Fatalf("fallback name = %q, want %q", app.GetName(), "My Application")
	}
	if app.GetIcon() != "application-x-executable" {
		t.Fatalf("fallback icon = %q, want application-x-executable", app.GetIcon())
	}
}

func TestNewWithTitleUsesSteamFallbackIcon(t *testing.T) {
	app, err := NewWithTitle("steam_app_1665460", "eFootball™")
	if err == nil {
		t.Fatal("unknown Steam game should report missing desktop metadata")
	}
	if app.GetName() != "eFootball™" || app.GetIcon() != "steam" {
		t.Fatalf("Steam fallback = name %q icon %q", app.GetName(), app.GetIcon())
	}
}

func TestBrowserHostHelpers(t *testing.T) {
	cases := []struct {
		className string
		want      string
	}{
		{"chrome-web.whatsapp.com__-Default", "web.whatsapp.com"},
		{"chromium-maps.google.com__-Default", "maps.google.com"},
		{"chrome-127.0.0.1__-Default", "127.0.0.1"},
	}
	for _, tc := range cases {
		got, ok := browserHostFromClass(tc.className)
		if !ok || got != tc.want {
			t.Fatalf("browserHostFromClass(%q) = %q, %v; want %q, true", tc.className, got, ok, tc.want)
		}
	}
}
