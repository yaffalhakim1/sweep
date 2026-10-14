package app

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"sweep/internal/disk"
	"sweep/internal/scan"
)

// homeDir is the directory a scan looks at when the user has not chosen
// one.
func homeDir() (string, error) { return os.UserHomeDir() }

// missingRoots returns the roots that are not directories, so a scan can
// report a folder that has been moved rather than showing an empty result.
func missingRoots(roots []string) []string {
	var missing []string
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			missing = append(missing, root)
		}
	}
	return missing
}

// View is which of the two presentations of the same data is showing.
type View int

const (
	// ViewMap is the treemap: where the space went.
	ViewMap View = iota
	// ViewList is the grouped list: what to do about it.
	ViewList
)

// App is the window's state. The view builds the interface from it on the
// main thread; a background scan changes it through Window.Update.
type App struct {
	win *mygo.Window

	// found is every artifact of the last completed scan; shown is that
	// list filtered by the query. During a scan, foundSoFar and visited
	// report its progress, so the window shows it working instead of
	// waiting silently.
	found      []scan.Artifact
	shown      []scan.Artifact
	foundSoFar int

	// selected is the set of artifact paths chosen for removal, keyed by
	// path so it survives a rescan that reorders the list.
	selected map[string]bool

	query string
	// viewIdx is bound to the segmented control, which writes the chosen
	// segment into it; view() reads it as the app's own type.
	viewIdx int

	// roots is where to scan. Empty means the user's home, which the app
	// resolves at scan time rather than storing a machine-specific path.
	roots []string

	// scanning is true while a walk is in flight, so the view can say so;
	// visited counts the directories read so far.
	scanning bool
	visited  int
	lastErr  error
	lastScan time.Time
	// status reports the last action's outcome, which is where the app
	// confirms a removal or explains why one failed.
	status string

	// removing is true while a removal is in flight, and removed/total
	// count the folders it has dealt with, so the window shows progress
	// instead of appearing hung while gigabytes are deleted.
	removing bool
	removed  int
	total    int

	// theme is the appearance the user asked for, which the button
	// switches directly. It starts from the desktop's appearance so the
	// first switch goes to the other one rather than to the same.
	theme bool
	// lastDark is what the desktop was showing when the app last resolved
	// the appearance, so a switch made before the first frame still starts
	// from the system's setting.
	lastDark bool
	// themeInit guards that first read, so it happens once.
	themeInit bool

	// mapPaintX and mapPaintY are the map's top-left in painted
	// coordinates, written by its draw callback. Hit testing needs them:
	// PointerPosition is relative to the element's box, while the tiles
	// are laid out in painted coordinates, and the two differ by exactly
	// this offset.
	mapPaintX float32
	mapPaintY float32

	// volume is the disk the scan is looking at, for the capacity meter.
	volume   disk.Volume
	haveDisk bool

	// confirm is true while the removal confirmation is open.
	confirm bool

	// cancel stops the scan in flight, so a new scan does not leave the
	// previous one walking the disk.
	cancel context.CancelFunc
	mu     sync.Mutex

	// mapBox is the treemap's element, kept so its committed geometry can
	// be read for hit testing after the draw pass.
	mapBox ui.Handle
	// mapCache is the laid-out treemap, kept so the layout runs when the
	// data or the box changes rather than on every pointer move.
	mapCache mapCache
	// list is the artifact list's scroll and identity state, which MyGo
	// keeps between frames.
	list ui.ListState
}

// New returns an app ready to be shown, with the remembered choices.
func New(s Settings) *App {
	a := &App{
		selected: map[string]bool{},
		viewIdx:  s.View,
		roots:    s.Roots,
	}
	// A remembered appearance is the user's own choice, so it wins over the
	// desktop's from the first frame.
	if s.Dark != nil {
		a.theme = *s.Dark
		a.themeInit = true
	}
	return a
}

// view is which presentation of the data is showing.
func (a *App) view() View {
	if a.viewIdx == int(ViewList) {
		return ViewList
	}
	return ViewMap
}

// SetWindow records the window the app draws into.
func (a *App) SetWindow(w *mygo.Window) { a.win = w }

// View builds the window's interface from the app's state. It is the
// window's content function, called on the main thread for every frame.
func (a *App) View(c *ui.Context) { a.viewFn(c) }

// Start scans once, and reports the volume the scan is looking at.
func (a *App) Start() {
	a.refreshVolume()
	a.StartScan()
}

// StartScan validates the roots and starts a scan. It must be called on
// the main thread, which every caller is: the check is cheap, and doing it
// here rather than inside the walk means a folder that has been moved is
// reported before any goroutine exists to race the state.
func (a *App) StartScan() {
	roots := a.roots
	if len(roots) == 0 {
		if home, err := homeDir(); err == nil {
			roots = []string{home}
		}
	}
	if missing := missingRoots(roots); len(missing) > 0 {
		a.scanning = false
		a.foundSoFar = 0
		a.lastErr = fmt.Errorf("%s no longer exists", missing[0])
		a.status = ""
		return
	}
	a.lastErr = nil
	go a.Scan()
}

// Scan walks the disk on a background goroutine and publishes what it
// finds as it goes. Starting a scan cancels any scan already running.
func (a *App) Scan() {
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.mu.Unlock()

	roots := a.roots
	if len(roots) == 0 {
		if home, err := homeDir(); err == nil {
			roots = []string{home}
		}
	}

	a.win.Update(func() {
		a.scanning = true
		a.visited = 0
		a.foundSoFar = 0
		a.lastErr = nil
		a.status = ""
	})

	// Progress is published at most five times a second. A per-directory
	// update would ask for thousands of frames a second and spend the
	// whole scan redrawing; the counters live on the scan's own goroutine
	// and only the published copies reach the window's state.
	var mu sync.Mutex
	visited, foundSoFar := 0, 0
	lastPush := time.Now()
	publish := func() {
		mu.Lock()
		due := time.Since(lastPush) > 200*time.Millisecond
		if due {
			lastPush = time.Now()
		}
		v, f := visited, foundSoFar
		mu.Unlock()
		if !due {
			return
		}
		a.win.Update(func() {
			a.visited, a.foundSoFar = v, f
		})
	}
	onFound := func(scan.Artifact) {
		mu.Lock()
		foundSoFar++
		mu.Unlock()
		publish()
	}
	onProgress := func() {
		mu.Lock()
		visited++
		mu.Unlock()
		publish()
	}

	found := scan.Walk(ctx, scan.Options{
		Roots:      roots,
		OnFound:    onFound,
		OnProgress: onProgress,
	})

	a.win.Update(func() {
		a.scanning = false
		a.lastScan = time.Now()
		a.found = found
		a.foundSoFar = 0
		a.pruneSelection()
		a.refresh()
		a.refreshVolume()
		if len(found) == 0 && a.lastErr == nil {
			a.status = "Nothing to clean in " + a.rootLabel()
		}
	})
}

// pruneSelection drops selected paths that the new scan no longer found,
// so a removal can never act on a stale path.
func (a *App) pruneSelection() {
	live := make(map[string]bool, len(a.found))
	for _, art := range a.found {
		live[art.Path] = true
	}
	for path := range a.selected {
		if !live[path] {
			delete(a.selected, path)
		}
	}
}

// refresh applies the query to the last scan.
func (a *App) refresh() {
	a.shown = scan.Filter(a.found, a.query)
}

// refreshVolume reads the disk the scan is looking at. A failure is not
// reported as an error: the capacity meter is context, not the task, and
// the app works without it.
func (a *App) refreshVolume() {
	path := a.rootLabel()
	vol, ok := disk.VolumeFor(path)
	if !ok {
		a.haveDisk = false
		return
	}
	a.volume = vol
	a.haveDisk = true
}

// rootLabel is what the window calls the place being scanned.
func (a *App) rootLabel() string {
	if len(a.roots) == 1 {
		return a.roots[0]
	}
	if len(a.roots) > 1 {
		return fmt.Sprintf("%d folders", len(a.roots))
	}
	if home, err := homeDir(); err == nil {
		return home
	}
	return "this computer"
}
