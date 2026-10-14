//go:build windows

package scan

import (
	"io/fs"
	"syscall"
)

// fileAttributeReparsePoint is FILE_ATTRIBUTE_REPARSE_POINT: the bit that
// marks a junction, a symlink or any other reparse point. Go reports a
// symlink through the file mode but a junction only through this bit, and
// junctions are what a Windows profile is full of.
const fileAttributeReparsePoint = 0x400

// isLink reports whether a directory entry is a link of any kind, so the
// walk never follows one.
func isLink(path string, d fs.DirEntry) bool {
	if d.Type()&fs.ModeSymlink != 0 {
		return true
	}
	info, err := d.Info()
	if err != nil {
		return false
	}
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return false
	}
	return data.FileAttributes&fileAttributeReparsePoint != 0
}
