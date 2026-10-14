package treemap

import "sort"

// Item is one rectangle's worth of data: what it is, and how much space it
// claims.
type Item struct {
	Key   string
	Value int64
	// Kind is carried through untouched so the view can colour by it
	// without a second lookup.
	Kind string
}

// Tile is a placed item: the data and the rectangle it occupies, in the
// same units as the box passed to Layout.
type Tile struct {
	Item
	X, Y, W, H float64
}

// Layout arranges items into box (x, y, w, h) so that each rectangle's
// area is proportional to its value, using the squarified algorithm of
// Bruls, Huizing and van Wijk. Squarified keeps rectangles close to
// square, which is what makes a treemap readable: a row of thin slivers
// is technically proportional and practically useless.
//
// Items with a non-positive value are dropped, because they cannot be
// given an area. Every tile lies within the box.
func Layout(items []Item, x, y, w, h float64) []Tile {
	kept := make([]Item, 0, len(items))
	for _, it := range items {
		if it.Value > 0 {
			kept = append(kept, it)
		}
	}
	if len(kept) == 0 || w <= 0 || h <= 0 {
		return nil
	}
	// Largest first: squarified grows a row until adding the next item
	// would make the row's worst aspect ratio worse, which only produces
	// good layouts when the input descends.
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].Value > kept[j].Value })

	var total float64
	for _, it := range kept {
		total += float64(it.Value)
	}
	area := w * h

	tiles := make([]Tile, 0, len(kept))
	var row []Item
	var rowSum float64

	for len(kept) > 0 {
		short := min(w, h)
		// Grow the row while doing so improves its worst aspect ratio.
		for len(kept) > 0 {
			if len(row) > 0 {
				with := append(append([]Item{}, row...), kept[0])
				if worst(row, rowSum, short, total, area) < worst(with, rowSum+float64(kept[0].Value), short, total, area) {
					break
				}
			}
			row = append(row, kept[0])
			rowSum += float64(kept[0].Value)
			kept = kept[1:]
		}

		placed, usedW, usedH := placeRow(row, rowSum, x, y, w, h, total, area)
		tiles = append(tiles, placed...)
		x, y = x+usedW, y+usedH
		w, h = w-usedW, h-usedH
		row, rowSum = row[:0], 0
		if w <= 1e-9 || h <= 1e-9 {
			break
		}
	}
	return tiles
}

// worst is the worst aspect ratio a row would have along a side of the
// given length, which is the quantity squarified minimises.
func worst(row []Item, sum, short, total, area float64) float64 {
	if len(row) == 0 || sum <= 0 || short <= 0 {
		return 0
	}
	thickness := sum / total * area / short
	if thickness <= 0 {
		return 0
	}
	var largest, smallest float64
	for i, it := range row {
		v := float64(it.Value)
		if i == 0 || v > largest {
			largest = v
		}
		if i == 0 || v < smallest {
			smallest = v
		}
	}
	longest := largest / total * area / thickness
	shortest := smallest / total * area / thickness
	if shortest <= 0 {
		return 0
	}
	return max(thickness/shortest, longest/thickness)
}

// placeRow lays one row along the shorter side of the box and reports the
// strip it consumed, so the caller can shrink the box by exactly that
// much.
func placeRow(row []Item, sum, x, y, w, h, total, area float64) (tiles []Tile, usedW, usedH float64) {
	if len(row) == 0 || sum <= 0 {
		return nil, 0, 0
	}
	tiles = make([]Tile, 0, len(row))
	if w >= h {
		// A column down the left edge: the strip is as wide as the row's
		// share of the box, and each item takes a band of it.
		thickness := sum / total * area / h
		if thickness > w {
			thickness = w
		}
		cy := y
		for i, it := range row {
			share := float64(it.Value) / sum
			height := share * h
			if i == len(row)-1 {
				height = y + h - cy // the last closes the strip exactly
			}
			tiles = append(tiles, Tile{Item: it, X: x, Y: cy, W: thickness, H: height})
			cy += height
		}
		return tiles, thickness, 0
	}
	// A band across the top: the strip is as tall as the row's share, and
	// each item takes a column of it.
	thickness := sum / total * area / w
	if thickness > h {
		thickness = h
	}
	cx := x
	for i, it := range row {
		share := float64(it.Value) / sum
		width := share * w
		if i == len(row)-1 {
			width = x + w - cx
		}
		tiles = append(tiles, Tile{Item: it, X: cx, Y: y, W: width, H: thickness})
		cx += width
	}
	return tiles, 0, thickness
}
