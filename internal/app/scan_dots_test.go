package app

import (
	"bytes"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// testCols is a plausible width in dots: the field is counted from the card,
// so the tests pick one width and check the rules that hold for any.
const testCols = 46

// scanningApp is the state the field exists for: a walk in flight with
// nothing found yet, so the page shows the empty card.
func scanningApp() *App {
	a := New(Settings{})
	a.scanning = true
	a.visited = 1204
	a.foundSoFar = 3
	a.refresh()
	return a
}

// TestScanningFieldAnimates checks the field is alive: it asks for another
// frame, and the next frame paints something different.
func TestScanningFieldAnimates(t *testing.T) {
	a := scanningApp()
	tt := frame(a, 1180, 800)

	if !tt.HasText("Scanning…") {
		t.Fatalf("the scanning state does not say so; texts: %v", tt.Texts())
	}
	if !tt.FramePending() {
		t.Fatal("the field does not ask for another frame, so it would stand still")
	}

	first := append([]byte(nil), tt.Image().Pix...)
	tt.Advance(180 * time.Millisecond)
	tt.Frame()
	second := tt.Image().Pix

	changed := 0
	for i := 0; i+4 <= len(first); i += 4 {
		if !bytes.Equal(first[i:i+4], second[i:i+4]) {
			changed++
		}
	}
	if changed < 40 {
		t.Errorf("only %d pixels changed 180ms later, so the field is not moving", changed)
	}
}

// TestEmptyStateWithoutAScanDoesNotAnimate keeps the animation where it
// belongs: a page with nothing to show must not ask for frames.
func TestEmptyStateWithoutAScanDoesNotAnimate(t *testing.T) {
	a := New(Settings{})
	a.refresh()
	tt := frame(a, 1180, 800)
	if tt.FramePending() {
		t.Error("the empty state asks for frames it does not need")
	}
}

// TestDotFieldIsDeterministicAndBounded checks the two properties the drawing
// depends on: the same moment always paints the same field, and a level is
// always a usable alpha.
func TestDotFieldIsDeterministicAndBounded(t *testing.T) {
	now := time.Unix(1700000000, 0)
	blinking, quiet := 0, 0

	for col := 0; col < testCols; col++ {
		for row := 0; row < dotRows; row++ {
			level := dotLevel(col, row, now)
			if level != dotLevel(col, row, now) {
				t.Fatalf("dot %d,%d changed between two reads of the same moment", col, row)
			}
			if level < 0 || level > 1 {
				t.Fatalf("dot %d,%d is %v, outside 0..1", col, row, level)
			}
			if _, _, isQuiet := dotBlink(col, row); isQuiet {
				quiet++
			} else {
				blinking++
			}
		}
	}

	if blinking == 0 || quiet == 0 {
		t.Errorf("the field is not sparse: %d blinking, %d quiet", blinking, quiet)
	}
}

// TestDotFieldMovesOverTime checks the field is not a still picture: a moment
// later, a good part of it has changed.
func TestDotFieldMovesOverTime(t *testing.T) {
	now := time.Unix(1700000000, 0)
	later := now.Add(120 * time.Millisecond)

	moved := 0
	for col := 0; col < testCols; col++ {
		for row := 0; row < dotRows; row++ {
			if dotLevel(col, row, now) != dotLevel(col, row, later) {
				moved++
			}
		}
	}
	if moved < 10 {
		t.Errorf("only %d dots changed in 120ms, so the field barely moves", moved)
	}
}

// TestDotFieldUsesThreeColoursPerAppearance checks the field is the three
// bands it is meant to be, in both appearances.
func TestDotFieldUsesThreeColoursPerAppearance(t *testing.T) {
	for _, dark := range []bool{false, true} {
		seen := map[ui.Color]bool{}
		for col := 0; col < testCols; col++ {
			seen[dotColor(col, testCols, dark)] = true
		}
		if len(seen) != 3 {
			t.Errorf("dark=%v: the field has %d colours, want 3", dark, len(seen))
		}
	}
}
