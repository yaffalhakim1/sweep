package main

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"sweep/internal/app"
)

// TestAppStarts checks that the window's view builds against a real app
// with nothing found yet, which is what the first frame shows.
func TestAppStarts(t *testing.T) {
	a := app.New(app.LoadSettings())
	tt := ui.NewTester(a.View, 980, 720)
	if !tt.HasText("Sweep") {
		t.Errorf("the window does not name the app; texts: %v", tt.Texts())
	}
}
