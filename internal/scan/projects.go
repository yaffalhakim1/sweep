package scan

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// Project is the artifacts that share one project folder.
//
// It is the unit the space map draws, because a Next.js app's .next and
// node_modules are two rows of one decision: "is this project still alive". A
// treemap of individual folders answers "which folder is biggest", which is a
// less useful question and produces a map with three tiles all called
// node_modules.
type Project struct {
	// Dir is the folder that holds the artifacts, and Name is its leaf.
	Dir  string
	Name string
	// Artifacts are the project's artifacts, largest first.
	Artifacts []Artifact
	Bytes     int64
	Files     int
}

// GroupByProject groups artifacts by the folder that holds them, largest
// project first, with each project's artifacts largest first.
func GroupByProject(artifacts []Artifact) []Project {
	byDir := map[string]*Project{}
	var order []string
	for _, art := range artifacts {
		dir := filepath.Dir(art.Path)
		p, ok := byDir[dir]
		if !ok {
			p = &Project{Dir: dir, Name: projectName(dir)}
			byDir[dir] = p
			order = append(order, dir)
		}
		p.Artifacts = append(p.Artifacts, art)
		p.Bytes += art.Bytes
		p.Files += art.Files
	}
	out := make([]Project, 0, len(order))
	for _, dir := range order {
		p := byDir[dir]
		sort.Slice(p.Artifacts, func(i, j int) bool {
			if p.Artifacts[i].Bytes != p.Artifacts[j].Bytes {
				return p.Artifacts[i].Bytes > p.Artifacts[j].Bytes
			}
			return p.Artifacts[i].Path < p.Artifacts[j].Path
		})
		out = append(out, *p)
	}
	// A tool's own store is one tile, named for what it is: the per-version
	// folders inside nvm are not projects, and a map that listed them as
	// "v22.21.1" and "v22.23.2" would be describing the tool's layout rather
	// than the user's work.
	out = mergeShared(out)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Bytes != out[j].Bytes {
			return out[i].Bytes > out[j].Bytes
		}
		return out[i].Dir < out[j].Dir
	})
	return out
}

// sharedProject is the name of the tile that holds every tool store.
const sharedProject = "Package managers"

// mergeShared folds every shared project into one, so the map has a single tile
// for "the package managers' own folders" instead of one per version.
func mergeShared(projects []Project) []Project {
	var out []Project
	var shared *Project
	for _, p := range projects {
		if !p.Shared() {
			out = append(out, p)
			continue
		}
		if shared == nil {
			shared = &Project{Dir: sharedProject, Name: sharedProject}
		}
		shared.Artifacts = append(shared.Artifacts, p.Artifacts...)
		shared.Bytes += p.Bytes
		shared.Files += p.Files
	}
	if shared != nil {
		sort.Slice(shared.Artifacts, func(i, j int) bool {
			if shared.Artifacts[i].Bytes != shared.Artifacts[j].Bytes {
				return shared.Artifacts[i].Bytes > shared.Artifacts[j].Bytes
			}
			return shared.Artifacts[i].Path < shared.Artifacts[j].Path
		})
		out = append(out, *shared)
	}
	return out
}

// projectName is the project folder's leaf name, falling back to the whole
// path for a directory that has no leaf of its own.
func projectName(dir string) string {
	name := filepath.Base(dir)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return dir
	}
	return name
}

// sharedMarkers are path segments that hold a tool's own installation or cache
// rather than a project: nvm's per-version folders, npm's global root, a
// package manager's store. A node_modules under one of these belongs to the
// tool, not to a project, and calling the folder "v22.21.1" would be a lie.
var sharedMarkers = []string{
	"nvm",
	"npm-cache",
	"pnpm-store",
	"yarn",
	"node_global",
	"AppData",
	"Library",
	"ProgramData",
}

// Shared reports whether a project is a tool's own install rather than a
// project on the disk. The map groups these under one tile, because "which
// project" has no answer for a package manager's global store.
func (p Project) Shared() bool {
	parts := strings.Split(filepath.ToSlash(p.Dir), "/")
	for _, part := range parts {
		for _, marker := range sharedMarkers {
			if strings.EqualFold(part, marker) {
				return true
			}
		}
	}
	return false
}

// Kind is the project's dominant build system: the kind of its largest
// artifact. It is what colours the project's tile.
func (p Project) Kind() Kind {
	if len(p.Artifacts) == 0 {
		return Other
	}
	return p.Artifacts[0].Kind
}

// Costly reports whether any of the project's artifacts takes real time to
// rebuild, which is the one caveat on "safe to remove".
func (p Project) Costly() bool {
	for _, art := range p.Artifacts {
		if art.Rule.Costly {
			return true
		}
	}
	return false
}

// Paths lists the project's artifact paths, which is what a selection holds.
func (p Project) Paths() []string {
	out := make([]string, 0, len(p.Artifacts))
	for _, art := range p.Artifacts {
		out = append(out, art.Path)
	}
	return out
}

// Summary describes a project's contents in one phrase, as a tile reads it:
// "Next.js · .next & node_modules". The build system comes first because it
// is what the tile's colour already says, and the folder names follow because
// they are what a developer recognises.
func (p Project) Summary() string {
	if len(p.Artifacts) == 0 {
		return ""
	}
	return TypeLabel(p.Artifacts[0]) + " · " + folderNames(p.Artifacts)
}

// folderNames joins artifact leaf names the way a person writes a short list:
// "a", "a & b", "a, b & c", and a count once the list stops being readable.
func folderNames(artifacts []Artifact) string {
	names := make([]string, 0, len(artifacts))
	for _, art := range artifacts {
		names = append(names, artifactName(art))
	}
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " & " + names[1]
	}
	if len(names) > 3 {
		return fmt.Sprintf("%s, %s & %d more", names[0], names[1], len(names)-2)
	}
	return strings.Join(names[:len(names)-1], ", ") + " & " + names[len(names)-1]
}

// artifactName is an artifact's leaf name, falling back to the path's last
// segment when the name was not recorded.
func artifactName(a Artifact) string {
	if a.Name != "" {
		return a.Name
	}
	return projectName(a.Path)
}

// FindProject returns the project with the given directory, or false.
func FindProject(projects []Project, dir string) (Project, bool) {
	for _, p := range projects {
		if p.Dir == dir {
			return p, true
		}
	}
	return Project{}, false
}
