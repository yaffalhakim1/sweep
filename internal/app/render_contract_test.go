package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestShotUIHasTheNewControls asserts the rendered UI actually contains the
// controls the README advertises, so the screenshots cannot go stale against
// the code.
func TestShotUIHasTheNewControls(t *testing.T) {
	a := New(Settings{})
	a.found = realWorld()
	a.minSize = 0
	a.refresh()
	a.haveDisk = true
	a.volume = diskFixture()

	tt := ui.NewTester(a.viewFn, 1180, 800)
	tt.SetDark(false)
	tt.Frame()

	for _, want := range []string{
		"Switch to the dark appearance", // the appearance toggle
		"Reclaim",                       // the action
		"Clean suggested",               // the page's main action
		"Build artifacts",
		"Total reclaimable",
		"Active drive",
		"Filter",
	} {
		if !tt.HasText(want) {
			t.Errorf("the rendered UI has no %q; texts: %v", want, tt.Texts())
		}
	}

	// Dark: the toggle must offer the way back.
	tt.SetDark(true)
	a.theme, a.themeInit = true, true
	tt.Frame()
	if !tt.HasText("Switch to the light appearance") {
		t.Errorf("the dark render does not offer the light appearance; texts: %v", tt.Texts())
	}
}

// TestHeaderShowsTheScanTargetInTheRender checks the rendered page names the
// disk being scanned, so a screenshot cannot silently lose it.
func TestHeaderShowsTheScanTargetInTheRender(t *testing.T) {
	a := New(Settings{})
	a.LoadVolumes()
	a.found = realWorld()
	a.refresh()
	a.haveDisk = true
	a.volume = diskFixture()
	a.updateRoot()

	tt := ui.NewTester(a.viewFn, 1180, 800)
	tt.SetDark(false)
	tt.Frame()

	if !tt.HasText(a.diskLabel()) {
		t.Errorf("the page does not show %q; texts: %v", a.diskLabel(), tt.Texts())
	}
	if !tt.HasText(a.rootLabel()) {
		t.Errorf("the breadcrumb does not show %q; texts: %v", a.rootLabel(), tt.Texts())
	}
}
