package settingsapp

import (
	"path/filepath"
	"reflect"
	"testing"

	"hypr-dock/internal/pkg/pinned"
	"hypr-dock/internal/settings"
)

func TestMovePinsPreservesSyntheticSlots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "order")
	if err := pinned.Save(path, []string{"app-a", "hypr-dock-trash", "app-b", "terminal-group"}); err != nil {
		t.Fatal(err)
	}
	p := &ApplicationsPage{app: &App{config: &settings.Settings{OrderPath: path}, pins: []string{"app-a", "app-b"}}}
	p.swapPins(0, 1)
	order, err := pinned.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"app-b", "hypr-dock-trash", "app-a", "terminal-group"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	if !reflect.DeepEqual(p.app.pins, []string{"app-b", "app-a"}) {
		t.Fatalf("pins = %v", p.app.pins)
	}
}
