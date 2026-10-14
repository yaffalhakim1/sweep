package app

import "github.com/egoist/mygo/ui"

// The app's colours. The surfaces and the accent are Apple's own system
// values, so the window sits beside the system's apps without a seam;
// everything else is derived from them. Hue is reserved for meaning: the
// only saturated colour in the chrome is the accent, and the only other
// colour anywhere is a treemap tile.
var (
	// Light: the system's grouped background and its raised white.
	lightCanvas       = ui.Hex("#F5F5F7")
	lightSurface      = ui.Hex("#FFFFFF")
	lightSurfaceHover = ui.Hex("#ECECEF")
	lightInset        = ui.Hex("#E8E8EB")
	lightBorder       = ui.Hex("#D8D8DC")
	lightText         = ui.Hex("#1D1D1F")
	lightTextMuted    = ui.Hex("#6E6E73")

	// Dark: the system's elevated surfaces.
	darkCanvas       = ui.Hex("#1C1C1E")
	darkSurface      = ui.Hex("#2C2C2E")
	darkSurfaceHover = ui.Hex("#3A3A3C")
	darkInset        = ui.Hex("#232325")
	darkBorder       = ui.Hex("#3A3A3C")
	darkText         = ui.Hex("#F5F5F7")
	darkTextMuted    = ui.Hex("#98989D")

	// The system accent blue. It colours the primary action and the focus
	// ring, and nothing else, so the one strong colour on screen always
	// means "this is the action".
	accentLight = ui.Hex("#007AFF")
	accentDark  = ui.Hex("#0A84FF")

	// The disk warning colours, which the capacity meter uses. They are
	// the only other saturated colours in the chrome, and they appear only
	// when the disk is genuinely getting full.
	warnLight   = ui.Hex("#B26A00")
	warnDark    = ui.Hex("#E0A030")
	dangerLight = ui.Hex("#C43A31")
	dangerDark  = ui.Hex("#E2726A")
	goodLight   = ui.Hex("#2F8F52")
	goodDark    = ui.Hex("#5FA463")
)

// theme builds the theme for an appearance. It starts from MyGo's default,
// which already follows the desktop's accent, contrast and text-size
// settings, and overrides only what this app has an opinion about.
func theme(dark bool) *ui.Theme {
	t := &ui.Theme{Dark: dark}
	if dark {
		t.Background = darkCanvas
		t.Surface = darkSurface
		t.SurfaceHover = darkSurfaceHover
		t.SurfacePressed = darkInset
		t.Border = darkBorder
		t.Text = darkText
		t.TextMuted = darkTextMuted
		t.Accent = accentDark
		t.AccentHover = accentDark
		t.AccentPressed = accentDark
		t.Success = goodDark
		t.Warning = warnDark
		t.Danger = dangerDark
		t.Scrollbar = darkBorder
		t.Selection = accentDark.Alpha(0.30)
		t.Focus = accentDark
	} else {
		t.Background = lightCanvas
		t.Surface = lightSurface
		t.SurfaceHover = lightSurfaceHover
		t.SurfacePressed = lightInset
		t.Border = lightBorder
		t.Text = lightText
		t.TextMuted = lightTextMuted
		t.Accent = accentLight
		t.AccentHover = accentLight
		t.AccentPressed = accentLight
		t.Success = goodLight
		t.Warning = warnLight
		t.Danger = dangerLight
		t.Scrollbar = lightBorder
		t.Selection = accentLight.Alpha(0.25)
		t.Focus = accentLight
	}
	// Text on the accent: both accents are bright enough that dark ink is
	// the readable choice in either appearance.
	t.AccentText = ui.Hex("#FFFFFF")
	t.Radius = 10
	return t
}
