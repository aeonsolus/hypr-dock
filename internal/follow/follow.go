package follow

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dlasky/gotk3-layershell/layershell"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	"hypr-dock/internal/layering"
	"hypr-dock/internal/state"
	"hypr-dock/pkg/ipc"
)

// Follow keeps the dock on the monitor that currently has the cursor and/or
// keyboard focus, moving it between monitors with a fast crisp slide-in from
// the screen edge. Disabled unless "FollowMouse = true" in hypr-dock.conf.

const (
	defaultCursorPollMs = 180
	animFrameMs         = 10 // ~100 fps
	// Off-screen stow distance when pre-hiding the dock before first show.
	preHideOffset = 4096
)

type Controller struct {
	appState      *state.State
	window        *gtk.Window
	layerctl      *layering.Control

	currentNative uintptr
	animSource    glib.SourceHandle
	lastAnim      time.Time
}

// Init starts the follow controller. Call after the window is shown.
func Init(appState *state.State) *Controller {
	c := &Controller{
		appState: appState,
		window:   appState.GetWindow(),
		layerctl: appState.GetLayerctl(),
	}

	if !c.enabled() {
		return c
	}

	ipc.AddEventListener("focusedmon", func(event string) {
		parts := strings.SplitN(event, ">>", 2)
		if len(parts) != 2 {
			return
		}
		name := strings.TrimSpace(strings.SplitN(parts[1], ",", 2)[0])
		glib.IdleAdd(func() {
			if m, err := c.monitorByName(name); err == nil {
				c.moveTo(m)
			}
		})
	}, true)

	go c.pollCursor()

	return c
}

func (c *Controller) enabled() bool {
	return c.appState.GetSettings().FollowMouse
}

func (c *Controller) pollCursor() {
	interval := time.Duration(defaultCursorPollMs) * time.Millisecond
	for {
		time.Sleep(interval)
		glib.IdleAdd(func() {
			if !c.enabled() {
				return
			}
			if m, err := c.monitorAtCursor(); err == nil {
				c.moveTo(m)
			}
		})
	}
}

func (c *Controller) monitorByName(name string) (*gdk.Monitor, error) {
	monitors, err := ipc.GetMonitors()
	if err != nil {
		return nil, err
	}
	for _, mon := range monitors {
		if mon.Name == name {
			return c.monitorAtPoint(mon.X, mon.Y)
		}
	}
	return nil, fmt.Errorf("monitor not found: %s", name)
}

func (c *Controller) monitorAtCursor() (*gdk.Monitor, error) {
	raw, err := ipc.Hyprctl("cursorpos")
	if err != nil {
		return nil, err
	}
	clean := strings.TrimSpace(string(raw))
	parts := strings.SplitN(clean, ",", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("bad cursorpos: %q", clean)
	}
	x, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	y, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("bad cursorpos: %q", clean)
	}
	return c.monitorAtPoint(x, y)
}

func (c *Controller) monitorAtPoint(x, y int) (*gdk.Monitor, error) {
	display, err := gdk.DisplayGetDefault()
	if err != nil {
		return nil, err
	}
	return display.GetMonitorAtPoint(x, y)
}

// moveTo moves the dock window to the target monitor and plays the slide-in.
func (c *Controller) moveTo(target *gdk.Monitor) {
	if target == nil {
		return
	}
	if c.currentNative != 0 && c.currentNative == target.Native() {
		return
	}

	layershell.SetMonitor(c.window, target)
	c.currentNative = target.Native()
	c.slideIn()
}

// PreHide moves the dock fully off its anchored edge before the window is
// first shown, so the appear animation plays exactly once (a single slide
// from off-screen) instead of a static pop followed by a re-slide.
func PreHide(window *gtk.Window, layerctl *layering.Control) {
	layershell.SetMargin(window, layerctl.GetEdge(), -preHideOffset)
}

// slideIn quickly slides the dock in from its anchored edge.
func (c *Controller) slideIn() {
	// Cancel any in-flight animation so two triggers can't overlap (the
	// appear animation used to play twice when both the focusedmon event
	// and the cursor poll fired around the same time).
	if c.animSource != 0 {
		glib.SourceRemove(c.animSource)
		c.animSource = 0
	}

	edge := c.layerctl.GetEdge()

	// Surface size along the anchored axis (height for top/bottom docks,
	// width for left/right docks).
	var hidden int
	if edge == layershell.LAYER_SHELL_EDGE_TOP || edge == layershell.LAYER_SHELL_EDGE_BOTTOM {
		hidden = c.window.GetAllocatedHeight()
	} else {
		hidden = c.window.GetAllocatedWidth()
	}
	if hidden <= 0 {
		hidden = 64
	}

	// Start off the edge, then animate the margin to rest (0; the inner app
	// box carries the configured gap).
	layershell.SetMargin(c.window, edge, -hidden)

	duration := c.appState.GetSettings().FollowAnimMs
	if duration <= 0 {
		duration = 150
	}

	start := time.Now()
	c.animSource = glib.TimeoutAdd(animFrameMs, func() bool {
		elapsed := time.Since(start).Milliseconds()
		progress := float64(elapsed) / float64(duration)
		if progress >= 1 {
			layershell.SetMargin(c.window, edge, 0)
			c.animSource = 0
			return false
		}
		// Cubic ease-out: fast start, crisp settle.
		eased := 1 - pow3(1-progress)
		margin := int(float64(hidden) * (eased - 1))
		layershell.SetMargin(c.window, edge, margin)
		return true
	})
}

func pow3(v float64) float64 {
	return v * v * v
}