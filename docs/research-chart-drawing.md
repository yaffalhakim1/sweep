# MyGo v0.4.8 — Charts and Custom Drawing in the Native `ui` Package

Source pinned at `C:/Users/yafit/go/pkg/mod/github.com/egoist/mygo@v0.4.8`.
Every signature below is quoted verbatim from that tree; every claim is backed by a
`file:line` citation or a run of the compile/render probes described in
[Verification](#verification).

**Imports** (the only two you need):

```go
import (
	"github.com/egoist/mygo/ui"        // elements, layout, Painter, Path, Canvas
	"github.com/egoist/mygo/ui/chart"  // scales, plot, marks, paths, stacking, pie
)
```

The `ui` package is one flat package — `ui.Box`, `ui.Painter`, `ui.Path`,
`ui.Rect`, `ui.Color`, `ui.Canvas` all live in it. There is no
`ui/chart` dependency on any renderer you have to import.

---

## 1. Chart package — exact API

`ui/chart` is **headless**: it computes scales and returns *paths* and
*elements*. It owns no colors, fonts or sizes
(`ui/chart/chart.go:1-24`, `docs/ui/chart.md:1-12`).

### 1.1 Data types

```go
type Span struct{ Lo, Hi float32 }                                   // ui/chart/scale.go:12
func (s Span) Mid() float32 { return (s.Lo + s.Hi) / 2 }             // ui/chart/scale.go:15

type Tick struct {                                                   // ui/chart/scale.go:19
	At    float32
	Value float64
	Label string
}
```

A `Span` is a stretch of the plot in **fractions**: 0→1 along x from the
left edge, and 0→1 along y **from the bottom up** (`ui/chart/scale.go:8-11`).
This y-up convention is the single most important thing to internalize; it
is the opposite of screen coordinates.

There is **no `Series`, `Mark`, `Dataset` or chart-object type**. The
package's complete exported type set is `Side, Axes, Span, Tick, Linear,
Band, Curve, Slice, Segment, Stacking, Time`
(`ui/chart/chart.go:33,44`, `scale.go:12,19,31,261`, `shape.go:19,265`,
`stack.go:6,10`, `time.go:16`). Marks are `ui.Element`s you build yourself.

### 1.2 Scales

**Linear** (`ui/chart/scale.go:31-46`):

```go
type Linear struct {
	Lo, Hi float64
	// Clamp keeps values outside the domain at its ends.
	Clamp bool
	// Format labels the ticks; nil labels them with as many decimals as
	// their step needs.
	Format func(v float64) string
}

func NewLinear(lo, hi float64) Linear { return Linear{Lo: lo, Hi: hi} }   // scale.go:41
func Extent(values ...float64) (lo, hi float64)                           // scale.go:48
func (s Linear) At(v float64) float32                                     // scale.go:66
func (s Linear) Span(a, b float64) Span { return Span{s.At(a), s.At(b)} } // scale.go:79
func (s Linear) Invert(f float32) float64                                 // scale.go:82
func (s Linear) Nice(count int) Linear                                    // scale.go:92
func (s Linear) Ticks(count int) []Tick                                   // scale.go:124
```

`Extent` skips NaN and returns `0, 1` for an empty or all-NaN input
(`scale.go:48-63`). `Nice` rounds the domain out d3-style (0–87 → 0–90 for
10 ticks); `Ticks` returns round values 1, 2 or 5 × a power of ten apart
(`scale.go:92,124`). `At` returns `0.5` for a zero-width domain
(`scale.go:66-68`).

**Band** (categorical, `ui/chart/scale.go:261-329`):

```go
type Band struct {
	Labels []string
	// Inner is the room between bands, and Outer the room before the
	// first and after the last, in fractions of a band's step.
	Inner, Outer float32
	// Align places the bands in the room the outer padding leaves: 0 at
	// the start, 0.5 (as NewBand) in the middle, 1 at the end.
	Align float32
}

func NewBand(labels ...string) Band { return Band{Labels: labels, Align: 0.5} } // scale.go:272
func (b Band) Padding(inner, outer float32) Band                                // scale.go:276
func (b Band) Band(i int) Span                                                  // scale.go:291
func (b Band) Width() float32                                                   // scale.go:298
func (b Band) At(i int) float32 { return b.Band(i).Mid() }                       // scale.go:304
func (b Band) Index(f float32) (i int, ok bool)                                 // scale.go:309
func (b Band) Ticks() []Tick                                                    // scale.go:323
```

`Padding(inner, outer)` clamps inner to `[0,1]` and outer to `>= 0`
(`scale.go:276-279`). `Ticks()` puts one tick in the middle of each band
with `Value = float64(i)` and `Label` = the band's label (`scale.go:323-329`).

**Time** (`ui/chart/time.go:16-43`):

```go
type Time struct {
	From, To time.Time
	// Format labels the ticks; nil labels each with what changed since
	// a round time of the coarser unit, as d3's multi-scale format:
	// "2026", "Mar", "Mar 5", "14:00", "14:05:30".
	Format func(t time.Time) string
}

func NewTime(from, to time.Time) Time { return Time{From: from, To: to} }  // time.go:25
func (s Time) At(t time.Time) float32                                      // time.go:28
func (s Time) Span(a, b time.Time) Span { return Span{s.At(a), s.At(b)} }  // time.go:37
func (s Time) Invert(f float32) time.Time                                  // time.go:40
func (s Time) Ticks(count int) []Tick                                      // time.go:160
```

Time ticks land on calendar boundaries chosen for the span (seconds,
quarters of an hour, midnights, Sundays, month firsts, years —
`time.go:59-104`).

### 1.3 The plot and its axes

```go
type Side int                                    // ui/chart/chart.go:33
const (
	Left Side = iota                             // chart.go:35-40
	Right
	Top
	Bottom
)

type Axes struct {                               // ui/chart/chart.go:44
	Left, Right, Top, Bottom []Tick
	// Label builds the label of a tick on a side, centered on the tick;
	// nil shows its Label as text. Labels sit in boxes of no width or
	// height, so keep their text on one line with NoWrap.
	Label func(side Side, t Tick)
}

func Plot(c *ui.Context, axes Axes, build func(area ui.Element)) ui.Element  // chart.go:67
```

`Plot` builds a `ui.Grid` whose middle cell is the plot area, with a
`FitContent` track added per non-empty axis side
(`chart.go:67-105`). The `build` callback receives the **area** — an
ordinary element you style, give a background/border, attach a `Draw` to,
and add marks to. Give the plot a `Height` (or `Fill`) and a `Gap`
(`docs/ui/chart.md:64-84`).

### 1.4 Marks

```go
func Rect(c *ui.Context, x, y Span) ui.Element                       // chart.go:174
func Point(c *ui.Context, x, y, w, h float32) ui.Element             // chart.go:187
func RuleX(c *ui.Context, x, width float32) ui.Element               // chart.go:195
func RuleY(c *ui.Context, y, height float32) ui.Element              // chart.go:202
func Anchor(c *ui.Context, x, y float32, h, v ui.Align, build func()) ui.Element // chart.go:216
func Select(c *ui.Context, area ui.Element, index *int, xs []float32, describe func(i int) string) bool // chart.go:246
```

All marks are `Absolute()` boxes placed with percentages of the parent —
no layout pass is needed, so they follow the plot as it resizes
(`chart.go:174-222`). `Rect` normalizes its spans with `min`/`max`, so
`y.Span(0, v)` works even for a negative `v` (`chart.go:175-183`).

`Select` makes the area focusable, stores `selection` state in
`ui.Local(area, "chart.Select", ...)`, reads the pointer with
`area.PointerPosition()` and `area.Bounds().W`, handles Left/Right/Home/End
and Escape as `Shortcut`s, and reports a changed choice (`chart.go:246-300`).
It returns `true` when `*index` changed this frame.

### 1.5 Paths for `Draw` — lines, areas, pies

```go
func X(r ui.Rect, f float32) float32 { return r.X + f*r.W }           // ui/chart/shape.go:11
func Y(r ui.Rect, f float32) float32 { return r.Y + (1-f)*r.H }       // ui/chart/shape.go:15

type Curve int                                                        // shape.go:19
const (
	CurveLinear Curve = iota                                          // shape.go:23
	CurveMonotone                                                     // shape.go:27
	CurveStepAfter                                                    // shape.go:30
	CurveStepBefore                                                   // shape.go:32
	CurveStep                                                         // shape.go:34
)

func Line(r ui.Rect, xs, ys []float32, k Curve) *ui.Path                          // shape.go:165
func Area(r ui.Rect, xs, lo, hi []float32, k Curve) *ui.Path                      // shape.go:187
func Nearest(xs []float32, f float32) int                                         // shape.go:240

type Slice struct {                                                               // shape.go:265
	Start, End, Pad float64
}
func Pie(values []float64, pad float64) []Slice                                   // shape.go:274
func (s Slice) Mid() float64 { return (s.Start + s.End) / 2 }                     // shape.go:301
func Polar(cx, cy, r float32, a float64) (x, y float32)                           // shape.go:305
func Sector(cx, cy, inner, outer float32, s Slice) *ui.Path                       // shape.go:319
func SliceAt(slices []Slice, x, y, inner, outer float32) (int, bool)              // shape.go:378
```

`Line`/`Area` take **fractions**, not pixels — you pass `x.At(...)`/
`y.At(...)` values and they convert through `X`/`Y` against the `ui.Rect`
the `Draw` callback hands you (`shape.go:165-186`). A NaN value **breaks**
the line/area there, which is how you render missing data
(`shape.go:131-139,172-175`). Nil `lo` in `Area` means the plot's bottom
(`shape.go:190-196`). `Sector` angles are radians clockwise from twelve
o'clock, and `inner > 0` yields a donut ring rather than a wedge
(`shape.go:265-272,319-322`).

### 1.6 Stacking

```go
type Segment struct{ Lo, Hi float64 }        // ui/chart/stack.go:6
type Stacking int                            // stack.go:10
const (
	StackStandard Stacking = iota            // stack.go:14
	StackNormalized                          // stack.go:17
	StackCenter                              // stack.go:19
)
func Stack(series [][]float64, how Stacking) [][]Segment   // stack.go:33
```

`Stack` returns `out[s][i]` = where series `s` sits in the `i`-th stack.
NaN values take no room (`stack.go:37-43`).

### 1.7 Compiling sample — bar chart with grid, tooltips, click, label

This is an abridged form of the probe in
`C:/Users/yafit/AppData/Local/Temp/chartprobe/apiprobe/main.go`, which
`go vet`s clean against `github.com/egoist/mygo v0.4.8`:

```go
func (a *appState) barChart(c *ui.Context) {
	t := c.Theme()
	x := chart.NewBand(a.months...).Padding(0.3, 0.15)
	_, hi := chart.Extent(a.sales...)
	y := chart.NewLinear(0, hi).Nice(4)
	y.Format = func(v float64) string { return fmt.Sprintf("$%.0fk", v) }
	ticks := y.Ticks(4)

	chart.Plot(c, chart.Axes{
		Left:   ticks,
		Bottom: x.Ticks(),
		Label: func(s chart.Side, tk chart.Tick) {
			ui.Text(c, tk.Label).NoWrap().FontSize(11).TextColor(t.TextMuted)
		},
	}, func(area ui.Element) {
		area.Draw(func(p *ui.Painter, r ui.Rect) {
			for _, tk := range ticks { // grid lines
				p.Line(r.X, chart.Y(r, tk.At), r.X+r.W, chart.Y(r, tk.At), 1, t.Border)
			}
		})
		for i, v := range a.sales {
			bar := chart.Rect(c.Key(i), x.Band(i), y.Span(0, v)).Radius(4).
				Background(t.Accent).Tooltip(fmt.Sprintf("%s: %.0f", a.months[i], v))
			if bar.Clicked() {
				a.month = i
			}
		}
		if i := a.month; i >= 0 && i < len(a.sales) {
			chart.Anchor(c, x.At(i), y.At(a.sales[i]), ui.Center, ui.End, func() {
				ui.Text(c, fmt.Sprint(a.sales[i])).NoWrap().Margin(0, 0, 4, 0)
			})
		}
	}).Label("Sales by month").Height(200).Gap(6)
}
```

Line + area + selection:

```go
area.Draw(func(p *ui.Painter, r ui.Rect) {
	p.FillPath(chart.Area(r, xs, nil, ys, chart.CurveMonotone), t.Accent.Alpha(0.15))
	p.StrokePath(chart.Line(r, xs, ys, chart.CurveMonotone), 2, t.Accent)
})
chart.Select(c, area, &a.selected, xs, describe)
if i := a.selected; i >= 0 && i < len(xs) {
	chart.RuleX(c, xs[i], 1).Background(t.TextMuted)
	chart.Point(c, xs[i], ys[i], 10, 10).Radius(5).Background(t.Accent)
}
```

Donut:

```go
slices := chart.Pie(a.shares, 0.025)
const size, inner, outer = 180, 56, 86
ring := ui.Box(c).Size(size, size).Label("Users by platform")
hover := -1
if px, py, over := ring.PointerPosition(); over {
	if i, ok := chart.SliceAt(slices, px-size/2, py-size/2, inner, outer); ok {
		hover = i
	}
}
ring.Draw(func(p *ui.Painter, r ui.Rect) {
	for i, s := range slices {
		out := float32(outer - 4)
		if i == hover {
			out = outer
		}
		p.FillPath(chart.Sector(r.X+size/2, r.Y+size/2, inner, out, s), colors[i])
	}
})
```

Stacked areas:

```go
stacked := chart.Stack([][]float64{a.search, a.direct}, chart.StackStandard)
for s, series := range stacked {
	lo, hi := make([]float32, n), make([]float32, n)
	for i, seg := range series {
		lo[i], hi[i] = y.At(seg.Lo), y.At(seg.Hi)
	}
	p.FillPath(chart.Area(r, xs, lo, hi, chart.CurveMonotone), colors[s].Alpha(0.75))
	p.StrokePath(chart.Line(r, xs, hi, chart.CurveMonotone), 1.5, colors[s])
}
```

The upstream gallery has the complete, production versions of all five
charts (bar, live line, stacked, scatter, donut) at
`examples/gallery/charts.go:1-323` — notably `gridLines` at `charts.go:70-80`
and `axisLabel` at `charts.go:62-68`.

### 1.8 Mark animation

Marks are elements, so `Element.Transition` gives them free data-change
animation (`docs/ui/chart.md:117-122`):

```go
chart.Rect(c, x.Band(i), y.Span(0, v)).Background(t.Accent).
	Transition(ui.ElementTransition{Duration: 300 * time.Millisecond})
```

```go
type ElementTransition struct {          // ui/transition.go:18
	Duration time.Duration
	Ease     Easing
	Animation *Animation
	Position, Size, Colors bool
	Enter *Motion
	Exit  *Motion
	Match any
}
```

A mark moves because its `LeftPercent`/`TopPercent`/`HeightPercent` change,
and `Position` + `Size` animate those (`ui/transition.go:28-32`,
`ui/chart/chart.go:174-183`). All three animate when none are set.

---

## 2. Custom drawing and canvas

### 2.1 The draw callback

```go
func (_handle Element) Draw(fn func(p *Painter, r Rect)) Element      // ui/api_gen.go:3047
func (_handle Element) DrawOver(fn func(p *Painter, r Rect)) Element  // ui/api_gen.go:3058
```

`Draw` paints **after** the element's background and **before** its
children; `DrawOver` after the children (`docs/ui/drawing.md:3-4`).
The `r` you receive is the element's border box in window DIPs, already in
the painter's coordinate space — you do **not** add the element's origin
yourself. The upstream docs are explicit: "`Draw` and `DrawOver` receive
the original layout coordinates... the painter applies the transform to
everything they draw" (`docs/ui/transforms.md:86-90`).

Draw functions **only paint, and may be called more than once per frame**
(`docs/ui/drawing.md:35`). Never mutate app state in one except behind an
idempotent guard.

### 2.2 Getting bounds

```go
func (_handle Element) Bounds() Rect        // ui/api_gen.go:2525
func (_handle Element) LayoutBounds() Rect  // ui/api_gen.go:3670
```

> `Bounds` returns the previous frame's transformed bounding rectangle in
> window DIPs; it is empty for an element the previous frame lacked.
> `LayoutBounds` returns its box before visual transforms.
> — `ui/api_gen.go:2522-2524`, `:3668-3670`

```go
type Rect struct{ X, Y, W, H float32 }   // ui/element.go:439
```

**In practice you do not need either inside a `Draw`** — the callback's `r`
parameter is the correct box and is always current. Use `Bounds()`/
`LayoutBounds()` only from view code, for hit-testing or anchoring overlays,
and remember they describe the **previous** frame.

Pointer position inside an element:

```go
func (_handle Element) PointerPosition() (float32, float32, bool)   // ui/api_gen.go:4115
```

Relative to the element's box; asking for it makes MyGo draw a frame on
every pointer move over the element (`api_gen.go:4112-4114`).

### 2.3 Painter — the full drawing surface

All coordinates are DIPs.

```go
// Rects
func (p *Painter) Fill(r Rect, c Color, radius float32)                              // ui/paint.go:915
func (p *Painter) FillGradient(r Rect, g LinearGradient, radius float32)             // paint.go:920
func (p *Painter) Stroke(r Rect, c Color, radius, width float32)                     // paint.go:929
func (p *Painter) StrokeDashed(r Rect, c Color, radius, width float32)               // paint.go:935
func (p *Painter) Shadow(r Rect, radius, x, y, blur, spread float32, c Color)        // paint.go:945
func (p *Painter) Line(x0, y0, x1, y1, width float32, c Color)                       // paint.go:992

// Paths
func (p *Painter) FillPath(path *Path, c Color)                                      // ui/path.go:342
func (p *Painter) FillPathGradient(path *Path, g LinearGradient)                     // path.go:350
func (p *Painter) StrokePath(path *Path, width float32, c Color)                     // path.go:358
func (p *Painter) StrokePathGradient(path *Path, width float32, g LinearGradient)    // path.go:376

// Text and images
func (p *Painter) Text(x, y float32, s string, size float32, c Color)                // paint.go:1005
func (p *Painter) RichText(x, y, width float32, spans ...Span) (w, h float32)        // ui/richtext.go:212
func (p *Painter) MeasureText(width float32, spans ...Span) (w, h float32)           // richtext.go:221
func (p *Painter) Image(src ImageSource, r Rect, fit Fit)                            // paint.go:1012
func (p *Painter) ImageRegion(b *Bitmap, src, dst Rect, o ImageOptions)              // paint.go:1041
func (p *Painter) Icon(s *SVG, r Rect, c Color)                                      // ui/svg.go:91
func (p *Painter) Glyphs(glyphs []Glyph, x, y float32, c Color)                      // ui/glyphs.go:85

// Grouping and state
func (p *Painter) Clip(r Rect, radius float32, fn func())                            // paint.go:1085
func (p *Painter) Transform(t Transform, ox, oy float32, fn func())                  // paint.go:1108
func (p *Painter) FocusRing(r Rect, radius [4]float32)                               // paint.go:1075

// Frame info
func (p *Painter) Now() time.Time      // ui/paint.go:77
func (p *Painter) AnimationFrame()     // ui/paint.go:85
func (p *Painter) After(d time.Duration) // ui/paint.go:92
func (p *Painter) Scale() float32      // ui/glyphs.go:194
func (p *Painter) Visible() Rect       // ui/glyphs.go:190
func (p *Painter) Theme() *Theme       // ui/material.go:35
```

Notes that matter for a dashboard:

- `Line` only paints horizontal or vertical lines — it collapses to a
  `Fill` of a thin rect (`paint.go:992-1003`). For a diagonal, build a
  `Path`.
- `Fill`'s `radius` is a **single** float; `FocusRing` takes per-corner
  radii as `[4]float32`.
- `ImageOptions` (`paint.go:1024-1030`) carries `FlipX, FlipY bool` and
  `Tint Color`.
- `Fit` is `Contain, Cover, FillBox, ScaleDown, NaturalSize`
  (`ui/element.go:445-460`).

### 2.4 Path — arbitrary shapes

```go
type Path struct{ /* unexported: cmds, trim */ }                        // ui/path.go:14
func (p *Path) MoveTo(x, y float32) *Path                               // path.go:27
func (p *Path) LineTo(x, y float32) *Path                               // path.go:33
func (p *Path) QuadTo(cx, cy, x, y float32) *Path                       // path.go:40
func (p *Path) CubeTo(c1x, c1y, c2x, c2y, x, y float32) *Path           // path.go:47
func (p *Path) Close() *Path                                            // path.go:53
func (p *Path) Trim(from, to float32) *Path                             // path.go:65
func (p *Path) Circle(cx, cy, r float32) *Path                          // path.go:72
```

Every builder returns `*Path`, so they chain. `Trim` keeps a length
fraction — this is how you animate a path drawing itself in
(`docs/ui/drawing.md:40-56`). There is **no** `RoundedRect`, `Arc` or
`Rect` helper on `Path`; for a rounded rect use `Painter.Fill` with a
radius, and for an arc build cubics (that is exactly what
`chart.Sector`/`chart.arc` does at `ui/chart/shape.go:319-376`).

### 2.5 Canvas — render to texture

```go
type Canvas struct { img *scene.Image; bm *Bitmap }                     // ui/canvas.go:17
func NewCanvas(w, h int) *Canvas                                        // canvas.go:23
func (cv *Canvas) Bitmap() *Bitmap                                      // canvas.go:30
func (cv *Canvas) Size() (w, h int)                                     // canvas.go:33
func (cv *Canvas) Draw(p *Painter, fn func(cp *Painter))                // canvas.go:40
```

A `Canvas` is a bitmap a painter draws, on the GPU where the window draws
there — render to texture (`ui/canvas.go:7-16`). `Draw` clears it to
transparent and repaints it with a painter whose units are the canvas's
**pixels**, at scale 1, clipped to the canvas
(`canvas.go:40-53`). It **keeps** what it holds until drawn anew, and shows
anywhere a bitmap does via `Bitmap()` (`canvas.go:28-30`).

This is the right tool for a chart's static grid or a map's ground: paint
it once, then `p.Image(grid.Bitmap(), r, ui.FillBox)` every frame for the
price of an image. The canonical pattern, from
`docs/ui/drawing.md:81-108` and exercised by `ui/canvas_test.go:10-51`:

```go
grid := ui.NewCanvas(800, 400)
drawn := false
ui.Box(c).Height(200).Draw(func(p *ui.Painter, r ui.Rect) {
	if !drawn { // once, then every frame for the price of an image
		drawn = true
		grid.Draw(p, func(cp *ui.Painter) { drawGrid(cp, 800, 400) })
	}
	p.Image(grid.Bitmap(), r, ui.FillBox)
})
```

Canvases nest and hold text (`ui/canvas_test.go:54-89`). Make the bitmap
`Pixelated()` (`ui/bitmap.go:42`) to scale pixel art up crisp.

**Caveat for a dashboard:** the canvas is a fixed pixel buffer, so it does
not re-render on resize — draw it at `Painter.Scale()` size and redraw when
the element's size changes, or keep canvases for content that genuinely
seldom changes.

### 2.6 Text shaping (only if you lay out text yourself)

`Painter.Text` and `Painter.RichText` are enough for chart labels.
For grid cells or a heat map where you place glyphs yourself:

```go
type Font struct{ /* Family, Size, ... */ }        // ui/glyphs.go:14
type FontMetrics struct {                          // glyphs.go:39
	Ascent, Descent, LineGap float32
}
func (f Font) Metrics() FontMetrics                // glyphs.go:44
func Shape(s string, f Font) []Glyph               // glyphs.go:72
```

`Shape` caches nothing, so keep the glyphs of text drawn in many frames
(`docs/ui/drawing.md:139-163`).

### 2.7 Animation from a Draw

For motion that does not change layout, compute from `p.Now()` and call
`p.AnimationFrame()` while it moves (`docs/ui/drawing.md:165-213`,
`docs/ui/motion.md:337-338`):

```go
ui.Box(c).Height(4).Draw(func(p *ui.Painter, r ui.Rect) {
	p.AnimationFrame()
	x := r.X + (r.W-20)*float32(p.Now().UnixMilli()%1000)/1000
	p.Fill(ui.Rect{X: x, Y: r.Y, W: 20, H: r.H}, c.Theme().Accent, 2)
})
```

Such frames repaint without running the view, so they cost only painting —
and an element out of view asks for none (`docs/ui/drawing.md:198-205`).
Honor `c.Preferences().ReduceMotion`.

---

## 3. Treemap strategy — **must be drawn manually**

### Verdict

**`ui/chart` cannot draw a treemap. A squarified treemap must be computed
by your own code and painted with the drawing API** (or, better for
interactivity, laid out as `chart.Rect` marks).

### Evidence

1. **No treemap code exists anywhere in the module.** A case-insensitive
   recursive grep for `treemap|squarif|squarified` over the entire
   `mygo@v0.4.8` tree (all `.go` and `.md` files) returns **0 matches**.
2. **No treemap-shaped API exists.** `go doc -all` on `ui/chart` lists the
   complete exported surface — 21 package-level functions and 11 types —
   and none of them takes a hierarchy, a parent/child tree, or a nesting
   weight. There is no `Hierarchy`, `Nest`, `Partition`, `Pack` or `Tree`
   symbol.
3. **The docs state the omissions explicitly.** `docs/ui/chart.md:221-222`:

   > What the package leaves out: logarithmic scales, zooming and panning,
   > legends (rows of your own, as above) and labels that avoid each other.

   Treemap is not even listed as a candidate — the package's scope is
   scales, axes, marks, line/area/pie paths, stacking and selection
   (`ui/chart/chart.go:1-24`, `docs/ui/chart.md:1-12`).
4. **Nothing in the package produces a 2-D nested rectangle partition.**
   The only partition primitives are 1-D: `Band` divides one axis into
   strips (`scale.go:261-329`), and `Stack` divides one axis into
   `Segment`s (`stack.go:33-80`). Neither recurses.

### What to build instead

**Step 1 — compute the layout.** Implement squarified treemap (Bruls,
Huizing, van Wijk 2000) over the unit square. This is ~80 lines and has no
MyGo dependency, so it is unit-testable in isolation. A verified working
version (see [Verification](#verification)):

```go
// treemapItem is one value to lay out.
type treemapItem struct {
	Label string
	Value float64
}

// treemapRect is a laid-out tile: a fraction of the plot's box.
type treemapRect struct {
	X, Y, W, H float32 // fractions of the plot, y down from the top
	Item       treemapItem
}

// squarify lays items out in the unit square with the squarified treemap
// algorithm (Bruls, Huizing, van Wijk 2000): values become areas, rows are
// laid along the shorter side, and a row closes when the next item would
// make the row's aspect ratios worse.
func squarify(items []treemapItem) []treemapRect {
	total := 0.0
	for _, it := range items {
		if it.Value > 0 {
			total += it.Value
		}
	}
	if total <= 0 {
		return nil
	}
	// Areas of the unit square, largest first, as the algorithm wants.
	areas := make([]float64, 0, len(items))
	keep := make([]treemapItem, 0, len(items))
	for _, it := range items {
		if it.Value > 0 {
			areas = append(areas, it.Value/total)
			keep = append(keep, it)
		}
	}
	for i := range areas { // sort descending by area
		for j := i + 1; j < len(areas); j++ {
			if areas[j] > areas[i] {
				areas[i], areas[j] = areas[j], areas[i]
				keep[i], keep[j] = keep[j], keep[i]
			}
		}
	}
	out := make([]treemapRect, 0, len(keep))
	rx, ry, rw, rh := 0.0, 0.0, 1.0, 1.0
	i := 0
	for i < len(areas) {
		short := math.Min(rw, rh)
		row := []float64{areas[i]}
		i++
		// Grow the row while the worst aspect ratio improves.
		for i < len(areas) {
			if worst(row, short) >= worst(append(append([]float64{}, row...), areas[i]), short) {
				row = append(row, areas[i])
				i++
				continue
			}
			break
		}
		sum := 0.0
		for _, a := range row {
			sum += a
		}
		if rw >= rh {
			// Lay the row in a column along the left edge.
			thick := sum / rh
			y := ry
			for _, a := range row {
				h := a / thick
				out = append(out, treemapRect{X: float32(rx), Y: float32(y), W: float32(thick), H: float32(h)})
				y += h
			}
			rx += thick
			rw -= thick
		} else {
			// Lay the row in a band along the top edge.
			thick := sum / rw
			x := rx
			for _, a := range row {
				w := a / thick
				out = append(out, treemapRect{X: float32(x), Y: float32(ry), W: float32(w), H: float32(thick)})
				x += w
			}
			ry += thick
			rh -= thick
		}
	}
	for k := range out { // restore the items after the sort
		out[k].Item = keep[k]
	}
	return out
}

// worst is the worst aspect ratio of a row of areas laid along side.
func worst(row []float64, side float64) float64 {
	if len(row) == 0 || side <= 0 {
		return math.Inf(1)
	}
	sum := 0.0
	mx, mn := 0.0, math.Inf(1)
	for _, a := range row {
		sum += a
		mx, mn = math.Max(mx, a), math.Min(mn, a)
	}
	if sum <= 0 {
		return math.Inf(1)
	}
	s2, sum2 := side*side, sum*sum
	return math.Max(s2*mx/sum2, sum2/(s2*mn))
}
```

Its verified output for `{usr:60, var:25, home:10, tmp:3, etc:2}`:

```
tiles=5 area=1.000000 overlap=0.000000000
  usr    x=0.0000 y=0.0000 w=0.6000 h=1.0000 aspect=1.67
  var    x=0.6000 y=0.0000 w=0.4000 h=0.6250 aspect=1.56
  home   x=0.6000 y=0.6250 w=0.2667 h=0.3750 aspect=1.41
  tmp    x=0.8667 y=0.6250 w=0.1333 h=0.2250 aspect=1.69
  etc    x=0.8667 y=0.8500 w=0.1333 h=0.1500 aspect=1.12
```

Areas sum to exactly 1.0 and overlap is exactly 0 — the invariants you
should assert in your own tests.

**Step 2 — paint it.** Two options, and the second is strictly better for
a dashboard.

*Option A — one `Draw` painting every tile.* Simplest, but the whole
treemap is one element: no per-tile tooltip, hover or click.

```go
area := ui.Box(c).Fill().Label("Disk usage")
area.Draw(func(p *ui.Painter, r ui.Rect) {
	for i, rc := range rects {
		box := ui.Rect{
			X: r.X + rc.X*r.W,
			Y: r.Y + rc.Y*r.H,
			W: rc.W * r.W,
			H: rc.H * r.H,
		}
		p.Fill(box, colors[i%len(colors)].Alpha(0.85), 2)
		if box.W > 48 && box.H > 20 {
			p.Text(box.X+6, box.Y+6, rc.Item.Label, 11, t.Text)
		}
	}
})
```

*Option B (recommended) — one `chart.Rect` mark per tile.* You get
`Tooltip`, `Clicked`, `Hovered`, `Label` (accessibility), `Cursor` and
`Transition` per tile for free, because marks are ordinary elements
(`docs/ui/chart.md:88-90`). `chart.Rect` needs no `Plot` — it is an
absolute box positioned in percentages of **any** parent
(`chart.go:174-183`), and it normalizes its spans, so a squarify result in
any corner order works directly.

Mind the axis flip: squarify emits `y` **down from the top**, while
`chart.Span` measures `y` **up from the bottom**
(`ui/chart/scale.go:8-11`). Convert with `ySpan = Span{Lo: 1 - (y + h), Hi: 1 - y}`.

```go
ui.Box(c).Size(400, 200).Children(func() {
	for i, rc := range rects {
		chart.Rect(c.Key(i),
			chart.Span{Lo: rc.X, Hi: rc.X + rc.W},
			chart.Span{Lo: 1 - (rc.Y + rc.H), Hi: 1 - rc.Y},
		).Background(colors[i%len(colors)]).Radius(2).
			Tooltip(rc.Item.Label)
	}
})
```

I verified this exact approach end-to-end: the headless tester
hit-tests all three tiles of a hand-computed squarify layout at the
expected coordinates (probe `TestTreemapMarksHitTest`, see
[Verification](#verification)). Tiles are also verified to paint real
pixels (probe `TestTreemapRenders`: 80 000 colored pixels of 80 000).

**What you must supply yourself** (no MyGo help): the squarify algorithm,
depth/child nesting if you want drill-down, label fitting/truncation, and
color assignment. The chart package's missing legend and label-collision
handling (`docs/ui/chart.md:221-222`) applies here too.

---

## 4. Layout primitives for a dashboard

### Constructors

```go
func Box(c *Context) Element      // ui/api_gen.go:624
func Column(c *Context) Element   // ui/api_gen.go:784
func Row(c *Context) Element      // ui/api_gen.go:1425
func Grid(c *Context) Element     // ui/api_gen.go:1048
func Text(c *Context, s string) Element  // ui/api_gen.go:1805
func Spacer(c *Context) Element   // ui/api_gen.go:1631
func Divider(c *Context) Element  // ui/api_gen.go:882
```

`Box` is a column (`docs/ui/layout.md:17`). `Column` stacks top-to-bottom
and stretches children to its width; `Row` places left-to-right and centers
vertically (`docs/ui/layout.md:16-18`).

### Children — the scoping callback

```go
func (e Element) Children(fn func()) Element   // ui/frame.go:122
```

> Children builds children with the same context, scoped to this element.
> An absent element skips fn.
> — `ui/frame.go:119-121`

Every container call must be followed by `.Children(...)` before building
its contents; the context's parent scope changes inside the callback
(`ui/doc.go:31-34`).

### Sizes

```go
func (_handle Element) Width(v float32) Element           // ui/api_gen.go:4943
func (_handle Element) Height(v float32) Element          // ui/api_gen.go:3454
func (_handle Element) Size(w float32, h float32) Element // ui/api_gen.go:4558
func (_handle Element) WidthPercent(p float32) Element    // ui/api_gen.go:4954
func (_handle Element) HeightPercent(p float32) Element   // ui/api_gen.go:3465
func (_handle Element) Fill() Element                     // ui/api_gen.go:3132
func (_handle Element) FillWidth() Element                // ui/api_gen.go:3154
func (_handle Element) FillHeight() Element               // ui/api_gen.go:3143
func (_handle Element) MinWidth(v float32) Element        // ui/api_gen.go:3920
func (_handle Element) MaxWidth(v float32) Element        // ui/api_gen.go:3855
func (_handle Element) MinHeight(v float32) Element       // ui/api_gen.go:3899
func (_handle Element) MaxHeight(v float32) Element       // ui/api_gen.go:3823
func (_handle Element) Grow(f float32) Element            // ui/api_gen.go:3406
func (_handle Element) Shrink(f float32) Element          // ui/api_gen.go:4535
func (_handle Element) Basis(v float32) Element           // ui/api_gen.go:2419
func (_handle Element) AspectRatio(r float32) Element     // ui/api_gen.go:2326
```

`Fill()` sets width and height to `100%` (`ui/element.go:644`).
`FillWidth`/`FillHeight` take the parent's whole content box
(`docs/ui/layout.md:36-37`). `Grow(1)` is CSS `flex: 1`
(`docs/ui/layout.md:38-41`).

### Spacing

```go
func (_handle Element) Gap(v float32) Element       // ui/api_gen.go:3334
func (_handle Element) GapX(v float32) Element      // ui/api_gen.go:3345
func (_handle Element) GapY(v float32) Element      // ui/api_gen.go:3356
func (_handle Element) Padding(v ...float32) Element  // ui/api_gen.go:4032
func (_handle Element) PaddingX(v float32) Element    // ui/api_gen.go:4043
func (_handle Element) PaddingY(v float32) Element    // ui/api_gen.go:4054
func (_handle Element) Margin(v ...float32) Element   // ui/api_gen.go:3791
func (_handle Element) MarginX(v float32) Element     // ui/api_gen.go:3802
func (_handle Element) MarginY(v float32) Element     // ui/api_gen.go:3813
```

`Padding` and `Margin` are variadic — one, two or four values, exactly as
in CSS (`docs/ui/layout.md:46-51`). `ui.Auto` as a margin takes the free
space on that side; `Margin(0, ui.Auto)` centers in a column
(`docs/ui/layout.md:49-52`).

### Alignment

```go
type Align uint8                       // ui/element.go:15
const (
	Start Align = iota                 // ui/element.go:19
	Center                             // element.go:20
	End                                // element.go:21
	Stretch                            // element.go:23
	SpaceBetween                       // element.go:26
	SpaceAround
	SpaceEvenly
)

func (_handle Element) Justify(a Align) Element      // ui/api_gen.go:3592
func (_handle Element) AlignItems(a Align) Element   // ui/api_gen.go:2208
func (_handle Element) AlignSelf(a Align) Element    // ui/api_gen.go:2221
func (_handle Element) AlignContent(a Align) Element // ui/api_gen.go:2197
func (_handle Element) Center() Element              // ui/api_gen.go:2536
func (_handle Element) Wrap() Element                // ui/api_gen.go:4977
func (_handle Element) Reverse() Element             // ui/api_gen.go:4197
```

`Justify` places along the main axis; `AlignItems` across it; `Center()`
does both (`docs/ui/layout.md:55-58`).

### Absolute positioning and percentages

```go
func (_handle Element) Absolute() Element               // ui/api_gen.go:2154
func (_handle Element) Top(v float32) Element           // ui/api_gen.go:4736
func (_handle Element) Right(v float32) Element         // ui/api_gen.go:4207
func (_handle Element) Bottom(v float32) Element        // ui/api_gen.go:2502
func (_handle Element) Left(v float32) Element          // ui/api_gen.go:3680
func (_handle Element) TopPercent(p float32) Element    // ui/api_gen.go:4749
func (_handle Element) RightPercent(p float32) Element  // ui/api_gen.go:4230
func (_handle Element) BottomPercent(p float32) Element // ui/api_gen.go:2512
func (_handle Element) LeftPercent(p float32) Element   // ui/api_gen.go:3690
```

`Absolute` takes the element out of flow; insets are measured from just
inside the parent's border, padding ignored — this is what `chart.Rect`
and friends rely on (`docs/ui/layout.md:60-72`, `ui/chart/chart.go:174-183`).

### Grid tracks

```go
type Track struct{ kind trackKind; v float32 }   // ui/grid.go:4
func Fixed(v float32) Track { return Track{trackFixed, v} }   // ui/grid.go:18
func Fr(f float32) Track { return Track{trackFr, max(f, 0)} } // ui/grid.go:25
func FitContent() Track { return Track{} }                    // ui/grid.go:30

func (_handle Element) ColumnTracks(tracks ...Track) Element  // ui/api_gen.go:2682
func (_handle Element) RowTracks(tracks ...Track) Element     // ui/api_gen.go:4336
func (_handle Element) Columns(n int) Element                 // ui/api_gen.go:2693
func (_handle Element) ColumnStart(n int) Element             // ui/api_gen.go:2669
func (_handle Element) RowStart(n int) Element                // ui/api_gen.go:4324
func (_handle Element) ColumnSpan(n int) Element              // ui/api_gen.go:2658
func (_handle Element) RowSpan(n int) Element                 // ui/api_gen.go:4313
```

Grid lines are numbered **from 1**; `0` means automatic placement
(`ui/grid.go:33-35`). `chart.Plot` builds exactly this grid internally
(`ui/chart/chart.go:82-107`), so you rarely need it by hand.

### Clipping, transitions

```go
func (_handle Element) Clip() Element                        // ui/api_gen.go:2612
func (_handle Element) ClipX() Element                       // ui/api_gen.go:2624
func (_handle Element) ClipY() Element                       // ui/api_gen.go:2634
func (_handle Element) Invisible() Element                   // ui/api_gen.go:3545
func (_handle Element) Transition(t ElementTransition) Element // ui/api_gen.go:4835
func (_handle Element) Opacity(o float32) Element            // ui/api_gen.go:4000
func (_handle Element) Radius(r ...float32) Element          // ui/api_gen.go:4157
```

### Theme

```go
type Theme struct{ /* ... */ }                 // ui/theme.go:7
func (_handle *Context) Theme() *Theme         // ui/api_gen.go:420
func (t *Theme) Space(n float32) float32       // ui/theme.go:61
```

Relevant fields for charts: `Background, Surface, Border, Text, TextMuted,
Accent, Danger, Warning, Success, Radius, Spacing, FontSize, Font`
(`ui/theme.go:7-57`). `Space(n)` sizes things in multiples of the theme's
spacing unit so custom chrome matches the widgets (`ui/theme.go:61-67`).

### A dashboard skeleton

```go
ui.Row(c).Gap(16).Padding(24).Wrap().AlignItems(ui.Start).Children(func() {
	card := ui.Column(c).Basis(320).Grow(1).MinWidth(0).Padding(24).Gap(8)
	card.Children(func() {
		ui.Text(c, "Sales").FontWeight(600)
		a.barChart(c)
	})
})
```

---

## 5. Verification

Everything in this report was checked against the pinned source, and the
API claims were compiled and rendered:

| Check | Result |
|---|---|
| `go vet ./apiprobe` — every signature in §1–§4 exercised | clean |
| `go vet ./finalcheck` — the §3 Option A and Option B snippets verbatim | clean |
| `go vet ./finalcheck2` — the §1.7 and §1.8 samples verbatim | clean |
| `go test ./render -run TestChartRenders` — `Plot` + `Rect` + `Sector` headless | PASS, 104 000/104 000 non-transparent px |
| `go test ./render -run TestTreemapRenders` — manual tiles via `Painter.Fill`/`Painter.Text` | PASS, 80 000 colored px |
| `go test ./render -run TestTreemapMarksHitTest` — tiles as `chart.Rect` marks, clicked at 3 coordinates | PASS, all 3 hit-tested correctly |
| `go run ./inlined` — the squarify code **as written in §3**, extracted from this file | compiles; area = 1.000000, overlap = 0.000000000 |
| `grep -ri "treemap\|squarif" mygo@v0.4.8` (all `.go`/`.md`) | 0 matches |

Probes live in `C:/Users/yafit/AppData/Local/Temp/chartprobe/`
(`apiprobe/`, `finalcheck/`, `finalcheck2/`, `render/`, `squarify/`,
`inlined/`), a throwaway module requiring `github.com/egoist/mygo v0.4.8`
— the same version as `sweep/go.mod`. The §3 squarify implementation was
extracted back out of this report and run, so it is known-good as printed.

### Gotchas worth pinning to the wall

1. **y is up in chart fractions, down in squarify output.** The single most
   common bug when combining the two.
2. **`Painter.Line` is axis-aligned only** (`ui/paint.go:992-1003`).
3. **Draw callbacks may run more than once a frame** — guard side effects
   (`docs/ui/drawing.md:35`).
4. **`chart.Select` must be called inside the `Plot` build callback**, on
   the `area` it hands you, because it attaches focus, shortcuts and local
   state to that element (`ui/chart/chart.go:246-262`).
5. **`ui.Element` values expire before the next build pass** — never store
   one in app state (`ui/doc.go:25-28`). Use `ui.Handle` for persistent
   identity.
