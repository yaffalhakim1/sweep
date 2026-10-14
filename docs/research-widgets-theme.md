# MyGo `ui` — theming, styling, collection & navigation widgets

Source of truth: `github.com/egoist/mygo@v0.4.8`, package `ui`
(`C:/Users/yafit/go/pkg/mod/github.com/egoist/mygo@v0.4.8/ui`).
Docs: `.../mygo@v0.4.8/docs/ui/*.md`.
Every signature below is quoted verbatim from that pinned source; `file:line` cites are from it.
Reference app on the same stack: `C:/Users/yafit/Documents/learn/go/ports` (mygo v0.3.5).

Package note: `ui` is generated twice — the exported surface lives in `ui/api_gen.go`
(thin wrappers over unexported `core*` builders in `ui/*.go`). Cite either, but the
`api_gen.go` line is the public one.

---

## 0. Imports

```go
import (
    "github.com/egoist/mygo"     // App, NewWindow, WindowOptions, Theme (appearance)
    "github.com/egoist/mygo/ui"  // Context, Element, all widgets
)
```

`ui` is self-contained: `Color`, `Theme`, `ListState`, `TableColumn`, `Toast`,
`Preferences`, `ScrollState`, `Selection[K]` all live in package `ui`.
`mygo.Theme` (the *appearance module*, not `ui.Theme`) is the only thing needed from
the root package for light/dark detection.

`ui` in `ports` (`go.mod`) requires `github.com/egoist/mygo v0.3.5`; the pinned
research version is v0.4.8. Signatures below are v0.4.8.

---

## 1. `ui.Theme` — exact struct fields

`ui/theme.go:7-57`:

```go
// Theme holds the colors and metrics widgets use. Change a copy of
// LightTheme or DarkTheme and set it with Context.SetTheme.
type Theme struct {
	Dark bool                                          // theme.go:8
	Background Color                                   // theme.go:10
	Surface        Color                               // theme.go:13
	SurfaceHover   Color                               // theme.go:14
	SurfacePressed Color                               // theme.go:15
	Border Color                                       // theme.go:17
	Text      Color                                    // theme.go:20
	TextMuted Color                                    // theme.go:21
	Accent        Color                                // theme.go:24
	AccentHover   Color                                // theme.go:25
	AccentPressed Color                                // theme.go:26
	AccentText    Color                                // theme.go:27
	Danger        Color                                // theme.go:28
	Warning Color                                      // theme.go:31
	Success Color                                      // theme.go:32
	Selection Color                                    // theme.go:34
	Focus Color                                        // theme.go:36
	Inverse     Color                                  // theme.go:41
	InverseText Color                                  // theme.go:42
	Scrollbar      Color                               // theme.go:45
	ScrollbarWidth float32                             // theme.go:46
	Radius float32                                     // theme.go:48
	Spacing float32                                    // theme.go:53
	FontSize float32                                   // theme.go:55
	Font     string                                    // theme.go:56
}
```

That is the complete field set — 19 `Color` fields plus `Dark bool`, `ScrollbarWidth`,
`Radius`, `Spacing`, `FontSize` (`float32`) and `Font string` (25 fields). There is no
`White`, no `Shadow` color, no per-widget color: widgets derive everything else
(e.g. `Badge` fills with `t.Text.Alpha(0.1)`, `ui/sidebar.go:226`).

Metrics semantics (`ui/theme.go`):

- `Space(n)`: `func (t *Theme) Space(n float32) float32` — theme.go:61. `n` units of
  `Spacing`; `Spacing <= 0` means 4.
- `Rem(n)`: `func (t *Theme) Rem(n float32) float32` — theme.go:91. `n` × `FontSize`;
  `FontSize <= 0` falls back to `defaultFontSize()` (13 on darwin, 14 elsewhere) — theme.go:98.
- `Spacing` — the unit every widget padding/gap/row height is a multiple of
  (`docs/ui/styling.md`); 4 default, 3 compact, 5 roomy.
- `Radius` — control corner radius; on macOS corners curve continuously
  (AppKit/SwiftUI), elsewhere as quarter circles.
- `ScrollbarWidth` — DIPs; `0` is 6 (`t.scrollbarWidth()`, theme.go:82).
- `Inverse`/`InverseText` — tooltips and toasts are the theme turned over. Left
  `Transparent` (zero `Color`), they fall back to `Text` / `Background`
  (`t.inverse()`, theme.go:70).

### 1.1 The two stock themes

```go
func LightTheme() *Theme   // theme.go:106
func DarkTheme() *Theme    // theme.go:132
```

`LightTheme` (theme.go:106-130) — Background `#ffffff`, Surface `#f4f4f5`,
SurfaceHover `#e9e9ec`, SurfacePressed `#dddde1`, Border `#d9d9de`, Text `#18181b`,
TextMuted `#71717a`, Accent `#2563eb`, AccentHover `#1d4ed8`, AccentPressed `#1e40af`,
AccentText `#ffffff`, Danger `#dc2626`, Warning `#d97706`, Success `#16a34a`,
Selection `RGBA(37,99,235,0.25)`, Focus `RGBA(37,99,235,0.55)`,
Scrollbar `RGBA(0,0,0,0.32)`, Radius 6, Spacing 4, FontSize `defaultFontSize()`.

`DarkTheme` (theme.go:132-155) — `Dark: true`, Background `#18181b`, Surface `#27272a`,
SurfaceHover `#323236`, SurfacePressed `#3c3c41`, Border `#3f3f46`, Text `#f4f4f5`,
TextMuted `#a1a1aa`, Accent `#3b82f6`, AccentHover `#60a5fa`, AccentPressed `#2563eb`,
AccentText `#ffffff`, Danger `#ef4444`, Warning `#f59e0b`, Success `#22c55e`,
Selection `RGBA(59,130,246,0.4)`, Focus `RGBA(96,165,250,0.6)`,
Scrollbar `RGBA(255,255,255,0.35)`, Radius 6, Spacing 4.

Neither sets `Inverse`/`InverseText` — they stay `Transparent` and the
`inverse()` fallback supplies `Text`/`Background`.

### 1.2 `c.Theme()` / `c.SetTheme()` — exact signatures

```go
func (_handle *Context) Theme() *Theme            // api_gen.go:420
func (_handle *Context) SetTheme(t *Theme)        // api_gen.go:387
func (c *context) Theme() *Theme                  // context.go:297  (impl)
func (c *context) SetTheme(t *Theme)              // context.go:301  (impl)
```

`SetTheme` impl (context.go:301-305):

```go
func (c *context) SetTheme(t *Theme) {
	c.theme = t
	c.root.bg = t.Background
	c.root.ts.color, c.root.ts.size, c.root.ts.family = t.Text, t.FontSize, t.Font
}
```

Docs: *"SetTheme makes the frame use t, for the window's root and the widgets created
after the call."* (api_gen.go:385-386). So **call `c.SetTheme` at the top of the view,
before building anything** — widgets built earlier in the pass keep the old theme.
`ports` does exactly this (`internal/app/view_header.go:15` `a.applyTheme(c)` is the
second statement of `view`, then `t := c.Theme()`).

`Theme()` returns the per-frame theme pointer: *"the light or dark theme following the
system's appearance, unless SetTheme replaced it"* (api_gen.go:419-420).

**Do not mutate the returned theme in place.** It points at the engine's
`rt.passTheme` (ui/runtime.go:363-365), so writing through it corrupts the pass.
Copy first, as `docs/ui/styling.md` shows:

```go
t := *c.Theme() // the default, which follows the system
t.Accent, t.Radius = ui.Hex("#7c3aed"), 8
t.Spacing = 3 // compact
c.SetTheme(&t)
```

`ports` builds a whole theme from scratch instead (`internal/app/theme.go:26`,
`func theme(dark bool) *ui.Theme { t := &ui.Theme{Dark: dark} ... }`), driven by
`c.Theme().Dark` (`internal/app/app.go:171`).

### 1.3 `Color` — the type in every theme field

`ui/color.go`:

```go
type Color struct { R, G, B, A uint8; wide wideRGB }   // color.go:17
var Transparent = Color{}                              // color.go:23
func RGB(r, g, b uint8) Color                          // color.go:26
func RGBA(r, g, b uint8, alpha float32) Color          // color.go:29
func Hex(s string) Color                               // color.go:33  panics on bad input
func (c Color) Alpha(a float32) Color                  // color.go:93
func (c Color) Mix(o Color, t float32) Color           // color.go:105
func (c Color) Over(base Color) Color                  // color.go:120
func Oklch(l, c, h float32) Color                      // widecolor.go:18
```

`Hex` accepts `"#rgb"`, `"#rgba"`, `"#rrggbb"`, `"#rrggbbaa"` (color.go:31-32).
`Color` carries straight (non-premultiplied) alpha; a color from `Oklch` may lie
outside sRGB and keeps both the wide value and the nearest sRGB fallback in `R,G,B`
— never assign `R`/`G`/`B` of such a color, build a new one (color.go:8-15).

---

## 2. Detecting the desktop light/dark appearance

Two independent routes; use the first inside a view, the second for app-level policy.

**Inside the view (recommended).** The theme already followed the system appearance:

```go
t := c.Theme()
if t.Dark { /* dark */ }
```

`Dark bool` is `ui/theme.go:8`. The engine picks it in `defaultTheme()`
(ui/runtime.go:355-366): `t := LightTheme(); if rt.dark { t = DarkTheme() }` where
`rt.dark = rt.host.isDark()` (ui/runtime.go:356, host impl ui/window.go:488
`func (h *windowHost) isDark() bool { return h.conn.IsDark != nil && h.conn.IsDark() }`).
`themeChanged()` (ui/runtime.go:369-373) invalidates the cached theme when the system
appearance or desktop preferences change, so a view that reads `c.Theme().Dark` each
frame re-styles itself automatically.

**App-level module.** `mygo.Theme` (root package, `modules.go`):

```go
var Theme = &ThemeModule{}                       // modules.go:237
func (t *ThemeModule) IsDark() bool              // modules.go:240
func (t *ThemeModule) Source() ThemeSource       // modules.go:246
func (t *ThemeModule) SetSource(s ThemeSource)   // modules.go:258
func (t *ThemeModule) OnUpdated(fn func()) (off func())  // modules.go:289

type ThemeSource string                          // modules.go:219
const (
	ThemeSystem ThemeSource = "system"           // modules.go:223
	ThemeLight  ThemeSource = "light"            // modules.go:224
	ThemeDark   ThemeSource = "dark"             // modules.go:225
)
```

`IsDark()` must run on the main thread (it calls `needsApp` + `onMainValue`,
modules.go:241-242). `SetSource` may be called before `App.Run` to start in a saved
appearance (modules.go:255-256). Documented at `docs/native.md:175-180`.

**Desktop preferences** (accent color, contrast, text scale, reduce motion):

```go
func (_handle *Context) Preferences() Preferences  // api_gen.go:356
```

```go
type Preferences struct {                          // preferences.go:7
	Accent Color                                   // preferences.go:10  (A == 0: none)
	ReduceMotion bool                              // preferences.go:15
	HighContrast bool                              // preferences.go:20
	TextScale float32                              // preferences.go:24  (1 == normal)
}
```

The default theme already follows these: `(*Theme).follow(p Preferences)`
(preferences.go:55-90) replaces `Accent`/`AccentHover`/`AccentPressed`/`AccentText`/
`Selection`/`Focus` with the desktop accent, darkens `Border`/`TextMuted` and makes
`Focus` opaque under `HighContrast`, and multiplies `FontSize` by `TextScale`.
`defaultTheme()` calls it (ui/runtime.go:362). **A hand-built theme does not get this
for free** — `ports` deliberately opts out and documents why
(`internal/app/theme.go:22-28`: "The desktop's own accent is deliberately not adopted").

---

## 3. Layout primitives for a dashboard

All on `Element`, all in `ui/api_gen.go` (impls on `*node` in `ui/element.go`).

### Containers

```go
func Row(c *Context) Element        // api_gen.go:1425   horizontal, centers vertically
func Column(c *Context) Element     // api_gen.go:784    vertical, stretches to width
func Box(c *Context) Element        // api_gen.go:624    same as Column
func Spacer(c *Context) Element     // api_gen.go:1631   takes the free space
func Divider(c *Context) Element    // api_gen.go:882    thin line across the axis
```

### Children (the callback container)

```go
func (e Element) Children(fn func()) Element   // ui/frame.go:122
func (e Element) Key(key any) Element          // ui/frame.go:131
```

`Children` builds with the same `*Context`, scoped to this element (frame.go:122-128).
`Element.Key` keys the container *before* its children or local state are built; for a
stateful control use `Context.Key` before construction instead
(`func (c *Context) Key(key any) *Context`, ui/frame.go:49).

### Size

```go
func (_handle Element) Fill() Element                     // api_gen.go:3132
func (_handle Element) FillWidth() Element                // api_gen.go:3154
func (_handle Element) FillHeight() Element               // api_gen.go:3143
func (_handle Element) Width(v float32) Element           // api_gen.go:4943
func (_handle Element) Height(v float32) Element          // api_gen.go:3454
func (_handle Element) Size(w float32, h float32) Element // api_gen.go:4558
func (_handle Element) MinWidth(v float32) Element        // api_gen.go:3920
func (_handle Element) MaxWidth(v float32) Element        // api_gen.go:3855
func (_handle Element) MinHeight(v float32) Element       // api_gen.go:3899
func (_handle Element) MaxHeight(v float32) Element       // api_gen.go:3823
func (_handle Element) Grow(f float32) Element            // api_gen.go:3406
func (_handle Element) Shrink(f float32) Element          // api_gen.go:4535
func (_handle Element) AspectRatio(r float32) Element      // api_gen.go:2326
func (_handle Element) WidthPercent(p float32) Element     // api_gen.go:4954
func (_handle Element) HeightPercent(p float32) Element    // api_gen.go:3465
func (_handle Element) Basis(v float32) Element            // api_gen.go:2419
func (_handle Element) Wrap() Element                      // api_gen.go:4977
func (_handle Element) AlignContent(a Align) Element       // api_gen.go:2197
func (_handle Element) Absolute() Element                  // api_gen.go:2154
func (_handle Element) Top(v float32) Element              // api_gen.go:4736
func (_handle Element) Left(v float32) Element             // api_gen.go:3680
```

`Fill` == "as large as its parent's content" (api_gen.go:3131). `Grow(1)` is CSS
`flex: 1`. **`Grow(1)` alone is not enough for a pane that must not push its siblings
out** — the vibrancy example pairs it with `MinWidth(0)` for exactly this
(`examples/vibrancy/main.go:162`, `ui.Column(c).Grow(1).MinWidth(0)`), and
`examples/code-editor/main.go:78` uses `.Grow(1).FillHeight()` for the pane beside its
fixed-width sidebar.

### Spacing

```go
func (_handle Element) Padding(v ...float32) Element  // api_gen.go:4032
func (_handle Element) PaddingX(v float32) Element    // api_gen.go:4043
func (_handle Element) PaddingY(v float32) Element    // api_gen.go:4054
func (_handle Element) Margin(v ...float32) Element   // api_gen.go:3791
func (_handle Element) Gap(v float32) Element         // api_gen.go:3334
func (_handle Element) GapX(v float32) Element        // api_gen.go:3345
func (_handle Element) GapY(v float32) Element        // api_gen.go:3356
```

`Padding`/`Margin` take CSS-style 1, 2 (v,h) or 4 (t,r,b,l) values (api_gen.go:4030-4031).
`Margin(0, ui.Auto)` centers in a column; `Margin(0, 0, 0, ui.Auto)` sends a row's
child and those after it to the end (`docs/ui/layout.md`).

### Alignment / direction

```go
func (_handle Element) Justify(a Align) Element     // api_gen.go:3592
func (_handle Element) AlignItems(a Align) Element  // api_gen.go:2208
func (_handle Element) AlignSelf(a Align) Element   // api_gen.go:2221
func (_handle Element) Center() Element             // api_gen.go:2536
func (_handle Element) Reverse() Element            // api_gen.go:4197
```

```go
type Align uint8                 // ui/element.go:15
const (
	Start Align = iota           // element.go:19
	Center                       // element.go:20
	End                          // element.go:21
	Stretch                      // element.go:23
	SpaceBetween                 // element.go:25
	SpaceAround                  // element.go:26
	SpaceEvenly                  // element.go:27
	alignAuto                    // element.go:28  (unexported)
)
```

`ui.Auto` (used with `Margin`) is `var Auto = float32(math.Inf(1))` — ui/element.go:49.

### Surface styling

```go
func (_handle Element) Background(c Color) Element                    // api_gen.go:2407
func (_handle Element) TextColor(c Color) Element                     // api_gen.go:4681
func (_handle Element) Border(width float32, c Color) Element         // api_gen.go:2455
func (_handle Element) BorderWidth(v ...float32) Element              // api_gen.go:2492
func (_handle Element) BorderColor(c Color) Element                   // api_gen.go:2466
func (_handle Element) Radius(r ...float32) Element                   // api_gen.go:4157
func (_handle Element) Shadow(x, y, blur, spread float32, c Color) Element  // api_gen.go:4509
func (_handle Element) InsetShadow(x, y, blur, spread float32, c Color) Element // api_gen.go:3532
func (_handle Element) Gradient(from Color, to Color, angle float32) Element    // api_gen.go:3369
func (_handle Element) Stripes(c Color, width, gap, angle float32) Element      // api_gen.go:4620
func (_handle Element) Opacity(o float32) Element                     // api_gen.go:4000
func (_handle Element) Clip() Element                                 // api_gen.go:2612
func (_handle Element) Material(m Material) Element                   // ui/frame.go:144
```

`Material` (ui/material.go:8-12) is an interface:

```go
type Material interface {
	PaintMaterial(p *Painter, box Rect, radii [4]float32)
}
```

`MaterialBuilder` (ui/frame.go:139) extends it so a material can follow the element it
fills:

```go
type MaterialBuilder interface {
	Material
	BuildMaterial(Element) Material
}
```

`Border` draws inside every edge (border-box semantics, `docs/ui/styling.md`).
`Divider`-style rules use `BorderWidth(0, 0, 1, 0)` + `BorderColor(t.Border)` — that is
exactly what `ports` does for its header rule
(`internal/app/view_header.go` pattern and `examples/code-editor/main.go:79`).

### Text

```go
func Text(c *Context, s string) Element                 // api_gen.go:1805
func Textf(c *Context, format string, args ...any) Element // api_gen.go:1891
func (_handle Element) FontSize(v float32) Element      // api_gen.go:3311
func (_handle Element) FontWeight(w int) Element        // api_gen.go:3322
func (_handle Element) Bold() Element                   // api_gen.go:2442
func (_handle Element) SingleLine() Element             // api_gen.go:4547
func (_handle Element) Ellipsis(s string) Element       // api_gen.go:3083
func (_handle Element) TextAlign(a Align) Element       // api_gen.go:4644
func (_handle Element) FontFeatures(features ...string) Element // api_gen.go:3299
```

### State queries (read *after* building the widget)

```go
func (_handle Element) Changed() bool     // api_gen.go:2549
func (_handle Element) Clicked() bool     // api_gen.go:2587
func (_handle Element) Submitted() bool   // api_gen.go:4633
func (_handle Element) Dismissed() bool   // api_gen.go:2854
func (_handle Element) Hovered() bool     // api_gen.go:3493
func (_handle Element) Focused() bool     // api_gen.go:3268
func (_handle Element) FocusVisible() bool // api_gen.go:3234
```

`Changed()` "applies pending input to the controls built so far and reports whether this
widget's value changed in this build pass" (api_gen.go:2546-2548). Configure controls
before querying.

---

## 4. `ui.Sidebar` — source list

```go
func Sidebar(c *Context, selected *string, fn func()) Element                      // api_gen.go:1565
func SidebarItem(c *Context, id string, icon *SVG, label string) Element           // api_gen.go:1575
func SidebarSection(c *Context, title string, open *bool, fn func()) Element       // api_gen.go:1587
```

Impls: `coreSidebar` ui/sidebar.go:41, `coreSidebarSection` ui/sidebar.go:103,
`coreSidebarItem` ui/sidebar.go:164.

Behaviour:

- `selected *string` holds the ID of the chosen item. `Changed()` reports a new choice
  (`docs/ui/sidebar.md:34`).
- The sidebar is itself a vertical `Scroll` (`coreScroll(c)` + `Background(t.Surface)`
  + `Focusable()` + `Shrink(0)`, ui/sidebar.go:42-45) with role `RoleTree`, widget
  `"Sidebar"`. **Give it a width** (`docs/ui/sidebar.md:7`).
- Keyboard: Up/Down/Home/End and first-letter type-select while it has focus; one Tab
  stop (ui/sidebar.go:64-101). Chosen item shows `Accent`+`AccentText` when the sidebar
  is focused, `SurfacePressed` when not (ui/sidebar.go:187-204) — AppKit's behaviour.
- `SidebarItem(c, id, icon, label)` — `icon *SVG` may be `nil`; label after the icon.
  Add trailing content with `.Children(...)`, e.g. a `ui.Badge`
  (`docs/ui/sidebar.md:24`).
- `SidebarSection(c, title, open *bool, fn)` — `open` nil keeps it always open; the
  title becomes a clickable disclosure row that toggles `*open`
  (ui/sidebar.go:114-133).
- With a router, IDs are paths: `if ui.Sidebar(c, &page, ...).Changed() { app.router.Push(page) }`
  (`docs/ui/sidebar.md:42-48`).

Live examples: `examples/code-editor/main.go:68-76`, `examples/vibrancy/main.go:139-150`,
`examples/gallery/main.go:443-455`.

`ui.SVG` is `ui/svg.go:31`; icons are `*ui.SVG`.

---

## 5. `ui.List` — virtual list, lazy building, `ItemKey`/`Rows`

```go
func List(c *Context, s *ListState, n int, row ...func(int)) Element  // ui/list_builder.go:22
func (e Element) ItemKey(key func(int) any) Element                   // ui/list_builder.go:41
func (e Element) Selection(selection Selector) Element                // ui/list_builder.go:49
func (e Element) Rows(row func(ListRow)) Element                      // ui/list_builder.go:57
```

```go
type ListRow struct {                    // ui/list_builder.go:4
	Context  *Context   // list_builder.go:5
	Index    int        // list_builder.go:6
	// unexported: owner Element, selected bool  (list_builder.go:7-8)
}
func (r ListRow) Selected() bool    // list_builder.go:11
func (r ListRow) ListFocused() bool // list_builder.go:12
```

**Laziness**: *"`ui.List` shows rows of a collection of any length: it builds only the
rows in view, and a few beyond, so a list of millions of rows is as fast as one of ten."*
(`docs/ui/list.md:5-7`). `ListState` has `const listMaxRows = 4096` (ui/list.go:29) as a
guard.

**Two construction styles** (both are the same function):

1. Classic, with the row callback:
   ```go
   ui.List(c, &app.files, len(app.files), func(i int) {
       ui.Text(c, app.files[i].Name).Padding(6, 12)
   }).Grow(1)
   ```
2. Deferred, when you need `ItemKey`/`Selection` first (`list_builder.go:19-21`:
   *"Omit it to configure ItemKey and Selection, then call Rows"*):
   ```go
   ui.List(c, &app.files, len(app.files)).Grow(1).
       ItemKey(func(i int) any { return app.files[i].Path }).
       Selection(&app.chosen).
       Rows(func(row ui.ListRow) { ... })
   ```
   `Rows` panics if called twice in one pass (`list_builder.go:66-68`), and `List`
   panics if given more than one row builder (`list_builder.go:27-29`).

`ItemKey` writes straight through to `s.state.Key` (list_builder.go:41-47) — it is a
convenience for `ListState.Key`, not a separate concept.

### 5.1 `ListState` — the fields that matter

```go
type ListState struct {                                     // ui/list.go:38
	Handle                                                  // list.go:40
	Key func(row int) any                                   // list.go:47
	FollowEnd bool                                          // list.go:52
	Selected *int                                           // list.go:58
	Selection Selector                                      // list.go:67
	Label func(row int) string                              // list.go:71
	Header func(row int) bool                               // list.go:76
	Reorder func(rows []int, to int)                        // list.go:82
	Sort *SortOrder                                         // list.go:87
	Columns TableLayout                                     // list.go:91
	RoundRows bool                                          // list.go:95
	// ... unexported place/measurement fields (list.go:98-143)
}
```

Give each list its own `ListState` and **set its fields before building the list**
(ui/list.go:35-36, `docs/ui/list.md:74`). `Key` gives rows a stable identity so focus,
in-progress edits, scroll place and selection survive insertions above.

Scrolling / loading API:

```go
func (s *ListState) ScrollTo(row int, align Align)  // ui/list.go:164
func (s *ListState) ScrollIntoView(row int)         // ui/list.go:170
func (s *ListState) ScrollToEnd()                   // ui/list.go:176
func (s *ListState) Visible() (first, last int)     // ui/list.go:182
func (s *ListState) AtEnd() bool                    // ui/list.go:192
```

Focus, generated on `*ListState`:

```go
func (_handle *ListState) Focus(c *Context) bool                                  // api_gen.go:21
func (_handle *ListState) FocusWithin(c *Context) bool                            // api_gen.go:32
func (_handle *ListState) Focused(c *Context) bool                                // api_gen.go:44
func (_handle *ListState) Shortcut(c *Context, mods Modifiers, key Key) bool      // api_gen.go:56
```

`Handle` (`ui/ref.go:7`, `type Handle struct{ state *handleState }`) supplies
persistent identity: `func (r *Handle) Focus(window ...*Context)` (ref.go:60),
`OnShortcut(c *Context, mods Modifiers, key Key, fn func())` (ref.go:172).

### 5.2 Multi-selection

```go
type Selection[K comparable] struct{ ... }        // ui/selection.go:23
func (s *Selection[K]) Has(k K) bool              // selection.go:28
func (s *Selection[K]) Len() int                  // selection.go:34
func (s *Selection[K]) Add(k K)                   // selection.go:37
func (s *Selection[K]) Remove(k K)                // selection.go:45
func (s *Selection[K]) Clear()                    // selection.go:48
func (s *Selection[K]) All() iter.Seq[K]          // selection.go:51

type Selector interface { ... }                   // ui/selection.go:62 (unexported methods)
```

`Modifiers` is `ui/keys.go:99` and `Key` is `ui/keys.go:11`.
`Selector` is satisfied only by `*Selection[K]` inside the package; you pass
`&app.chosen` and store `ui.Selection[string]` (or any comparable key type).
Without `Key`, rows are indices and `K` is `int` (`docs/ui/list.md:84-85`).

### 5.3 List layout extras

```go
func (_handle Element) RowWidth(w float32) Element       // api_gen.go:4355
func (_handle Element) ScrollWith(scroller Element) Element // api_gen.go:4405
func (_handle Element) Justify(a Align) Element          // api_gen.go:3592
func (_handle Element) Dividers(width float32, c Color) Element // api_gen.go:2870
```

`RowWidth` makes rows at least `w` DIPs and adds sideways scrolling
(`docs/ui/list.md:37-46`). `ScrollWith` moves a header sideways with a scroller — that
is how `ui.Table` keeps its header aligned.

---

## 6. `ui.Table`

```go
func Table(c *Context, s *ListState, columns []TableColumn, n int, cell func(row int, col int)) Element // api_gen.go:1750
```

Impl `coreTable` ui/table.go:142. It is a `List` with columns: same `ListState`, same
`Key`/`Label`/`Selection`/`Selected`/`Sort`/`Columns`/`Reorder`, only the rows in view
are built, and rows are as tall as their tallest cell (`docs/ui/table.md:5-10`).

```go
type TableColumn struct {                     // ui/table.go:9
	Title, ID string                          // table.go:13
	Width, MinWidth, MaxWidth float32         // table.go:18
	Align Align                               // table.go:20
	Sortable bool                             // table.go:23
	Fixed bool                                // table.go:26
}

type SortOrder struct {                       // ui/table.go:38
	Column     string
	Descending bool
}

type TableLayout struct {                     // ui/table.go:46
	Order  []string
	Widths map[string]float32
}
```

- `Width == 0` shares the room the other columns leave (`docs/ui/table.md:5-7`).
- `ID` defaults to `Title` (`func (col *TableColumn) id()`, table.go:29-34) and is the
  key for `Columns`/`Sort`.
- `Fixed` keeps a column pinned and unresizable — `ports` pins the port number
  (`internal/app/view_table.go:15-25`).
- The user resizes (drag right edge, double-click to fit), moves, and scrolls columns
  sideways; `ListState.Columns` persists order+widths, `ListState.Sort` drives
  `Sortable` headers. **The table renders rows in the order `cell` gets them**, so
  sorting is the view's job (`docs/ui/table.md:62-64`).
- `ListState.RoundRows` rounds chosen/hovered rows at the theme `Radius`; default is
  square edge-to-edge (ui/table.go:239-240, `docs/ui/table.md:36-38`).

Full working table + sort wiring: `ports/internal/app/view_table.go:30-58` and
`examples/gallery/main.go:993-1012`.

---

## 7. `ui.Scroll` / `ScrollHorizontal` / `ScrollBoth`

```go
func Scroll(c *Context) Element           // api_gen.go:1435   vertical
func ScrollHorizontal(c *Context) Element // api_gen.go:1455   horizontal row
func ScrollBoth(c *Context) Element       // api_gen.go:1446   both ways
```

Impls: ui/widgets.go:478, 485, 494 — `Scroll` is a `Box` with `flagScrollY|flagHover`,
`ScrollHorizontal` a `Row` with `flagScrollX`, `ScrollBoth` a `Box` with both.
Give them a size or `Grow` them in the parent.

```go
type ScrollState struct {                       // ui/scroll.go:7
	Handle                                      // scroll.go:8
	X, Y float32                                // scroll.go:13  (set to scroll)
	MaxX, MaxY float32                          // scroll.go:16  (container writes)
	// unexported: glide *scrollGlide  (scroll.go:18)
}
func (s *ScrollState) ScrollTo(x, y float32, a Animation)  // scroll.go:41
func (s *ScrollState) Scrolling() bool                     // scroll.go:53

func (_handle Element) TrackScroll(s *ScrollState) Element // api_gen.go:4784
func (_handle Element) ScrollbarInsets(v ...float32) Element // api_gen.go:4424
func (_handle Element) ScrollIntoView() Element            // api_gen.go:4392
```

`Y = 0` is the top and `math.MaxFloat32` the end; the container clamps
(`docs/ui/scroll.md:29-31`). `TrackScroll` binds the state to the container: the
container shows content from `s.X`/`s.Y`, writes them as the user scrolls, and sets
`MaxX`/`MaxY`. Follow-end idiom (`docs/ui/scroll.md:35-38`):

```go
if app.log.Y >= app.log.MaxY {
	app.log.Y = math.MaxFloat32
}
ui.Scroll(c).TrackScroll(&app.log).Grow(1).Children(app.lines)
```

`ScrollTo` animates via `ui.Smooth` (`ui/motion.go:33`, a `ui.Animation` — `ui/motion.go:16`) and stops if the user scrolls;
`ScrollbarInsets` moves the bars in from the edges, CSS-style padding order
(api_gen.go:4415-4422). Scroll bar width/color come from `Theme.ScrollbarWidth` /
`Theme.Scrollbar` (`docs/ui/scroll.md:72-73`).

---

## 8. `ui.Segmented` and `ui.Tabs`

```go
func Segmented(c *Context, selected *int, labels ...string) Element  // api_gen.go:1487
func SegmentedBase(c *Context, selected *int, n int) SegmentedParts  // api_gen.go:1500

type SegmentedParts struct {   // api_gen.go:5168
	Track Element
	// unexported: part uint32  (api_gen.go:5170)
}
func (_handle SegmentedParts) Segment(i int) Element  // api_gen.go:5001
```

```go
func Tabs(c *Context, selected *int, labels ...string) Element   // api_gen.go:1770
func TabsBase(c *Context, selected *int, n int) TabsParts        // api_gen.go:1796

type TabsParts struct {   // api_gen.go:5221
	List Element
	// unexported: part uint32  (api_gen.go:5223)
}
func (_handle TabsParts) Tab(i int) Element   // api_gen.go:5065
```

Both bind `*int` to the chosen index; a click or the arrows/Home/End choose;
`Changed()` reports a new choice. `Segmented` is a radio group (accessible as one,
name it with `.Label("View")`); `Tabs` is a tab list. Build the page below them by
switching on the index (`docs/ui/segmented.md:9-16`, `docs/ui/tabs.md:9-18`).
For icons/custom looks use the `*Base` variants and style `Track`/`Segment`
(or `List`/`Tab`) — `docs/ui/segmented.md:22-32`, `docs/ui/tabs.md:29-40`.

---

## 9. `ui.Badge`, `ui.Progress`, `ui.Meter`

```go
func Badge(c *Context, text string) Element                                // api_gen.go:615
func Progress(c *Context, value float64) Element                           // api_gen.go:1306
func Meter(c *Context, value float64, lo float64, hi float64, levels *MeterLevels) Element // api_gen.go:1160
```

```go
type MeterLevels struct {          // ui/indicators.go:47
	Warning, Critical float64
}
```

- **Badge** — a pill: `Padding(0, t.Space(1.5)).Radius(999).Background(t.Text.Alpha(0.1))`
  with text at `FontSize - 2` (ui/sidebar.go:224-229). Style it:
  `ui.Badge(c, "New").Background(t.Accent).TextColor(t.AccentText)`.
- **Progress** — `value` between 0 and 1; negative shows indeterminate activity
  (a sliding segment). `.Reverse()` fills from the right; `.Label("Uploading")` names it
  for assistive tech (api_gen.go:1303-1305). Impl ui/widgets.go:447; the indeterminate
  bar repaints without rebuilding the view (`p.AnimationFrame()`, widgets.go:464).
- **Meter** — a value between `lo` and `hi`, drawn in `t.Success`, or `t.Warning` /
  `t.Danger` past the levels. `levels` may be `nil` (and levels with `Warning ==
  Critical` are ignored); a `Critical` below `Warning` inverts the sense (a battery) —
  ui/indicators.go:63-71.

Both `Progress` and `Meter` stretch across their column; give a `Width` inside a row.

---

## 10. Toast

```go
func (_handle *Context) Toast(message string)                                    // api_gen.go:456
func (_handle *Context) ToastAction(message string, label string, action func()) // api_gen.go:471
func (_handle *Context) AddToast(t Toast) string                                 // api_gen.go:228
func (_handle *Context) CloseToast(id string)                                    // api_gen.go:273
```

```go
type Toast struct {                    // ui/toast.go:21
	ID string                          // toast.go:25
	Title, Description string          // toast.go:28
	Type string                        // toast.go:31
	Timeout time.Duration              // toast.go:36
	Action   string                    // toast.go:39
	OnAction func()                    // toast.go:40
	OnClose func()                     // toast.go:43
	Data any                           // toast.go:45
}
```

Semantics (`docs/ui/toast.md`, ui/toast.go:13-18, 60-70):

- `Timeout == 0` → 4 s (`toastTime`); `< 0` → until closed; `ToastAction` uses 8 s.
- At most 3 show (`toastLimit`); older ones wait hidden with their time running.
- Adding with an existing `ID` updates that toast and shows it anew.
- `CloseToast("")` closes every toast.
- The timer pauses while the pointer rests on the toasts, focus is in them, or the
  window is in the background. Escape closes the focused toast.
- Look: `Theme.Inverse` / `InverseText` (falling back to `Text`/`Background`).

Custom look (opt-in, replaces the theme's):

```go
func ToastViewportBase(c *Context, fn func(viewport Element, toasts []Toast)) Element // api_gen.go:1949
func ToastBase(c *Context, t Toast) ToastParts                                        // api_gen.go:1918

type ToastParts struct {   // api_gen.go:5246
	Root Element
	// unexported: part uint32  (api_gen.go:5248)
}
func (_handle ToastParts) ActionButton() Element      // api_gen.go:5078
func (_handle ToastParts) CloseButton() Element       // api_gen.go:5090
func (_handle ToastParts) Left() time.Duration        // api_gen.go:5104
```

Call `ToastViewportBase` once per view, anywhere; `toasts` are the showing ones,
newest three, oldest first (`docs/ui/toast.md:94`).

---

## 11. `ui.AlertDialog`, `ui.Modal`, `ui.DialogBase` — and how to get a yes/no

### 11.1 AlertDialog — the yes/no answer

```go
func AlertDialog(c *Context, open *bool, title string, message string, buttons ...string) int // api_gen.go:567
const AlertDismissed = -2                                                                     // ui/feedback.go:154
```

Impl `coreAlertDialog` ui/feedback.go:171-218. The return value is the whole answer:

- `i >= 0` — index of the button clicked; the alert sets `*open = false` in that same
  frame (feedback.go:209-212).
- `ui.AlertDismissed` (`-2`) — the frame Escape closed it; `*open` is set to false
  (feedback.go:206-208, 209-212).
- `-1` — otherwise (still open, or `*open` was false on entry, feedback.go:172-174).

The last button is the default: accent-colored, focused, Enter clicks it
(feedback.go:193-194, `docs/ui/alert-dialog.md:7-8`). A click outside does nothing.

Canonical yes/no:

```go
switch ui.AlertDialog(c, &app.asking, "Delete “Notes”?", "You can't undo this.", "Cancel", "Delete") {
case 1:
	app.delete()
}
```

Force an answer (must be answered — Escape reopens):

```go
if ui.AlertDialog(c, &app.updating, "Update required", "Restart to finish updating.", "Restart") == ui.AlertDismissed {
	app.updating = true
}
```

### 11.2 Modal — a dialog of your own content

```go
func Modal(c *Context, open *bool, fn func()) Element   // api_gen.go:1171
```

`Dismissed()` (api_gen.go:2854) reports the frame Escape or an outside click closed it
(`docs/ui/dialog.md:40-42`). The dialog traps focus, Tab cycles inside it, and focus
returns to the opener on close.

```go
ui.Modal(c, &app.renaming, func() {
	ui.Text(c, "Rename").Bold()
	if ui.TextInput(c, &app.name).AutoFocus().Submitted() {
		app.renaming = false
	}
	ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
		if ui.Button(c, "Cancel").Clicked() {
			app.renaming = false
		}
		if ui.PrimaryButton(c, "Rename").Clicked() {
			app.rename()
			app.renaming = false
		}
	})
})
```

### 11.3 DialogBase — a dialog without a look

```go
func DialogBase(c *Context, open *bool, fn func(backdrop Element, panel Element)) Element // api_gen.go:873
```

`fn` styles the backdrop (which centers the panel) and the panel, and builds the panel's
content; it returns the panel element (`docs/ui/dialog.md:55-60`).

---

## 12. Minimal compiling sidebar + detail-pane layout

Verified: built with `go build ./...` against `github.com/egoist/mygo v0.4.8`
(temp module, Go 1.27.2) — compiles clean.

```go
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// app is the state the window shows, kept across frames.
type app struct {
	section string
	open    bool
	files   []string
	chosen  int
	list    ui.ListState
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		// The sidebar chooses the section ID into a.section.
		ui.Sidebar(c, &a.section, func() {
			ui.SidebarSection(c, "Library", &a.open, func() {
				ui.SidebarItem(c, "all", nil, "All files").Children(func() {
					ui.Badge(c, "12")
				})
				ui.SidebarItem(c, "recent", nil, "Recent")
			})
		}).Width(220).Changed()

		// The detail pane fills the rest.
		ui.Column(c).Grow(1).MinWidth(0).Background(t.Background).Children(func() {
			a.list.Selected = &a.chosen
			a.list.Key = func(i int) any { return a.files[i] }
			ui.List(c, &a.list, len(a.files)).Grow(1).Rows(func(row ui.ListRow) {
				text := ui.Text(row.Context, a.files[row.Index]).Padding(6, 12)
				if row.Selected() {
					text.TextColor(t.Accent)
				}
			})
		})
	})
}

func main() {
	a := &app{open: true, files: []string{"a.txt", "b.txt"}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Files",
			Width:   800,
			Height:  600,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
```

Key points in the sample:

- `ui.View` — `func View(view func(c *Context)) *Content` (api_gen.go:2116) — is what
  goes in `mygo.WindowOptions.Content` (`window.go:162`, `docs/windows.md:25`).
- `ui.Row(c).Fill().AlignItems(ui.Stretch)` + `Width(220)` on the sidebar +
  `Column(c).Grow(1).MinWidth(0)` on the pane is the standard split. `MinWidth(0)` is
  required or the pane's intrinsic width can push the sidebar out
  (see `examples/code-editor/main.go:68-84` for the same shape).
- The sidebar already draws `Theme.Surface`; the pane draws `Theme.Background`.
- `ListState.Key` is set before `ui.List` (ui/list.go:35-36). `Rows` (the deferred
  form) is used so the row can style from `row.Selected()`.

A second, wider sample exercising `c.SetTheme`, `ItemKey`+`Selection` multi-select,
`TrackScroll`, `AlertDialog`, `Modal`, `Tabs`, `Segmented`, `Progress` and `Meter` was
also built clean against v0.4.8; the pattern in `ports` (`internal/app/app.go:164-180`,
`internal/app/theme.go:26-74`, `internal/app/view_header.go:13-29`) is the production
reference for a full app of this shape.

---

## 13. File:line index

| Symbol | Location |
|---|---|
| `type Theme struct` | ui/theme.go:7 |
| `func (t *Theme) Space` | ui/theme.go:61 |
| `func (t *Theme) Rem` | ui/theme.go:91 |
| `func LightTheme` | ui/theme.go:106 |
| `func DarkTheme` | ui/theme.go:132 |
| `type Color struct` | ui/color.go:17 |
| `func Hex` / `RGB` / `RGBA` / `Alpha` / `Mix` / `Over` | ui/color.go:33 / 26 / 29 / 93 / 105 / 120 |
| `func Oklch` | ui/widecolor.go:18 |
| `(*Context).Theme` | ui/api_gen.go:420, impl ui/context.go:297 |
| `(*Context).SetTheme` | ui/api_gen.go:387, impl ui/context.go:301 |
| `(*Context).Preferences` | ui/api_gen.go:356, ui/preferences.go:7 |
| `(*Context).Toast` / `ToastAction` / `AddToast` / `CloseToast` | ui/api_gen.go:456 / 471 / 228 / 273 |
| `(*Context).Vibrancy` | ui/api_gen.go:497, ui/context.go:344 |
| `mygo.Theme` (`IsDark`/`SetSource`/`OnUpdated`) | modules.go:237 / 240 / 258 / 289 |
| `type ListState` | ui/list.go:38 |
| `func List` | ui/list_builder.go:22 |
| `Element.ItemKey` / `Selection` / `Rows` | ui/list_builder.go:41 / 49 / 57 |
| `type ListRow` | ui/list_builder.go:4 |
| `(*ListState).ScrollTo`/`ScrollIntoView`/`ScrollToEnd`/`Visible`/`AtEnd` | ui/list.go:164 / 170 / 176 / 182 / 192 |
| `(*ListState).Focus`/`FocusWithin`/`Focused`/`Shortcut` | ui/api_gen.go:21 / 32 / 44 / 56 |
| `type Selection[K]` | ui/selection.go:23 |
| `type Selector` | ui/selection.go:62 |
| `func Table` | ui/api_gen.go:1750, ui/table.go:142 |
| `type TableColumn` | ui/table.go:9 |
| `type SortOrder` / `TableLayout` | ui/table.go:38 / 46 |
| `func Scroll`/`ScrollHorizontal`/`ScrollBoth` | ui/api_gen.go:1435 / 1455 / 1446 |
| `type ScrollState` | ui/scroll.go:7 |
| `(*ScrollState).ScrollTo`/`Scrolling` | ui/scroll.go:41 / 53 |
| `Element.TrackScroll`/`ScrollbarInsets`/`ScrollIntoView` | ui/api_gen.go:4784 / 4424 / 4392 |
| `func Sidebar`/`SidebarItem`/`SidebarSection` | ui/api_gen.go:1565 / 1575 / 1587 |
| `coreSidebar`/`coreSidebarItem`/`coreSidebarSection` | ui/sidebar.go:41 / 164 / 103 |
| `func Segmented`/`SegmentedBase`/`SegmentedParts.Segment` | ui/api_gen.go:1487 / 1500 / 5001 |
| `func Tabs`/`TabsBase`/`TabsParts.Tab` | ui/api_gen.go:1770 / 1796 / 5065 |
| `func Badge` / `coreBadge` | ui/api_gen.go:615 / ui/sidebar.go:224 |
| `func Progress` / `coreProgress` | ui/api_gen.go:1306 / ui/widgets.go:447 |
| `func Meter` / `MeterLevels` / `coreMeter` | ui/api_gen.go:1160 / ui/indicators.go:47 / 57 |
| `type Toast` | ui/toast.go:21 |
| `func ToastViewportBase`/`ToastBase` | ui/api_gen.go:1949 / 1918 |
| `type ToastParts` | ui/api_gen.go:5246 |
| `func AlertDialog` / `AlertDismissed` | ui/api_gen.go:567 / ui/feedback.go:154 |
| `func Modal` / `DialogBase` | ui/api_gen.go:1171 / 873 |
| `Element.Children` / `Key` | ui/frame.go:122 / 131 |
| `(*Context).Key` | ui/frame.go:49 |
| `Element.Fill`/`FillWidth`/`FillHeight` | ui/api_gen.go:3132 / 3154 / 3143 |
| `Element.Width`/`Height`/`Size`/`Grow`/`Shrink` | ui/api_gen.go:4943 / 3454 / 4558 / 3406 / 4535 |
| `Element.Padding`/`PaddingX`/`PaddingY`/`Margin` | ui/api_gen.go:4032 / 4043 / 4054 / 3791 |
| `Element.Gap`/`GapX`/`GapY` | ui/api_gen.go:3334 / 3345 / 3356 |
| `Element.Justify`/`AlignItems`/`AlignSelf`/`Center`/`Reverse` | ui/api_gen.go:3592 / 2208 / 2221 / 2536 / 4197 |
| `type Align` | ui/element.go:15 |
| `Element.Background`/`TextColor`/`Border`/`Radius`/`Shadow` | ui/api_gen.go:2407 / 4681 / 2455 / 4157 / 4509 |
| `type Material` | ui/material.go:8 |
| `Element.Changed`/`Clicked`/`Submitted`/`Dismissed`/`Hovered` | ui/api_gen.go:2549 / 2587 / 4633 / 2854 / 3493 |
| `func View` | ui/api_gen.go:2116 |
