//go:build windows

package disk

import (
	"strings"
	"syscall"
	"unsafe"
)

var (
	procGetLogicalDriveStrings = kernel32.NewProc("GetLogicalDriveStringsW")
	procGetVolumeInformation   = kernel32.NewProc("GetVolumeInformationW")
	procGetDriveType           = kernel32.NewProc("GetDriveTypeW")
)

// driveFixed and driveRemote are the DRIVE_* types worth offering. A
// removable or optical drive is left out: it has no build output to find,
// and reading it can block or spin up media.
const (
	driveRemovable = 2
	driveFixed     = 3
	driveRemote    = 4
)

// Volumes lists the mounted filesystems worth scanning, with the label
// each one carries and how full it is.
//
// A volume that cannot be read is left out rather than reported with zero
// capacity: a network share that is down would otherwise appear as a disk
// with nothing on it.
func Volumes() []Volume {
	needed, _, _ := procGetLogicalDriveStrings.Call(0, 0)
	if needed == 0 {
		return nil
	}
	buf := make([]uint16, needed+1)
	n, _, _ := procGetLogicalDriveStrings.Call(uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])))
	if n == 0 {
		return nil
	}

	var out []Volume
	// The result is a run of NUL-terminated roots ended by an empty one,
	// so it is split on the NULs rather than decoded as one string:
	// UTF16ToString stops at the first NUL and would return only the first
	// drive.
	for _, units := range splitUTF16(buf[:n], 0) {
		root := syscall.UTF16ToString(units)
		if root == "" {
			continue
		}
		ptr, err := syscall.UTF16PtrFromString(root)
		if err != nil {
			continue
		}
		kind, _, _ := procGetDriveType.Call(uintptr(unsafe.Pointer(ptr)))
		if kind != driveFixed && kind != driveRemote {
			continue
		}
		v, ok := VolumeFor(root)
		if !ok || v.Total <= 0 {
			continue
		}
		v.Label = volumeName(root)
		out = append(out, v)
	}
	return out
}

// splitUTF16 cuts a NUL-separated run of UTF-16 strings into its parts,
// keeping the terminator on each so each part decodes as one string.
func splitUTF16(buf []uint16, sep uint16) [][]uint16 {
	var parts [][]uint16
	start := 0
	for i, u := range buf {
		if u == sep {
			if i > start {
				parts = append(parts, buf[start:i+1])
			}
			start = i + 1
		}
	}
	return parts
}

// volumeName is the name the volume carries, as "Windows" for a system
// drive, or the drive's own root when it has none. A volume with no label
// reports "New Volume" through the shell, which is not what the filesystem
// stores, so the root is the honest fallback.
func volumeName(root string) string {
	ptr, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return root
	}
	buf := make([]uint16, 261) // MAX_PATH + 1, the documented maximum
	var serial, maxLen, flags uint32
	ok, _, _ := procGetVolumeInformation.Call(
		uintptr(unsafe.Pointer(ptr)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&serial)),
		uintptr(unsafe.Pointer(&maxLen)),
		uintptr(unsafe.Pointer(&flags)),
		0, 0,
	)
	if ok == 0 {
		return root
	}
	name := strings.TrimSpace(syscall.UTF16ToString(buf))
	if name == "" {
		return root
	}
	return name
}

// Friendly is what the window calls a volume: its label with the root in
// parentheses, as "Windows (C:)", so a drive is named rather than lettered.
func (v Volume) Friendly() string {
	if v.Label == "" || v.Label == v.Root {
		return v.Root
	}
	return v.Label + " (" + strings.TrimSuffix(v.Root, `\`) + ")"
}
