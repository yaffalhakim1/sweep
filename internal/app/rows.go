package app

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"sweep/internal/scan"
)

// artifactRow is one line of the artifact table: an artifact, or a heading
// that introduces a build system. The table is a flat list rather than a tree
// so it can be virtualized, and the grouping is a heading row rather than a
// structure to expand.
type artifactRow struct {
	Artifact scan.Artifact
	// Header marks a row that introduces a kind rather than being an
	// artifact.
	Header bool
	Kind   scan.Kind
	Bytes  int64
	Count  int
}

// tableState is the artifact table's presentation state: how its rows are
// ordered. It is separate from the app's data because it survives a rescan.
type tableState struct {
	// sort is the column the rows are ordered by, and whether it descends.
	sort      sortKey
	ascending bool
}

// sortKey names a column the table can order by.
type sortKey int

const (
	sortSize sortKey = iota
	sortName
	sortModified
	sortType
)

// sortKeys are the sortable columns in the order they appear.
var sortKeys = []struct {
	key   sortKey
	title string
}{
	{sortName, "Name"},
	{sortType, "Ecosystem"},
	{sortSize, "Size"},
	{sortModified, "Modified"},
}

// defaultMinSize hides the long tail of small caches. A disk with hundreds of
// artifacts is a list nobody reads; 100 MB is where a folder becomes worth a
// decision.
const defaultMinSize = 100 << 20

// sizeSteps are the minimum-size filter's stops, in bytes. A slider needs a
// bounded, finite set of stops to be predictable.
var sizeSteps = []int64{
	0,
	10 << 20,
	50 << 20,
	100 << 20,
	250 << 20,
	500 << 20,
	1 << 30,
	2 << 30,
	5 << 30,
}

// sizeStepIndex returns where a byte threshold sits among the stops, so the
// slider can bind to an index rather than to a byte count.
func sizeStepIndex(n int64) int {
	best := 0
	for i, step := range sizeSteps {
		if n >= step {
			best = i
		}
	}
	return best
}

// sizeStepLabel names a stop for the rail.
func sizeStepLabel(i int) string {
	if i <= 0 {
		return "Any size"
	}
	return "> " + scan.HumanBytes(sizeSteps[i])
}

// rows returns the table's rows for the current data, ordering them by the
// chosen column. Grouping is by build system when the rows are ordered by
// kind, and flat otherwise: a group heading under a size ordering would be a
// lie, because the rows would not be sorted inside it.
func (a *App) rows() []artifactRow {
	key := a.rowCacheKey()
	if key == a.tableKey && a.tableRows != nil {
		return a.tableRows
	}
	rows := buildRows(a.shown, a.order)
	a.tableRows, a.tableKey = rows, key
	return rows
}

// rowCacheKey is everything the rows depend on, so the cache cannot serve a
// list that no longer matches the data.
func (a *App) rowCacheKey() string {
	return fmt.Sprintf("%d/%d/%v/%v", len(a.shown), a.totalShown(),
		a.order.sort, a.order.ascending)
}

// buildRows orders and groups the artifacts for the table.
func buildRows(artifacts []scan.Artifact, t tableState) []artifactRow {
	ordered := make([]scan.Artifact, len(artifacts))
	copy(ordered, artifacts)
	sort.SliceStable(ordered, func(i, j int) bool {
		return rowLess(ordered[i], ordered[j], t)
	})

	if t.sort != sortType {
		// A flat table: one row per artifact, no headings.
		rows := make([]artifactRow, 0, len(ordered))
		for _, art := range ordered {
			rows = append(rows, artifactRow{Artifact: art})
		}
		return rows
	}

	var rows []artifactRow
	for _, g := range scan.GroupByKind(ordered) {
		rows = append(rows, artifactRow{
			Header: true, Kind: g.Kind, Bytes: g.Bytes, Count: len(g.Artifacts),
		})
		for _, art := range g.Artifacts {
			rows = append(rows, artifactRow{Artifact: art})
		}
	}
	return rows
}

// rowLess orders two artifacts by the table's sort. Size and modified descend
// by default: the useful question about a disk is "what is biggest", never
// "what is smallest". Name and type ascend, because a name list is read from
// the top.
func rowLess(a, b scan.Artifact, t tableState) bool {
	switch t.sort {
	case sortName:
		an, bn := strings.ToLower(a.Name), strings.ToLower(b.Name)
		if an != bn {
			return (an < bn) == t.ascending
		}
		return a.Path < b.Path
	case sortType:
		at, bt := scan.TypeLabel(a), scan.TypeLabel(b)
		if at != bt {
			return (at < bt) == t.ascending
		}
		return a.Bytes > b.Bytes
	case sortModified:
		if !a.Touched.Equal(b.Touched) {
			return a.Touched.After(b.Touched) != t.ascending
		}
		return a.Bytes > b.Bytes
	default:
		if a.Bytes != b.Bytes {
			return (a.Bytes < b.Bytes) == t.ascending
		}
		return a.Path < b.Path
	}
}

// totalShown is the size of everything the table shows.
func (a *App) totalShown() int64 { return scan.Total(a.shown) }

// dormantCount counts the artifacts nothing has touched in months.
func (a *App) dormantCount() int {
	now := time.Now()
	n := 0
	for _, art := range a.shown {
		if scan.Dormant(art, now) {
			n++
		}
	}
	return n
}
