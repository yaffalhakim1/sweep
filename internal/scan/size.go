package scan

import (
	"strconv"
	"strings"
	"time"
)

type Artifact struct {
	Path  string
	Name  string
	Kind  Kind
	Rule  Rule
	Bytes int64
	Files int
	// Touched is the newest modification time inside the directory, which
	// is what tells a project in use from one abandoned a year ago.
	Touched time.Time
}

// Human renders the size the way the list shows it.
func (a Artifact) Human() string { return HumanBytes(a.Bytes) }

// HumanBytes writes a byte count as a short size, with the unit that keeps
// it readable: 1.4 GB, 812 MB, 46 KB.
func HumanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return strconv.FormatInt(n, 10) + " B"
	}
	value := float64(n)
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	i := -1
	for value >= unit && i < len(units)-1 {
		value /= unit
		i++
	}
	// One decimal below 100, so "1.4 GB" and "812 MB" both stay short.
	if value < 100 {
		s := strconv.FormatFloat(value, 'f', 1, 64)
		s = strings.TrimSuffix(s, ".0")
		return s + " " + units[i]
	}
	return strconv.FormatFloat(value, 'f', 0, 64) + " " + units[i]
}

// skipNames are directories that never hold an artifact and can be very
// large, so descending into them only costs time. They are matched by leaf
// name at any depth.
var skipNames = map[string]bool{
	".git":                      true,
	".hg":                       true,
	".svn":                      true,
	"$RECYCLE.BIN":              true,
	"System Volume Information": true,
	"Windows":                   true,
	"Program Files":             true,
	"Program Files (x86)":       true,
	"ProgramData":               true,
}
