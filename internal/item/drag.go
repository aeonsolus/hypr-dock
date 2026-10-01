package item

import (
	"slices"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"

	"hypr-dock/internal/pkg/pinned"
)

// Drag-to-reorder support.
//
// Items are plain gtk.Buttons inside the dock's items box. We track a left
// press + motion beyond a small threshold as a drag, reorder the box live
// while the pointer moves, and on release persist the new relative order of
// pinned apps back to the pinned file. A drag is a global session state so
// moving over neighbouring icons keeps updating the same drag; normal clicks
// are unaffected (they never cross the threshold).

const dragThresholdSq = 100.0 // 10px squared

var (
	dragBox   *gtk.Box
	dragState *dragContext
)

type dragContext struct {
	item   *Item
	startX float64
	startY float64
	active bool
}

// InitDrag stores the dock's item container. Call once after the items box
// is created (before/after rendering items is both fine).
func InitDrag(box *gtk.Box) {
	dragBox = box
}

// DragActive reports whether a drag is currently engaged. The dock's click
// handlers use it to suppress the click that would otherwise fire on the icon
// the pointer happens to be over when the drag is released.
func DragActive() bool {
	return dragState != nil && dragState.active
}

// AttachDrag wires press/motion/release handlers on an item's button.
func (i *Item) AttachDrag() {
	if i == nil || i.Button == nil {
		return
	}

	btn := i.Button
	btn.AddEvents(int(gdk.POINTER_MOTION_MASK))

	btn.Connect("button-press-event", func(w *gtk.Button, e *gdk.Event) bool {
		eb := gdk.EventButtonNewFromEvent(e)
		if eb.Button() != 1 {
			return false
		}
		dragState = &dragContext{
			item:   i,
			startX: eb.X(),
			startY: eb.Y(),
		}
		// keep the event flowing so the button's normal press handling runs
		return false
	})

	btn.Connect("motion-notify-event", func(w *gtk.Button, e *gdk.Event) bool {
		if dragState == nil {
			return false
		}
		em := gdk.EventMotionNewFromEvent(e)
		x, y := em.MotionVal()

		if !dragState.active {
			dx := x - dragState.startX
			dy := y - dragState.startY
			if dx*dx+dy*dy < dragThresholdSq {
				return false
			}
			dragState.active = true
			dragState.item.Button.SetOpacity(0.55)
		}

		reorderUnderPointer(w, x, y)
		return false
	})

	btn.Connect("button-release-event", func(w *gtk.Button, e *gdk.Event) bool {
		eb := gdk.EventButtonNewFromEvent(e)
		if eb.Button() != 1 {
			return false
		}
		if dragState != nil {
			if dragState.active {
				dragState.item.Button.SetOpacity(1)
				persistOrder()
			}
			dragState = nil
		}
		return false
	})
}

// reorderUnderPointer moves the dragged item to the position under the
// pointer. x/y are widget-relative to the button currently under the pointer.
func reorderUnderPointer(under *gtk.Button, x, y float64) {
	if dragBox == nil || dragState == nil || dragState.item == nil || dragState.item.ButtonBox == nil {
		return
	}

	horizontal := dragBox.GetOrientation() == gtk.ORIENTATION_HORIZONTAL

	// Convert pointer position to items-box coordinates.
	btnAlloc := under.GetAllocation()
	boxAlloc := dragBox.GetAllocation()
	var pos float64
	if horizontal {
		pos = float64(btnAlloc.GetX()) + x - float64(boxAlloc.GetX())
	} else {
		pos = float64(btnAlloc.GetY()) + y - float64(boxAlloc.GetY())
	}

	// Find the index whose leading edge the pointer has crossed.
	acc := 0.0
	target := -1
	idx := 0
	for l := dragBox.GetChildren(); l != nil && l.Data() != nil; l = l.Next() {
		w, ok := l.Data().(gtk.IWidget)
		if !ok {
			idx++
			continue
		}
		alloc := w.ToWidget().GetAllocation()
		var size int
		if horizontal {
			size = alloc.GetWidth()
		} else {
			size = alloc.GetHeight()
		}
		acc += float64(size)
		if pos < acc {
			target = idx
			break
		}
		idx++
	}
	if target == -1 {
		target = idx
	}

	// Find the dragged item's current index. GetChildren() wraps children in
	// fresh Go wrappers, so identity must be compared by native pointer.
	cur := -1
	idx2 := 0
	for l := dragBox.GetChildren(); l != nil && l.Data() != nil; l = l.Next() {
		if w, ok := l.Data().(gtk.IWidget); ok && w.ToWidget().Native() == dragState.item.ButtonBox.ToWidget().Native() {
			cur = idx2
			break
		}
		idx2++
	}
	if cur >= 0 && target != cur {
		dragBox.ReorderChild(dragState.item.ButtonBox, target)
	}
}

// persistOrder writes the new relative order of pinned apps to disk so the
// rearrangement survives a dock restart. Running (unpinned) apps keep their
// session position but are not persisted.
func persistOrder() {
	if dragBox == nil || dragState == nil || dragState.item == nil {
		return
	}

	list := dragState.item.List
	if list == nil {
		return
	}

	var order []string
	for l := dragBox.GetChildren(); l != nil && l.Data() != nil; l = l.Next() {
		w, ok := l.Data().(gtk.IWidget)
		if !ok {
			continue
		}
		targetNative := w.ToWidget().Native()
		for cn, it := range list {
			if it.ButtonBox != nil && it.ButtonBox.ToWidget().Native() == targetNative {
				order = append(order, cn)
				break
			}
		}
	}

	pinnedSet := make(map[string]bool, len(*dragState.item.PinnedList))
	for _, cn := range *dragState.item.PinnedList {
		pinnedSet[cn] = true
	}

	var pins []string
	for _, cn := range order {
		if pinnedSet[cn] {
			pins = append(pins, cn)
		}
	}
	// Keep any pinned app that is not currently rendered (e.g. hidden for some
	// reason) at the end of the list.
	for _, cn := range *dragState.item.PinnedList {
		if !slices.Contains(pins, cn) {
			pins = append(pins, cn)
		}
	}

	if slices.Equal(pins, *dragState.item.PinnedList) {
		return
	}

	*dragState.item.PinnedList = pins
	if err := pinned.Save(dragState.item.Settings.PinnedPath, pins); err != nil {
		dragState.item.log.Error("Failed to save pinned order", "file", dragState.item.Settings.PinnedPath, "error", err)
	}
}