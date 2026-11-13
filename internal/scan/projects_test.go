package scan

import (
	"path/filepath"
	"testing"
)

// TestGroupByProjectMergesOneProject is the map's premise: a Next.js app's
// .next and node_modules are one decision, so they are one tile.
func TestGroupByProjectMergesOneProject(t *testing.T) {
	arts := []Artifact{
		{Path: `/w/app/.next`, Name: ".next", Kind: Node, Bytes: 300},
		{Path: `/w/app/node_modules`, Name: "node_modules", Kind: Node, Bytes: 700},
		{Path: `/w/other/target`, Name: "target", Kind: Rust, Bytes: 500},
	}
	projects := GroupByProject(arts)
	if len(projects) != 2 {
		t.Fatalf("got %d projects, want 2", len(projects))
	}
	// Largest project first.
	if projects[0].Name != "app" {
		t.Errorf("first project is %q, want app", projects[0].Name)
	}
	if projects[0].Bytes != 1000 {
		t.Errorf("app totals %d, want 1000", projects[0].Bytes)
	}
	if len(projects[0].Artifacts) != 2 {
		t.Errorf("app holds %d artifacts, want 2", len(projects[0].Artifacts))
	}
	// Each project's artifacts are largest first, which is the order a tile's
	// summary reads them in.
	if projects[0].Artifacts[0].Name != "node_modules" {
		t.Errorf("the project's artifacts are not largest first: %v", projects[0].Artifacts)
	}
}

// TestProjectSummaryReadsLikeASentence checks the line under a tile's name: the
// build system, then the folders, in a form a person would write. The folders
// are listed largest first, because that is the order the tile's own size is
// made of.
func TestProjectSummaryReadsLikeASentence(t *testing.T) {
	cases := []struct {
		arts []Artifact
		want string
	}{
		{
			arts: []Artifact{{Path: "/w/app/target", Name: "target", Kind: Rust, Bytes: 10}},
			want: "Rust · target",
		},
		{
			arts: []Artifact{
				{Path: "/w/app/node_modules", Name: "node_modules", Kind: Node, Bytes: 70},
				{Path: "/w/app/.next", Name: ".next", Kind: Node, Bytes: 30},
			},
			want: "Node · node_modules & .next",
		},
		{
			arts: []Artifact{
				{Path: "/w/app/a", Name: "a", Kind: Node, Bytes: 40},
				{Path: "/w/app/b", Name: "b", Kind: Node, Bytes: 30},
				{Path: "/w/app/c", Name: "c", Kind: Node, Bytes: 20},
				{Path: "/w/app/d", Name: "d", Kind: Node, Bytes: 10},
			},
			want: "Node · a, b & 2 more",
		},
	}
	for _, c := range cases {
		p := GroupByProject(c.arts)[0]
		if got := p.Summary(); got != c.want {
			t.Errorf("Summary() = %q, want %q", got, c.want)
		}
	}
}

// TestProjectCostlyIsTheCaveat checks the one warning a size cannot carry.
func TestProjectCostlyIsTheCaveat(t *testing.T) {
	costly := GroupByProject([]Artifact{
		{Path: "/w/a/target", Name: "target", Kind: Rust, Bytes: 10, Rule: Rule{Costly: true}},
	})
	if !costly[0].Costly() {
		t.Error("a Cargo target was not called costly")
	}
	cheap := GroupByProject([]Artifact{
		{Path: "/w/a/.next", Name: ".next", Kind: Node, Bytes: 10},
	})
	if cheap[0].Costly() {
		t.Error("a .next cache was called costly")
	}
}

// TestProjectPathsIsWhatASelectionHolds checks the map's main action: choosing a
// project chooses exactly its artifacts, no more.
func TestProjectPathsIsWhatASelectionHolds(t *testing.T) {
	p := GroupByProject([]Artifact{
		{Path: "/w/a/.next", Name: ".next", Bytes: 1},
		{Path: "/w/a/node_modules", Name: "node_modules", Bytes: 2},
	})[0]
	paths := p.Paths()
	if len(paths) != 2 {
		t.Fatalf("Paths() returned %d paths", len(paths))
	}
	for _, want := range []string{"/w/a/.next", "/w/a/node_modules"} {
		found := false
		for _, got := range paths {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Errorf("Paths() is missing %q", want)
		}
	}
}

// TestFindProject checks the inspector's lookup, which decides which project the
// banner describes.
func TestFindProject(t *testing.T) {
	projects := GroupByProject([]Artifact{
		{Path: filepath.Join("/w", "a", "target"), Name: "target", Bytes: 5},
		{Path: filepath.Join("/w", "b", "target"), Name: "target", Bytes: 9},
	})
	want := filepath.Join("/w", "a")
	got, ok := FindProject(projects, want)
	if !ok || got.Name != "a" {
		t.Errorf("FindProject(%q) = %q, %v", want, got.Name, ok)
	}
	if _, ok := FindProject(projects, filepath.Join("/w", "gone")); ok {
		t.Error("FindProject found a project that does not exist")
	}
}
