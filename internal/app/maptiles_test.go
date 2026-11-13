package app

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"sweep/internal/scan"
)

// TestMapPageFitsOneScreen measures the map page, because the page's whole
// premise is that the partition and the project it describes are on one screen.
// The map's height is computed from a chrome constant, and a constant that
// drifts silently clips the inspector — which is what happened.
func TestMapPageFitsOneScreen(t *testing.T) {
	a := New(Settings{})
	a.found = realWorld()
	a.minSize = 0
	a.refresh()
	a.page = PageMap
	a.haveDisk = true
	a.volume = diskFixture()

	tt := ui.NewTester(a.viewFn, 1180, 800)
	tt.SetDark(false)
	tt.Frame()

	// The inspector is the last thing on the page and the thing that gets
	// clipped, so its bottom edge is the test.
	inspector, ok := tt.Find("Open in Explorer")
	if !ok {
		t.Fatal("the inspector is missing")
	}
	// The footer bar starts at the window's height less its own height; the
	// inspector must end above it, and above the window.
	if bottom := inspector.Y + inspector.H; bottom > 800 {
		t.Errorf("the inspector ends at %.0f, past the window's 800", bottom)
	}
	// The partition's own title must be above the inspector, which is only
	// true if the map did not push it down.
	title, ok := tt.Find("Interactive space partition")
	if !ok {
		t.Fatal("the partition card is missing")
	}
	if title.Y >= inspector.Y {
		t.Errorf("the partition title is at %.0f, below the inspector at %.0f",
			title.Y, inspector.Y)
	}
	// The page must not scroll: a scrollbar on a map page means part of the
	// map, or the project it describes, is off screen.
	tt.Scroll(600, 400, 0, -400)
	tt.Frame()
	after, ok := tt.Find("Open in Explorer")
	if !ok {
		t.Fatal("the inspector disappeared after scrolling")
	}
	if after.Y != inspector.Y {
		t.Errorf("the map page scrolled: the inspector moved from %.0f to %.0f",
			inspector.Y, after.Y)
	}
	t.Logf("partition title y=%.0f, inspector y=%.0f h=%.0f bottom=%.0f",
		title.Y, inspector.Y, inspector.H, inspector.Y+inspector.H)
}

// TestMapTilesStayInsideTheBox is the regression test for the map that drew
// past its card. The partition box takes the room the card leaves, and the
// layout has to be built for that same height: the tiles used to be laid out
// for a height summed from the window, which was taller than the card, so the
// bottom row painted over the border.
func TestMapTilesStayInsideTheBox(t *testing.T) {
	a := New(Settings{})
	a.found = realWorld()
	a.minSize = 0
	a.refresh()
	a.page = PageMap
	a.haveDisk = true
	a.volume = diskFixture()

	tt := ui.NewTester(a.viewFn, 1180, 800)
	tt.Frame()

	if a.mapRoom <= 0 {
		t.Fatalf("the partition box was never measured: %v", a.mapRoom)
	}
	bottom := 0.0
	for _, tile := range a.mapCache.tiles {
		if tile.Y+tile.H > bottom {
			bottom = tile.Y + tile.H
		}
	}
	if d := bottom - float64(a.mapRoom); d > 1 || d < -1 {
		t.Errorf("the tiles were laid out to %.1f but the box is %.1f tall", bottom, a.mapRoom)
	}

	// The measurement has to settle: a map that asks for a frame every frame
	// would paint the window for as long as it is open.
	before := a.mapRoom
	tt.Frame()
	if a.mapRoom != before {
		t.Errorf("the box height moved from %.1f to %.1f on the next frame", before, a.mapRoom)
	}
	if tt.FramePending() {
		t.Error("the settled map still asks for another frame")
	}
}

// TestMapShowsSeveralProjects checks the partition actually uses the room it
// has: a map that shows four tiles in a wide card is a list with extra steps.
func TestMapShowsSeveralProjects(t *testing.T) {
	a := New(Settings{})
	a.found = realWorld()
	a.minSize = 0
	a.refresh()
	a.page = PageMap

	tt := ui.NewTester(a.viewFn, 1180, 800)
	tt.Frame()

	projects := a.projects()
	if len(projects) < 6 {
		t.Fatalf("the fixture has only %d projects", len(projects))
	}
	drawn := 0
	for _, p := range projects {
		if tt.HasText(p.Name) {
			drawn++
		}
	}
	if drawn < 5 {
		t.Errorf("the map drew %d of %d projects; texts: %v", drawn, len(projects), tt.Texts())
	}
}

// TestMapTileOpensTheInspector is the map's core interaction: clicking a tile
// names the project in the banner, with its path and its actions.
func TestMapTileOpensTheInspector(t *testing.T) {
	a := New(Settings{})
	a.found = realWorld()
	a.minSize = 0
	a.refresh()
	a.page = PageMap

	tt := ui.NewTester(a.viewFn, 1180, 800)
	tt.Frame()

	// The first tile is the biggest project, which is the one the inspector
	// shows before any click.
	projects := a.projects()
	want := projects[0]
	if a.mapSelected != "" {
		t.Fatalf("the map started with %q chosen", a.mapSelected)
	}
	if err := tt.Click(want.Name); err != nil {
		t.Fatalf("the biggest tile is not clickable: %v", err)
	}
	if a.mapSelected != want.Dir {
		t.Errorf("clicking %q chose %q", want.Dir, a.mapSelected)
	}
	if !tt.HasText(want.Dir) {
		t.Errorf("the inspector does not show the project's path; texts: %v", tt.Texts())
	}
	if !tt.HasText("Reclaim " + scan.HumanBytes(want.Bytes)) {
		t.Errorf("the inspector has no reclaim action for %s", want.Name)
	}
}

// TestMapZoomShowsMoreProjects checks the zoom control, which is the map's only
// way to see past the largest projects.
func TestMapZoomShowsMoreProjects(t *testing.T) {
	a := New(Settings{})
	a.found = realWorld()
	a.minSize = 0
	a.refresh()
	a.page = PageMap

	if a.mapZoom != 0 {
		t.Fatalf("the map started at zoom %d", a.mapZoom)
	}
	tt := ui.NewTester(a.viewFn, 1180, 800)
	tt.Frame()
	if err := tt.Click("Show more projects"); err != nil {
		t.Fatalf("no zoom-in control: %v", err)
	}
	if a.mapZoom != 1 {
		t.Errorf("zoom = %d after one press", a.mapZoom)
	}
	if err := tt.Click("Reset the map"); err != nil {
		t.Fatal(err)
	}
	if a.mapZoom != 0 {
		t.Errorf("reset left zoom at %d", a.mapZoom)
	}
}
