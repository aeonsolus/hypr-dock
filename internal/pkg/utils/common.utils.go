package utils

import (
	"regexp"
	"strings"

	"hypr-dock/pkg/ipc"

	"github.com/hashicorp/go-hclog"
)

func GetSingleValue[K comparable, V any](m map[K]V) (V, bool) {
	for _, v := range m {
		return v, true
	}
	var zero V
	return zero, false
}

// GetFocusedValue picks the entry with the highest FocusHistoryID — Hyprland
// focus history for ipc.Clients. Falls back to the first value when the type
// has no focus metadata.
func GetFocusedValue[K comparable, V any](m map[K]V) (V, bool) {
	var best V
	found := false
	bestHistory := -1
	for _, v := range m {
		if !found {
			best, found = v, true
		}
		if client, ok := any(v).(ipc.Client); ok {
			if client.FocusHistoryID > bestHistory {
				bestHistory = client.FocusHistoryID
				best = v
			}
		} else if clientPtr, ok := any(v).(*ipc.Client); ok && clientPtr != nil {
			if clientPtr.FocusHistoryID > bestHistory {
				bestHistory = clientPtr.FocusHistoryID
				best = v
			}
		}
	}
	return best, found
}

func СreateLogger(logLevel string) hclog.Logger {
	level := hclog.LevelFromString(logLevel)

	if level == hclog.NoLevel {
		level = hclog.Info
	}

	return hclog.New(&hclog.LoggerOptions{
		Name:  "hypr-dock",
		Level: level,
		Color: hclog.AutoColor,
	})
}

func NormaliseTitle(title string) string {
	re := regexp.MustCompile(`^[a-zA-Z-]+`)
	firstWord := re.FindString(title)

	return strings.ToLower(firstWord)
}
