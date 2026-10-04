package conf

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"

	"hypr-dock/pkg/ini"
)

type General struct {
	CurrentTheme  string `def:"lotos"`
	IconSize      int    `def:"23" min:"12" max:"128"`
	Layer         string `def:"top" valid:"background,bottom,top,overlay"`
	Exclusive     bool   `def:"true"`
	SmartView     bool   `def:"false"`
	Position      string `def:"bottom" valid:"top,bottom,left,right"`
	AutoHideDelay int    `def:"400" min:"0" max:"10000"`
	SystemGapUsed bool   `def:"true"`
	Margin        int    `def:"8" min:"0" max:"64"`
	ContextPos    int    `def:"5" min:"0" max:"64"`
	FollowMouse   bool   `def:"true"`
	FollowAnimMs  int    `def:"150" min:"0" max:"2000"`
	HiddenApps    string `def:""`

	// Behavior: section membership and default visibility.
	// ShowRecentApps stays false: the macOS-style Recent Applications
	// section is opt-in only.
	ShowPinnedApps   bool `def:"true"`
	ShowRunningApps  bool `def:"true"`
	ShowRecentApps   bool `def:"false"`
	ShowTrash        bool `def:"true"`
	ShowHomeFolder   bool `def:"false"`
	ShowSettingsIcon bool `def:"false"`

	// Window indicators (running dots) and badge.
	ShowWindowCount bool `def:"true"`

	// Click semantics for running applications.
	ClickAction        string `def:"focus" valid:"launch,focus,minimize,show,cycle,none"`
	ClickActionFocused string `def:"minimize" valid:"minimize,show,cycle,none"`
	MiddleClickAction  string `def:"launch" valid:"launch,none"`

	// Launcher button.
	ShowLauncherButton bool   `def:"false"`
	LauncherCommand    string `def:""`
	LauncherPosition   string `def:"start" valid:"start,end"`

	// Running-indicator size as a percent of IconSize.
	IndicatorSize int `def:"56" min:"20" max:"100"`

	// Follow the active Omarchy theme's colors (detects Omarchy; when
	// enabled, colors.toml changes restyle the dock live).
	FollowOmarchy bool `def:"false"`
}

// IsHidden reports whether the class name is blacklisted. HiddenApps is a
// comma-separated list; entries may end or start with '*' for wildcard match.
func (c *Config) IsHidden(className string) bool {
	if c == nil || className == "" {
		return false
	}
	for _, entry := range strings.Split(c.HiddenApps, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		switch {
		case entry == className:
			return true
		case strings.HasPrefix(entry, "*") && strings.HasSuffix(entry, "*"):
			if strings.Contains(className, strings.Trim(entry, "*")) {
				return true
			}
		case strings.HasPrefix(entry, "*"):
			if strings.HasSuffix(className, strings.TrimPrefix(entry, "*")) {
				return true
			}
		case strings.HasSuffix(entry, "*"):
			if strings.HasPrefix(className, strings.TrimSuffix(entry, "*")) {
				return true
			}
		}
	}
	return false
}

type Preview struct {
	Mode       string `def:"none" valid:"none, static, live"`
	FPS        int    `def:"30" min:"1" max:"120"`
	BufferSize int    `def:"5" max:"20" min:"1"`
	ShowDelay  int    `def:"500" min:"0" max:"5000"`
	HideDelay  int    `def:"350" min:"0" max:"5000"`
	MoveDelay  int    `def:"100" min:"0" max:"2000"`
}

type PreviewStyle struct {
	Size         int `def:"120" min:"40" max:"512"`
	BorderRadius int `def:"0" min:"0" max:"128"`
	Padding      int `def:"10" min:"0" max:"64"`
}

type ThemeGeneral struct {
	Spacing int `def:"5" min:"0" max:"64"`
}

// Appearance holds structured visual overrides. A value of 0 means "keep the
// theme as-is" for numeric fields; overrides are applied through a generated
// runtime CSS provider so the user's style.css is never rewritten.
type Appearance struct {
	PanelOpacity int    `def:"0" min:"0" max:"100"`
	BorderRadius int    `def:"0" min:"0" max:"128"`
	BorderWidth  int    `def:"0" min:"0" max:"16"`
	PanelPadding int    `def:"0" min:"0" max:"64"`
	HoverEffects bool   `def:"true"`
	ActiveTint   bool   `def:"true"`
	Accent       string `def:""`
}

// Displays selects where the dock is shown. The current architecture hosts a
// single layer surface, so "all" is reserved for future per-monitor instances;
// "specific" pins the dock to one named output.
type Displays struct {
	Mode        string `def:"focused" valid:"focused,primary,specific,all"`
	MonitorName string `def:""`
}

type Theme struct {
	ThemeGeneral `section:"Theme"`
	PreviewStyle PreviewStyle `section:"Theme.preview"`
}

type Config struct {
	General `section:"General"`
	Preview Preview `section:"General.preview"`

	Appearance Appearance `section:"Appearance"`
	Displays   Displays   `section:"Displays"`

	Theme
	ThemeDir  string
	ThemeConf string

	configPath string
	themesDir  string
	logger     hclog.Logger
}

// ConfigPath returns the file the config was loaded from.
func (c *Config) ConfigPath() string { return c.configPath }

// ThemesDirPath returns the themes directory used for the current theme.
func (c *Config) ThemesDirPath() string { return c.themesDir }

// SetPaths pins the config/theme paths — used when constructing a config
// outside of New (e.g. defaults fallback after a load failure).
func (c *Config) SetPaths(configPath, themesDir string) {
	c.configPath = configPath
	c.themesDir = themesDir
}

func New(configPath string, themesDir string, logger hclog.Logger) (*Config, error) {
	var err error

	if logger == nil {
		logger = hclog.NewNullLogger()
	}

	// MAIN CONFIG
	conf := ini.New(configPath, logger)

	var config Config
	err = conf.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// THEME
	themeDir := filepath.Join(themesDir, config.CurrentTheme)
	themeConf := filepath.Join(themeDir, "theme.conf")
	th := ini.New(themeConf, logger)

	var theme Theme
	err = th.Unmarshal(&theme)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	config.Theme = theme
	config.ThemeDir = themeDir
	config.ThemeConf = themeConf

	config.configPath = configPath
	config.themesDir = themesDir
	config.logger = logger

	return &config, nil
}

// type Config struct {
// 	CurrentTheme    string
// 	IconSize        int
// 	Layer           string
// 	Exclusive       string
// 	SmartView       string
// 	Position        string
// 	Spacing         int
// 	AutoHideDelay   int
// 	SystemGapUsed   string
// 	Margin          int
// 	ContextPos      int
// 	Preview         string
// 	PreviewAdvanced struct {
// 		FPS        int
// 		BufferSize int
// 		ShowDelay  int
// 		HideDelay  int
// 		MoveDelay  int
// 	}
// 	PreviewStyle struct {
// 		Size         int
// 		BorderRadius int
// 		Padding      int
// 	}
// }

// type ThemeConfig struct {
// 	Blur         string
// 	Spacing      int
// 	PreviewStyle struct {
// 		Size         int
// 		BorderRadius int
// 		Padding      int
// 	}
// }

// type ItemList struct {
// 	Pinned []string
// }

// func GetDefaultConfig() Config {
// 	return Config{
// 		CurrentTheme:  "lotos",
// 		IconSize:      21,
// 		Layer:         "top",
// 		Exclusive:     "true",
// 		SmartView:     "false",
// 		Position:      "bottom",
// 		Spacing:       8,
// 		SystemGapUsed: "true",
// 		AutoHideDelay: 400,
// 		Margin:        8,
// 		ContextPos:    5,
// 		Preview:       "none",
// 		PreviewAdvanced: struct {
// 			FPS        int
// 			BufferSize int
// 			ShowDelay  int
// 			HideDelay  int
// 			MoveDelay  int
// 		}{
// 			FPS:        30,
// 			BufferSize: 5,
// 			ShowDelay:  600,
// 			HideDelay:  300,
// 			MoveDelay:  200,
// 		},
// 		PreviewStyle: struct {
// 			Size         int
// 			BorderRadius int
// 			Padding      int
// 		}{
// 			Size:         120,
// 			BorderRadius: 0,
// 			Padding:      10,
// 		},
// 	}
// }

// func ReadConfig(jsoncFile string, themesDir string) Config {
// 	// Read jsonc
// 	config := Config{}
// 	err := ReadJsonc(jsoncFile, &config)
// 	if err != nil {
// 		log.Println(err)
// 		log.Println("Load default config")
// 		return GetDefaultConfig()
// 	}

// 	// Set default values ​​if not specified
// 	if config.CurrentTheme == "" {
// 		config.CurrentTheme = GetDefaultConfig().CurrentTheme
// 		log.Println("The theme is not set, the default theme is currently used - \"lotos\"")
// 	}

// 	if !validate.Layer(config.Layer, false) {
// 		ValidWarn("Layer", config.Layer, GetDefaultConfig().Layer, jsoncFile)
// 		config.Layer = GetDefaultConfig().Layer
// 	}

// 	if !validate.Boolen(config.Exclusive) {
// 		ValidWarn("Exclusive", config.Exclusive, GetDefaultConfig().Exclusive, jsoncFile)
// 		config.Exclusive = GetDefaultConfig().Exclusive
// 	}

// 	if !validate.Boolen(config.SmartView) {
// 		ValidWarn("SmartView", config.SmartView, GetDefaultConfig().SmartView, jsoncFile)
// 		config.SmartView = GetDefaultConfig().SmartView
// 	}

// 	if !validate.Position(config.Position, false) {
// 		config.Position = GetDefaultConfig().Position
// 	}

// 	if !validate.SystemGapUsed(config.SystemGapUsed, false) {
// 		config.SystemGapUsed = GetDefaultConfig().SystemGapUsed
// 	}

// 	if !validate.Preview(config.Preview, false) {
// 		config.Preview = GetDefaultConfig().Preview
// 	}

// 	if config.PreviewAdvanced.FPS == 0 {
// 		config.PreviewAdvanced.FPS = GetDefaultConfig().PreviewAdvanced.FPS
// 	}

// 	if config.PreviewAdvanced.BufferSize < 1 || config.PreviewAdvanced.BufferSize > 20 {
// 		config.PreviewAdvanced.BufferSize = GetDefaultConfig().PreviewAdvanced.BufferSize
// 	}

// 	if config.Spacing < 1 {
// 		config.Spacing = GetDefaultConfig().Spacing
// 	}

// 	if config.IconSize < 1 {
// 		config.IconSize = GetDefaultConfig().IconSize
// 	}

// 	return config
// }

// func ReadTheme(jsoncFile string, config Config) *ThemeConfig {
// 	// Read jsonc
// 	themeConfig := ThemeConfig{}
// 	err := ReadJsonc(jsoncFile, &themeConfig)
// 	if err != nil {
// 		log.Println(err)
// 		log.Println("Load default config")
// 		return nil
// 	}

// 	// Set default values ​​if not specified
// 	if themeConfig.Spacing < 0 {
// 		themeConfig.Spacing = config.Spacing
// 	}

// 	if themeConfig.PreviewStyle.Size < 20 {
// 		themeConfig.PreviewStyle.Size = GetDefaultConfig().PreviewStyle.Size
// 	}

// 	return &themeConfig
// }

// func ReadItemList(jsonFile string) []string {
// 	itemList := ItemList{}

// 	if !utils.FileExists(jsonFile) {
// 		itemList.Pinned = CreateEmptyPinnedFile(jsonFile)
// 		return itemList.Pinned
// 	}

// 	err := ReadJson(jsonFile, &itemList)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	return itemList.Pinned
// }

// func ReadJsonc(jsoncFile string, v interface{}) error {
// 	file, err := os.ReadFile(jsoncFile)
// 	if err != nil {
// 		return errors.Wrapf(err, "file %q not found", jsoncFile)
// 	}

// 	// Парсим JSONC
// 	standardized, err := hujson.Standardize(file)
// 	if err != nil {
// 		return errors.Wrapf(err, "failed to standardize JSONC")
// 	}

// 	if err := json.Unmarshal(standardized, &v); err != nil {
// 		return errors.Wrapf(err, "file %q has a syntax error", jsoncFile)
// 	}

// 	return nil
// }

// func ChangeJsonPinnedApps(apps []string, jsonFile string) error {
// 	itemList := ItemList{
// 		Pinned: apps,
// 	}

// 	if err := WriteItemList(jsonFile, itemList); err != nil {
// 		log.Println("Error", jsonFile, "writing: ", err)
// 		return err
// 	}

// 	return nil
// }

// func ReadJson(jsonFile string, v interface{}) error {
// 	file, err := os.Open(jsonFile)
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()

// 	decoder := json.NewDecoder(file)
// 	if err := decoder.Decode(&v); err != nil {
// 		return err
// 	}

// 	return nil
// }

// func CreateEmptyPinnedFile(jsonFile string) []string {
// 	initialData := ItemList{
// 		Pinned: []string{},
// 	}

// 	if err := WriteItemList(jsonFile, initialData); err != nil {
// 		log.Fatalf("Failed to create file %q: %v", jsonFile, err)
// 		return nil
// 	}

// 	return initialData.Pinned
// }

// func WriteItemList(jsonFile string, data ItemList) error {
// 	file, err := os.Create(jsonFile)
// 	if err != nil {
// 		return errors.Wrapf(err, "failed to create file %q", jsonFile)
// 	}
// 	defer file.Close()

// 	encoder := json.NewEncoder(file)
// 	encoder.SetIndent("", "  ")
// 	if err := encoder.Encode(data); err != nil {
// 		return errors.Wrapf(err, "failed to encode data to file %q", jsonFile)
// 	}

// 	return nil
// }

// func ValidWarn(key string, value string, defaultVal, file string) {
// 	log.Printf("ERROR: config value \"%s\":\"%s\" is not valid in file: %s", key, value, file)
// 	log.Printf("DEBUG: default value \"%s\":\"%s\" is used", key, defaultVal)
// }
