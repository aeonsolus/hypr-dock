package settings

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"

	"hypr-dock/internal/pkg/conf"
	"hypr-dock/internal/pkg/flags"
	"hypr-dock/internal/pkg/pinned"
	"hypr-dock/internal/pkg/utils"
)

const APP_NAME = "hypr-dock"
const LOCAL = ".local/share"

type Settings struct {
	*conf.Config
	LocalDir       string
	ConfigDir      string
	ConfigPath     string
	PinnedPath     string
	ThemesDir      string
	ThemeStyle     string
	ThemeStyleHash string
	PinnedApps     []string
}

func Init(flags flags.Flags, log hclog.Logger) (*Settings, error) {
	var err error

	// get local app dir
	localDir := filepath.Join(os.Getenv("HOME"), LOCAL, APP_NAME)

	// main configs dir
	configDir, isCreate, err := GetConfigDir(flags.DevMode)
	log.Debug("Config dir init", "path", configDir, "created", isCreate, "error", err)

	// main config file
	configPath := filepath.Join(configDir, APP_NAME+".conf")
	if flags.Config != "~/.config/hypr-dock" {
		configPath = expand(flags.Config)
	}

	return Load(configPath, configDir, localDir, log)
}

// Load reads pinned apps, the main config and the active theme from fixed
// paths. Reload-safe: it re-derives everything from disk and never mutates
// global state, so the dock can re-load settings at runtime.
func Load(configPath, configDir, localDir string, log hclog.Logger) (*Settings, error) {
	var err error

	if log == nil {
		log = hclog.NewNullLogger()
	}

	if localDir == "" {
		localDir = filepath.Join(os.Getenv("HOME"), LOCAL, APP_NAME)
	}

	// read pinned file
	pinnedPath := filepath.Join(localDir, "pinned")
	pinnedApps, err := pinned.Open(pinnedPath)
	if err != nil {
		log.Error("Failed to create/write pinned list", "file", pinnedPath, "error", err)
	}

	// themes dir
	themesDir := filepath.Join(configDir, "themes")

	// read main config and current theme config
	config, err := conf.New(configPath, themesDir, log)
	if err != nil {
		// Fail open: run with defaults rather than crashing the dock.
		log.Error("Config load failed, using defaults", "path", configPath, "error", err)
		config = conf.Defaults()
		config.ThemeDir = filepath.Join(themesDir, config.CurrentTheme)
		config.ThemeConf = filepath.Join(config.ThemeDir, "theme.conf")
		config.SetPaths(configPath, themesDir)
	}

	// theme style file
	themeStyle := filepath.Join(config.ThemeDir, "style.css")

	return &Settings{
		Config:         config,
		LocalDir:       localDir,
		ConfigDir:      configDir,
		ConfigPath:     configPath,
		PinnedPath:     pinnedPath,
		ThemesDir:      themesDir,
		ThemeStyle:     themeStyle,
		ThemeStyleHash: hashFile(themeStyle),
		PinnedApps:     pinnedApps,
	}, nil
}

// LauncherCommandOrDefault resolves the launcher command: the configured
// value, else the first known launcher binary on PATH.
func (s *Settings) LauncherCommandOrDefault() string {
	if strings.TrimSpace(s.LauncherCommand) != "" {
		return s.LauncherCommand
	}
	return DetectLauncher()
}

// DetectLauncher probes PATH for known launchers, preferring Omarchy's own.
func DetectLauncher() string {
	candidates := []string{
		"omarchy-launcher",
		"omarchy-menu",
		"vicinae",
		"fuzzel",
		"walker",
		"wofi",
		"rofi",
		"tofi",
	}
	for _, name := range candidates {
		if path, err := execLookPath(name); err == nil && path != "" {
			return name
		}
	}
	return ""
}

func hashFile(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

func GetConfigDir(dev bool) (string, bool, error) {
	var target, source string

	if dev {
		exe, _ := os.Executable()
		exeDir := filepath.Dir(exe)
		configs := filepath.Join(filepath.Dir(exeDir), "configs")

		target = filepath.Join(configs, "dev")
		source = filepath.Join(configs, "default")
	} else {
		target = filepath.Join(os.Getenv("HOME"), ".config", APP_NAME)
		source = filepath.Join("/etc", APP_NAME)
	}

	created := utils.DirExists(target)

	if created {
		return target, created, nil
	}

	err := utils.CopyDir(target, source)
	if err != nil {
		return source, created, err
	}

	return target, created, nil
}

func expand(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func execLookPath(name string) (string, error) {
	return exec.LookPath(name)
}
