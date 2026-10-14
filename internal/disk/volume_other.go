//go:build !windows

package disk

import "syscall"

// VolumeFor reports the capacity of the volume that holds path.
func VolumeFor(path string) (Volume, bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return Volume{}, false
	}
	// Bavail is the space unprivileged processes may use, which is what
	// the user can actually write to.
	total := int64(st.Blocks) * int64(st.Bsize)
	free := int64(st.Bavail) * int64(st.Bsize)
	return Volume{
		Root:  "/",
		Label: "/",
		Total: total,
		Free:  free,
	}, true
}
