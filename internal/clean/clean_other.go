//go:build !windows

package clean

import "os"

// isWindowsPermission is always false where there is no such failure.
func isWindowsPermission(error) bool { return false }

// isReparsePoint reports whether a path is a symbolic link.
func isReparsePoint(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}
