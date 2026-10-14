package app

import (
	"github.com/egoist/mygo/ui"

	"sweep/internal/scan"
)

// listLimit is how many rows the list builds. It is a virtual list, so
// only the rows in view are built; the cap is there to keep the row index
// arithmetic and the keyboard walk bounded on a pathological disk.
const listLimit = 2000

// listRow is one line of the list: the artifacts shown largest first,
// which is the order a user wants when looking for space.
type listRow struct {
	Artifact scan.Artifact
	// Header marks a row that introduces a kind rather than being an
	// artifact, which is how the list groups without a tree widget.
	Header bool
	Kind   scan.Kind
	Bytes  int64
	Count  int
}

// listView shows the artifacts grouped by build system, largest first,
// each with a checkbox. This is the actionable view: the map answers where
// the space went, the list answers what to do about it.
func (a *App) listView(c *ui.Context) {
	rows := a.listRows()
	card(c, func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, "Build output").FontSize(13).FontWeight(600)
			ui.Text(c, "·").FontSize(13).TextColor(t2(c))
			ui.Text(c, "largest first, grouped by build system").
				FontSize(12).TextColor(t2(c))
			ui.Spacer(c)
			if b := ui.Button(c, "Select all").Padding(5, 10); b.Clicked() {
				a.SelectAll()
			}
			if b := ui.Button(c, "None").Padding(5, 10); b.Clicked() {
				a.ClearSelection()
			}
		})
		ui.Column(c).Gap(0).Children(func() {
			ui.List(c, &a.list, len(rows)).ItemKey(func(i int) any {
				if rows[i].Header {
					return "h:" + string(rows[i].Kind)
				}
				return rows[i].Artifact.Path
			}).Rows(func(row ui.ListRow) {
				r := rows[row.Index]
				if r.Header {
					a.kindHeader(row.Context, r)
					return
				}
				a.artifactRow(row.Context, r.Artifact)
			})
		})
	})
}

// t2 is the muted text colour, named short so the layout code above stays
// readable.
func t2(c *ui.Context) ui.Color { return c.Theme().TextMuted }

// listRows flattens the grouped artifacts into the list's rows, so the
// list is a flat virtual list rather than a tree: the grouping is a
// heading, not a structure to expand.
func (a *App) listRows() []listRow {
	groups := scan.GroupByKind(a.shown)
	var rows []listRow
	for _, g := range groups {
		rows = append(rows, listRow{Header: true, Kind: g.Kind, Bytes: g.Bytes, Count: len(g.Artifacts)})
		for _, art := range g.Artifacts {
			if len(rows) >= listLimit {
				return rows
			}
			rows = append(rows, listRow{Artifact: art})
		}
	}
	return rows
}

// kindHeader introduces a group, with what it totals.
func (a *App) kindHeader(c *ui.Context, r listRow) {
	t := c.Theme()
	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(6, 4).
		BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
		ui.Box(c).Size(9, 9).Radius(2).Background(KindColor(t.Dark, r.Kind))
		ui.Text(c, string(r.Kind)).FontSize(12).FontWeight(600)
		ui.Text(c, plural(r.Count, "folder", "folders")).
			FontSize(12).TextColor(t.TextMuted)
		ui.Spacer(c)
		ui.Text(c, scan.HumanBytes(r.Bytes)).FontSize(12).TextColor(t.TextMuted)
	})
}

// artifactRow is one artifact: a checkbox, its name and path, its size,
// and how long ago it changed. The whole row is the hit target for the
// checkbox, so choosing one does not require hitting a 16-pixel square.
func (a *App) artifactRow(c *ui.Context, art scan.Artifact) {
	t := c.Theme()
	chosen := a.selected[art.Path]
	row := ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(6, 4).Radius(6).
		Cursor(ui.CursorPointer)
	if chosen {
		row.Background(t.Selection)
	} else if row.Hovered() {
		row.Background(t.SurfaceHover)
	}
	row.OnClick(func() { a.Toggle(art.Path) })

	row.Children(func() {
		// The box is drawn rather than using a Checkbox, because the row
		// owns the click: a nested control would take the click first and
		// the row would only respond outside the box.
		box := ui.Box(c).Size(16, 16).Radius(4)
		if chosen {
			box.Background(t.Accent)
		} else {
			box.Background(t.Background).Border(1, t.Border)
		}
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			if !chosen {
				return
			}
			check := (&ui.Path{}).
				MoveTo(r.X+4, r.Y+8).
				LineTo(r.X+7, r.Y+11).
				LineTo(r.X+12, r.Y+5)
			p.StrokePath(check, 2, t.AccentText)
		})
		ui.Column(c).Gap(1).Grow(1).Shrink(1).MinWidth(0).Children(func() {
			ui.Text(c, artifactLabel(art)).FontSize(13).SingleLine().Ellipsis("…")
			// On a selected row the path cannot stay muted: the selection
			// wash lifts the background toward the muted grey, which
			// measured 4.05:1. Full-contrast ink keeps it readable.
			path := ui.Text(c, art.Path).FontSize(11).
				SingleLine().Ellipsis("…")
			if !chosen {
				path.TextColor(t.TextMuted)
			}
		})
		if when := timeLabel(art.Touched); when != "" {
			// The same rule as the path: on a selected row the wash lifts
			// the background, so muted ink no longer clears 4.5:1.
			stamp := ui.Text(c, when).FontSize(11).Shrink(0)
			if !chosen {
				stamp.TextColor(t.TextMuted)
			}
		}
		ui.Text(c, scan.HumanBytes(art.Bytes)).FontSize(13).
			FontWeight(600).Shrink(0).Width(78).TextAlign(ui.End)
	})
}

// countLabel writes "N folders" for a selection summary.
func countLabel(n int) string { return plural(n, "folder", "folders") }
