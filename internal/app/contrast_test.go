package app

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"sweep/internal/scan"
)

// TestContrastAudit checks every text colour the theme can produce against the
// surface it is drawn on. The app has no CSS, so the pairs are enumerated from
// the theme rather than measured from a render.
//
// This exists because both appearances fail quietly: a grey that reads fine on
// the canvas measures under 4.5:1 on a hover surface, and nothing about the code
// says so. Both appearances are checked, because the light one regressed as
// soon as the accent became a dark green.
func TestContrastAudit(t *testing.T) {
	for _, dark := range []bool{false, true} {
		tm := theme(dark)
		type pair struct {
			name     string
			fg, bg   ui.Color
			minRatio float64
		}
		pairs := []pair{
			{"text on canvas", tm.Text, tm.Background, 4.5},
			{"text on surface", tm.Text, tm.Surface, 4.5},
			{"muted on canvas", tm.TextMuted, tm.Background, 4.5},
			{"muted on surface", tm.TextMuted, tm.Surface, 4.5},
			{"muted on surface hover", tm.TextMuted, tm.SurfaceHover, 4.5},
			// The accent face carries the chosen page and the main action, so
			// its label is body text, not decoration.
			{"accent text on accent", tm.AccentText, tm.Accent, 4.5},
			// The label goes on the lifted face too, and the hover is the
			// lightest the accent ever gets: a lift that clears the bar at rest
			// can still fail under the pointer.
			{"accent text on accent hover", tm.AccentText, tm.AccentHover, 4.5},
			{"accent text on accent pressed", tm.AccentText, tm.AccentPressed, 4.5},
			// A selected row's wash lifts the surface toward the accent. The
			// row draws its secondary text in full ink for this reason; the
			// primary ink must clear the bar on the wash itself.
			{"text on selection", tm.Text, tm.Selection.Over(tm.Surface), 4.5},
			{"danger on surface", tm.Danger, tm.Surface, 4.5},
			{"warning on surface", tm.Warning, tm.Surface, 4.5},
			{"success on surface", tm.Success, tm.Surface, 4.5},
			// Decoration, not text.
			{"scrollbar on surface", tm.Scrollbar, tm.Surface, 0},
			{"border on canvas", tm.Border, tm.Background, 0},
		}
		for _, p := range pairs {
			got := contrast(p.fg, p.bg)
			t.Logf("dark=%-5v %-28s %.2f:1", dark, p.name, got)
			if p.minRatio > 0 && got < p.minRatio {
				t.Errorf("dark=%v: %s is %.2f:1, under %.1f:1", dark, p.name, got, p.minRatio)
			}
		}
	}
}

// TestKindColorsContrastOnBothThemes checks the treemap palette, whose labels
// are drawn in onColor(fill) — the ink chosen per fill.
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

// TestMutedInkIsNotUsedOnASelectedRow is the reason rowMuted exists: the wash
// lifts the background far enough that muted ink stops clearing 4.5:1, so the
// row must switch to full ink. Asserting the failure keeps the fix honest — if
// the wash is ever dimmed enough that muted ink would pass, this test says so
// and rowMuted can be simplified.
func TestMutedInkIsNotUsedOnASelectedRow(t *testing.T) {
	for _, dark := range []bool{false, true} {
		tm := theme(dark)
		wash := tm.Selection.Over(tm.Surface)
		if got := contrast(tm.TextMuted, wash); got >= 4.5 {
			t.Logf("dark=%v: muted ink on the wash is %.2f:1, so it would pass", dark, got)
		}
		if got := contrast(tm.Text, wash); got < 4.5 {
			t.Errorf("dark=%v: full ink on the wash is %.2f:1, under 4.5:1", dark, got)
		}
	}
}
