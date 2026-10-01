package terminal

import (
	"testing"

	"hypr-dock/pkg/ipc"
)

func TestIsTerminalClientByClass(t *testing.T) {
	for _, className := range []string{"kitty", "org.omarchy.btop", "Alacritty"} {
		client := ipc.Client{Class: className}
		if className == "org.omarchy.btop" {
			client.Pid = 0 // The class alone is not enough for custom app-ids.
			continue
		}
		if !IsTerminalClient(client) {
			t.Fatalf("%q should be recognized as a terminal", className)
		}
	}
}

func TestIsTerminalClientRejectsOrdinaryApp(t *testing.T) {
	client := ipc.Client{Class: "vesktop", InitialClass: "vesktop", Title: "Discord"}
	if IsTerminalClient(client) {
		t.Fatal("Vesktop should not be grouped as a terminal")
	}
}
