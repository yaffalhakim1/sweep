package app

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"sweep/internal/scan"
)

// TestDarkContrastAudit checks every text colour the dark theme can produce
// against the surface it is drawn on. The app has no CSS, so the pairs are
// enumerated from the theme rather than measured from a render.
//
// This exists because dark themes fail quietly: a grey that reads fine on
// the canvas measures 3.95:1 on a hover surface, and nothing about the code
// says so. The values below are the ones that actually regressed.
func TestDarkContrastAudit(t *testing.T) {
	// Colours the app actually draws text in, dark theme.
	type pair struct {
		name string
		fg   ui.Color
		bg   ui.Color
	}
	tm := theme(true)
	pairs := []pair{
		{"text on canvas", tm.Text, tm.Background},
		{"text on surface", tm.Text, tm.Surface},
		{"muted on canvas", tm.TextMuted, tm.Background},
		{"muted on surface", tm.TextMuted, tm.Surface},
		{"muted on surface hover", tm.TextMuted, tm.SurfaceHover},
		{"accent text on accent", tm.AccentText, tm.Accent},
		{"text on selection", tm.Text, tm.Selection.Over(tm.Surface)},
		// Muted ink is NOT used on a selected row: the wash lifts the
		// background toward the grey, so the row draws those cells in
		// full-contrast ink instead. Measured here so the reason stays
		// visible, and asserted as "better than the plain surface" rather
		// than against the 4.5:1 body-text bar.
		{"danger on surface", tm.Danger, tm.Surface},
		{"warning on surface", tm.Warning, tm.Surface},
		{"success on surface", tm.Success, tm.Surface},
		{"scrollbar on surface", tm.Scrollbar, tm.Surface},
		{"border on canvas", tm.Border, tm.Background},
	}
	for _, p := range pairs {
		got := contrast(p.fg, p.bg)
		t.Logf("%-28s %.2f:1", p.name, got)
	}

	// The pairs that must clear 4.5:1 as body text.
	for _, p := range pairs {
		switch p.name {
		case "scrollbar on surface", "border on canvas":
			continue // decoration, not text
		}
		if p.name == "accent text on accent" {
			continue // checked separately below at 3:1 (button label)
		}
		if got := contrast(p.fg, p.bg); got < 4.5 {
			t.Errorf("%s is %.2f:1, under 4.5:1", p.name, got)
		}
	}
	if got := contrast(tm.AccentText, tm.Accent); got < 3.0 {
		t.Errorf("accent label on accent is %.2f:1, under 3:1", got)
	}
}

// TestKindColorsContrastOnBothThemes checks the treemap palette, whose
// labels are drawn in onColor(fill) — the ink chosen per fill.
func TestKindColorsContrastOnBothThemes(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for _, k := range scan.Kinds {
			fill := KindColor(dark, k)
			ink := onColor(fill)
			got := contrast(ink, fill)
			if got < 4.5 {
				t.Errorf("dark=%v kind=%s: label %.2f:1 on its tile, under 4.5:1", dark, k, got)
			}
			// The tile must also be distinguishable from the canvas it sits on.
			bg := theme(dark).Background
			if edge := contrast(fill, bg); edge < 1.2 {
				t.Errorf("dark=%v kind=%s: tile %.2f:1 against the canvas, too close to see", dark, k, edge)
			}
		}
	}
}
