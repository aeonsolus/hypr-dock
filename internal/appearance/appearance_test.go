package appearance

import "testing"

const sampleTheme = `#app {
	background-color: rgba(12, 18, 30, 0.85);
	border-radius: 14px;
	border: 2px solid rgb(90, 100, 110);
	padding: 4px 8px 4px 8px;
}`

func TestParseAppBlockExtractsColors(t *testing.T) {
	block := ParseAppBlock(sampleTheme)

	if !block.HasBg {
		t.Fatal("HasBg not set")
	}
	if block.BgR != 12 || block.BgG != 18 || block.BgB != 30 {
		t.Fatalf("bg = %v/%v/%v", block.BgR, block.BgG, block.BgB)
	}
	if block.BgA != 0.85 {
		t.Fatalf("alpha = %v", block.BgA)
	}
	if block.Radius != 14 {
		t.Fatalf("radius = %d", block.Radius)
	}
	if !block.HasBorder || block.BorderWidth != 2 {
		t.Fatalf("border = %+v", block)
	}
	if block.PaddingTop != 4 || block.PaddingRight != 8 {
		t.Fatalf("padding = %+v", block)
	}
}

func TestParseAppBlockEmptyOnMissingRule(t *testing.T) {
	if block := ParseAppBlock("nothing here"); block.HasBg {
		t.Fatal("parsed bg from css without #app rule")
	}
}

func TestOverridePanelOpacityScalesAlpha(t *testing.T) {
	// PanelOpacity is the panel's alpha in percent (50 -> 0.50); 0 keeps the
	// theme's own value.
	got := Override(sampleTheme, 50, 0, 0, 0, false, false, "", nil)
	if !contains(got, "background-color: rgba(12, 18, 30, 0.50)") {
		t.Fatalf("opacity override not applied:\n%s", got)
	}
}

func TestOverrideOpacityZeroMeansKeepTheme(t *testing.T) {
	got := Override(sampleTheme, 0, 0, 0, 0, false, false, "", nil)
	if contains(got, "0.50") {
		t.Fatalf("zero override must not alter alpha:\n%s", got)
	}
}

func TestOverrideAccentUsedForRunningDot(t *testing.T) {
	// The accent tints the active item — only emitted when ActiveTint is on.
	got := Override(sampleTheme, 0, 0, 0, 0, false, true, "#ff8800", nil)
	if !contains(got, "255, 136, 0, 0.18") {
		t.Fatalf("accent not used:\n%s", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
