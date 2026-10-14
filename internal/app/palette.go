package app

import (
	"math"

	"github.com/egoist/mygo/ui"

	"sweep/internal/scan"
)

// kindColors is the map's categorical palette: one hue per build system,
// so a treemap reads as "where the space went" before any label is read.
//
// The hues are mid-tone on purpose. A tile is filled solid and labelled on
// top, so the label colour has to be chosen per fill rather than fixed; a
// mid-tone keeps that choice from being marginal for any of them, and the
// contrast test asserts the result for every pair in both appearances.
var kindColors = map[bool]map[scan.Kind]ui.Color{
	false: { // light
		scan.Rust:   ui.Hex("#B4551F"),
		scan.Node:   ui.Hex("#3D7D40"),
		scan.Python: ui.Hex("#3A6A9C"),
		scan.JVM:    ui.Hex("#744D91"),
		scan.Xcode:  ui.Hex("#3B768C"),
		scan.CMake:  ui.Hex("#4C5661"),
		scan.DotNet: ui.Hex("#8E4C72"),
		scan.Other:  ui.Hex("#6E675C"),
	},
	true: { // dark
		scan.Rust:   ui.Hex("#D2762F"),
		scan.Node:   ui.Hex("#5FA463"),
		scan.Python: ui.Hex("#5C90C7"),
		scan.JVM:    ui.Hex("#9B72B8"),
		scan.Xcode:  ui.Hex("#5AA0B8"),
		scan.CMake:  ui.Hex("#7C8899"),
		scan.DotNet: ui.Hex("#B8789E"),
		scan.Other:  ui.Hex("#9C9484"),
	},
}

// KindColor is the fill for a kind in one appearance.
func KindColor(dark bool, k scan.Kind) ui.Color {
	if c, ok := kindColors[dark][k]; ok {
		return c
	}
	return kindColors[dark][scan.Other]
}

// onColor returns readable text for a solid fill: the near-black or the
// near-white ink, whichever contrasts more. Choosing per fill rather than
// per palette entry means a new hue cannot ship an unreadable label.
func onColor(fill ui.Color) ui.Color {
	ink := ui.Hex("#1A1A1A")
	paper := ui.Hex("#FFFFFF")
	if contrast(fill, paper) > contrast(fill, ink) {
		return paper
	}
	return ink
}

// contrast is the WCAG contrast ratio between two opaque colours, from 1
// (identical) to 21 (black on white).
func contrast(a, b ui.Color) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// luminance is the WCAG relative luminance of a colour: 0 for black, 1 for
// white, with the sRGB transfer function applied first.
func luminance(c ui.Color) float64 {
	channel := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(c.R) + 0.7152*channel(c.G) + 0.0722*channel(c.B)
}
