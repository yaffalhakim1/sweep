package scan

import (
	"sort"
	"strings"
)

// Group is the artifacts of one kind, which is how the list shows them.
type Group struct {
	Kind      Kind
	Artifacts []Artifact
	Bytes     int64
}

// GroupByKind groups artifacts by kind, keeping the kind order stable and
// each group's artifacts largest first.
func GroupByKind(artifacts []Artifact) []Group {
	byKind := map[Kind][]Artifact{}
	for _, a := range artifacts {
		byKind[a.Kind] = append(byKind[a.Kind], a)
	}
	groups := make([]Group, 0, len(Kinds))
	for _, k := range Kinds {
		list := byKind[k]
		if len(list) == 0 {
			continue
		}
		sort.Slice(list, func(i, j int) bool {
			if list[i].Bytes != list[j].Bytes {
				return list[i].Bytes > list[j].Bytes
			}
			return list[i].Path < list[j].Path
		})
		var total int64
		for _, a := range list {
			total += a.Bytes
		}
		groups = append(groups, Group{Kind: k, Artifacts: list, Bytes: total})
	}
	return groups
}

// Total sums a set of artifacts.
func Total(artifacts []Artifact) int64 {
	var n int64
	for _, a := range artifacts {
		n += a.Bytes
	}
	return n
}

// Filter returns the artifacts whose path or kind contains the query,
// case-insensitively. An empty query keeps everything.
func Filter(artifacts []Artifact, query string) []Artifact {
	query = strings.TrimSpace(query)
	if query == "" {
		return artifacts
	}
	needle := strings.ToLower(query)
	out := make([]Artifact, 0, len(artifacts))
	for _, a := range artifacts {
		if strings.Contains(strings.ToLower(a.Path), needle) ||
			strings.Contains(strings.ToLower(string(a.Kind)), needle) {
			out = append(out, a)
		}
	}
	return out
}
