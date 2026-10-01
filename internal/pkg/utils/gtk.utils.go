package utils

import (
	"math"
	"strings"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

func CreateImageWidthTransform(source string, size int, scaleFactor float64, rotate bool) (*gtk.Image, error) {
	scaleSize := int(math.Round(float64(size) * math.Max(scaleFactor, 0)))

	return CreateImage0(source, scaleSize, rotate)
}

func CreateImage(source string, size int) (*gtk.Image, error) {
	image, err := CreateImage0(source, size, false)
	if err == nil {
		return image, nil
	}
	return CreateImage0("image-missing", size, false)
}

func CreateImage0(source string, size int, rotate bool) (*gtk.Image, error) {
	var err error
	var pixbuf *gdk.Pixbuf

	image, err := gtk.ImageNew()
	if err != nil {
		return nil, err
	}

	scaleFactor := image.GetScaleFactor()
	physicalSize := size * scaleFactor

	if strings.Contains(source, "/") {
		pixbuf, err = gdk.PixbufNewFromFileAtSize(source, physicalSize, physicalSize)
	} else {
		theme, _ := gtk.IconThemeGetDefault()
		pixbuf, err = theme.LoadIcon(source, physicalSize, gtk.ICON_LOOKUP_FORCE_SIZE)
	}
	if err != nil {
		return nil, err
	}

	if rotate {
		rotPixbuf, err := pixbuf.RotateSimple(gdk.PIXBUF_ROTATE_COUNTERCLOCKWISE)
		if err == nil {
			pixbuf = rotPixbuf
		}
	}

	surface, err := gdk.CairoSurfaceCreateFromPixbuf(pixbuf, scaleFactor, nil)
	if err != nil {
		return nil, err
	}

	image.SetFromSurface(surface)
	image.SetPixelSize(size)

	return image, nil
}

// CreateIconBalanced loads an app icon and normalizes its opaque artwork to a
// uniform fill ratio inside the canvas. Full-bleed logos (e.g. Chrome, which
// fills the tile edge to edge) and sparse glyphs (terminal app icons with lots
// of padding) otherwise render at visibly different sizes.
func CreateIconBalanced(source string, size int) (*gtk.Image, error) {
	var err error
	var pixbuf *gdk.Pixbuf

	image, err := gtk.ImageNew()
	if err != nil {
		return nil, err
	}

	scaleFactor := image.GetScaleFactor()
	physicalSize := size * scaleFactor

	if strings.Contains(source, "/") {
		pixbuf, err = gdk.PixbufNewFromFileAtSize(source, physicalSize, physicalSize)
	} else {
		theme, _ := gtk.IconThemeGetDefault()
		pixbuf, err = theme.LoadIcon(source, physicalSize, gtk.ICON_LOOKUP_FORCE_SIZE)
	}
	if err != nil {
		return CreateImage0("image-missing", size, false)
	}

	pixbuf, err = normalizeIconContent(pixbuf, physicalSize)
	if err != nil {
		return CreateImage0("image-missing", size, false)
	}

	surface, err := gdk.CairoSurfaceCreateFromPixbuf(pixbuf, scaleFactor, nil)
	if err != nil {
		return nil, err
	}

	image.SetFromSurface(surface)
	image.SetPixelSize(size)

	return image, nil
}

func normalizeIconContent(src *gdk.Pixbuf, canvas int) (*gdk.Pixbuf, error) {
	w, h := src.GetWidth(), src.GetHeight()
	if w <= 0 || h <= 0 {
		return src, nil
	}

	ch := src.GetNChannels()
	hasAlpha := src.GetHasAlpha()
	px := src.GetPixels()
	stride := src.GetRowstride()

	// Opaque bounding box of the artwork.
	minX, minY, maxX, maxY := w, h, -1, -1
	for y := 0; y < h; y++ {
		row := y * stride
		for x := 0; x < w; x++ {
			if hasAlpha && px[row+x*ch+ch-1] == 0 {
				continue
			}
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if maxX < 0 {
		return src, nil // fully transparent artwork: leave untouched
	}

	bw := maxX - minX + 1
	bh := maxY - minY + 1

	// Map the artwork's longest side to a uniform share of the canvas.
	const fill = 0.72
	scale := (float64(canvas) * fill) / float64(max(bw, bh))

	scaled, err := src.ScaleSimple(int(math.Round(float64(w)*scale)), int(math.Round(float64(h)*scale)), gdk.INTERP_BILINEAR)
	if err != nil {
		return src, nil
	}
	sw, sh := scaled.GetWidth(), scaled.GetHeight()

	dest, err := gdk.PixbufNew(gdk.COLORSPACE_RGB, true, 8, canvas, canvas)
	if err != nil {
		return src, nil
	}

	dpx := dest.GetPixels()
	dstride := dest.GetRowstride()
	for i := range dpx {
		dpx[i] = 0
	}

	spx := scaled.GetPixels()
	sstride := scaled.GetRowstride()
	sch := scaled.GetNChannels()

	dx0 := (canvas - sw) / 2
	dy0 := (canvas - sh) / 2

	// Compose centered; crop any overflow (zero padding round the artwork).
	vx0, vy0 := max(0, dx0), max(0, dy0)
	vx1, vy1 := min(canvas, dx0+sw), min(canvas, dy0+sh)

	for dy := vy0; dy < vy1; dy++ {
		sy := dy - dy0
		doffRow := dy*dstride + vx0*4
		soffRow := sy * sstride
		for dx := vx0; dx < vx1; dx++ {
			sx := dx - dx0
			doff := doffRow + (dx-vx0)*4
			soff := soffRow + sx*sch
			dpx[doff] = spx[soff]
			dpx[doff+1] = spx[soff+1]
			dpx[doff+2] = spx[soff+2]
			if sch == 4 {
				dpx[doff+3] = spx[soff+3]
			} else {
				dpx[doff+3] = 255
			}
		}
	}

	return dest, nil
}

func AddStyle(widget gtk.IWidget, style string) (*gtk.CssProvider, error) {
	provider, err := gtk.CssProviderNew()
	if err != nil {
		return nil, err
	}

	err = provider.LoadFromData(style)
	if err != nil {
		return nil, err
	}

	context, err := widget.ToWidget().GetStyleContext()
	if err != nil {
		return nil, err
	}

	context.AddProvider(provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)

	return provider, nil
}

// AddCssProvider loads the dock theme. The provider is registered at USER
// priority (highest) so rules like `window { background-color: transparent }`
// beat the desktop GTK theme (e.g. Adwaita-dark), whose window background
// would otherwise paint an opaque slab behind the dock. GTK3 cannot parse
// !important, so priority is the lever.
// The provider is returned so callers can swap themes live via
// RemoveCssProvider.
func AddCssProvider(cssFile string) (*gtk.CssProvider, error) {
	cssProvider, err := gtk.CssProviderNew()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create CSS provider")
	}

	if err := cssProvider.LoadFromPath(cssFile); err != nil {
		return nil, errors.Wrapf(err, "failed to load CSS from %q", cssFile)
	}

	if err := registerProvider(cssProvider, int(gtk.STYLE_PROVIDER_PRIORITY_USER)); err != nil {
		return nil, err
	}

	return cssProvider, nil
}

// AddCssData registers a generated CSS fragment (e.g. appearance overrides)
// at the given provider priority. Priority 900 sits above the theme file's
// USER priority (800) so overrides win without touching the file itself.
func AddCssData(css string, priority int) (*gtk.CssProvider, error) {
	provider, err := gtk.CssProviderNew()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create CSS provider")
	}

	if err := provider.LoadFromData(css); err != nil {
		return nil, errors.Wrapf(err, "failed to load CSS data: %q", css)
	}

	if err := registerProvider(provider, priority); err != nil {
		return nil, err
	}

	return provider, nil
}

func registerProvider(provider *gtk.CssProvider, priority int) error {
	screen, err := gdk.ScreenGetDefault()
	if err != nil {
		return errors.Wrap(err, "failed to get default screen")
	}

	gtk.AddProviderForScreen(screen, provider, uint(priority))
	return nil
}

// RemoveCssProvider unregisters a screen-level provider previously added by
// AddCssProvider/AddCssData. Used for live theme swaps.
func RemoveCssProvider(provider *gtk.CssProvider) {
	if provider == nil {
		return
	}

	screen, err := gdk.ScreenGetDefault()
	if err != nil {
		return
	}

	gtk.RemoveProviderForScreen(screen, provider)
}

func RemoveStyleProvider(widget *gtk.Box, provider *gtk.CssProvider) error {
	if provider == nil {
		return errors.New("provider is nil")
	}

	styleContext, err := widget.GetStyleContext()
	if err != nil {
		return err
	}

	styleContext.RemoveProvider(provider)
	return nil
}

func GetFirstAvailableImage(sources []string, fallback ...string) string {
	fallbackImg := "image-missing"
	if len(fallback) > 0 {
		fallbackImg = fallback[0]
	}

	theme, err := gtk.IconThemeGetDefault()
	if err != nil {
		return fallbackImg
	}

	for _, source := range sources {
		if strings.Contains(source, "/") && FileExists(source) {
			return source
		}

		if theme.HasIcon(source) {
			return source
		}
	}

	return fallbackImg
}

func SetCursorPointer(v *gtk.Widget) error {
	display, err := gdk.DisplayGetDefault()
	if err != nil {
		return err
	}

	pointer, _ := gdk.CursorNewFromName(display, "pointer")
	arrow, _ := gdk.CursorNewFromName(display, "default")

	v.Connect("enter-notify-event", func() {
		win, _ := v.GetWindow()
		if win != nil {
			win.SetCursor(pointer)
		}
	})

	v.Connect("leave-notify-event", func(_ interface{}, e *gdk.Event) {
		event := gdk.EventCrossingNewFromEvent(e)
		win, _ := v.GetWindow()

		if win != nil && event.Detail() != 2 {
			win.SetCursor(arrow)
		}
	})

	return nil
}

func SetAutoHover(v *gtk.Widget, context *gtk.StyleContext) {
	v.Connect("enter-notify-event", func() {
		context.AddClass("hover")
	})
	v.Connect("leave-notify-event", func(_ interface{}, e *gdk.Event) {
		event := gdk.EventCrossingNewFromEvent(e)
		isInWindow := event.Detail() == 3 || event.Detail() == 0

		if isInWindow {
			context.RemoveClass("hover")
		}
	})
}
