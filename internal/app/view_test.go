package app

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"sweep/internal/scan"
)

// testApp is an app with a fixed list of artifacts, so the view can be
// exercised without walking a disk.
func testApp() *App {
	a := New(Settings{})
	a.found = []scan.Artifact{
		{
			Path: "/home/dev/rust-app/target", Name: "target", Kind: scan.Rust,
			Bytes: 8 << 30, Files: 42000, Touched: time.Now().Add(-3 * time.Hour),
			Rule: scan.Rule{Costly: true, Note: "Rust build output."},
		},
		{
			Path: "/home/dev/web-app/node_modules", Name: "node_modules", Kind: scan.Node,
			Bytes: 900 << 20, Files: 31000, Touched: time.Now().Add(-48 * time.Hour),
			Rule: scan.Rule{Costly: true, Note: "Installed npm packages."},
		},
		{
			Path: "/home/dev/py-app/__pycache__", Name: "__pycache__", Kind: scan.Python,
			Bytes: 4 << 20, Files: 90, Touched: time.Now().Add(-30 * 24 * time.Hour),
		},
	}
	a.refresh()
	return a
}

// TestViewShowsWhatWasFound checks the summary line, which is the app's
// single claim about what it found.
func TestViewShowsWhatWasFound(t *testing.T) {
	a := testApp()
	tt := ui.NewTester(a.viewFn, 980, 720)
	if !tt.HasText(a.summary()) {
		t.Errorf("summary %q is not in the view; texts: %v", a.summary(), tt.Texts())
	}
	if !tt.HasText("Space map") {
		t.Errorf("the map card is missing; texts: %v", tt.Texts())
	}
}

// TestChoosingAnArtifactUpdatesTheFooter is the core interaction: choose a
// folder, and the footer says what would be freed and offers the action.
func TestChoosingAnArtifactUpdatesTheFooter(t *testing.T) {
	a := testApp()
	tt := ui.NewTester(a.viewFn, 980, 720)

	// The action is present but disabled with nothing chosen, so the
	// footer never changes shape between states.
	if !tt.HasText("Remove…") {
		t.Errorf("the remove action is missing; texts: %v", tt.Texts())
	}
	if !tt.HasText("Nothing chosen yet") {
		t.Errorf("the footer does not say the selection is empty; texts: %v", tt.Texts())
	}

	// The list is the actionable view.
	a.viewIdx = int(ViewList)
	tt.Frame()
	if err := tt.Click("/home/dev/rust-app/target"); err != nil {
		t.Fatalf("could not click the artifact row: %v", err)
	}
	if !a.selected["/home/dev/rust-app/target"] {
		t.Fatalf("clicking the row did not choose it; selection %v", a.selected)
	}
	if !tt.HasText("1 folder · 8 GB") {
		t.Errorf("footer does not total the choice; texts: %v", tt.Texts())
	}
	if !tt.HasText("Rebuilding these will take a while") {
		t.Errorf("footer does not warn about the rebuild cost; texts: %v", tt.Texts())
	}

	// Choosing it again unchoses it, which is the only way back.
	if err := tt.Click("/home/dev/rust-app/target"); err != nil {
		t.Fatal(err)
	}
	if a.selected["/home/dev/rust-app/target"] {
		t.Error("clicking a chosen row did not unchoose it")
	}
}

// TestSelectionTotals checks the arithmetic the footer shows.
func TestSelectionTotals(t *testing.T) {
	a := testApp()
	a.Toggle("/home/dev/rust-app/target")
	a.Toggle("/home/dev/web-app/node_modules")
	chosen, total := a.selection()
	if len(chosen) != 2 {
		t.Fatalf("chose %d, want 2", len(chosen))
	}
	if want := int64(8<<30 + 900<<20); total != want {
		t.Errorf("total %d, want %d", total, want)
	}
}

// TestFilterNarrowsTheList checks that the filter is applied to what the
// views show, not just to a label.
func TestFilterNarrowsTheList(t *testing.T) {
	a := testApp()
	a.query = "node"
	a.refresh()
	if len(a.shown) != 1 || a.shown[0].Name != "node_modules" {
		t.Fatalf("filter kept %v, want only node_modules", a.shown)
	}
	a.query = "Rust"
	a.refresh()
	if len(a.shown) != 1 || a.shown[0].Kind != scan.Rust {
		t.Fatalf("filtering by kind kept %v, want only Rust", a.shown)
	}
}

// TestSelectAllAndNone checks the bulk actions the list offers.
func TestSelectAllAndNone(t *testing.T) {
	a := testApp()
	a.SelectAll()
	if _, total := a.selection(); total != scan.Total(a.found) {
		t.Errorf("Select all chose %d bytes, want %d", total, scan.Total(a.found))
	}
	a.ClearSelection()
	if chosen, _ := a.selection(); len(chosen) != 0 {
		t.Errorf("None left %d chosen", len(chosen))
	}
}

// TestPruneSelectionDropsVanishedPaths checks that a rescan cannot leave a
// stale path chosen, which would let a removal act on something the scan
// no longer vouches for.
func TestPruneSelectionDropsVanishedPaths(t *testing.T) {
	a := testApp()
	a.Toggle("/home/dev/rust-app/target")
	a.Toggle("/home/dev/gone/node_modules")

	// The second artifact is no longer found by the new scan.
	a.found = a.found[:1]
	a.pruneSelection()

	if !a.selected["/home/dev/rust-app/target"] {
		t.Error("a path the scan still found was dropped")
	}
	if a.selected["/home/dev/gone/node_modules"] {
		t.Error("a path the scan no longer found stayed chosen")
	}
}

// TestRemovalIsRefusedWhenTheSelectionNests checks the end-to-end guard:
// the app refuses a selection that contains a parent and its child rather
// than removing the parent and reporting the child's bytes twice.
func TestRemovalIsRefusedWhenTheSelectionNests(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "proj", "node_modules")
	child := filepath.Join(parent, "dep", "node_modules")
	for _, p := range []string{parent, child} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	a := New(Settings{})
	a.found = []scan.Artifact{
		{Path: parent, Bytes: 100, Kind: scan.Node},
		{Path: child, Bytes: 50, Kind: scan.Node},
	}
	a.refresh()
	a.SelectAll()
	a.Clean()

	if len(a.selected) != 2 {
		t.Error("a refused selection was still cleared, which hides the refusal")
	}
	if a.status == "" {
		t.Error("a refused removal said nothing about it")
	}
	if _, err := os.Stat(parent); err != nil {
		t.Errorf("a refused removal deleted %s anyway", parent)
	}
}

// TestMissingRootIsReported checks that a chosen folder that has been
// moved says so, rather than showing an empty result that reads as a clean
// disk.
func TestMissingRootIsReported(t *testing.T) {
	a := New(Settings{Roots: []string{filepath.Join(t.TempDir(), "gone")}})
	a.StartScan() // validates the roots on the calling thread
	if a.lastErr == nil {
		t.Fatal("a missing root was not reported")
	}
	if a.scanning {
		t.Error("a failed scan left the app claiming to be scanning")
	}
	if !strings.Contains(a.summary(), "no longer exists") {
		t.Errorf("summary %q does not explain the failure", a.summary())
	}
}

// TestHoverPicksTheTileUnderThePointer is the regression test for the bug
// that made the tooltip name the wrong artifact: the tiles are laid out in
// painted coordinates while PointerPosition is relative to the element's
// box, and comparing the two directly shifted the hit test by the map's own
// position, so hovering one tile reported a neighbour above it.
//
// The box is deliberately placed at a non-zero origin, because a map at the
// window's top-left would make the two spaces coincide and hide the bug.
func TestHoverPicksTheTileUnderThePointer(t *testing.T) {
	a := New(Settings{})
	a.found = realWorld()
	a.refresh()

	arts := a.largest(mapTileLimit)
	items := mapItems(arts)
	box := ui.Rect{X: 20, Y: 347, W: 940, H: 380} // as the real layout puts it

	tiles := a.layout(items, box)
	if len(tiles) == 0 {
		t.Fatal("the map laid out no tiles")
	}

	for _, tile := range tiles {
		// A point comfortably inside the tile, in painted coordinates.
		px := float32(tile.X + tile.W/2)
		py := float32(tile.Y + tile.H/2)
		// PointerPosition reports the same point relative to the box.
		got, ok := a.hoveredTileAt(items, box, px-box.X, py-box.Y, true)
		if !ok {
			t.Errorf("no tile found at the centre of %q", tile.Key)
			continue
		}
		if got != tile.Key {
			t.Errorf("the centre of %q reported %q", tile.Key, got)
		}
	}

	// The largest artifact is the first tile, which a treemap puts in the
	// top-left corner of the box.
	biggest := arts[0]
	if tiles[0].Key != biggest.Path {
		t.Errorf("the first tile is %q, want the largest artifact %q", tiles[0].Key, biggest.Path)
	}
	// A pointer outside the box reports nothing.
	if _, ok := a.hoveredTileAt(items, box, -5, -5, true); ok {
		t.Error("a pointer outside the box found a tile")
	}
	if _, ok := a.hoveredTileAt(items, box, 10, 10, false); ok {
		t.Error("a pointer that is not over the element found a tile")
	}
}

// TestMapLayoutIsCached checks that the map does not relay itself out on
// every frame, which is what made hovering lag: a pointer move produced a
// frame, and every frame ran the whole squarify pass.
func TestMapLayoutIsCached(t *testing.T) {
	a := New(Settings{})
	a.found = realWorld()
	a.refresh()

	r := ui.Rect{X: 20, Y: 100, W: 900, H: 380}
	items := mapItems(a.largest(mapTileLimit))

	first := a.layout(items, r)
	if len(first) == 0 {
		t.Fatal("no tiles")
	}
	// The same inputs must return the very same slice, not an equal one.
	second := a.layout(items, r)
	if &first[0] != &second[0] {
		t.Error("the layout was recomputed for unchanged inputs")
	}
	// A different box must relay out.
	moved := r
	moved.W = 800
	third := a.layout(items, moved)
	if &first[0] == &third[0] {
		t.Error("the layout was reused for a different box")
	}
	// So must different data.
	shorter := items[:len(items)-1]
	fourth := a.layout(shorter, moved)
	if &third[0] == &fourth[0] {
		t.Error("the layout was reused for different data")
	}
}

// TestRemovingFooterReplacesTheActions checks the loading state: while a
// removal runs the footer reports progress instead of offering the action
// again.
func TestRemovingFooterReplacesTheActions(t *testing.T) {
	a := testApp()
	a.SelectAll()
	tt := ui.NewTester(a.viewFn, 980, 720)
	if !tt.HasText("Remove…") {
		t.Fatalf("the action is missing before a removal; texts: %v", tt.Texts())
	}

	a.removing = true
	a.removed, a.total = 2, 5
	tt.Frame()

	if !tt.HasText("Removing 2 of 5…") {
		t.Errorf("the footer does not report progress; texts: %v", tt.Texts())
	}
	if tt.HasText("Remove…") {
		t.Error("the action is still offered while a removal runs")
	}
	if !tt.HasText("Removing build output") {
		t.Errorf("the spinner is not labelled; texts: %v", tt.Texts())
	}
}

// TestThemeButtonSwitchesAppearance checks the button a user presses to go
// between light and dark, and that the choice survives.
func TestThemeButtonSwitchesAppearance(t *testing.T) {
	a := testApp()
	tt := ui.NewTester(a.viewFn, 980, 720)
	tt.SetDark(false)
	tt.Frame()
	if a.appearanceOf(nil) {
		t.Fatal("the app did not start from the desktop's light appearance")
	}

	if err := tt.Click("Switch to the dark appearance"); err != nil {
		t.Fatalf("the theme button is missing: %v", err)
	}
	if !a.appearanceOf(nil) {
		t.Error("pressing the button did not switch to dark")
	}
	if !tt.HasText("Switch to the light appearance") {
		t.Errorf("the button does not offer the way back; texts: %v", tt.Texts())
	}

	if err := tt.Click("Switch to the light appearance"); err != nil {
		t.Fatal(err)
	}
	if a.appearanceOf(nil) {
		t.Error("pressing the button again did not switch back to light")
	}
}

// TestRememberedAppearanceWinsOverTheDesktop checks that a stored choice is
// honoured from the first frame rather than being overwritten by the
// system's setting.
func TestRememberedAppearanceWinsOverTheDesktop(t *testing.T) {
	dark := true
	a := New(Settings{Dark: &dark})
	tt := ui.NewTester(a.viewFn, 980, 720)
	tt.SetDark(false) // the desktop is light
	tt.Frame()
	if !a.appearanceOf(nil) {
		t.Error("the remembered dark appearance was replaced by the desktop's")
	}
}

// TestEmptyStateOffersAWayOut checks that a scan which finds nothing says
// what it searched and offers another place to look, rather than only
// reporting emptiness.
func TestEmptyStateOffersAWayOut(t *testing.T) {
	a := New(Settings{})
	a.refresh()
	tt := ui.NewTester(a.viewFn, 980, 720)
	if !tt.HasText("Nothing to clean") {
		t.Errorf("no empty state; texts: %v", tt.Texts())
	}
	if !tt.HasText("Choose a folder…") {
		t.Errorf("the empty state offers no way out; texts: %v", tt.Texts())
	}
}

// TestRendersBothAppearances writes the real view to PNGs, which is how
// the layout is checked without a display.
func TestRendersBothAppearances(t *testing.T) {
	if testing.Short() {
		t.Skip("rendering is the slow part")
	}
	for _, tc := range []struct {
		name string
		dark bool
		view View
	}{
		{"map-light", false, ViewMap},
		{"map-dark", true, ViewMap},
		{"list-light", false, ViewList},
		{"list-dark", true, ViewList},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := New(Settings{})
			// The real shape of a developer's disk, not three tidy rows:
			// the map has to stay readable with a long tail of small
			// artifacts, which is the case that breaks a naive treemap.
			a.found = realWorld()
			a.refresh()
			a.theme = tc.dark
			a.themeInit = true
			a.viewIdx = int(tc.view)
			a.haveDisk = true
			a.volume = diskFixture()
			a.Toggle(`C:\Users\dev\Documents\Learn\rust\kaku-gui\target`)

			tt := ui.NewTester(a.viewFn, 980, 720)
			tt.SetDark(tc.dark)
			tt.Frame()
			img := tt.Image()
			if img == nil {
				t.Fatal("no image")
			}
			if err := writePNG(filepath.Join("testdata", tc.name+".png"), img); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// writePNG writes a rendered frame, so it can be looked at.
func writePNG(path string, img *image.RGBA) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
