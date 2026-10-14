package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestShotUIHasTheNewControls asserts the rendered UI actually contains
// the controls the README advertises, so the screenshots cannot go stale
// against the code.
func TestShotUIHasTheNewControls(t *testing.T) {
	a := New(Settings{})
	a.found = realWorld()
	a.refresh()
	a.haveDisk = true
	a.volume = diskFixture()

	tt := ui.NewTester(a.viewFn, 980, 720)
	tt.SetDark(false)
	tt.Frame()

	for _, want := range []string{
		"Switch to the dark appearance", // the theme toggle
		"Remove…",                       // the action
		"Rescan",
		"Space map",
		"Disk in use",
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

// TestHeaderShowsTheScanTargetInTheRender checks the rendered header names
// the disk being scanned, so a screenshot cannot silently lose it.
func TestHeaderShowsTheScanTargetInTheRender(t *testing.T) {
	a := New(Settings{})
	a.LoadVolumes()
	a.found = realWorld()
	a.refresh()
	a.haveDisk = true
	a.volume = diskFixture()
	a.updateRoot()

	tt := ui.NewTester(a.viewFn, 980, 720)
	tt.SetDark(false)
	tt.Frame()

	if !tt.HasText(a.rootLabel()) {
		t.Errorf("the header does not show %q; texts: %v", a.rootLabel(), tt.Texts())
	}
	if !tt.HasText("Scan") {
		t.Errorf("the scan-target button is missing; texts: %v", tt.Texts())
	}
}
