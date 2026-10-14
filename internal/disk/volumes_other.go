//go:build !windows

package disk

import "strings"

// Volumes lists the mounted filesystems worth scanning. Off Windows the
// root is the only volume this app knows how to name without pulling in a
// mount table, and it is the one a scan starts from anyway.
func Volumes() []Volume {
	v, ok := VolumeFor("/")
	if !ok {
		return nil
	}
	v.Label = "/"
	return []Volume{v}
}

// Friendly is what the window calls a volume.
func (v Volume) Friendly() string {
	if v.Label == "" || v.Label == v.Root {
		return v.Root
	}
	return v.Label + " (" + strings.TrimSuffix(v.Root, "/") + ")"
}
