package conf

import "testing"

func TestIsHiddenMatchesExactAndWildcards(t *testing.T) {
	config := &Config{General: General{HiddenApps: "scratch-* , *-test, *steam*, foot"}}

	cases := map[string]bool{
		"foot":               true,
		"scratch-app":        true, // prefix wildcard
		"something-test":     true, // suffix wildcard
		"my-steam-game":      true, // contains wildcard
		"FootClient":         false,
		"org.gnome.Nautilus": false,
		"":                   false,
	}

	for class, want := range cases {
		if got := config.IsHidden(class); got != want {
			t.Fatalf("IsHidden(%q) = %v, want %v", class, got, want)
		}
	}
}

func TestIsHiddenEmptyBlacklist(t *testing.T) {
	config := &Config{General: General{HiddenApps: "  ,  "}}
	if config.IsHidden("anything") {
		t.Fatal("blank blacklist hides everything")
	}
}
