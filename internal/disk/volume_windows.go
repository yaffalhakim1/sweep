//go:build windows

package disk

import (
	"syscall"
	"unsafe"
)

var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procGetDiskFreeSpaceEx = kernel32.NewProc("GetDiskFreeSpaceExW")
)

// VolumeFor reports the capacity of the volume that holds path. The path
// is resolved to its volume by the system, so "C:\Users\me\project" and
// "C:\" report the same volume.
func VolumeFor(path string) (Volume, bool) {
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return Volume{}, false
	}
	var freeToCaller, total, free uint64
	r, _, _ := procGetDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(ptr)),
		uintptr(unsafe.Pointer(&freeToCaller)),
		uintptr(unsafe.Pointer(&total)),
		uintptr(unsafe.Pointer(&free)),
	)
	if r == 0 {
		return Volume{}, false
	}
	root := volumeRoot(path)
	return Volume{
		Root:  root,
		Label: root,
		Total: int64(total),
		Free:  int64(free),
	}, true
}

// volumeRoot returns the root of the volume a path is on, as "C:\".
func volumeRoot(path string) string {
	if len(path) >= 2 && path[1] == ':' {
		return string(path[0]) + ":\\"
	}
	return path
}
