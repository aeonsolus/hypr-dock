package updater

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewer(t *testing.T) {
	for _, tc := range []struct {
		remote, installed string
		want              bool
	}{
		{"1.4.3-custom", "1.4.2-custom", true},
		{"1.10.0-custom", "1.9.9-custom", true},
		{"2.0.0-custom", "1.99.99-custom", true},
		{"1.4.2-custom", "1.4.2-custom", false},
		{"1.3.9-custom", "1.4.2-custom", false},
		{"v1.4.3-custom", "1.4.2-custom", true},
	} {
		got, err := Newer(tc.remote, tc.installed)
		if err != nil || got != tc.want {
			t.Errorf("Newer(%q, %q)=%t, %v; want %t", tc.remote, tc.installed, got, err, tc.want)
		}
	}
	for _, value := range []string{"", "<html>error</html>", "1.4", "1.4.2; rm -rf /", "999999999999999999999.1.1"} {
		if _, err := Newer(value, "1.4.2-custom"); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
}

func blobSHA(data []byte) string {
	h := sha1.New()
	fmt.Fprintf(h, "blob %d%c", len(data), 0)
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func TestBinaryVerification(t *testing.T) {
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyBinary(path, blobSHA(data)); err != nil {
		t.Fatal(err)
	}
	if err := verifyBinary(path, "0000000000000000000000000000000000000000"); err == nil {
		t.Fatal("accepted checksum mismatch")
	}
	bad := filepath.Join(t.TempDir(), "bad")
	data = []byte("not a binary")
	if err := os.WriteFile(bad, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyBinary(bad, blobSHA(data)); err == nil {
		t.Fatal("accepted non-executable response")
	}
}

func TestGitHubDownload(t *testing.T) {
	if os.Getenv("HYPR_DOCK_UPDATER_LIVE") != "1" {
		t.Skip("opt-in private GitHub integration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	release, err := Check(ctx, "0.0.0-custom")
	if err != nil {
		t.Fatal(err)
	}
	if !release.Available {
		t.Fatal("did not detect newer version")
	}
	current, err := Check(ctx, release.Version)
	if err != nil {
		t.Fatal(err)
	}
	if current.Commit == release.Commit && current.Available {
		t.Fatal("same version incorrectly offered as update")
	}
	paths, err := Download(ctx, *release, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 4 {
		t.Fatalf("downloaded %d artifacts", len(paths))
	}
	t.Logf("Downloaded and verified all four binaries for %s at %s", release.Version, release.Commit)
}

// Explicit deployment check: never runs as part of the normal test suite.
func TestGitHubInstall(t *testing.T) {
	if os.Getenv("HYPR_DOCK_UPDATER_INSTALL") != "1" {
		t.Skip("opt-in installation integration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	installed, err := command(ctx, "", "/usr/bin/hypr-dockctl", "version")
	if err != nil {
		t.Fatal(err)
	}
	release, err := Check(ctx, strings.TrimPrefix(installed, "hypr-dock "))
	if err != nil {
		t.Fatal(err)
	}
	if !release.Available {
		t.Skip("latest Git version already installed")
	}
	if err := Install(ctx, *release); err != nil {
		t.Fatal(err)
	}
	actual, err := command(ctx, "", "/usr/bin/hypr-dockctl", "version")
	if err != nil {
		t.Fatal(err)
	}
	if actual != "hypr-dock "+release.Version {
		t.Fatalf("installed version = %q", actual)
	}
	t.Logf("Installed verified GitHub binaries for %s", release.Version)
}
