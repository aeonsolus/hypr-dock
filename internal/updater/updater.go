// Package updater checks the private fork and installs a verified Git snapshot.
package updater

import (
	"bytes"
	"context"
	"crypto/sha1"
	"debug/elf"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const Repository = "aeonsolus/hypr-dock"

type Release struct {
	Version   string
	Commit    string
	Available bool
}

var versionPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-[A-Za-z0-9.-]+)?$`)
var commitPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)

func versionParts(value string) ([3]uint64, error) {
	var parts [3]uint64
	match := versionPattern.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return parts, fmt.Errorf("invalid version %q", value)
	}
	for i := range parts {
		n, err := strconv.ParseUint(match[i+1], 10, 64)
		if err != nil {
			return parts, err
		}
		parts[i] = n
	}
	return parts, nil
}

// Newer compares this fork's numbered releases; the -custom suffix is its
// distribution label. A different commit at the same version is not an update.
func Newer(remote, installed string) (bool, error) {
	r, err := versionParts(remote)
	if err != nil {
		return false, err
	}
	l, err := versionParts(installed)
	if err != nil {
		return false, err
	}
	for i := range r {
		if r[i] != l[i] {
			return r[i] > l[i], nil
		}
	}
	return false, nil
}

func command(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		if len(text) > 2400 {
			text = text[len(text)-2400:]
		}
		return "", fmt.Errorf("%s: %w\n%s", name, err, text)
	}
	return text, nil
}

func Check(ctx context.Context, installed string) (*Release, error) {
	commit, err := command(ctx, "", "gh", "api", "repos/"+Repository+"/commits/main", "--jq", ".sha")
	if err != nil {
		return nil, err
	}
	if !commitPattern.MatchString(commit) {
		return nil, fmt.Errorf("GitHub returned an invalid commit")
	}
	remote, err := command(ctx, "", "gh", "api", "-H", "Accept: application/vnd.github.raw+json", "repos/"+Repository+"/contents/VERSION?ref="+commit)
	if err != nil {
		return nil, err
	}
	available, err := Newer(remote, installed)
	if err != nil {
		return nil, err
	}
	return &Release{Version: remote, Commit: commit, Available: available}, nil
}

func Install(ctx context.Context, release Release) error {
	if !commitPattern.MatchString(release.Commit) {
		return fmt.Errorf("invalid update commit")
	}
	if _, err := versionParts(release.Version); err != nil {
		return err
	}
	for _, tool := range []string{"gh", "pkexec", "install"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("update requires %s: %w", tool, err)
		}
	}
	dir, err := os.MkdirTemp("", "hyprdock-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	paths, err := Download(ctx, release, dir)
	if err != nil {
		return err
	}
	args := []string{"--user", "root", "install", "-m", "755"}
	args = append(args, paths...)
	args = append(args, "/usr/bin/")
	_, err = command(ctx, "", "pkexec", args...)
	return err
}

// Download obtains prebuilt artifacts from the checked immutable commit. All
// artifacts are verified before any installed binary is replaced.
func Download(ctx context.Context, release Release, dir string) ([]string, error) {
	if !commitPattern.MatchString(release.Commit) {
		return nil, fmt.Errorf("invalid update commit")
	}
	if _, err := versionParts(release.Version); err != nil {
		return nil, err
	}
	var paths []string
	for _, binary := range []string{"hypr-dock", "hypr-dock-settings", "hypr-dockctl", "hypr-alttab"} {
		endpoint := "repos/" + Repository + "/contents/bin/" + binary + "?ref=" + release.Commit
		sha, err := command(ctx, "", "gh", "api", endpoint, "--jq", ".sha")
		if err != nil {
			return nil, err
		}
		if !commitPattern.MatchString(sha) {
			return nil, fmt.Errorf("invalid checksum for %s", binary)
		}
		path := filepath.Join(dir, binary)
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
		cmd := exec.CommandContext(ctx, "gh", "api", "-H", "Accept: application/vnd.github.raw", endpoint)
		var stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = file, &stderr
		err = cmd.Run()
		closeErr := file.Close()
		if err != nil {
			return nil, fmt.Errorf("download %s: %w: %s", binary, err, strings.TrimSpace(stderr.String()))
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if err := verifyBinary(path, sha); err != nil {
			return nil, err
		}
		if err := os.Chmod(path, 0700); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	text, err := command(ctx, "", filepath.Join(dir, "hypr-dockctl"), "version")
	if err != nil {
		return nil, err
	}
	if text != "hypr-dock "+release.Version {
		return nil, fmt.Errorf("downloaded binaries report %q instead of %s", text, release.Version)
	}
	return paths, nil
}

func verifyBinary(path, wantSHA string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	hash := sha1.New()
	fmt.Fprintf(hash, "blob %d%c", info.Size(), 0)
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != wantSHA {
		return fmt.Errorf("checksum mismatch: %s", filepath.Base(path))
	}
	image, err := elf.Open(path)
	if err != nil {
		return fmt.Errorf("invalid Linux executable %s: %w", filepath.Base(path), err)
	}
	defer image.Close()
	machine := map[string]elf.Machine{"amd64": elf.EM_X86_64, "arm64": elf.EM_AARCH64}[runtime.GOARCH]
	if runtime.GOOS != "linux" || machine == 0 || image.Machine != machine {
		return fmt.Errorf("%s is not built for this machine (%s/%s)", filepath.Base(path), runtime.GOOS, runtime.GOARCH)
	}
	return nil
}
