package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestTableColumnsLineUp measures the header against the cells beneath it,
// because a table whose columns drift is a table nobody can read: the header
// says "Size" over a column of dates.
//
// The two right-aligned columns are the ones that regressed, so the assertion
// is about their right edges rather than their left: a title and a value of
// different widths must still end in the same place.
func TestTableColumnsLineUp(t *testing.T) {
	a := New(Settings{})
	a.found = realWorld()
	a.minSize = 0
	a.refresh()
	a.page = PageArtifacts
	tt := ui.NewTester(a.viewFn, 1180, 800)
	tt.SetDark(false)
	tt.Frame()

	pairs := []struct {
		header, cell string
	}{
		{"Size", "14.4 GB"},
		{"Modified", "today"},
	}
	for _, p := range pairs {
		h, ok := tt.Find(p.header)
		if !ok {
			t.Fatalf("the %q header is missing", p.header)
		}
		v, ok := tt.Find(p.cell)
		if !ok {
			t.Fatalf("the %q cell is missing", p.cell)
		}
		if h.X+h.W != v.X+v.W {
			t.Errorf("%q ends at %.0f but %q ends at %.0f, so the column drifted",
				p.header, h.X+h.W, p.cell, v.X+v.W)
		}
	}

	// A left-aligned column's cell sits inside its column: the chip has its own
	// padding, so it starts a little after the title rather than exactly on it.
	h, ok := tt.Find("Ecosystem")
	if !ok {
		t.Fatal("the ecosystem header is missing")
	}
	v, ok := tt.Find("Node")
	if !ok {
		t.Fatal("an ecosystem cell is missing")
	}
	if v.X < h.X || v.X+v.W > h.X+colType {
		t.Errorf("the ecosystem cell spans %.0f..%.0f, outside its column %.0f..%.0f",
			v.X, v.X+v.W, h.X, h.X+colType)
	}
}
