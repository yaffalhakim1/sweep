//go:build windows

package clean

import (
	"errors"
	"os"
	"syscall"
)

// errorSharingViolation is ERROR_SHARING_VIOLATION, which the syscall
// package does not name. A file another process still holds open fails
// this way, and it is worth one retry after clearing attributes.
const errorSharingViolation syscall.Errno = 32

// isWindowsPermission reports whether an error is a Windows failure that
// clearing the read-only bit can fix.
func isWindowsPermission(err error) bool {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}
	return errno == syscall.ERROR_ACCESS_DENIED || errno == errorSharingViolation
}

// isReparsePoint reports whether a path is a link, a junction or another
// reparse point.
func isReparsePoint(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return false
	}
	const fileAttributeReparsePoint = 0x400
	return data.FileAttributes&fileAttributeReparsePoint != 0
}
