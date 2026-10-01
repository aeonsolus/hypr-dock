package conf

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"hypr-dock/pkg/ini"
)

// mainSection is the section keys land in before the first header of the
// dock's config file.
const mainSection = "General"

// walkSections visits every struct field pair that maps to an ini section.
// fn receives the fully-qualified section name and the value of the struct
// holding that section's leaves. Nested subsections recurse.
func walkSections(v reflect.Value, path []string, fn func(section string, v reflect.Value)) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() || field.Type.Kind() != reflect.Struct {
			continue
		}

		name, hasSection := field.Tag.Lookup("section")
		if hasSection {
			fn(sectionPath(append(path, name)), v.Field(i))
			walkSections(v.Field(i), append(path, name), fn)
			continue
		}

		if field.Anonymous {
			walkSections(v.Field(i), path, fn)
		}
	}
}

// walkLeaves visits the tagged leaf fields of a section struct.
func walkLeaves(v reflect.Value, fn func(field reflect.StructField, value reflect.Value)) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		if field.Type.Kind() == reflect.Struct {
			continue
		}
		if _, ok := field.Tag.Lookup("def"); !ok {
			continue
		}
		fn(field, v.Field(i))
	}
}

// sectionPath joins path parts with dots; empty paths collapse to mainSection.
func sectionPath(path []string) string {
	if len(path) == 0 {
		return mainSection
	}
	return strings.Join(path, ".")
}

// MarshalMap converts the typed config back into section/key/value form so it
// can be written with ini.Update, which preserves comments, unknown keys and
// ordering. Only fields carrying a `def` tag participate.
func (c *Config) MarshalMap() map[string]map[string]string {
	out := make(map[string]map[string]string)

	walkSections(reflect.ValueOf(c), nil, func(section string, v reflect.Value) {
		keys := make(map[string]string)
		walkLeaves(v, func(field reflect.StructField, value reflect.Value) {
			keys[field.Name] = formatValue(value)
		})
		if len(keys) > 0 {
			out[section] = keys
		}
	})

	return out
}

func formatValue(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			return "true"
		}
		return "false"
	case reflect.Int:
		return strconv.Itoa(int(v.Int()))
	case reflect.String:
		return v.String()
	default:
		return fmt.Sprint(v.Interface())
	}
}

// Save writes the typed config to path. Values keep the file's comments and
// unknown keys; the write is atomic. Use the same path the config was loaded
// from unless migrating.
func (c *Config) Save(path string) error {
	return ini.Update(path, mainSection, c.MarshalMap())
}

// SaveTheme writes the theme-specific values (Spacing, preview geometry) to
// the theme's theme.conf.
func (c *Config) SaveTheme() error {
	theme := make(map[string]map[string]string)

	walkSections(reflect.ValueOf(&c.Theme), nil, func(section string, v reflect.Value) {
		keys := make(map[string]string)
		walkLeaves(v, func(field reflect.StructField, value reflect.Value) {
			keys[field.Name] = formatValue(value)
		})
		if len(keys) > 0 {
			theme[section] = keys
		}
	})

	return ini.Update(c.ThemeConf, "Theme", theme)
}

// Defaults builds a fully default configuration. Parsed from an empty file so
// defaults always come from the same tag source the loader uses.
func Defaults() *Config {
	conf := ini.NewManagerEmpty()
	var config Config
	_ = conf.Unmarshal(&config)
	return &config
}

// ResetSection restores one section (e.g. "General", "General.preview",
// "Appearance", "Displays", "Theme", "Theme.preview") to its defaults in
// place and reports whether the section exists.
func (c *Config) ResetSection(name string) bool {
	var found bool

	walkSections(reflect.ValueOf(c), nil, func(section string, v reflect.Value) {
		if section != name {
			return
		}
		found = true

		def := Defaults()
		walkSections(reflect.ValueOf(def), nil, func(dSection string, dV reflect.Value) {
			if dSection != name {
				return
			}
			walkLeaves(dV, func(field reflect.StructField, value reflect.Value) {
				target := v.FieldByName(field.Name)
				if target.IsValid() && target.CanSet() {
					target.Set(value)
				}
			})
		})
	})

	return found
}

// Validate checks the typed values against their tag constraints and returns
// a sorted list of human-readable problems. The loader already falls back to
// defaults for malformed entries, so this is about surfacing bad values.
func (c *Config) Validate() []string {
	var problems []string

	check := func(section, key, value, valid string) {
		if valid == "" {
			return
		}
		for _, candidate := range strings.Split(valid, ",") {
			if strings.TrimSpace(candidate) == value {
				return
			}
		}
		problems = append(problems, fmt.Sprintf("%s.%s: invalid value %q", section, key, value))
	}

	check("General", "Layer", c.Layer, "background,bottom,top,overlay")
	check("General", "Position", c.Position, "top,bottom,left,right")
	check("General", "ClickAction", c.ClickAction, "launch,focus,minimize,show,cycle,none")
	check("General", "ClickActionFocused", c.ClickActionFocused, "minimize,show,cycle,none")
	check("General", "MiddleClickAction", c.MiddleClickAction, "launch,none")
	check("General", "LauncherPosition", c.LauncherPosition, "start,end")
	check("General.preview", "Mode", strings.TrimSpace(c.Preview.Mode), "none,static,live")
	check("Displays", "Mode", c.Displays.Mode, "focused,primary,specific,all")

	inRange := func(section, key string, v, min, max int) {
		if v < min {
			problems = append(problems, fmt.Sprintf("%s.%s: %d below minimum %d", section, key, v, min))
		}
		if v > max {
			problems = append(problems, fmt.Sprintf("%s.%s: %d above maximum %d", section, key, v, max))
		}
	}

	inRange("General", "IconSize", c.IconSize, 12, 128)
	inRange("General", "Margin", c.Margin, 0, 64)
	inRange("General", "ContextPos", c.ContextPos, 0, 64)
	inRange("General", "AutoHideDelay", c.AutoHideDelay, 0, 10000)
	inRange("General", "FollowAnimMs", c.FollowAnimMs, 0, 2000)
	inRange("General", "IndicatorSize", c.IndicatorSize, 20, 100)
	inRange("General.preview", "FPS", c.Preview.FPS, 1, 120)
	inRange("General.preview", "BufferSize", c.Preview.BufferSize, 1, 20)
	inRange("Theme", "Spacing", c.Spacing, 0, 64)
	inRange("Theme.preview", "Size", c.PreviewStyle.Size, 40, 512)
	inRange("Theme.preview", "BorderRadius", c.PreviewStyle.BorderRadius, 0, 128)
	inRange("Theme.preview", "Padding", c.PreviewStyle.Padding, 0, 64)
	inRange("Appearance", "PanelOpacity", c.Appearance.PanelOpacity, 0, 100)
	inRange("Appearance", "BorderRadius", c.Appearance.BorderRadius, 0, 128)
	inRange("Appearance", "BorderWidth", c.Appearance.BorderWidth, 0, 16)
	inRange("Appearance", "PanelPadding", c.Appearance.PanelPadding, 0, 64)

	sort.Strings(problems)
	return problems
}
