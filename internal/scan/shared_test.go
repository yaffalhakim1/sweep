package scan

import (
	"path/filepath"
	"testing"
)

// TestSharedStoresAreOneTile is the bug the real disk found: nvm keeps a
// node_modules under every installed Node version, so the map drew "v22.21.1"
// and "v22.23.2" as if they were projects. A tool's own store is one tile.
func TestSharedStoresAreOneTile(t *testing.T) {
	arts := []Artifact{
		{Path: filepath.Join("/u/AppData/Local/nvm/v22.21.1/node_modules"), Name: "node_modules", Bytes: 3},
		{Path: filepath.Join("/u/AppData/Local/nvm/v22.23.2/node_modules"), Name: "node_modules", Bytes: 2},
		{Path: filepath.Join("/w/myapp/.next"), Name: ".next", Bytes: 5},
	}
	projects := GroupByProject(arts)
	if len(projects) != 2 {
		t.Fatalf("got %d projects, want the app and one shared tile: %+v", len(projects), projects)
	}
	var shared, app *Project
	for i := range projects {
		if projects[i].Name == sharedProject {
			shared = &projects[i]
		} else {
			app = &projects[i]
		}
	}
	if shared == nil {
		t.Fatal("no shared tile")
	}
	if app == nil || app.Name != "myapp" {
		t.Fatalf("the real project is %v", app)
	}
	if len(shared.Artifacts) != 2 || shared.Bytes != 5 {
		t.Errorf("the shared tile holds %d artifacts totalling %d, want 2 and 5",
			len(shared.Artifacts), shared.Bytes)
	}
}

// TestSharedDoesNotSwallowRealProjects checks the guard on the other side: a
// project that merely lives somewhere with a common name is still a project.
func TestSharedDoesNotSwallowRealProjects(t *testing.T) {
	for _, dir := range []string{
		filepath.Join("/w", "clicky-frontend"),
		filepath.Join("/w", "b2c-v3", "apps", "client"),
		filepath.Join("/Users", "me", "projects", "portfolio"),
	} {
		p := Project{Dir: dir, Name: filepath.Base(dir)}
		if p.Shared() {
			t.Errorf("%q was called a tool store", dir)
		}
	}
}
