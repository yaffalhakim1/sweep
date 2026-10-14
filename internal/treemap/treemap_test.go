package treemap

import (
	"math"
	"testing"
)

// TestLayoutAreaIsProportional is the treemap's whole contract: a tile's
// area is its share of the total.
func TestLayoutAreaIsProportional(t *testing.T) {
	items := []Item{
		{Key: "a", Value: 50},
		{Key: "b", Value: 25},
		{Key: "c", Value: 15},
		{Key: "d", Value: 10},
	}
	tiles := Layout(items, 0, 0, 100, 100)
	if len(tiles) != 4 {
		t.Fatalf("got %d tiles, want 4", len(tiles))
	}
	var total float64
	for _, it := range items {
		total += float64(it.Value)
	}
	for _, tile := range tiles {
		want := float64(tile.Value) / total * 10000
		got := tile.W * tile.H
		if math.Abs(got-want) > want*0.02 {
			t.Errorf("tile %s area %.1f, want about %.1f", tile.Key, got, want)
		}
	}
}

// TestLayoutStaysInBox checks that no tile escapes the box, which the
// view relies on for its hit testing.
func TestLayoutStaysInBox(t *testing.T) {
	items := []Item{
		{Key: "a", Value: 900},
		{Key: "b", Value: 70},
		{Key: "c", Value: 20},
		{Key: "d", Value: 9},
		{Key: "e", Value: 1},
	}
	const x, y, w, h = 12.5, 30, 640, 240
	for _, tile := range Layout(items, x, y, w, h) {
		if tile.X < x-1e-6 || tile.Y < y-1e-6 ||
			tile.X+tile.W > x+w+1e-6 || tile.Y+tile.H > y+h+1e-6 {
			t.Errorf("tile %s at (%.2f,%.2f,%.2f,%.2f) escapes the box", tile.Key, tile.X, tile.Y, tile.W, tile.H)
		}
	}
}

// TestLayoutCoversTheBox checks that the tiles together fill the box, so
// there is no unexplained empty area in the map.
func TestLayoutCoversTheBox(t *testing.T) {
	items := []Item{
		{Key: "a", Value: 7},
		{Key: "b", Value: 5},
		{Key: "c", Value: 3},
		{Key: "d", Value: 2},
		{Key: "e", Value: 1},
	}
	var sum float64
	for _, tile := range Layout(items, 0, 0, 300, 200) {
		sum += tile.W * tile.H
	}
	if math.Abs(sum-60000) > 60 {
		t.Errorf("tiles cover %.1f, want 60000", sum)
	}
}

// TestLayoutKeepsRectanglesSquareish checks the point of squarifying: no
// tile is a sliver. A linear layout of the same data would produce
// extreme ratios here.
func TestLayoutKeepsRectanglesSquareish(t *testing.T) {
	items := []Item{
		{Key: "a", Value: 100},
		{Key: "b", Value: 100},
		{Key: "c", Value: 100},
		{Key: "d", Value: 100},
	}
	for _, tile := range Layout(items, 0, 0, 200, 200) {
		ratio := math.Max(tile.W, tile.H) / math.Min(tile.W, tile.H)
		if ratio > 2.5 {
			t.Errorf("tile %s aspect %.2f, want near 1 (squarified)", tile.Key, ratio)
		}
	}
}

// TestLayoutDropsEmptyItems checks that a zero-byte artifact does not
// produce a zero-area rectangle, which would be an unclickable sliver.
func TestLayoutDropsEmptyItems(t *testing.T) {
	tiles := Layout([]Item{
		{Key: "a", Value: 10},
		{Key: "zero", Value: 0},
		{Key: "neg", Value: -5},
	}, 0, 0, 100, 100)
	if len(tiles) != 1 {
		t.Fatalf("got %d tiles, want 1", len(tiles))
	}
}

// TestLayoutDegenerateBox checks the guards against a box with no area,
// which happens before the first layout pass and while a window is tiny.
func TestLayoutDegenerateBox(t *testing.T) {
	items := []Item{{Key: "a", Value: 10}}
	if got := Layout(items, 0, 0, 0, 100); got != nil {
		t.Errorf("got %v for a zero-width box, want nil", got)
	}
	if got := Layout(items, 0, 0, 100, -1); got != nil {
		t.Errorf("got %v for a negative-height box, want nil", got)
	}
	if got := Layout(nil, 0, 0, 100, 100); got != nil {
		t.Errorf("got %v for no items, want nil", got)
	}
}

// TestLayoutCarriesKind checks that the view's colour key survives the
// layout, since it is what the map encodes.
func TestLayoutCarriesKind(t *testing.T) {
	tiles := Layout([]Item{{Key: "p", Value: 1, Kind: "Rust"}}, 0, 0, 10, 10)
	if len(tiles) != 1 || tiles[0].Kind != "Rust" {
		t.Errorf("got %+v, want the kind carried through", tiles)
	}
}
