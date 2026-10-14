package app

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"sweep/internal/scan"
	"sweep/internal/treemap"
)

// mapTileLimit is how many artifacts the map draws. A treemap of every
// artifact on a large disk is a texture rather than a map, and the tiles
// past the largest few hundred are unreadable and unclickable; the list
// shows all of them.
const mapTileLimit = 300

// mapHeight is the map's height. A treemap needs area to be legible, and a
// fixed height keeps the layout stable across scans: the tiles move, the
// card does not.
const mapHeight = 380

// tileGap is the space left around each tile, which is what separates
// neighbours without a border between them.
const tileGap = 1.5

// mapCache holds a laid-out treemap and the inputs it was built from, so
// the layout runs when the data or the box changes rather than on every
// frame the pointer moves.
type mapCache struct {
	tiles []treemap.Tile
	items []treemap.Item
	rect  ui.Rect
	valid bool
}

// layout returns the map's tiles for the box, relaying them out only when
// the artifacts, the box, or the canvas have changed.
//
// This is what keeps the map responsive: the layout is O(n log n) with a
// squarify pass over up to 300 tiles, and a pointer that merely moves
// produces frames where nothing about it changed. Both the draw callback
// and the hit test read this one result, so the tile drawn under the
// pointer is always the tile the hit test reports.
func (a *App) layout(items []treemap.Item, r ui.Rect) []treemap.Tile {
	if a.mapCache.valid && a.mapCache.rect == r && sameItems(a.mapCache.items, items) {
		return a.mapCache.tiles
	}
	tiles := treemap.Layout(items, float64(r.X), float64(r.Y), float64(r.W), float64(r.H))
	a.mapCache = mapCache{tiles: tiles, items: items, rect: r, valid: true}
	return tiles
}

// sameItems reports whether two item lists hold the same values in the
// same order, which is what the layout depends on.
func sameItems(a, b []treemap.Item) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// tileAt returns the tile under a point in painted coordinates. The tiles
// do not overlap, so the first hit is the only hit.
func tileAt(tiles []treemap.Tile, x, y float32) (string, bool) {
	for _, tile := range tiles {
		if float64(x) >= tile.X && float64(x) < tile.X+tile.W &&
			float64(y) >= tile.Y && float64(y) < tile.Y+tile.H {
			return tile.Key, true
		}
	}
	return "", false
}

// mapView draws the treemap: every tile's area is its share of the
// reclaimable total, and its colour is the build system it belongs to.
// This is the "where did it all go" answer, which a list of paths cannot
// give.
func (a *App) mapView(c *ui.Context) {
	t := c.Theme()
	arts := a.largest(mapTileLimit)
	items := mapItems(arts)
	byPath := make(map[string]scan.Artifact, len(arts))
	for _, art := range arts {
		byPath[art.Path] = art
	}

	card(c, func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, "Space map").FontSize(13).FontWeight(600)
			ui.Text(c, "·").FontSize(13).TextColor(t.TextMuted)
			ui.Text(c, "area is size, colour is build system").FontSize(12).TextColor(t.TextMuted)
			ui.Spacer(c)
			if len(a.found) > len(arts) {
				ui.Textf(c, "showing the largest %d of %d", len(arts), len(a.found)).
					FontSize(12).TextColor(t.TextMuted)
			}
		})

		// FillWidth, not Fill: Fill would also stretch the height to the
		// parent's content box, which here is the card's, collapsing the
		// explicit height the map is drawn at.
		box := ui.Box(c).Height(mapHeight).FillWidth().Radius(8).Clip().
			Background(t.Background).Bind(&a.mapBox)
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			tiles := a.layout(items, r)
			// PointerPosition is relative to this element's box, so it is
			// turned into painted coordinates before it is compared with
			// the tiles, which the layout places in those coordinates.
			px, py, over := box.PointerPosition()
			hover := ""
			if over {
				if key, ok := tileAt(tiles, r.X+px, r.Y+py); ok {
					hover = key
				}
			}
			for _, tile := range tiles {
				art := byPath[tile.Key]
				fill := KindColor(t.Dark, art.Kind)
				rect := ui.Rect{
					X: float32(tile.X) + tileGap,
					Y: float32(tile.Y) + tileGap,
					W: float32(tile.W) - 2*tileGap,
					H: float32(tile.H) - 2*tileGap,
				}
				if rect.W <= 0 || rect.H <= 0 {
					continue
				}
				p.Fill(rect, fill, 4)
				switch {
				case a.selected[art.Path]:
					// A chosen tile keeps its colour and takes a ring, so
					// the selection reads as a state of the same tile
					// rather than as a different kind.
					p.Stroke(rect, t.Text, 4, 2.5)
				case tile.Key == hover:
					p.Stroke(rect, onColor(fill), 4, 2)
				}
				// A label only where it fits and can be read: text in a
				// 12-pixel tile is noise, not information.
				if rect.W > 52 && rect.H > 26 {
					label := artifactLabel(art)
					if w, h := p.MeasureText(0, ui.Span{Text: label, Weight: 600, Size: 12}); w < rect.W-10 && h < rect.H-8 {
						p.RichText(rect.X+6, rect.Y+5, 0, ui.Span{
							Text: label, Weight: 600, Size: 12, Color: onColor(fill),
						})
						if rect.H > 44 {
							p.RichText(rect.X+6, rect.Y+21, 0, ui.Span{
								Text: scan.HumanBytes(art.Bytes), Size: 11,
								Color: onColor(fill).Alpha(0.85),
							})
						}
					}
				}
			}
		})
		// The tooltip and the click are resolved from the same cached
		// layout the draw callback used, so what the tooltip names is what
		// is under the pointer.
		px, py, over := box.PointerPosition()
		if path, ok := a.hoveredTileAt(items, a.mapBox.Bounds(c), px, py, over); ok {
			box.Tooltip(tileTooltip(byPath[path]))
			if box.Clicked() {
				a.Toggle(path)
			}
		}
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			a.legend(c, a.kindsPresent())
			ui.Spacer(c)
			ui.Text(c, "Click a tile to choose it").
				FontSize(12).TextColor(t.TextMuted).Shrink(0)
		})
	})
}

// hoveredTileAt returns the artifact under a pointer, given the map's
// committed box and the pointer's position in the element's own
// coordinates, which is what PointerPosition reports.
//
// The two spaces differ by the box's own origin, and getting that wrong
// shifts the hit test by exactly that much: the bug this signature exists
// to make testable named the tile above the one under the pointer.
func (a *App) hoveredTileAt(items []treemap.Item, r ui.Rect, x, y float32, over bool) (string, bool) {
	if !over || r.W <= 0 || r.H <= 0 {
		return "", false
	}
	return tileAt(a.layout(items, r), r.X+x, r.Y+y)
}

// mapItems turns artifacts into the map's items, which is what both the
// layout and the cache key are built from.
func mapItems(arts []scan.Artifact) []treemap.Item {
	items := make([]treemap.Item, 0, len(arts))
	for _, art := range arts {
		items = append(items, treemap.Item{
			Key:   art.Path,
			Value: art.Bytes,
			Kind:  string(art.Kind),
		})
	}
	return items
}

// tileTooltip describes one artifact in a sentence, so a hover answers
// "what is this and is it safe" without leaving the map.
func tileTooltip(a scan.Artifact) string {
	parts := []string{a.Path, scan.HumanBytes(a.Bytes)}
	if a.Files > 0 {
		parts = append(parts, plural(a.Files, "file", "files"))
	}
	if a.Rule.Note != "" {
		parts = append(parts, a.Rule.Note)
	}
	if when := timeLabel(a.Touched); when != "" {
		parts = append(parts, "Last changed "+when)
	}
	return strings.Join(parts, "\n")
}

// plural writes a count with its noun.
func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return itoa(n) + " " + many
}

// legend is the map's key: one chip per kind present, so the colours are
// named rather than guessed.
func (a *App) legend(c *ui.Context, kinds []scan.Kind) {
	t := c.Theme()
	ui.Row(c).Gap(12).Wrap().Children(func() {
		for _, k := range kinds {
			ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(9, 9).Radius(2).Background(KindColor(t.Dark, k))
				ui.Text(c, string(k)).FontSize(12).TextColor(t.TextMuted)
			})
		}
	})
}

// kindsPresent lists the kinds the map is showing, in the app's kind
// order, which is the order the list groups them in: the same build system
// then sits in the same relative place in both views, and the legend does
// not shuffle between scans.
func (a *App) kindsPresent() []scan.Kind {
	seen := map[scan.Kind]bool{}
	for _, art := range a.shown {
		seen[art.Kind] = true
	}
	var out []scan.Kind
	for _, k := range scan.Kinds {
		if seen[k] {
			out = append(out, k)
		}
	}
	return out
}

// itoa writes an int without pulling in strconv for one call site.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
