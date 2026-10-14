package disk

// Volume is a mounted filesystem and how full it is.
type Volume struct {
	// Root is the path that names the volume, as "C:\" on Windows and
	// "/" on Unix.
	Root string
	// Label is what to call it in the window.
	Label string
	Total int64
	Free  int64
}

// Used is the space in use.
func (v Volume) Used() int64 { return v.Total - v.Free }

// UsedRatio is the share of the volume in use, from 0 to 1. A volume
// whose size is unknown reports 0 rather than a division by zero.
func (v Volume) UsedRatio() float64 {
	if v.Total <= 0 {
		return 0
	}
	return float64(v.Used()) / float64(v.Total)
}
