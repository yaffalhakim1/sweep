# Sweep — implementation plan

A desktop app that finds regenerable build artifacts across a disk and removes
them, so a full drive goes back to green.

Built with MyGo native UI (`github.com/egoist/mygo` v0.4.8), pure Go, no cgo,
no webview. Same stack and conventions as `../ports`.

## Problem

Rust/GPUI development fills a disk with `target/` directories — often many GB
each, one per project. Cargo rebuilds them on demand, so they are pure waste.
The same is true of `node_modules`, `.next`, Python caches, Gradle and Android
build trees.

Existing answers are bad for this job:

- `cargo clean` is per-project; the user has to remember where each project is.
- Deleting by hand in Explorer is slow and has no sense of total size.
- Disk visualizers (DiskTree, DaisyDisk) show *where* bytes are, not *what is
  safe to remove*.

Mole's design is the right reference: conservative, category-based cleanup with
an explicit safety story. DiskTree's treemap is the right reference for the
"where did my space go" question. Sweep takes both: a treemap scoped to
*reclaimable artifacts only*, and a list that is directly actionable.

## Architecture

```
sweep/
  main.go                 window wiring
  internal/disk/          volume capacity and free space
  internal/scan/          artifact rules, the walker, directory sizing
  internal/clean/         permanent deletion, with the path guards
  internal/treemap/       squarified treemap layout (pure geometry)
  internal/app/           state, theme, palette, views
```

Layering is strict: `internal/app` (UI) never touches the filesystem itself;
it calls `scan` and `clean`, and every package below it is pure Go with no `ui`
import, so all of it is unit-testable without a window. `internal/treemap`
depends on nothing at all.

### Scan model

A **rule** matches a directory leaf name and optionally verifies the match.

```go
type Rule struct {
    Name   string            // leaf name, or a prefix when Prefix
    Prefix bool
    Kind   Kind              // Rust, Node, Python, JVM, Xcode, CMake, DotNet
    Verify func(string) bool // optional; confirms a generic name
    Costly bool              // rebuilding takes real time
    Note   string            // what the list says the directory holds
}
```

When a directory matches, its whole subtree is sized and recorded as one
**artifact**, and the walker does not descend into it. Nested artifacts inside
an artifact are therefore not double-counted.

Rules ship conservative by default. Generic names are matched only with a
`Verify` that confirms a real build system, because a wrong match here deletes
user data:

| Kind | Names | Verified by |
|---|---|---|
| Rust | `target` | `CACHEDIR.TAG` (which Cargo writes) |
| Node | `node_modules`, `.next`, `.nuxt`, `.svelte-kit`, `.turbo`, `.parcel-cache`, `.vite` | — |
| Python | `__pycache__`, `.pytest_cache`, `.mypy_cache`, `.ruff_cache` | — |
| Python | `.venv`, `venv` | `pyvenv.cfg` |
| JVM | `.gradle` | — |
| JVM | `build` | sibling `build.gradle[.kts]` |
| .NET | `obj` | sibling `*.csproj` |
| Xcode | `DerivedData` | — |
| Xcode | `Pods` | sibling `Podfile` |
| CMake | `cmake-build-*`, `.cxx` | — |

`.cxx` is the Android NDK tree. It is recorded whole rather than descending
into it, because the NDK writes a `CMakeFiles` directory per ABI and reporting
each one buries the real space users — that was a defect found only by running
the scanner against the real disk.

### Deletion safety

1. Only paths discovered by the scanner can be deleted. The cleaner re-checks
   that a path is absolute, is not a volume root, is at least two levels deep,
   is not the user's home or an ancestor of it, and is not a link.
2. A selection containing a parent and something inside it is refused
   outright, because removing the parent would take the child with it and the
   reported total would count those bytes twice.
3. Deletion is **permanent** by default — moving to the Recycle Bin does not
   free space until the bin is emptied, which defeats the purpose. The
   confirmation names the count and the total size, and the list shows the
   exact paths.
4. Read-only attributes are cleared when a removal fails (Cargo and NuGet
   leave these behind). Long paths need no special handling: Go's `os` package
   applies the `\\?\` prefix itself.
5. Failures are collected per path and reported; one locked directory does not
   abort the run.

### Concurrency

Sizing a disk is I/O bound. The walker runs on a bounded worker pool
(`runtime.NumCPU()` workers) fed by an **unbounded** work queue. Unbounded is
deliberate: a bounded channel deadlocks when every worker blocks sending newly
discovered directories and none is left to drain them, which is exactly what a
wide tree produces.

Progress is published to the window at most five times a second. A
per-directory update would ask for thousands of frames a second and spend the
whole scan redrawing.

All app state is written on the main thread through `Window.Update`, so there
is no shared mutable state to race. The scan's own counters live on its
goroutine behind a mutex, and only published copies reach the window. A scan is
cancellable by `context.Context`; starting a new scan cancels the previous one.

Junctions and symlinks are never followed (a loop guard, not a nicety: Windows
ships junctions such as `C:\Users\All Users` that otherwise recurse forever).

## UI

One window, one scrolling column of cards, no navigation. Apple's *Simplicity*
principle: show the common path first.

```
┌──────────────────────────────────────────────────────────────┐
│ ⛁ Sweep  C:\Users\me            [Rescan] [◐]                 │  header
├──────────────────────────────────────────────────────────────┤
│  C:\   28 GB free of 476 GB                                  │  capacity
│  ████████████████████████████████████████░░░░░░░░░░░░░░░░░░  │  meter
│  94% full — critically full, cleaning now matters            │
├──────────────────────────────────────────────────────────────┤
│  30.1 GB reclaimable · 44 folders        [filter] [Map|List] │  toolbar
├──────────────────────────────────────────────────────────────┤
│  ┌────────────────────────────────────────────────────────┐  │
│  │  squarified treemap: area = bytes, colour = kind        │  │  map
│  │  hover → tooltip, click → choose                        │  │
│  └────────────────────────────────────────────────────────┘  │
│  ■ Rust ■ Node ■ Python ■ CMake ■ JVM      Click a tile…     │  legend
├──────────────────────────────────────────────────────────────┤
│  2 folders · 21.2 GB · Rebuilding these will take a while    │  footer
│                              [Reveal]  [Remove…]             │
└──────────────────────────────────────────────────────────────┘
```

- **Header** — the app's identity, the folder being scanned, Rescan, and the
  appearance menu (System / Light / Dark), one click each, as in `ports`.
- **Capacity** — `ui.Meter` per volume with `MeterLevels` so it turns amber at
  80% and red at 95%, plus a sentence naming the state so colour is never the
  only signal. This is the "storage going red" signal, in the app.
- **Toolbar** — total reclaimable, folder count, a filter, and a segmented
  control between the two views.
- **Map** — squarified treemap (Bruls/Huizing/van Wijk), drawn with
  `Element.Draw`. Area is proportional to bytes, colour is the build system, and
  a legend names the colours. A label is drawn only where it fits and measures,
  so a 12-pixel tile is a colour rather than noise.
- **List** — the same data grouped by kind with checkboxes, exact paths, sizes,
  and how long ago each changed. This is the actionable view.
- **Footer** — selection total, the rebuild-cost warning, Reveal in the file
  manager, and Remove, which opens the confirmation.

The layout model that this depends on, learned by measurement: `.Fill()` sizes
an element against its parent's *definite* size on both axes, so inside a
`Scroll` — which has none — a filling child collapses to zero. The content
column is therefore given an explicit width from `c.Size()`, and cards use
`.FillWidth()`. A box that needs a fixed height uses `.FillWidth().Height(h)`,
never `.Fill()` after `.Height(h)`, which overrides it.

### Apple design decisions, concretely

- **Response** — the treemap tile highlights under the pointer, and the whole
  list row is the hit target for its checkbox, so choosing one never requires
  hitting a 16-pixel square.
- **Materials and hierarchy** — one background, one raised surface per card,
  hairline borders, no stacked translucency.
- **Motion** — no animated transitions. Nothing here is gesture-driven, so
  springs would be decoration; the only motion is the system's own focus ring
  and scroll.
- **Typography** — one family (the system's), three sizes, hierarchy from
  weight and size together.
- **Colour** — near-greyscale surfaces with hue reserved for meaning: the
  accent appears only on the primary action, and the only other colour anywhere
  is a treemap tile. Label ink on a tile is chosen per fill by measured
  contrast, so a new hue cannot ship an unreadable label.
- **Destructive action** — one confirmation, naming the count and the size, and
  stating that removal is permanent and that the rebuild will take longer. Not
  used anywhere else.

## Verification

- Unit tests for the rule table (positive and negative matches, especially the
  generic names that need a proof file), the treemap's invariants (area
  proportional, contained in the box, tiles cover it, no slivers), the deletion
  guards, and the walker's queue under concurrency.
- `ui.NewTester` view tests: choose a row, check the footer total and the
  rebuild warning, filter by name and by kind, bulk select, and confirm the
  nested-selection refusal.
- `Tester.Image()` renders the real view to PNGs headlessly — both appearances,
  both views, against a realistic 28-artifact disk — so the layout and the
  palette are verified without a display.
- The scanner was run against the real `~/Documents`: 30.1 GB reclaimable
  across 44 folders, including a 5.7 GB Rust `target`. That run is what found
  the two rule defects the synthetic tests missed: 58 fragmented `CMakeFiles`
  rows for one Android build tree, and a `Pods` verify that could never pass.
- `go tool mygo vet .`, `go vet ./...`, `gofmt`, and `go tool mygo build` are
  all clean, and the packaged app was launched and closed cleanly.
