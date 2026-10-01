package appearance

import (
	"regexp"
	"strconv"
	"strings"
)

// AppBlock holds the visual properties hypr-dock themes express in the #app
// rule of style.css. Parsed so structured overrides can recompute derived
// values (e.g. opacity from the theme's base color) without rewriting the
// user's CSS.
type AppBlock struct {
	BgR, BgG, BgB float64
	BgA           float64
	HasBg         bool

	Radius                    int
	BorderWidth               int
	BorderR, BorderG, BorderB float64
	BorderA                   float64
	HasBorder                 bool

	PaddingTop, PaddingRight, PaddingBottom, PaddingLeft int
	HasPadding                                           bool
}

var (
	appBlockRe    = regexp.MustCompile(`(?m)(?:^|\n)\s*#app\s*\{([^}]*)\}`)
	bgColorRe     = regexp.MustCompile(`(?is)background-color\s*:\s*rgba?\(([^)]*)\)`)
	radiusRe      = regexp.MustCompile(`(?is)border-radius\s*:\s*(\d+)`)
	borderRe      = regexp.MustCompile(`(?is)(?:^|;|\n)\s*border\s*:\s*([^;]+)`)
	borderColorRe = regexp.MustCompile(`(?is)rgba?\(([^)]*)\)|#[0-9a-fA-F]{6}`)
	paddingRe     = regexp.MustCompile(`(?is)padding\s*:\s*([^;]+)`)
	hexColorRe    = regexp.MustCompile(`#[0-9a-fA-F]{6}\b`)
	tomlKeyRe     = regexp.MustCompile(`(?m)^\s*` + `([a-zA-Z_]+)` + `\s*=\s*"?\s*(#[0-9a-fA-F]{6})` + `\s*"?`)
)

// parseColor converts a borderColorRe match to RGBA components (0-255 /
// 0-1). rgb() colors default to alpha 1.
func parseColor(match []string) (r, g, b, a float64, ok bool) {
	if match[1] != "" {
		vals := splitNumbers(match[1])
		if len(vals) < 3 {
			return 0, 0, 0, 0, false
		}
		r, g, b = vals[0], vals[1], vals[2]
		a = 1
		if len(vals) >= 4 {
			a = vals[3]
		}
		return r, g, b, a, true
	}

	hex := match[0]
	if !strings.HasPrefix(hex, "#") {
		return 0, 0, 0, 0, false
	}
	r, g, b = hexToComponents(hex)
	return r, g, b, 1, true
}

func hexToComponents(hex string) (r, g, b float64) {
	r = float64(hexByte(hex, 1))
	g = float64(hexByte(hex, 3))
	b = float64(hexByte(hex, 5))
	return r, g, b
}

func hexByte(hex string, offset int) float64 {
	if len(hex) < offset+2 {
		return 0
	}
	v, err := strconv.ParseUint(hex[offset:offset+2], 16, 8)
	if err != nil {
		return 0
	}
	return float64(v)
}

// ParseAppBlock extracts the #app rule's relevant properties from theme CSS.
func ParseAppBlock(css string) AppBlock {
	var block AppBlock

	m := appBlockRe.FindStringSubmatch(css)
	if m == nil {
		return block
	}
	body := m[1]

	if bg := bgColorRe.FindStringSubmatch(body); bg != nil {
		if vals := splitNumbers(bg[1]); len(vals) >= 3 {
			block.BgR, block.BgG, block.BgB = vals[0], vals[1], vals[2]
			block.BgA = 1
			if len(vals) >= 4 {
				block.BgA = vals[3]
			}
			block.HasBg = true
		}
	}

	if r := radiusRe.FindStringSubmatch(body); r != nil {
		block.Radius, _ = strconv.Atoi(r[1])
	}

	if b := borderRe.FindStringSubmatch(body); b != nil {
		line := b[1]
		width := regexp.MustCompile(`(\d+)px`).FindStringSubmatch(line)
		if width != nil {
			block.BorderWidth, _ = strconv.Atoi(width[1])
		}

		// Border colors appear as rgba(), rgb() or #hex after the width.
		if c := borderColorRe.FindStringSubmatch(line); c != nil {
			if r, g, bl, a, ok := parseColor(c); ok {
				block.BorderR, block.BorderG, block.BorderB = r, g, bl
				block.BorderA = a
				block.HasBorder = true
			}
		}
	}

	if p := paddingRe.FindStringSubmatch(body); p != nil {
		vals := strings.Fields(p[1])
		nums := make([]int, 0, 4)
		for _, v := range vals {
			trimmed := strings.TrimSuffix(v, "px")
			n, err := strconv.Atoi(trimmed)
			if err != nil {
				break
			}
			nums = append(nums, n)
		}
		switch len(nums) {
		case 1:
			block.PaddingTop, block.PaddingRight, block.PaddingBottom, block.PaddingLeft = nums[0], nums[0], nums[0], nums[0]
			block.HasPadding = true
		case 2:
			block.PaddingTop, block.PaddingBottom = nums[0], nums[0]
			block.PaddingRight, block.PaddingLeft = nums[1], nums[1]
			block.HasPadding = true
		case 3:
			block.PaddingTop = nums[0]
			block.PaddingRight, block.PaddingLeft = nums[1], nums[1]
			block.PaddingBottom = nums[2]
			block.HasPadding = true
		case 4:
			block.PaddingTop, block.PaddingRight, block.PaddingBottom, block.PaddingLeft = nums[0], nums[1], nums[2], nums[3]
			block.HasPadding = true
		}
	}

	return block
}

func splitNumbers(s string) []float64 {
	var out []float64
	for _, field := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	}) {
		n, err := strconv.ParseFloat(strings.TrimSpace(field), 64)
		if err != nil {
			return nil
		}
		out = append(out, n)
	}
	return out
}

// Palette is a small set of accent colors, usually from the Omarchy theme.
type Palette struct {
	BG     string // hex
	FG     string // hex
	Accent string // hex
}

// ParsePalette reads hex colors out of a colors.toml-style file, keyed by
// name. Background values may be quoted or bare.
func ParsePalette(content string) Palette {
	var pal Palette

	matches := tomlKeyRe.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		key, hex := strings.ToLower(m[1]), m[2]
		switch key {
		case "background":
			pal.BG = hex
		case "foreground":
			pal.FG = hex
		case "accent", "accentcolor", "accent_color":
			pal.Accent = hex
		}
	}
	if pal.Accent == "" {
		pal.Accent = pal.FG
	}

	return pal
}

// HexToRGB converts "#rrggbb" to 0-255 components.
func HexToRGB(hex string) (r, g, b int, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(v >> 16), int((v >> 8) & 0xFF), int(v & 0xFF), true
}

// Override builds the generated override CSS fragment for one appearance
// configuration. themeCSS is the active theme's style.css contents; pal may
// be nil. The fragment only restates properties the user explicitly set, so
// hand-tuned theme CSS survives untouched.
func Override(themeCSS string, panelOpacity, borderRadius, borderWidth, panelPadding int, hoverEffects, activeTint bool, accent string, pal *Palette) string {
	block := ParseAppBlock(themeCSS)

	var rules []string

	// Base panel color: omarchy palette wins when provided, else derive from
	// the theme's own rgba.
	baseR, baseG, baseB := block.BgR, block.BgG, block.BgB
	baseA := block.BgA
	if block.HasBg && baseA <= 0 {
		baseA = 0.22
	}
	if pal != nil {
		if r, g, b, ok := HexToRGB(pal.BG); ok {
			baseR, baseG, baseB = float64(r), float64(g), float64(b)
		}
		if block.BgA <= 0 {
			baseA = 0.22
		}
	}

	bgParts := make([]string, 0, 4)
	// Emit background-color only when the user set an explicit opacity or an
	// omarchy palette re-colors the panel; otherwise leave the theme rule.
	if panelOpacity > 0 {
		bgParts = append(bgParts, "background-color: "+fmtRGBA(baseR, baseG, baseB, float64(panelOpacity)/100))
	} else if pal != nil && (block.HasBg || pal != nil) {
		bgParts = append(bgParts, "background-color: "+fmtRGBA(baseR, baseG, baseB, baseA))
	}

	radius := block.Radius
	if borderRadius > 0 {
		radius = borderRadius
	}
	if radius > 0 {
		bgParts = append(bgParts, "border-radius: "+strconv.Itoa(radius)+"px")
	}

	if borderWidth > 0 {
		br, bg, bb := block.BorderR, block.BorderG, block.BorderB
		ba := block.BorderA
		if pal != nil {
			if r, g, b, ok := HexToRGB(pal.FG); ok {
				br, bg, bb = float64(r), float64(g), float64(b)
				ba = 0.14
			}
		} else if !block.HasBorder {
			br, bg, bb, ba = 255, 255, 255, 0.08
		}
		bgParts = append(bgParts, "border: "+strconv.Itoa(borderWidth)+"px solid "+fmtRGBA(br, bg, bb, ba))
	}

	padding := -1
	if panelPadding > 0 {
		padding = panelPadding
	}
	if padding >= 0 {
		bgParts = append(bgParts, "padding: "+strconv.Itoa(padding)+"px")
	}

	if len(bgParts) > 0 {
		rules = append(rules, "#app {\n  "+strings.Join(bgParts, ";\n  ")+";\n}")
	}

	if !hoverEffects {
		rules = append(rules, "button:hover {\n  background-color: rgba(0, 0, 0, 0);\n}")
	}

	if activeTint {
		a := accent
		if a == "" && pal != nil {
			a = pal.Accent
		}
		if r, g, b, ok := HexToRGB(a); ok {
			rules = append(rules, "#app button.active {\n  background-color: rgba("+strconv.Itoa(r)+", "+strconv.Itoa(g)+", "+strconv.Itoa(b)+", 0.18);\n}")
		}
	}

	return strings.Join(rules, "\n\n")
}

func fmtRGBA(r, g, b, a float64) string {
	return "rgba(" +
		strconv.Itoa(clamp255(int(r))) + ", " +
		strconv.Itoa(clamp255(int(g))) + ", " +
		strconv.Itoa(clamp255(int(b))) + ", " +
		strconv.FormatFloat(clampA(a), 'f', 2, 64) + ")"
}

func clamp255(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

func clampA(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
