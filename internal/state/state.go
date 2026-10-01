package state

import (
	"hypr-dock/internal/itemsctl"
	"hypr-dock/internal/layering"
	"hypr-dock/internal/pvctl"
	"hypr-dock/internal/settings"
	"sync"

	"github.com/gotk3/gotk3/gtk"
	"github.com/hashicorp/go-hclog"
)

type State struct {
	logger hclog.Logger

	settings *settings.Settings
	window   *gtk.Window
	layerctl *layering.Control
	appBox   *gtk.Box
	itemsBox *gtk.Box
	list     *itemsctl.List
	pv       *pvctl.PV

	themeProvider    *gtk.CssProvider
	overrideProvider *gtk.CssProvider

	activeAddress string
	mu            sync.Mutex
}

func New(settings *settings.Settings, logger hclog.Logger) *State {
	return &State{
		settings: settings,
		list:     itemsctl.New(),
		pv:       pvctl.New(settings, logger),
		logger:   logger,
	}
}

func (s *State) SetLogger(logger hclog.Logger) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger = logger
}

func (s *State) GetLogger() hclog.Logger {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.logger
}

func (s *State) SetLayerctl(ctl *layering.Control) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.layerctl = ctl
}

func (s *State) GetLayerctl() *layering.Control {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.layerctl
}

func (s *State) GetList() *itemsctl.List {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.list
}

func (s *State) SetSettings(settings *settings.Settings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = settings
}

func (s *State) GetSettings() *settings.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings
}

func (s *State) GetPinned() *[]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &s.settings.PinnedApps
}

func (s *State) SetWindow(window *gtk.Window) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.window = window
}

func (s *State) SetItemsBox(box *gtk.Box) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.itemsBox = box
}

func (s *State) GetWindow() *gtk.Window {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.window
}

func (s *State) GetItemsBox() *gtk.Box {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.itemsBox
}

func (s *State) GetPV() *pvctl.PV {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.pv
}

func (s *State) SetAppBox(box *gtk.Box) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.appBox = box
}

func (s *State) GetAppBox() *gtk.Box {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.appBox
}

func (s *State) SetThemeProvider(provider *gtk.CssProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.themeProvider = provider
}

func (s *State) GetThemeProvider() *gtk.CssProvider {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.themeProvider
}

func (s *State) SetOverrideProvider(provider *gtk.CssProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.overrideProvider = provider
}

func (s *State) GetOverrideProvider() *gtk.CssProvider {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.overrideProvider
}

// SetActiveAddress records the focused window address ("0x..." or "").
func (s *State) SetActiveAddress(address string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeAddress = address
}

func (s *State) GetActiveAddress() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeAddress
}
