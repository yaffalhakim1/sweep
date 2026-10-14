package clean

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Target is a directory the user asked to remove, with the size the scan
// measured for it so the app can report what was freed without walking
// the tree a second time.
type Target struct {
	Path  string
	Bytes int64
}

// Failure is one directory that could not be removed, with the reason.
type Failure struct {
	Path string
	Err  error
}

// Result is what a removal did.
type Result struct {
	Removed  int
	Freed    int64
	Failures []Failure
}

// ErrUnsafe reports a path the cleaner refuses to touch, whatever the
// caller asked. It is the last line of defence, not the first: the
// scanner only ever yields real artifacts, and the confirmation names
// them before anything is removed.
type ErrUnsafe struct{ Path, Reason string }

func (e ErrUnsafe) Error() string {
	return fmt.Sprintf("refusing to remove %s: %s", e.Path, e.Reason)
}

// Remove deletes every target permanently and reports what happened. It
// keeps going after a failure, so one locked directory does not stop the
// run, and it never touches a path the guards reject.
//
// Removal is permanent by design. Moving an artifact to the Recycle Bin
// does not free the space until the bin is emptied, which is the whole
// point of the app.
func Remove(targets []Target) Result {
	return RemoveProgress(targets, nil)
}

// RemoveProgress is Remove, calling progress after each target is dealt
// with. Deleting several gigabytes takes long enough that a window which
// says nothing looks hung, and the count is what a progress line reports.
//
// progress runs on the calling goroutine, so a caller that shows its
// result in a window must publish it thread-safely.
func RemoveProgress(targets []Target, progress func(done, total int)) Result {
	var res Result
	for i, t := range targets {
		if err := Guard(t.Path); err != nil {
			res.Failures = append(res.Failures, Failure{Path: t.Path, Err: err})
			if progress != nil {
				progress(i+1, len(targets))
			}
			continue
		}
		if err := removeAll(t.Path); err != nil {
			res.Failures = append(res.Failures, Failure{Path: t.Path, Err: err})
			if progress != nil {
				progress(i+1, len(targets))
			}
			continue
		}
		res.Removed++
		res.Freed += t.Bytes
		if progress != nil {
			progress(i+1, len(targets))
		}
	}
	return res
}

// Guard reports why a path must not be removed, or nil when it is a
// legitimate artifact directory.
//
// The checks are deliberately redundant with the scanner. The scanner
// decides what to offer; the guard decides what may actually be deleted,
// and it is the one that has to hold even if the scanner is changed.
func Guard(path string) error {
	if strings.TrimSpace(path) == "" {
		return ErrUnsafe{path, "empty path"}
	}
	if !filepath.IsAbs(path) {
		return ErrUnsafe{path, "not an absolute path"}
	}
	clean := filepath.Clean(path)

	// A volume root, or anything shallower than two levels, is a whole
	// user profile or a whole drive rather than a build directory.
	if isRoot(clean) {
		return ErrUnsafe{path, "a volume root"}
	}
	if depth(clean) < 2 {
		return ErrUnsafe{path, "too close to the volume root"}
	}
	if home, err := os.UserHomeDir(); err == nil {
		home = filepath.Clean(home)
		if strings.EqualFold(clean, home) {
			return ErrUnsafe{path, "the user's home directory"}
		}
		// An ancestor of the home directory, such as C:\Users, would take
		// the whole profile with it.
		if contains(clean, home) {
			return ErrUnsafe{path, "a parent of the user's home directory"}
		}
	}
	// A reparse point is a link: removing it removes the link, and the
	// app's model of "this directory holds these bytes" does not hold.
	if isReparsePoint(clean) {
		return ErrUnsafe{path, "a link or junction"}
	}
	return nil
}

// GuardSet checks the whole selection, including that no target contains
// another: removing a parent would silently take a child with it, and the
// reported total would count those bytes twice.
func GuardSet(targets []Target) error {
	for i, t := range targets {
		if err := Guard(t.Path); err != nil {
			return err
		}
		for j, other := range targets {
			if i == j {
				continue
			}
			if contains(filepath.Clean(t.Path), filepath.Clean(other.Path)) {
				return ErrUnsafe{other.Path, "inside " + t.Path + ", which is also selected"}
			}
		}
	}
	return nil
}

// isRoot reports whether a path is a volume root, as "C:\" or "/".
func isRoot(path string) bool {
	vol := filepath.VolumeName(path)
	rest := strings.TrimPrefix(path, vol)
	return rest == "" || rest == `\` || rest == "/"
}

// depth counts the path's components below its volume root.
func depth(path string) int {
	vol := filepath.VolumeName(path)
	rest := strings.Trim(strings.TrimPrefix(path, vol), `\/`)
	if rest == "" {
		return 0
	}
	return len(strings.FieldsFunc(rest, func(r rune) bool { return r == '\\' || r == '/' }))
}

// contains reports whether parent is an ancestor of child.
func contains(parent, child string) bool {
	if strings.EqualFold(parent, child) {
		return false
	}
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// removeAll deletes a directory tree, clearing the attributes that stop a
// plain removal. Cargo, NuGet and MSBuild leave read-only files behind,
// and Windows refuses to delete those without the attribute cleared.
func removeAll(path string) error {
	err := os.RemoveAll(path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrPermission) && !isWindowsPermission(err) {
		return err
	}
	clearReadOnly(path)
	return os.RemoveAll(path)
}

// clearReadOnly clears the read-only bit throughout a tree, best effort:
// what cannot be reached is reported by the removal that follows, not
// here.
func clearReadOnly(root string) {
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Mode().Perm()&0o200 == 0 {
			_ = os.Chmod(path, info.Mode().Perm()|0o600)
		}
		return nil
	})
}
