package app

import "github.com/egoist/mygo/ui"

// Sweep's palette. It keeps Onevra's structure — warm neutral surfaces, one
// accent, muted semantic colours — and takes its accent from Apple's system
// blue, the family the app's mark already leans on, so the window reads as a
// tool rather than as a generic desktop app.
//
// Hue is reserved for meaning: the accent marks the one primary action and the
// current page, and the only other saturated colours are a treemap tile and a
// status badge. Surfaces stay near-greyscale so those two always read.
var (
	// Light: warm paper, white raised surfaces.
	lightCanvas       = ui.Hex("#F7F7F4")
	lightSurface      = ui.Hex("#FFFFFF")
	lightSurfaceHover = ui.Hex("#EFEFEA")
	lightInset        = ui.Hex("#E7E7E0")
	lightBorder       = ui.Hex("#E2E2D9")
	lightText         = ui.Hex("#1E211C")
	// The muted ink is dark enough to clear 4.5:1 on the hover surface too,
	// which is the lightest surface it is ever drawn on. A lighter grey reads
	// fine on white and fails as soon as a row is hovered.
	lightTextMuted = ui.Hex("#63655D")

	// Dark: warm charcoal, lifted surfaces.
	darkCanvas       = ui.Hex("#171916")
	darkSurface      = ui.Hex("#212420")
	darkSurfaceHover = ui.Hex("#292D28")
	darkInset        = ui.Hex("#31362F")
	darkBorder       = ui.Hex("#3A4038")
	darkText         = ui.Hex("#ECEDE8")
	darkTextMuted    = ui.Hex("#A8ACA3")

	// The accent is Apple's system blue. The light appearance uses one step
	// deeper than the system's #007AFF: that value carries white text at
	// 4.0:1, under the 4.5:1 the label on a filled button needs. The dark
	// appearance is deeper again than the system's #0A84FF for the same
	// reason, because a filled button reads better with white on it than with
	// dark ink on a bright blue.
	accentLight = ui.Hex("#0066DB")
	accentDark  = ui.Hex("#0A6ACE")

	warnLight   = ui.Hex("#A85B0A")
	warnDark    = ui.Hex("#E6B567")
	dangerLight = ui.Hex("#B4444C")
	dangerDark  = ui.Hex("#E58F95")
	goodLight   = ui.Hex("#1C7A55")
	goodDark    = ui.Hex("#79BFA1")
)

// theme builds the theme for an appearance. It starts from MyGo's default,
// which already follows the desktop's accent, contrast and text-size settings,
// and overrides only what this app has an opinion about.
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
		// A lift of 8% rather than 12%: white on the lifted blue still has to
		// clear 4.5:1, and the hover is the lightest the accent ever gets.
		t.AccentHover = accentDark.Mix(ui.RGB(255, 255, 255), 0.08)
		t.AccentPressed = accentDark.Mix(ui.RGB(0, 0, 0), 0.14)
		t.Success = goodDark
		t.Warning = warnDark
		t.Danger = dangerDark
		t.Scrollbar = darkBorder
		// The selection wash is dim: it lifts the surface toward the text, so
		// every step of alpha costs contrast for the row's own label. The
		// selected row therefore draws its secondary text in full ink.
		t.Selection = accentDark.Alpha(0.16)
		t.Focus = accentDark
		// White on the accent, as the light appearance does. The blue is one
		// step deeper than the system's so that the label clears 4.5:1.
		t.AccentText = ui.Hex("#FFFFFF")
	} else {
		t.Background = lightCanvas
		t.Surface = lightSurface
		t.SurfaceHover = lightSurfaceHover
		t.SurfacePressed = lightInset
		t.Border = lightBorder
		t.Text = lightText
		t.TextMuted = lightTextMuted
		t.Accent = accentLight
		t.AccentHover = accentLight.Mix(ui.RGB(255, 255, 255), 0.10)
		t.AccentPressed = accentLight.Mix(ui.RGB(0, 0, 0), 0.18)
		t.Success = goodLight
		t.Warning = warnLight
		t.Danger = dangerLight
		t.Scrollbar = lightBorder
		t.Selection = accentLight.Alpha(0.20)
		t.Focus = accentLight
		t.AccentText = ui.Hex("#FFFFFF")
	}
	t.Radius = 8
	return t
}

// radiusCard is the corner of the large panels: a card is a different object
// from a control, and the reference design rounds it further than one.
const radiusCard = 12
