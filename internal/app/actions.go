package app

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"

	"sweep/internal/clean"
	"sweep/internal/disk"
	"sweep/internal/scan"
)

// LoadVolumes reads the mounted filesystems. It runs once at startup and
// again whenever the picker opens, so a drive plugged in after launch is
// offered without restarting the app.
func (a *App) LoadVolumes() { a.setVolumes(disk.Volumes()) }

// setVolumes takes a freshly read volume list, keeping the previous one
// when the read came back empty: a transient failure must not empty the
// picker and take the current target's name with it.
func (a *App) setVolumes(volumes []disk.Volume) {
	if len(volumes) == 0 && len(a.volumes) > 0 {
		return
	}
	a.volumes = volumes
}

// ChooseRoot asks the user for a folder to scan instead of the home
// directory, which is how the app is pointed at a projects drive.
func (a *App) ChooseRoot() {
	dirs, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:    a.win,
		Title:     "Choose a folder to scan",
		Directory: true,
		Multiple:  true,
	})
	if err != nil || len(dirs) == 0 {
		return
	}
	a.SetRoots(dirs)
}

// ChooseVolume scans a whole mounted volume, which is what a user means by
// "scan the other disk".
func (a *App) ChooseVolume(root string) { a.SetRoots([]string{root}) }

// SetRoots points the scan somewhere else and starts it. The window's name
// for the target is worked out here rather than when the scan finishes, so
// the header says where it is about to look, not where it used to.
func (a *App) SetRoots(roots []string) {
	a.roots = roots
	a.selected = map[string]bool{}
	a.updateRoot()
	a.saveSettings()
	a.StartScan()
}

// ResetRoot goes back to scanning the home directory.
func (a *App) ResetRoot() { a.SetRoots(nil) }

// Toggle chooses or unchoses one artifact.
func (a *App) Toggle(path string) {
	if a.selected[path] {
		delete(a.selected, path)
		return
	}
	a.selected[path] = true
}

// selection returns the chosen artifacts and their total size.
func (a *App) selection() ([]scan.Artifact, int64) {
	chosen := make([]scan.Artifact, 0, len(a.selected))
	var total int64
	for _, art := range a.found {
		if a.selected[art.Path] {
			chosen = append(chosen, art)
			total += art.Bytes
		}
	}
	return chosen, total
}

// SelectAll chooses every artifact currently shown.
func (a *App) SelectAll() {
	for _, art := range a.shown {
		a.selected[art.Path] = true
	}
}

// ClearSelection unchoses everything.
func (a *App) ClearSelection() {
	clear(a.selected)
}

// Clean removes the chosen artifacts. It is only ever called from the
// confirmation, which is what makes a permanent removal acceptable.
func (a *App) Clean() {
	chosen, _ := a.selection()
	if len(chosen) == 0 {
		return
	}
	targets := make([]clean.Target, 0, len(chosen))
	for _, art := range chosen {
		targets = append(targets, clean.Target{Path: art.Path, Bytes: art.Bytes})
	}
	// Guarding the whole set before removing anything means a selection
	// that contains a nested pair is refused outright rather than half
	// applied.
	if err := clean.GuardSet(targets); err != nil {
		a.status = "Refused: " + err.Error()
		return
	}
	// The window shows the removal running from this instant, so the
	// action that was clicked reports back immediately rather than after
	// the first multi-gigabyte directory is gone.
	a.removing = true
	a.removed, a.total = 0, len(targets)
	a.confirm = false
	a.status = ""
	go func() {
		res := clean.RemoveProgress(targets, func(done, total int) {
			a.win.Update(func() {
				a.removed, a.total = done, total
			})
		})
		a.win.Update(func() {
			a.removing = false
			a.removed, a.total = 0, 0
			a.selected = map[string]bool{}
			switch {
			case len(res.Failures) > 0 && res.Removed == 0:
				a.status = fmt.Sprintf("Could not remove %s: %v",
					res.Failures[0].Path, res.Failures[0].Err)
			case len(res.Failures) > 0:
				a.status = fmt.Sprintf("Freed %s from %d folders; %d could not be removed",
					scan.HumanBytes(res.Freed), res.Removed, len(res.Failures))
			default:
				a.status = fmt.Sprintf("Freed %s from %d folders",
					scan.HumanBytes(res.Freed), res.Removed)
			}
			// The rescan starts from the main thread, which is where the
			// app's state belongs: the removal ran elsewhere, but
			// everything the window shows is written here.
			a.StartScan()
		})
	}()
}

// Reveal opens the chosen artifact's folder in the system's file manager,
// so the user can look before removing anything.
func (a *App) Reveal(path string) {
	mygo.Shell.ShowItemInFolder(path)
}

// revealTarget is the single artifact the list is focused on, which is
// what Reveal acts on.
func (a *App) revealTarget() (string, bool) {
	chosen, _ := a.selection()
	if len(chosen) != 1 {
		return "", false
	}
	return chosen[0].Path, true
}

// groups returns the shown artifacts grouped by kind, which is how the
// list presents them.
func (a *App) groups() []scan.Group { return scan.GroupByKind(a.shown) }

// largest returns the biggest artifacts, which is what the map shows: a
// treemap of ten thousand tiles is a texture, not a map.
func (a *App) largest(n int) []scan.Artifact {
	arts := make([]scan.Artifact, len(a.shown))
	copy(arts, a.shown)
	sort.Slice(arts, func(i, j int) bool { return arts[i].Bytes > arts[j].Bytes })
	if len(arts) > n {
		arts = arts[:n]
	}
	return arts
}

// totalShown is the size of everything the list shows.
func (a *App) totalShown() int64 { return scan.Total(a.shown) }

// summary says what the scan found, in one line.
func (a *App) summary() string {
	if a.scanning {
		if a.foundSoFar > 0 {
			return fmt.Sprintf("Scanning %s · %s folders read · %d found",
				a.rootLabel(), withCommas(a.visited), a.foundSoFar)
		}
		return fmt.Sprintf("Scanning %s · %s folders read",
			a.rootLabel(), withCommas(a.visited))
	}
	if a.lastErr != nil {
		return "Scan failed: " + a.lastErr.Error()
	}
	if len(a.shown) == 0 {
		return "Nothing to clean"
	}
	return fmt.Sprintf("%s reclaimable · %d folders",
		scan.HumanBytes(a.totalShown()), len(a.shown))
}

// age says how long ago the last scan finished, which is the window's only
// claim about freshness.
func (a *App) age() string {
	if a.lastScan.IsZero() {
		return ""
	}
	d := time.Since(a.lastScan)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
}

// withCommas writes a count with thousands separators, as a status line
// reads it.
func withCommas(n int) string {
	s := fmt.Sprint(n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}
