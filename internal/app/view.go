package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"sweep/internal/scan"
)

// Layout metrics. The window is a single column of cards sharing one
// margin, so every card's edge lines up with every other's, and the
// header and footer share it too.
const (
	pageMargin  = 20
	pagePad     = 18
	cardGap     = 14
	cardPad     = 18
	controlPad  = 7
	controlPadH = 12
)

// viewFn builds the window from the app's state. It runs on the main
// thread for every frame, so it only reads state and asks for actions; the
// scan that fills that state runs elsewhere.
func (a *App) viewFn(c *ui.Context) {
	a.applyTheme(c)
	t := c.Theme()

	ui.Column(c).Fill().Background(t.Background).Children(func() {
		a.header(c)
		// The content column is given an explicit width rather than
		// filling: a scroll container has no definite width of its own, so
		// a child that fills inside it collapses to nothing. Sizing from
		// the window keeps every card the same width as the header.
		width, _ := c.Size()
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Width(width).Padding(pageMargin, pagePad).
				Gap(cardGap).Children(func() {
				a.capacity(c)
				a.toolbar(c)
				switch {
				case len(a.shown) == 0:
					a.empty(c)
				case a.view() == ViewMap:
					a.mapView(c)
				default:
					a.listView(c)
				}
			})
		})
		a.footer(c)
	})
}

// applyTheme installs the theme for the current appearance. It is called
// at the top of the view, so a change of appearance takes effect on the
// next frame.
func (a *App) applyTheme(c *ui.Context) {
	c.SetTheme(theme(a.appearanceOf(c)))
}

// appearanceOf resolves the appearance to show: the user's choice, or what
// the desktop asks for when the choice is to follow it.
func (a *App) appearanceOf(c *ui.Context) bool {
	if a.themeInit {
		return a.theme
	}
	// On the first frame the app follows the desktop, so the switch beside
	// the title starts from whatever the system is showing and the first
	// press goes to the other one rather than to the same.
	a.theme = c.Theme().Dark
	a.lastDark = a.theme
	a.themeInit = true
	return a.theme
}

// ToggleTheme switches between the light and the dark appearance. It is the
// one press a day-to-day user needs, so it is a button rather than a menu.
func (a *App) ToggleTheme() {
	if !a.themeInit {
		a.theme = a.lastDark
	}
	a.theme = !a.theme
	a.themeInit = true
	a.saveSettings()
}

// header is the window's top bar: what the app is, where it is looking,
// and the controls that act on the whole window.
func (a *App) header(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(pageMargin, 13).
		Background(t.Background).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).
		Children(func() {
			ui.Icon(c, iconDisk).Size(17, 17).TextColor(t.Accent)
			ui.Text(c, "Sweep").FontSize(16).FontWeight(600).Shrink(0)
			ui.Text(c, a.rootLabel()).FontSize(13).TextColor(t.TextMuted).
				SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
			ui.Spacer(c)
			a.diskButton(c)
			a.scanButton(c)
			a.themeButton(c)
		})
}

// diskButton chooses what to scan, beside the button that rescans it. It
// carries the current target as its label, so the header reads
// "Home · C:\Users\me  [Scan ▾] [Rescan]", and the menu offers every
// mounted volume plus a folder of the user's own choosing.
func (a *App) diskButton(c *ui.Context) {
	label := "Scan"
	b := ui.MenuButton(c, label, func(m *ui.Menu) {
		// The menu's build callback runs when the menu opens, which is the
		// right moment to re-read the drives: a disk plugged in since
		// launch appears here, and the cost is one enumeration per open
		// rather than one per frame.
		a.LoadVolumes()
		for _, v := range a.volumes {
			name := v.Friendly()
			if m.Item(name).Checked(a.scansVolume(v.Root)).Chosen() {
				a.ChooseVolume(v.Root)
			}
		}
		if len(a.volumes) > 0 {
			m.Separator()
		}
		if m.Item("Home folder").Checked(len(a.roots) == 0).Chosen() {
			a.ResetRoot()
		}
		m.Separator()
		if m.Item("Choose a folder…").Chosen() {
			a.ChooseRoot()
		}
	}).Padding(controlPad, controlPadH)
	b.Tooltip("Choose a disk or folder to scan")
}

// scansVolume reports whether the scan is looking at a whole volume, so
// the picker can mark the current one.
func (a *App) scansVolume(root string) bool {
	return len(a.roots) == 1 && a.roots[0] == root
}

// scanButton re-reads the disk on demand, and says that a scan is running
// so a manual refresh has a visible result even when nothing changed.
func (a *App) scanButton(c *ui.Context) {
	label := "Rescan"
	if a.scanning {
		label = "Scanning…"
	}
	b := ui.Button(c, label).Padding(controlPad, controlPadH).Disabled(a.scanning)
	b.Tooltip("Search again for build output that can be removed")
	if b.Clicked() {
		a.StartScan()
	}
}

// themeButton switches between the light and the dark appearance in one
// press, which is what a day-to-day user needs; a menu of three choices
// costs two presses for the same result. The icon shows the appearance in
// use, so the button reads as a state rather than only as an action.
func (a *App) themeButton(c *ui.Context) {
	t := c.Theme()
	dark := a.themeInit && a.theme
	label := "Switch to the dark appearance"
	icon := iconMoon
	if dark {
		label = "Switch to the light appearance"
		icon = iconSun
	}
	b := ui.ButtonBase(c).Padding(controlPad, 9).Radius(8).Focusable().Shrink(0).
		Background(t.Surface).Border(1, t.Border).
		Label(label).Tooltip(label)
	if b.Clicked() {
		a.ToggleTheme()
	}
	b.Children(func() {
		ui.Icon(c, icon).Size(14, 14)
	})
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
}

// toolbar carries the one line that answers "how much is there", and the
// switch between the map and the list.
func (a *App) toolbar(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).AlignItems(ui.Center).Gap(12).Children(func() {
		ui.Column(c).Gap(1).Shrink(1).MinWidth(0).Children(func() {
			ui.Text(c, a.summary()).FontSize(15).FontWeight(600).
				SingleLine().Ellipsis("…")
			if a.status != "" {
				ui.Text(c, a.status).FontSize(12).TextColor(t.TextMuted).
					SingleLine().Ellipsis("…")
			} else if age := a.age(); age != "" {
				ui.Text(c, "Scanned "+age).FontSize(12).TextColor(t.TextMuted)
			}
		})
		ui.Spacer(c)
		if len(a.found) > 0 {
			ui.SearchField(c, &a.query).Width(200).Placeholder("Filter").
				OnChange(a.refresh)
		}
		seg := ui.Segmented(c, &a.viewIdx, "Map", "List").Label("View")
		if seg.Changed() {
			a.saveSettings()
		}
	})
}

// empty is what the window shows when a scan found nothing, which is also
// where a failed scan is reported. It says what was searched and offers a
// way to search somewhere else, rather than only reporting emptiness.
func (a *App) empty(c *ui.Context) {
	t := c.Theme()
	title := "Nothing to clean"
	detail := "No build output was found in " + a.rootLabel() + "."
	if a.scanning {
		title = "Scanning…"
		detail = fmt.Sprintf("%s folders read, %d found so far.",
			withCommas(a.visited), a.foundSoFar)
	} else if a.lastErr != nil {
		title = "The scan failed"
		detail = a.lastErr.Error()
	}
	ui.Column(c).Padding(cardPad, 44).Gap(10).Center().Radius(t.Radius).
		Background(t.Surface).Border(1, t.Border).Children(func() {
		ui.Icon(c, iconCheck).Size(26, 26).TextColor(t.Success)
		ui.Text(c, title).FontSize(16).FontWeight(600)
		ui.Text(c, detail).FontSize(13).TextColor(t.TextMuted).TextAlign(ui.Center)
		ui.Row(c).Gap(8).Margin(0, 8, 0, 0).Children(func() {
			if b := ui.Button(c, "Choose a folder…"); b.Clicked() {
				a.ChooseRoot()
			}
			if len(a.roots) > 0 {
				if b := ui.Button(c, "Use my home folder"); b.Clicked() {
					a.ResetRoot()
				}
			}
		})
	})
}

// card is the shared surface every panel sits on, so the window reads as
// one column of cards rather than a set of unrelated boxes. It fills the
// width of the content column, which is what makes every card the same
// width as the header above them.
func card(c *ui.Context, build func()) ui.Element {
	t := c.Theme()
	return ui.Column(c).FillWidth().Padding(cardPad).Gap(12).Radius(t.Radius).
		Background(t.Surface).Border(1, t.Border).Children(build)
}

// timeLabel is the newest modification inside an artifact as a short
// phrase: it is what tells a project in use from one left behind.
func timeLabel(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < 24*time.Hour:
		return "today"
	case d < 48*time.Hour:
		return "yesterday"
	case d < 30*24*time.Hour:
		return plural(int(d.Hours()/24), "day ago", "days ago")
	case d < 365*24*time.Hour:
		return plural(int(d.Hours()/(24*30)), "month ago", "months ago")
	default:
		return plural(int(d.Hours()/(24*365)), "year ago", "years ago")
	}
}

// artifactLabel is what a tile or row calls an artifact: the folder name,
// which is the part that identifies it.
func artifactLabel(a scan.Artifact) string {
	if a.Name != "" {
		return a.Name
	}
	return a.Path
}
