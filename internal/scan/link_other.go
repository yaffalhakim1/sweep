//go:build !windows

package scan

import "io/fs"

// isLink reports whether a directory entry is a symbolic link. Unix
// filesystems have no other kind of link to guard against.
func isLink(_ string, d fs.DirEntry) bool {
	return d.Type()&fs.ModeSymlink != 0
}
