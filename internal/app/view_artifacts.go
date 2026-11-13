package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"

	"sweep/internal/scan"
)

// artifactsPage is the list view the design is built around: three summary
// cards, a filter rail, and the table of everything that was found.
//
// The summary cards are drawn even when nothing was found, because the drive
// card is the way out of that state: a page that hides its own picker when the
// pick was wrong is a dead end.
func (a *App) artifactsPage(c *ui.Context, arts []scan.Artifact, title, emptyNote string) {
	ui.Column(c).FillWidth().FillHeight().Gap(cardGap).Children(func() {
		a.summaryCards(c, arts)
		if len(arts) == 0 {
			a.empty(c, emptyNote)
			return
		}
		a.filterRail(c, arts)
		a.table(c, arts)
	})
}

// summaryCards are the three answers a user wants before reading a single row:
// how much can go, how full the disk is, and which disk it is.
func (a *App) summaryCards(c *ui.Context, arts []scan.Artifact) {
	t := c.Theme()
	// Three cards side by side need room for three readable numbers. Below
	// that they stack, which is a structural change, not a scale: the window's
	// own width decides it, because the sidebar takes a fixed column.
	//
	// The threshold is deliberately low: stacking three cards costs about
	// 300px of height, which on a short window pushes the table itself off
	// screen. Squeezing three numbers is the lesser evil until there is truly
	// no room for them.
	width, _ := c.Size()
	content := width - sidebarWidth - 2*pagePad
	stack := content < 480
	row := ui.Row(c).FillWidth().Gap(cardGap)
	if stack {
		row = ui.Column(c).FillWidth().Gap(cardGap)
	}
	row.Children(func() {
		card := func(build func()) {
			panel := ui.Column(c).Padding(cardPad).Gap(t.Space(1)).Radius(radiusCard).
				Background(t.Surface).Border(1, t.Border).Children(build)
			if stack {
				panel.FillWidth()
			} else {
				panel.Grow(1).Basis(0).MinWidth(0)
			}
		}
		card(func() { a.reclaimableCard(c, arts) })
		card(func() { a.freeCard(c) })
		card(func() { a.driveCard(c) })
	})
}

// cardHead is the small label and the icon a summary card leads with, so the
// three cards read as one row of the same object.
func cardHead(c *ui.Context, label string, icon *ui.SVG) {
	t := c.Theme()
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(t.Space(2)).Children(func() {
		components.Text(c, label).FontSize(12).FontWeight(500).TextColor(t.TextMuted).
			Grow(1).SingleLine().Ellipsis("…")
		ui.Box(c).Size(30, 30).Shrink(0).Radius(8).Center().Background(t.Text.Alpha(0.05)).
			Children(func() {
				components.Icon(c, icon).Size(15, 15).TextColor(t.TextMuted)
			})
	})
}

// reclaimableCard is the total the page can free, with what it is made of.
func (a *App) reclaimableCard(c *ui.Context, arts []scan.Artifact) {
	t := c.Theme()
	cardHead(c, "Total reclaimable", lucide.Eraser())
	components.Text(c, scan.HumanBytes(scan.Total(arts))).FontSize(26).Bold().SingleLine()
	ui.Row(c).AlignItems(ui.Center).Gap(t.Space(1)).Children(func() {
		components.Icon(c, lucide.CircleCheck()).Size(13, 13).Shrink(0).TextColor(t.Success)
		ui.Textf(c, "%s · %s dormant",
			plural(len(arts), "folder", "folders"), itoa(a.dormantCount())).
			FontSize(12).TextColor(t.TextMuted).SingleLine().Ellipsis("…")
	})
}

// freeCard is the disk's free space, which is the number the user came for.
func (a *App) freeCard(c *ui.Context) {
	t := c.Theme()
	cardHead(c, "Free storage", lucide.Gauge())
	if !a.haveDisk {
		components.Text(c, "—").FontSize(26).Bold()
		components.Text(c, "No capacity reported for this target").FontSize(12).TextColor(t.TextMuted)
		return
	}
	components.Text(c, scan.HumanBytes(a.volume.Free)).FontSize(26).Bold().SingleLine()
	ui.Textf(c, "%s of %s in use",
		scan.HumanBytes(a.volume.Used()), scan.HumanBytes(a.volume.Total)).
		FontSize(12).TextColor(t.TextMuted).SingleLine().Ellipsis("…")
}

// driveCard is which disk is being scanned, and the picker to change it. It
// is a button rather than a card that happens to contain one, because the
// whole card is the target a user aims at.
func (a *App) driveCard(c *ui.Context) {
	t := c.Theme()
	cardHead(c, "Active drive", lucide.HardDrive())
	label := a.diskLabel()
	b := ui.ButtonBase(c).FillWidth().Justify(ui.Start).Gap(t.Space(1.5)).
		Padding(0).Radius(t.Radius).Cursor(ui.CursorPointer)
	b.Label("Choose a drive or folder to scan")
	b.Menu(func(m *ui.Menu) {
		// The menu's build callback runs when the menu opens, which is the
		// right moment to re-read the drives: a disk plugged in since launch
		// appears here, and the cost is one enumeration per open rather than
		// one per frame.
		a.LoadVolumes()
		for _, v := range a.volumes {
			name := v.Friendly()
			if m.Item(name).Checked(a.scansVolume(v.Root)).Chosen() {
				a.ChooseVolume(v.Root)
			}
		}
		if len(a.volumes) > 0 {
			m.Separator()
		}
		if m.Item("Home folder").Checked(len(a.roots) == 0).Chosen() {
			a.ResetRoot()
		}
		m.Separator()
		if m.Item("Choose a folder…").Chosen() {
			a.ChooseRoot()
		}
	})
	b.Children(func() {
		components.Text(c, label).FontSize(20).FontWeight(600).Grow(1).Shrink(1).
			MinWidth(0).SingleLine().Ellipsis("…")
		components.Icon(c, lucide.ChevronsUpDown()).Size(15, 15).Shrink(0).TextColor(t.TextMuted)
	})
	components.Text(c, a.driveSubtitle()).FontSize(12).TextColor(t.TextMuted).
		SingleLine().Ellipsis("…")
}

// driveSubtitle describes the target under the drive's name: a volume says
// what it is, a folder says its path, and both say when the scan last ran.
func (a *App) driveSubtitle() string {
	kind := "Home folder"
	switch {
	case len(a.roots) > 1:
		kind = fmt.Sprintf("%d folders", len(a.roots))
	case len(a.roots) == 1 && a.haveDisk && a.scansVolume(a.volume.Root):
		kind = "Whole volume"
	case len(a.roots) == 1:
		kind = "Folder"
	}
	return kind + " · scanned " + a.age()
}

// filterRail is the row above the table: what kind of artifact to show, and
// how small an artifact is too small to bother with.
//
// The chips count the page's own rows, not the whole scan: on the cache page a
// chip that said "Node.js 18" while the table showed two would be a lie.
func (a *App) filterRail(c *ui.Context, arts []scan.Artifact) {
	t := c.Theme()
	ui.Row(c).FillWidth().Wrap().AlignItems(ui.Center).Gap(t.Space(2)).Children(func() {
		components.Text(c, "Filter").FontSize(12).TextColor(t.TextMuted).Shrink(0)
		a.filterChip(c, "", "All", len(arts))
		for _, kind := range kindsIn(arts) {
			a.filterChip(c, kind, kindFilterLabel(kind), countKind(arts, kind))
		}
		ui.Spacer(c)
		a.sizeFilter(c)
	})
}

// kindsIn lists the kinds present in a set of artifacts, in the app's kind
// order, so the chips do not shuffle between scans.
func kindsIn(arts []scan.Artifact) []scan.Kind {
	seen := map[scan.Kind]bool{}
	for _, art := range arts {
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

// countKind counts the artifacts of one kind.
func countKind(arts []scan.Artifact, k scan.Kind) int {
	n := 0
	for _, art := range arts {
		if art.Kind == k {
			n++
		}
	}
	return n
}

// kindFilterLabel names a kind the way the rail reads it: the ecosystem's own
// name, not the internal one.
func kindFilterLabel(k scan.Kind) string {
	switch k {
	case scan.Node:
		return "Node.js"
	case scan.JVM:
		return "JVM"
	case scan.CMake:
		return "CMake"
	case scan.DotNet:
		return ".NET"
	default:
		return string(k)
	}
}

// filterChip is one value of the kind filter. The chosen chip is the accent
// face; the rest are quiet until hovered.
func (a *App) filterChip(c *ui.Context, kind scan.Kind, label string, count int) {
	t := c.Theme()
	on := a.kindFilter == kind
	b := ui.ButtonBase(c).Padding(t.Space(1), t.Space(2.5)).Radius(999).
		Gap(t.Space(1.5)).AlignItems(ui.Center).Cursor(ui.CursorPointer).Shrink(0)
	switch {
	case on:
		b.Background(t.Accent).TextColor(t.AccentText)
	default:
		b.Background(t.Text.Alpha(0.05)).TextColor(t.TextMuted)
		if b.Hovered() {
			b.Background(t.Text.Alpha(0.09)).TextColor(t.Text)
		}
	}
	b.Label(label)
	b.OnClick(func() {
		a.kindFilter = kind
		a.refresh()
	})
	b.Children(func() {
		components.Text(c, label).FontSize(12).FontWeight(500).SingleLine()
		countColor := t.TextMuted.Alpha(0.85)
		if on {
			countColor = t.AccentText.Alpha(0.75)
		}
		components.Text(c, itoa(count)).FontSize(11).TextColor(countColor)
	})
}

// sizeFilter is the minimum-size rail. It is an index into a fixed set of
// stops rather than a byte count, so the slider is predictable and the label
// is always a size a person would say.
func (a *App) sizeFilter(c *ui.Context) {
	t := c.Theme()
	index := sizeStepIndex(a.minSize)
	ui.Row(c).AlignItems(ui.Center).Gap(t.Space(2)).Shrink(0).Children(func() {
		components.Text(c, "Min size").FontSize(12).TextColor(t.TextMuted).Shrink(0)
		value := float64(index)
		slider := components.Slider(c, &value, 0, float64(len(sizeSteps)-1),
			components.SliderOptions{Step: 1})
		slider.Label("Minimum artifact size")
		slider.Width(120)
		if slider.Changed() {
			a.minSize = sizeSteps[int(value)]
			a.refresh()
			a.saveSettings()
		}
		components.Text(c, sizeStepLabel(index)).FontSize(12).FontWeight(600).
			Shrink(0).Width(74).TextAlign(ui.End).SingleLine()
	})
}

// table is the artifact table. The header and the rows are built from the
// same column widths, so a column cannot drift out of line with its cells.
//
// It fills the room the rail leaves and scrolls inside its own card, so the
// column titles stay put while the rows move under them.
func (a *App) table(c *ui.Context, arts []scan.Artifact) {
	t := c.Theme()
	rows := a.rowsFor(arts)
	// The columns are chosen from the room the table actually has, which is the
	// window's width minus the sidebar and the page's margins.
	width, _ := c.Size()
	cols := columnsFor(width - sidebarWidth - 2*pagePad)
	card := ui.Column(c).FillWidth().Grow(1).MinHeight(0).Radius(radiusCard).
		Background(t.Surface).Border(1, t.Border).Clip()
	// Rows paint their own background right up to the card's edge, which would
	// cover the hairline border on the last visible row. Stroking it again on
	// top keeps the card an object rather than a list that leaks out of one.
	card.DrawOver(func(p *ui.Painter, r ui.Rect) {
		p.Stroke(r, t.Border, radiusCard, 1)
	})
	card.Children(func() {
		a.tableToolbar(c, arts)
		ui.Box(c).FillWidth().Height(1).Shrink(0).Background(t.Border)
		if len(rows) == 0 {
			a.tableEmpty(c)
			return
		}
		a.tableHeader(c, cols)
		ui.List(c, &a.list, len(rows)).Grow(1).MinHeight(0).ItemKey(func(i int) any {
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
			a.artifactRow(row.Context, r.Artifact, cols)
		})
	})
}

// rowsFor returns the rows for the artifacts a page shows, reusing the cached
// build when it is still current.
func (a *App) rowsFor(arts []scan.Artifact) []artifactRow {
	if len(arts) == len(a.shown) {
		return a.rows()
	}
	return buildRows(arts, a.order)
}

// Column widths, shared by the header and the rows so a column can never drift
// out of line with its cells. The name column is the one that takes the
// remaining room, which is why it is sized with a zero basis rather than by its
// content: a long path would otherwise set the width.
const (
	colCheck  = 28
	colType   = 150
	colSize   = 96
	colDate   = 116
	colAction = 112
)

// columns is which columns the table shows at a given width. A narrow window
// drops the least useful columns rather than squeezing every one: a table whose
// name column reads "node_m…" has stopped answering the question it exists for.
type columns struct {
	typeChip bool
	size     bool
	modified bool
	action   bool
	// typeWidth narrows the ecosystem chip before the column is dropped.
	typeWidth float32
}

// columnsFor picks the columns that fit. The thresholds are on the table's own
// width, because that is what the columns have to share: the window's width
// minus the sidebar says nothing about the padding around it.
func columnsFor(width float32) columns {
	switch {
	case width >= 760:
		return columns{typeChip: true, size: true, modified: true, action: true, typeWidth: colType}
	case width >= 560:
		return columns{typeChip: true, size: true, action: true, typeWidth: 120}
	default:
		return columns{size: true, action: true}
	}
}

// fixed is the room the columns that are not the name take.
func (c columns) fixed() float32 {
	var w float32
	if c.typeChip {
		w += c.typeWidth
	}
	if c.size {
		w += colSize
	}
	if c.modified {
		w += colDate
	}
	if c.action {
		w += colAction
	}
	return w
}

// tableToolbar is the strip above the columns: where the rows came from, how
// many there are, and the bulk selection actions.
func (a *App) tableToolbar(c *ui.Context, arts []scan.Artifact) {
	t := c.Theme()
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(t.Space(2)).
		Padding(cardPad, cardPad, t.Space(3), cardPad).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(t.Space(1)).Shrink(1).MinWidth(0).Children(func() {
			a.crumb(c, "Sweep")
			components.Icon(c, lucide.ChevronRight()).Size(12, 12).TextColor(t.TextMuted)
			a.crumb(c, a.rootLabel())
			components.Icon(c, lucide.ChevronRight()).Size(12, 12).TextColor(t.TextMuted)
			components.Text(c, a.pageTitleOnly()).FontSize(12).FontWeight(600).
				SingleLine().Ellipsis("…")
		})
		ui.Box(c).Padding(1, 6).Radius(6).Background(t.Text.Alpha(0.06)).Shrink(0).Children(func() {
			ui.Textf(c, "%d found", len(arts)).FontSize(11).TextColor(t.TextMuted)
		})
		ui.Spacer(c)
		chosen := len(a.selected)
		if chosen > 0 {
			ui.Textf(c, "%s selected", plural(chosen, "folder", "folders")).
				FontSize(12).TextColor(t.TextMuted).Shrink(0)
			if b := components.Button(c, "Deselect", components.ButtonLink); b.Clicked() {
				a.ClearSelection()
			}
		} else {
			if b := components.Button(c, "Select all", components.ButtonLink); b.Clicked() {
				a.SelectAll()
			}
		}
	})
}

// crumb is one step of the table's path. Only the last step is a label, so
// the path reads as position rather than as three buttons.
func (a *App) crumb(c *ui.Context, label string) {
	t := c.Theme()
	components.Text(c, label).FontSize(12).TextColor(t.TextMuted).
		SingleLine().Ellipsis("…").Shrink(1)
}

// pageTitleOnly is the page's name without its description, for the
// breadcrumb's last step.
func (a *App) pageTitleOnly() string {
	title, _ := a.pageTitle()
	return title
}

// tableEmpty is what the table shows when the filters leave nothing. It names
// the filter that is most likely responsible rather than only saying "empty".
func (a *App) tableEmpty(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).FillWidth().Padding(cardPad, 40).Gap(t.Space(2)).Center().Children(func() {
		components.Icon(c, lucide.SearchX()).Size(22, 22).TextColor(t.TextMuted)
		components.Text(c, "No folders match the filters").FontSize(13).FontWeight(600)
		components.Text(c, "Lower the minimum size, or choose a different ecosystem.").
			FontSize(12).TextColor(t.TextMuted).TextAlign(ui.Center)
		if a.minSize > 0 || a.kindFilter != "" || a.query != "" {
			if b := components.Button(c, "Reset filters", components.ButtonSecondary); b.Clicked() {
				a.resetFilters()
			}
		}
	})
}

// resetFilters returns the list to everything the scan found.
func (a *App) resetFilters() {
	a.query, a.minSize, a.kindFilter = "", 0, ""
	a.refresh()
	a.saveSettings()
}

// tableHeader is the column titles. Each sortable title orders the rows and
// shows which column is ordering them, so the order is never a mystery.
func (a *App) tableHeader(c *ui.Context, cols columns) {
	t := c.Theme()
	ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(0, cardPad).
		MinHeight(t.Space(9)).Shrink(0).Children(func() {
		// The checkbox column, the same width as the rows'.
		ui.Box(c).Width(colCheck).Shrink(0)
		for _, col := range sortKeys {
			if !cols.shows(col.key) {
				continue
			}
			on := a.order.sort == col.key
			b := ui.ButtonBase(c).Gap(t.Space(1)).AlignItems(ui.Center).
				Padding(t.Space(1), 0).Cursor(ui.CursorPointer).Shrink(0)
			b.Label("Sort by " + col.title)
			b.OnClick(func() {
				if a.order.sort == col.key {
					a.order.ascending = !a.order.ascending
				} else {
					a.order.sort, a.order.ascending = col.key, defaultAscending(col.key)
				}
				a.tableKey = ""
			})
			b.Children(func() {
				color := t.TextMuted
				if on {
					color = t.Text
				}
				// An end-aligned column leads with its arrow so the label still
				// ends at the column's edge: an arrow after the label would
				// shift every title 15px left of the values beneath it.
				end := col.key == sortSize || col.key == sortModified
				arrow := func() {
					if !on {
						return
					}
					icon := lucide.ArrowUp()
					if !a.order.ascending {
						icon = lucide.ArrowDown()
					}
					components.Icon(c, icon).Size(11, 11).Shrink(0).TextColor(color)
				}
				if end {
					arrow()
				}
				components.Text(c, col.title).FontSize(11).FontWeight(500).
					SingleLine().TextColor(color)
				if !end {
					arrow()
				}
			})
			// One width per column, from the same constants the cells use.
			switch col.key {
			case sortName:
				b.Basis(0).Grow(1).Shrink(1).MinWidth(0).Justify(ui.Start)
			case sortType:
				b.Width(cols.typeWidth).Justify(ui.Start)
			case sortSize:
				b.Width(colSize).Justify(ui.End)
			case sortModified:
				b.Width(colDate).Justify(ui.End)
			}
		}
		if cols.action {
			ui.Box(c).Width(colAction).Shrink(0)
		}
	})
	ui.Box(c).FillWidth().Height(1).Shrink(0).Background(t.Border)
}

// shows reports whether the chosen column set includes a column.
func (c columns) shows(k sortKey) bool {
	switch k {
	case sortType:
		return c.typeChip
	case sortSize:
		return c.size
	case sortModified:
		return c.modified
	default:
		return true
	}
}

// defaultAscending is how a column sorts the first time it is chosen. Size and
// modified start at the top of their range, which is the question a user has
// about a disk.
func defaultAscending(k sortKey) bool {
	switch k {
	case sortSize, sortModified:
		return false
	default:
		return true
	}
}

// kindHeader introduces a build system in the table, with what it totals. It
// is a row rather than a column because the grouping is a reading aid, not a
// structure to expand.
func (a *App) kindHeader(c *ui.Context, r artifactRow) {
	t := c.Theme()
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(t.Space(2)).
		Padding(t.Space(2), cardPad).Background(t.Text.Alpha(0.025)).Children(func() {
		ui.Box(c).Size(9, 9).Radius(2.5).Shrink(0).Background(KindColor(t.Dark, r.Kind))
		components.Text(c, string(r.Kind)).FontSize(11).FontWeight(600).SingleLine()
		ui.Textf(c, "%s · %s",
			plural(r.Count, "folder", "folders"), scan.HumanBytes(r.Bytes)).
			FontSize(11).TextColor(t.TextMuted).SingleLine()
		ui.Spacer(c)
		ui.Row(c).Gap(t.Space(1)).Shrink(0).Children(func() {
			if b := components.Button(c, "Select these", components.ButtonLink); b.Clicked() {
				a.selectKind(r.Kind)
			}
		})
	})
}

// selectKind chooses every artifact of one kind among those shown.
func (a *App) selectKind(k scan.Kind) {
	for _, art := range a.shown {
		if art.Kind == k {
			a.selected[art.Path] = true
		}
	}
}

// artifactRow is one artifact: a checkbox, its name and project with the path
// beneath, its ecosystem, its size, when it last changed, and its actions.
//
// The whole row is the hit target for the checkbox, so choosing one never
// requires hitting a 16-pixel square.
func (a *App) artifactRow(c *ui.Context, art scan.Artifact, cols columns) {
	t := c.Theme()
	chosen := a.selected[art.Path]
	row := ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(t.Space(2), cardPad).
		MinHeight(rowHeight).Cursor(ui.CursorPointer)
	switch {
	case chosen:
		row.Background(t.Selection)
	case row.Hovered():
		row.Background(t.SurfaceHover)
	}
	row.OnClick(func() { a.Toggle(art.Path) })
	row.Children(func() {
		ui.Box(c).Width(colCheck).Shrink(0).Children(func() {
			a.checkbox(c, chosen)
		})
		// Name, project and path: the leaf name alone does not identify an
		// artifact, because every Node project has a node_modules.
		ui.Row(c).Basis(0).Grow(1).Shrink(1).MinWidth(0).Gap(t.Space(2)).
			AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(30, 30).Shrink(0).Radius(8).Center().
				Background(t.Text.Alpha(0.05)).Children(func() {
				components.Icon(c, folderIcon(art)).Size(15, 15).TextColor(KindColor(t.Dark, art.Kind))
			})
			ui.Column(c).Gap(0).Basis(0).Grow(1).Shrink(1).MinWidth(0).Children(func() {
				ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).MinWidth(0).Children(func() {
					components.Text(c, artifactLabel(art)).FontSize(13).FontWeight(600).
						SingleLine().Ellipsis("…").Shrink(1)
					if project := scan.ProjectName(art); project != "" {
						components.Text(c, "("+project+")").FontSize(12).
							TextColor(rowMuted(c, chosen)).SingleLine().Ellipsis("…").Shrink(1)
					}
				})
				components.Text(c, art.Path).FontSize(11).
					TextColor(rowMuted(c, chosen)).SingleLine().Ellipsis("…")
			})
		})
		if cols.typeChip {
			ui.Box(c).Width(cols.typeWidth).Shrink(0).Children(func() {
				a.typeChip(c, art)
			})
		}
		if cols.size {
			ui.Box(c).Width(colSize).Shrink(0).Justify(ui.End).Children(func() {
				components.Text(c, scan.HumanBytes(art.Bytes)).FontSize(13).FontWeight(600).
					SingleLine()
			})
		}
		if cols.modified {
			ui.Box(c).Width(colDate).Shrink(0).Justify(ui.End).Children(func() {
				components.Text(c, modifiedLabel(art)).FontSize(11).
					TextColor(rowMuted(c, chosen)).SingleLine()
			})
		}
		if cols.action {
			ui.Box(c).Width(colAction).Shrink(0).Justify(ui.End).Children(func() {
				ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
					// Purge is a hover action: it is destructive, so it should
					// not sit in the table shouting at every row.
					if row.Hovered() || chosen {
						if b := components.Button(c, "Purge", components.ButtonGhost); b.Clicked() {
							a.purgeOne(art.Path)
						}
					}
					more := components.IconButton(c, lucide.Ellipsis(), "Row actions", components.IconButtonPlain)
					more.Menu(func(m *ui.Menu) {
						if m.Item("Reveal in file manager").Chosen() {
							a.Reveal(art.Path)
						}
						if m.Item("Choose this folder only").Chosen() {
							a.selected = map[string]bool{art.Path: true}
						}
						m.Separator()
						if m.Item("Purge now…").Chosen() {
							a.purgeOne(art.Path)
						}
					})
				})
			})
		}
	})
}

// rowMuted is the secondary ink for a row. On a selected row the wash lifts
// the background toward the accent, where muted ink no longer clears 4.5:1,
// so the row draws those cells in full-contrast ink instead.
func rowMuted(c *ui.Context, chosen bool) ui.Color {
	t := c.Theme()
	if chosen {
		return t.Text
	}
	return t.TextMuted
}

// checkbox is the row's selection box. It is drawn rather than built from
// components.Checkbox because the row owns the click: a nested control would
// take the click first and the row would only respond outside the box.
func (a *App) checkbox(c *ui.Context, on bool) {
	t := c.Theme()
	box := ui.Box(c).Size(16, 16).Radius(4).Shrink(0).PassThrough()
	if on {
		box.Background(t.Accent)
	} else {
		box.Background(t.Surface).Border(1, t.Border.Mix(t.Text, 0.25))
	}
	box.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if !on {
			return
		}
		check := (&ui.Path{}).
			MoveTo(r.X+4, r.Y+8).
			LineTo(r.X+7, r.Y+11).
			LineTo(r.X+12, r.Y+5)
		p.StrokePath(check, 2, t.AccentText)
	})
}

// typeChip names the artifact's ecosystem in a bordered chip whose dot is the
// treemap colour, so the list and the map use the same key.
func (a *App) typeChip(c *ui.Context, art scan.Artifact) {
	t := c.Theme()
	dormant := scan.Dormant(art, timeNow())
	label := scan.TypeLabel(art)
	bg, fg, border := t.Surface, t.Text, t.Border
	if dormant {
		// A dormant artifact is the one worth removing, so it gets the danger
		// tint and its own chip. The word goes in a second chip rather than
		// into the ecosystem's label: "Node modules · dormant" truncates to
		// "Node modules · dorm…" in a narrow column, which is worse than two
		// short ones.
		bg, fg, border = t.Danger.Alpha(0.10), t.Danger.Mix(t.Text, 0.15), t.Danger.Alpha(0.30)
	}
	ui.Row(c).AlignSelf(ui.Start).AlignItems(ui.Center).Gap(t.Space(1)).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(t.Space(1.5)).Padding(t.Space(0.5), t.Space(2)).
			Radius(6).Background(bg).Border(1, border).Shrink(0).Children(func() {
			ui.Box(c).Size(6, 6).Radius(3).Shrink(0).Background(KindColor(t.Dark, art.Kind))
			components.Text(c, label).FontSize(11).FontWeight(500).TextColor(fg).
				SingleLine()
		})
		if dormant {
			ui.Row(c).AlignItems(ui.Center).Padding(t.Space(0.5), t.Space(1.5)).Radius(6).
				Background(t.Danger.Alpha(0.14)).Shrink(0).Children(func() {
				components.Text(c, "dormant").FontSize(11).FontWeight(500).
					TextColor(t.Danger.Mix(t.Text, 0.2)).SingleLine()
			})
		}
	})
}

// folderIcon picks the glyph that names the artifact's shape: a zip for build
// output, a stack for packages, a cache box for caches.
func folderIcon(art scan.Artifact) *ui.SVG {
	switch {
	case art.Rule.Cache:
		return lucide.Database()
	case art.Name == "node_modules" || art.Name == ".venv" || art.Name == "venv" || art.Name == "Pods":
		return lucide.Package()
	default:
		return lucide.FolderArchive()
	}
}

// pageFootNote is the closing line of an artifacts page: the safety
// statement. It sits in the window's footer bar, beside the action, so it is
// visible without scrolling to the end of a long list.
//
// footNote and reclaimLabel live in view.go, with the footer that draws them.
// footNote says what the selection is, or that nothing is chosen yet, in one
// line. It is where the app's safety claim belongs: beside the action.
func (a *App) footNote(chosen int, total int64) string {
	if chosen == 0 {
		return "All artifacts are safe to delete — project source code stays intact"
	}
	if note := a.selectionNote(a.chosenArtifacts()); note != "" {
		return note
	}
	return "Source code stays intact; the next build takes longer"
}

// reclaimLabel is the footer button's text: what it will free, or what it
// needs first.
func (a *App) reclaimLabel(chosen int, total int64) string {
	if chosen == 0 {
		return "Reclaim"
	}
	return "Reclaim " + scan.HumanBytes(total)
}

// purgeOne asks to remove a single artifact by choosing only it and opening
// the confirmation, so the row action and the footer action share one path.
func (a *App) purgeOne(path string) {
	a.selected = map[string]bool{path: true}
	a.confirm = true
}
