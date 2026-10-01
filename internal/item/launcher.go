package item

import (
	"os/exec"
	"strings"

	"github.com/gotk3/gotk3/gtk"
	"github.com/hashicorp/go-hclog"

	"hypr-dock/internal/desktop"
	"hypr-dock/internal/pkg/utils"
	"hypr-dock/internal/settings"
	"hypr-dock/pkg/ipc"
)

// Launcher opens the user's preferred application launcher. It is not a dock
// application: it takes no windows, is excluded from drag persistence, and
// only appears when ShowLauncherButton is enabled.

const LauncherName = "hypr-dock-launcher"

// NewLauncher builds the launcher dock item.
func NewLauncher(s *settings.Settings, log hclog.Logger) (*Item, error) {
	orientation := gtk.ORIENTATION_VERTICAL
	switch s.Position {
	case "left", "right":
		orientation = gtk.ORIENTATION_HORIZONTAL
	}

	box, err := gtk.BoxNew(orientation, 0)
	if err != nil {
		return nil, err
	}

	button, err := gtk.ButtonNew()
	if err != nil {
		return nil, err
	}

	iconName := utils.GetFirstAvailableImage([]string{
		"applications-other",
		"view-grid-symbolic",
		"open-menu-symbolic",
		"application-x-executable",
	})
	if image, err := utils.CreateImage(iconName, s.IconSize); err == nil {
		button.SetImage(image)
	} else {
		log.Error("Unable to create launcher image", "error", err)
	}

	button.SetTooltipText("Applications")
	utils.SetCursorPointer(button.ToWidget())

	box.Add(button)

	self := &Item{
		Windows:        map[string]*ipc.Client{},
		Button:         button,
		ButtonBox:      box,
		App:            desktop.NewVirtual("Applications", iconName),
		ClassName:      LauncherName,
		Settings:       s,
		IndicatorImage: nil,
		log:            log,
	}

	button.Connect("clicked", func() {
		command := s.LauncherCommandOrDefault()
		if strings.TrimSpace(command) == "" {
			log.Warn("No launcher command available")
			return
		}
		if err := exec.Command("sh", "-c", command).Start(); err != nil {
			log.Error("Failed to launch", "command", command, "error", err)
		}
	})

	return self, nil
}
