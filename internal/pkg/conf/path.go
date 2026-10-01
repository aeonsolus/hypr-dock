package conf

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// SetByPath writes one typed value through a dotted path such as
// "General.IconSize" or "General.preview.FPS". The path resolves against the
// same section/leaf mapping used for marshalling, so every writable key has
// exactly one representation. Unknown paths or values are rejected; the
// field's min/max and valid tags are enforced so IPC writes cannot persist
// values the doctor would flag.
func (c *Config) SetByPath(path, value string) error {
	target, field, ok := c.resolve(path)
	if !ok {
		return fmt.Errorf("unknown setting: %s", path)
	}

	parsed, err := parseValue(field.Type, value)
	if err != nil {
		return fmt.Errorf("invalid value for %s: %w", path, err)
	}

	if err := enforceTagBounds(field, parsed, value); err != nil {
		return fmt.Errorf("invalid value for %s: %w", path, err)
	}

	target.Set(parsed)
	return nil
}

// enforceTagBounds checks a parsed value against the field's min/max (ints)
// and valid (enum) tags when present.
func enforceTagBounds(field reflect.StructField, parsed reflect.Value, raw string) error {
	if minStr, ok := field.Tag.Lookup("min"); ok {
		if minV, err := strconv.Atoi(minStr); err == nil && parsed.Int() < int64(minV) {
			return fmt.Errorf("%q is below minimum %s", raw, minStr)
		}
	}
	if maxStr, ok := field.Tag.Lookup("max"); ok {
		if maxV, err := strconv.Atoi(maxStr); err == nil && parsed.Int() > int64(maxV) {
			return fmt.Errorf("%q is above maximum %s", raw, maxStr)
		}
	}

	if valid, ok := field.Tag.Lookup("valid"); ok {
		value := strings.TrimSpace(raw)
		for _, candidate := range strings.Split(valid, ",") {
			if strings.TrimSpace(candidate) == value {
				return nil
			}
		}
		return fmt.Errorf("%q is not one of: %s", raw, strings.Join(strings.Split(valid, ","), "/"))
	}

	return nil
}

// GetByPath reads one value through a dotted path, formatted the way it is
// stored in the ini file.
func (c *Config) GetByPath(path string) (string, error) {
	target, _, ok := c.resolve(path)
	if !ok {
		return "", fmt.Errorf("unknown setting: %s", path)
	}
	return formatValue(target), nil
}

// resolve finds the addressable field value for a dotted path.
func (c *Config) resolve(path string) (reflect.Value, reflect.StructField, bool) {
	root := reflect.ValueOf(c).Elem()

	var (
		result      reflect.Value
		resultField reflect.StructField
		found       bool
	)

	walkSections(root, nil, func(section string, v reflect.Value) {
		walkLeaves(v, func(field reflect.StructField, value reflect.Value) {
			p := section + "." + field.Name
			if p == path && !found {
				found = true
				result = value
				resultField = field
			}
		})
	})

	return result, resultField, found
}

// Paths returns every writable dotted path, sorted. Used by doctor and tests.
func (c *Config) Paths() []string {
	var out []string

	root := reflect.ValueOf(c).Elem()
	walkSections(root, nil, func(section string, v reflect.Value) {
		walkLeaves(v, func(field reflect.StructField, value reflect.Value) {
			out = append(out, section+"."+field.Name)
		})
	})

	sort.Strings(out)
	return out
}

func parseValue(t reflect.Type, value string) (reflect.Value, error) {
	switch t.Kind() {
	case reflect.Bool:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "true", "1", "yes", "on":
			return reflect.ValueOf(true), nil
		case "false", "0", "no", "off":
			return reflect.ValueOf(false), nil
		default:
			return reflect.Value{}, fmt.Errorf("%q is not a boolean", value)
		}
	case reflect.Int:
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return reflect.Value{}, fmt.Errorf("%q is not a number", value)
		}
		return reflect.ValueOf(n), nil
	case reflect.String:
		return reflect.ValueOf(strings.TrimSpace(value)), nil
	default:
		return reflect.Value{}, fmt.Errorf("unsupported type %s", t)
	}
}
