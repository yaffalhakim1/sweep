package app

import (
	"github.com/egoist/mygo/ui"
)

// ViewTheme is the Shell's Theme option. It installs Sweep's own palette rather
// than Onevra's terracotta one, and it resolves the appearance the app is
// showing so the window's chrome and its content always agree.
//
// The Shell calls it before building anything, which is the only point at which
// a theme can be installed for a whole frame.
func (a *App) ViewTheme(c *ui.Context) *ui.Theme {
	return themeFor(c, a.appearanceOf(c))
}

// themeFor builds the palette for an appearance and applies the desktop's
// accessibility preferences to it, which is the same adjustment MyGo's default
// theme and Onevra's make: higher contrast and a text scale must reach an app's
// own colours too, or they stop working exactly where they are needed.
func themeFor(c *ui.Context, dark bool) *ui.Theme {
	t := theme(dark)
	if c == nil {
		return t
	}
	p := c.Preferences()
	if p.HighContrast {
		t.Border = t.Border.Mix(t.Text, 0.45)
		t.TextMuted = t.TextMuted.Mix(t.Text, 0.40)
		t.Focus.A = 255
		t.Scrollbar = t.Scrollbar.Alpha(1.6)
	}
	if s := p.TextScale; s > 0 && s != 1 {
		t.FontSize *= s
	}
	return t
}
