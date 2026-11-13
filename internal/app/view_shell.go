package app

import "github.com/egoist/mygo/ui"

// The window is Onevra's sidebar Shell, which builds its own chrome and calls
// these three builders for the regions it does not own. Keeping the split here
// means the app never has to know how the window is drawn, and a test can call
// the same builders without a window.

// ViewLeft builds the navigation column. It is the Shell's Left region.
func (a *App) ViewLeft(c *ui.Context) { a.sidebar(c) }

// ViewContent builds the page area. It is the Shell's Content region.
func (a *App) ViewContent(c *ui.Context) { a.content(c) }

// ViewLeftHeader builds the trailing controls of the sidebar's header. The
// appearance switch is drawn with the app's mark, so this region stays empty;
// it is kept so the Shell's option set is complete and a future window-wide
// control has an obvious home.
func (a *App) ViewLeftHeader(c *ui.Context) {}

// Dark reports the appearance the app is showing, so the native window chrome
// can follow the same choice the drawn content does.
func (a *App) Dark() bool { return a.themeInit && a.theme }

// viewFn builds the whole window as one row. It is not what the Shell calls; it
// exists so the view can be exercised without a window, in tests and in the
// rendered screenshots, with the same two regions the Shell would draw.
//
// The sidebar width is the same constant the Shell is asked for, so a rendered
// frame matches the real window instead of approximating it.
func (a *App) viewFn(c *ui.Context) {
	a.applyTheme(c)
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Width(sidebarWidth).FillHeight().Shrink(0).
			BorderWidth(0, 1, 0, 0).BorderColor(t.Border).
			Children(func() { a.sidebar(c) })
		ui.Column(c).Grow(1).MinWidth(0).FillHeight().Children(func() { a.content(c) })
	})
}
