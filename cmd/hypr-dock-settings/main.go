package main

import (
	"fmt"
	"os"

	"github.com/allan-simon/go-singleinstance"
	"github.com/gotk3/gotk3/gtk"
	"github.com/hashicorp/go-hclog"

	"hypr-dock/internal/pkg/utils"
	"hypr-dock/internal/settingsapp"
)

func main() {
	logger := utils.СreateLogger("info")

	// One settings window at a time keeps edits predictable.
	lockFilePath := fmt.Sprintf("%s/hypr-dock-settings.lock", os.TempDir())
	lock, err := singleinstance.CreateLockFile(lockFilePath)
	if err != nil {
		logger.Info("Settings already open — focusing existing window is not supported; exiting")
		return
	}
	defer lock.Close()

	gtk.Init(nil)

	window, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		logger.Error("Unable to create window", "error", err)
		os.Exit(2)
	}

	settingsapp.New(window, hclog.NewNullLogger())

	window.Connect("destroy", func() { gtk.MainQuit() })
	window.ShowAll()

	gtk.Main()
}
