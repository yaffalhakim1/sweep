package scan

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"
)

// Options shapes a scan.
type Options struct {
	// Roots are the directories to search. Empty means the user's home.
	Roots []string
	// Workers bounds concurrent filesystem calls. Zero uses the CPU count.
	Workers int
	// OnFound is called from a worker goroutine for every artifact found,
	// so the window can fill as the scan runs.
	OnFound func(Artifact)
	// OnProgress is called for every directory visited, for a progress
	// count. It runs on a worker goroutine.
	OnProgress func()
}

// Walk finds every artifact under the given roots. It runs a bounded pool
// of workers, never follows a symbolic link or junction, and never
// descends into a directory it has already recorded, so a nested build
// directory is not counted twice.
//
// Walk returns the artifacts it found, sorted largest first. Cancelling
// ctx stops the walk and returns what it found so far.
func Walk(ctx context.Context, opts Options) []Artifact {
	roots := opts.Roots
	if len(roots) == 0 {
		if home, err := os.UserHomeDir(); err == nil {
			roots = []string{home}
		}
	}
	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	q := newQueue()
	w := &walker{
		ctx:      ctx,
		queue:    q,
		onFound:  opts.OnFound,
		progress: opts.OnProgress,
	}

	// Cancelling the scan releases the workers: none of them waits for
	// work that will never arrive.
	stop := context.AfterFunc(ctx, q.close)
	defer stop()

	for _, root := range roots {
		if abs, err := filepath.Abs(root); err == nil {
			q.add(abs)
		}
	}

	var pool sync.WaitGroup
	for range workers {
		pool.Add(1)
		go func() {
			defer pool.Done()
			for {
				dir, ok := q.next()
				if !ok {
					return
				}
				w.visit(dir)
				q.done()
			}
		}()
	}
	pool.Wait()

	found := w.results()
	sort.Slice(found, func(i, j int) bool {
		if found[i].Bytes != found[j].Bytes {
			return found[i].Bytes > found[j].Bytes
		}
		return found[i].Path < found[j].Path
	})
	return found
}

// queue is the walk's work list: unbounded, so a worker that discovers
// many directories never blocks on a full buffer while the other workers
// are all blocked the same way. It closes itself when the last queued
// directory has been processed.
type queue struct {
	mu          sync.Mutex
	cond        *sync.Cond
	items       []string
	outstanding int
	closed      bool
}

func newQueue() *queue {
	q := &queue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// add queues a directory and counts it as outstanding work.
func (q *queue) add(dir string) {
	q.mu.Lock()
	q.items = append(q.items, dir)
	q.outstanding++
	q.cond.Signal()
	q.mu.Unlock()
}

// done reports that one queued directory has been fully processed, which
// closes the queue once nothing is left.
func (q *queue) done() {
	q.mu.Lock()
	q.outstanding--
	if q.outstanding == 0 {
		q.closed = true
		q.cond.Broadcast()
	}
	q.mu.Unlock()
}

// next takes a directory to process, waiting while the queue is empty and
// work is still outstanding. It reports false once the walk is over.
func (q *queue) next() (string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.items) == 0 && !q.closed {
		q.cond.Wait()
	}
	if len(q.items) == 0 {
		return "", false
	}
	last := len(q.items) - 1
	dir := q.items[last]
	q.items = q.items[:last]
	return dir, true
}

// close ends the walk early. Directories already queued are still
// processed, but no new work is waited for.
func (q *queue) close() {
	q.mu.Lock()
	q.closed = true
	q.cond.Broadcast()
	q.mu.Unlock()
}

// walker holds the shared state of one scan.
type walker struct {
	ctx      context.Context
	queue    *queue
	onFound  func(Artifact)
	progress func()

	mu        sync.Mutex
	artifacts []Artifact
}

// visit reads one directory: records what it matches, and queues what to
// descend into.
func (w *walker) visit(dir string) {
	if w.cancelled() {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return // unreadable directories are ordinary, not failures
	}
	if w.progress != nil {
		w.progress()
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		child := filepath.Join(dir, name)
		if skipNames[name] {
			continue
		}
		// A link or junction is never followed: Windows ships junctions
		// such as "C:\Users\All Users" that would otherwise recurse
		// forever, and a linked build directory is not this project's to
		// remove.
		if isLink(child, entry) {
			continue
		}
		if rule, ok := Match(child, name); ok {
			w.record(child, name, rule)
			continue // its contents belong to the artifact, not to the walk
		}
		w.queue.add(child)
	}
}

// record sizes a matched directory and publishes it.
func (w *walker) record(path, name string, rule Rule) {
	bytes, files, touched := dirSize(w.ctx, path)
	if w.cancelled() {
		return
	}
	a := Artifact{
		Path:    path,
		Name:    name,
		Kind:    rule.Kind,
		Rule:    rule,
		Bytes:   bytes,
		Files:   files,
		Touched: touched,
	}
	w.mu.Lock()
	w.artifacts = append(w.artifacts, a)
	w.mu.Unlock()
	if w.onFound != nil {
		w.onFound(a)
	}
}

func (w *walker) cancelled() bool {
	select {
	case <-w.ctx.Done():
		return true
	default:
		return false
	}
}

func (w *walker) results() []Artifact {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]Artifact, len(w.artifacts))
	copy(out, w.artifacts)
	return out
}

// dirSize sums the files under a directory without following links, and
// reports the newest modification time it saw.
func dirSize(ctx context.Context, dir string) (int64, int, time.Time) {
	var (
		bytes   int64
		files   int
		touched time.Time
	)
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip what cannot be read
		}
		select {
		case <-ctx.Done():
			return filepath.SkipAll
		default:
		}
		if d.IsDir() {
			// WalkDir descends into junctions, because Go sees them as
			// directories; they are cut here instead.
			if path != dir && isLink(path, d) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil // a link or a device file holds no bytes of its own
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		bytes += info.Size()
		files++
		if info.ModTime().After(touched) {
			touched = info.ModTime()
		}
		return nil
	})
	return bytes, files, touched
}
