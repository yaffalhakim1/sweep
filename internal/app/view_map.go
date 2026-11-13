package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"

	"sweep/internal/scan"
	"sweep/internal/treemap"
)

// mapTileLimit is how many projects the map draws at its first zoom level. A
// partition of every project on a large disk is a texture rather than a map,
// and the tiles past the largest few dozen are unreadable and unclickable; the
// list shows all of them.
const mapTileLimit = 24

// mapHeight is the map's height at its largest. The partition needs area to be
// legible, but it also has to share one screen with the summary cards and the
// inspector, so mapHeightFor shrinks it rather than making the page scroll.
const mapHeight = 400

// mapMinHeight is the smallest map worth drawing: below it the tiles stop being
// readable, and scrolling is the better answer.
const mapMinHeight = 220

// mapHeightFor is the map's height for this window, which is the room the
// partition card has left once the page's own furniture is placed.
//
// It is only the first frame's estimate. The box then takes the room the card
// actually gives it (see drawPartition), and mapRoomHeight reports that from
// the second frame on. The sum below is a guess at the page's chrome, and a
// guess that drifts is harmless here: it decides one frame, not the layout.
func mapHeightFor(c *ui.Context) float32 {
	_, height := c.Size()
	h := float32(height) - mapPageChrome
	return clampFloat(h, mapMinHeight, mapHeight)
}

// mapPageChrome is everything on the map page that is not the partition:
// the page header and its gap, the three summary cards, the filter rail, the
// partition's title row and padding, the gaps between the cards, the inspector,
// the footer bar, and the page's bottom padding.
const mapPageChrome = 66 + 130 + 40 + 34 + 2*cardPad + 96 + 5*cardGap + 58 + pagePad

// clampFloat bounds v to [lo, hi].
func clampFloat(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// mapGap is the space left between tiles, which is what separates neighbours
// without a border between them.
const mapGap = 8

// mapMinTile is the smallest tile that can still show a name. A tile below it
// is a colour, not information, so the layout gives its space to the tiles that
// can be read.
const mapMinTile = 62

// mapAspect is the width-to-height ratio the partition is laid out in. A
// squarified treemap of a wide, short box produces a few very wide tiles and a
// column of slivers; a ratio close to the card's own shape gives tiles that can
// all hold a name.
const mapAspect = 2.6

// mapZoomSteps are the zoom levels the map offers, as how many projects each
// one shows.
var mapZoomSteps = []int{mapTileLimit, mapTileLimit * 2, mapTileLimit * 4}

// mapCache holds a laid-out partition and the inputs it was built from, so the
// layout runs when the data or the box changes rather than on every frame.
type mapCache struct {
	tiles []treemap.Tile
	items []treemap.Item
	rect  ui.Rect
	valid bool
}

// mapView draws the space map: a partition of the projects that hold build
// output, each tile sized by what it would reclaim.
//
// It is a project view rather than a folder treemap on purpose. The question a
// full disk asks is "which project is eating it", and a Next.js app's .next and
// node_modules are one answer, not two.
//
// The page does not scroll: the partition fills the room the cards, the rail and
// the inspector leave, so the map and the project it describes are always on one
// screen. That is the whole point of a map.
func (a *App) mapView(c *ui.Context) {
	if len(a.shown) == 0 {
		a.summaryCards(c, nil)
		a.empty(c, "Nothing to map: no build output was found here.")
		return
	}
	projects := a.projects()
	limit := mapZoomSteps[min(a.mapZoom, len(mapZoomSteps)-1)]
	shown := projects
	if len(shown) > limit {
		shown = shown[:limit]
	}

	ui.Column(c).FillWidth().FillHeight().Gap(cardGap).Children(func() {
		a.summaryCards(c, a.shown)
		a.filterRail(c, a.shown)
		// The partition takes the room that is left, and the inspector keeps
		// its own height: a banner clipped by the map would hide the path,
		// which is the one thing a tile cannot show.
		a.partitionCard(c, shown, len(projects))
		a.inspector(c, projects)
	})
}

// projects groups the shown artifacts by the folder that holds them, which is
// what the map draws.
func (a *App) projects() []scan.Project {
	return scan.GroupByProject(a.shown)
}

// partitionCard is the map's card: its title, the zoom controls, and the
// partition itself.
func (a *App) partitionCard(c *ui.Context, projects []scan.Project, total int) {
	t := c.Theme()
	// The layout runs before the title so the subtitle can report how many
	// projects it could not label. Computing it after would make the note a
	// frame behind, and a stale count is worse than none.
	tiles, hidden := a.partitionTiles(c, projects)
	ui.Column(c).FillWidth().Grow(1).MinHeight(mapMinHeight).Padding(cardPad).
		Gap(t.Space(3)).Radius(radiusCard).
		Background(t.Surface).Border(1, t.Border).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(t.Space(3)).Shrink(0).Children(func() {
			ui.Box(c).Size(34, 34).Shrink(0).Radius(9).Center().
				Background(t.Accent.Alpha(0.10)).Children(func() {
				components.Icon(c, lucide.LayoutGrid()).Size(17, 17).TextColor(t.Accent)
			})
			ui.Column(c).Gap(0).Grow(1).Shrink(1).MinWidth(0).Children(func() {
				components.Text(c, "Interactive space partition").FontSize(13).FontWeight(600).
					SingleLine().Ellipsis("…")
				// The subtitle carries the one caveat the map has: tiles too
				// small to label are not drawn, so a partition of four is not
				// a disk with four projects.
				subtitle := "Click a tile to inspect the project, then reclaim it."
				if hidden > 0 {
					subtitle = fmt.Sprintf(
						"%s more too small to label — zoom in, or use the list.",
						itoa(hidden))
				}
				components.Text(c, subtitle).
					FontSize(11).TextColor(t.TextMuted).SingleLine().Ellipsis("…")
			})
			a.zoomControls(c, len(projects), total)
		})
		a.drawPartition(c, tiles)
	})
}

// zoomControls are the map's three view controls: more projects, fewer, and
// back to the start. They are real buttons, so the map can be explored without
// a scroll wheel.
func (a *App) zoomControls(c *ui.Context, shown, total int) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(1)).Shrink(0).AlignItems(ui.Center).Children(func() {
		more := a.mapZoom < len(mapZoomSteps)-1
		fewer := a.mapZoom > 0
		up := components.IconButton(c, lucide.ZoomIn(), "Show more projects", components.IconButtonPlain)
		up.Disabled(!more)
		if up.Clicked() && more {
			a.mapZoom++
		}
		down := components.IconButton(c, lucide.ZoomOut(), "Show fewer projects", components.IconButtonPlain)
		down.Disabled(!fewer)
		if down.Clicked() && fewer {
			a.mapZoom--
		}
		if b := components.IconButton(c, lucide.RotateCcw(), "Reset the map", components.IconButtonPlain); b.Clicked() {
			a.mapZoom, a.mapSelected = 0, ""
		}
		_ = t
	})
}

// partitionTiles lays the projects out and reports which tiles are too small to
// label, so the card can say so and the partition can draw the rest.
//
// The tiles are absolutely positioned inside a fixed-height box: the layout
// already knows every rectangle, and a flex row would not let a tile be both
// wide and tall in the proportions its share of the disk asks for.
func (a *App) partitionTiles(c *ui.Context, projects []scan.Project) ([]mapTile, int) {
	items := projectItems(projects)
	byDir := make(map[string]scan.Project, len(projects))
	for _, p := range projects {
		byDir[p.Dir] = p
	}
	// The box is the room the card actually gives it, measured last frame,
	// rather than a height worked out from the window: the partition card
	// grows into whatever the summary cards and the inspector leave, so a
	// computed height drifts from it and the tiles draw past the card's
	// border. The first frame has no measurement yet and uses the estimate.
	h := a.mapRoomHeight(c)
	width, _ := c.Size()
	r := ui.Rect{
		W: float32(width) - sidebarWidth - 2*pagePad - 2*cardPad - 16,
		H: h,
	}
	if r.W < 320 {
		r.W = 320
	}
	var out []mapTile
	hidden := 0
	for _, tile := range a.layout(items, r) {
		project, ok := byDir[tile.Key]
		if !ok {
			continue
		}
		w, ht := float32(tile.W)-mapGap, float32(tile.H)-mapGap
		if w < mapMinTile || ht < mapMinTile {
			hidden++
			continue
		}
		out = append(out, mapTile{
			Project: project,
			Rect: ui.Rect{
				X: float32(tile.X) + mapGap/2,
				Y: float32(tile.Y) + mapGap/2,
				W: w,
				H: ht,
			},
		})
	}
	return out, hidden
}

// mapTile is one laid-out project: what to draw and where.
type mapTile struct {
	Project scan.Project
	Rect    ui.Rect
}

// drawPartition paints the tiles inside the map's box.
// drawPartition paints the tiles inside the map's box. The box grows into the
// room the card has left, and records that height for the next frame's layout,
// which is what keeps the tiles inside the card instead of drawing past it.
// Clip is the backstop: the one frame that a resize leaves stale cannot paint
// outside the box either.
// drawPartition paints the tiles inside the map's box. The box grows into the
// room the card has left, and records that height for the next frame's layout,
// which is what keeps the tiles inside the card instead of drawing past it.
//
// The layout runs before the box is measured, so the first frame lays tiles out
// for an estimate and the box clips them. Asking for one more frame when the
// measurement differs is what stops that first frame from being the one the
// user sees: MyGo draws nothing while nothing changes, so the clipped map would
// otherwise stay on screen.
func (a *App) drawPartition(c *ui.Context, tiles []mapTile) {
	box := ui.Box(c).FillWidth().Grow(1).Shrink(1).Clip()
	box.Children(func() {
		for _, tile := range tiles {
			a.tile(c, tile.Project, tile.Rect)
		}
	})

	room := box.Bounds().H
	if d := room - a.mapRoom; room > 0 && (d > 0.5 || d < -0.5) {
		a.mapRoom = room
		c.Invalidate()
	}
}

// mapRoomHeight is the height the partition box had last frame, or the
// estimate for the window when there has not been one yet.
func (a *App) mapRoomHeight(c *ui.Context) float32 {
	if a.mapRoom > 0 {
		return a.mapRoom
	}
	return mapHeightFor(c)
}

// tile is one project in the partition: a tinted card with the project's name,
// what it holds, and its size. The tint is the build system's hue at low
// strength, because the label carries the meaning and the hue is only the key.
func (a *App) tile(c *ui.Context, project scan.Project, r ui.Rect) {
	t := c.Theme()
	chosen := a.isChosen(project)
	selected := a.mapSelected == project.Dir
	hue := KindColor(t.Dark, project.Kind())

	b := ui.ButtonBase(c).Key(project.Dir).Label(project.Name).
		Absolute().Left(r.X).Top(r.Y).Width(r.W).Height(r.H).
		Padding(t.Space(2.5)).Radius(radiusCard).Clip().
		Justify(ui.Start).AlignItems(ui.Stretch).Cursor(ui.CursorPointer)
	b.Background(hue.Alpha(0.10)).Border(1, hue.Alpha(0.22))
	switch {
	case chosen:
		b.Background(hue.Alpha(0.20)).Border(2, t.Accent)
	case selected:
		b.Border(2, t.Accent)
	case b.Hovered():
		b.Background(hue.Alpha(0.16)).Border(1, hue.Alpha(0.35))
	}
	b.OnClick(func() { a.mapSelected = project.Dir })
	b.Tooltip(projectTooltip(project))
	b.Children(func() {
		ui.Column(c).FillWidth().FillHeight().Gap(t.Space(1)).Children(func() {
			// The head: a folder mark, the project's name, and its size. A
			// narrow tile drops the mark and then the size, because the name
			// is what identifies the project and the tooltip carries the rest.
			roomy := r.W >= 150
			ui.Row(c).FillWidth().AlignItems(ui.Start).Gap(t.Space(2)).Children(func() {
				if roomy {
					ui.Box(c).Size(30, 30).Shrink(0).Radius(8).Center().
						Background(hue.Alpha(0.16)).Children(func() {
						components.Icon(c, projectIcon(project)).Size(15, 15).TextColor(hue)
					})
				}
				ui.Column(c).Gap(0).Grow(1).Shrink(1).MinWidth(0).Children(func() {
					components.Text(c, project.Name).FontSize(13).FontWeight(600).
						SingleLine().Ellipsis("…")
					if r.H >= 90 {
						components.Text(c, project.Summary()).FontSize(11).TextColor(t.TextMuted).
							SingleLine().Ellipsis("…")
					}
				})
				if r.W >= 210 {
					components.Text(c, scan.HumanBytes(project.Bytes)).FontSize(13).
						FontWeight(600).TextColor(hue).Shrink(0).SingleLine()
				}
			})
			ui.Spacer(c)
			// The foot only appears where there is room for it: a tile that
			// clipped its own foot would look broken rather than compact.
			if r.H < 132 {
				return
			}
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(t.Space(2)).Children(func() {
				components.Icon(c, lucide.ShieldCheck()).Size(13, 13).Shrink(0).
					TextColor(t.Success)
				components.Text(c, projectFoot(project)).FontSize(11).
					TextColor(t.TextMuted).Grow(1).Shrink(1).MinWidth(0).
					SingleLine().Ellipsis("…")
				if r.W >= 210 {
					components.Text(c, tileAction(chosen)).FontSize(11).FontWeight(600).
						TextColor(t.Accent).Shrink(0)
				}
			})
		})
	})
}

// projectIcon picks the glyph for a project's shape: a package for a
// dependency tree, an archive for build output, a database for a cache.
func projectIcon(p scan.Project) *ui.SVG {
	for _, art := range p.Artifacts {
		if art.Rule.Cache {
			return lucide.Database()
		}
	}
	for _, art := range p.Artifacts {
		switch art.Name {
		case "node_modules", ".venv", "venv", "Pods":
			return lucide.Package()
		}
	}
	return lucide.FolderArchive()
}

// projectFoot says what removing the project costs, which is the one thing a
// size cannot tell you.
func projectFoot(p scan.Project) string {
	switch {
	case len(p.Artifacts) == 0:
		return "Safe to purge"
	case p.Costly():
		return "Rebuilds next time"
	default:
		return "Safe to purge"
	}
}

// tileAction is the tile's trailing word: what clicking it will do.
func tileAction(chosen bool) string {
	if chosen {
		return "Selected ✓"
	}
	return "Purge"
}

// isChosen reports whether every artifact of a project is selected, which is
// what a selected tile means.
func (a *App) isChosen(p scan.Project) bool {
	if len(p.Artifacts) == 0 {
		return false
	}
	for _, art := range p.Artifacts {
		if !a.selected[art.Path] {
			return false
		}
	}
	return true
}

// inspector is the banner under the map: the project the last click chose, with
// its full path and the two actions that can be taken on it.
//
// It is always drawn, because the map's tiles are small and the path is the one
// piece of information a tile cannot show.
func (a *App) inspector(c *ui.Context, projects []scan.Project) {
	t := c.Theme()
	project, ok := scan.FindProject(projects, a.mapSelected)
	if !ok && len(projects) > 0 {
		project, ok = projects[0], true
	}
	ui.Column(c).FillWidth().Padding(cardPad).Radius(radiusCard).
		Background(t.Surface).Border(1, t.Border).Children(func() {
		if !ok {
			ui.Row(c).AlignItems(ui.Center).Gap(t.Space(2)).Children(func() {
				components.Icon(c, lucide.FolderOpen()).Size(16, 16).TextColor(t.TextMuted)
				components.Text(c, "Click a tile to inspect a project.").
					FontSize(12).TextColor(t.TextMuted)
			})
			return
		}
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(t.Space(3)).Wrap().Children(func() {
			ui.Box(c).Size(40, 40).Shrink(0).Radius(10).Center().
				Background(t.Accent.Alpha(0.10)).Children(func() {
				components.Icon(c, lucide.FolderOpen()).Size(19, 19).TextColor(t.Accent)
			})
			ui.Column(c).Gap(t.Space(1)).Grow(1).Shrink(1).MinWidth(0).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(t.Space(2)).Wrap().Children(func() {
					components.Text(c, project.Name).FontSize(14).FontWeight(600).
						SingleLine().Ellipsis("…").Shrink(1)
					components.Badge(c, "Safe to clean", components.BadgeOptions{
						Tone: components.BadgeSuccess,
						Icon: lucide.ShieldCheck(),
					})
					components.Text(c, "Size: "+scan.HumanBytes(project.Bytes)).
						FontSize(11).TextColor(t.TextMuted).Shrink(0)
				})
				components.Text(c, project.Dir).FontSize(11).TextColor(t.TextMuted).
					SingleLine().Ellipsis("…")
			})
			ui.Row(c).Gap(t.Space(2)).Shrink(0).AlignItems(ui.Center).Children(func() {
				if b := components.Button(c, "Open in Explorer", components.ButtonSecondary); b.Clicked() {
					a.Reveal(project.Dir)
				}
				label := "Reclaim " + scan.HumanBytes(project.Bytes)
				b := components.Button(c, label, components.ButtonPrimary)
				b.Tooltip("Choose this project's build output and review it before removing")
				if b.Clicked() {
					a.selectProject(project)
				}
			})
		})
	})
}

// selectProject chooses a project's artifacts and opens the confirmation, so
// the map's main action is one press rather than a selection the user then has
// to finish elsewhere.
func (a *App) selectProject(p scan.Project) {
	if len(p.Artifacts) == 0 {
		return
	}
	a.selected = map[string]bool{}
	for _, path := range p.Paths() {
		a.selected[path] = true
	}
	a.confirm = true
}

// projectItems turns projects into the map's items, which is what both the
// layout and the cache key are built from.
func projectItems(projects []scan.Project) []treemap.Item {
	items := make([]treemap.Item, 0, len(projects))
	for _, p := range projects {
		items = append(items, treemap.Item{Key: p.Dir, Value: p.Bytes, Kind: string(p.Kind())})
	}
	return items
}

// layout returns the partition's tiles for the box, relaying them out only when
// the projects or the box have changed.
//
// This is what keeps the map responsive: the layout is O(n log n) with a
// squarify pass over up to a few dozen tiles, and a pointer that merely moves
// produces frames where nothing about it changed.
func (a *App) layout(items []treemap.Item, r ui.Rect) []treemap.Tile {
	if a.mapCache.valid && a.mapCache.rect == r && sameItems(a.mapCache.items, items) {
		return a.mapCache.tiles
	}
	tiles := treemap.Layout(items, float64(r.X), float64(r.Y), float64(r.W), float64(r.H))
	a.mapCache = mapCache{tiles: tiles, items: items, rect: r, valid: true}
	return tiles
}

// sameItems reports whether two item lists hold the same values in the same
// order, which is what the layout depends on.
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

// tileAt returns the tile under a point in painted coordinates. The tiles do
// not overlap, so the first hit is the only hit.
func tileAt(tiles []treemap.Tile, x, y float32) (string, bool) {
	for _, tile := range tiles {
		if float64(x) >= tile.X && float64(x) < tile.X+tile.W &&
			float64(y) >= tile.Y && float64(y) < tile.Y+tile.H {
			return tile.Key, true
		}
	}
	return "", false
}

// hoveredTileAt returns the tile under a pointer, given the map's committed box
// and the pointer's position in the element's own coordinates.
//
// The two spaces differ by the box's own origin, and getting that wrong shifts
// the hit test by exactly that much: the bug this signature exists to make
// testable named the tile above the one under the pointer.
func (a *App) hoveredTileAt(items []treemap.Item, r ui.Rect, x, y float32, over bool) (string, bool) {
	if !over || r.W <= 0 || r.H <= 0 {
		return "", false
	}
	return tileAt(a.layout(items, r), r.X+x, r.Y+y)
}

// projectTooltip describes a project in a sentence, so a hover answers "what is
// this and is it safe" without leaving the map.
func projectTooltip(p scan.Project) string {
	parts := []string{p.Dir, scan.HumanBytes(p.Bytes)}
	if p.Files > 0 {
		parts = append(parts, plural(p.Files, "file", "files"))
	}
	for _, art := range p.Artifacts {
		if art.Rule.Note != "" {
			parts = append(parts, art.Rule.Note)
			break
		}
	}
	return strings.Join(parts, "\n")
}
