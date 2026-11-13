package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"github.com/oneo-me/onevra/source/shells/sidebar"

	"sweep/internal/scan"
)

// Layout metrics. The window is a sidebar and a content column; the content
// column pads its pages by pagePad and every panel inside a page shares that
// margin, so a card's edge lines up with the page title above it.
const (
	pagePad   = 24
	cardPad   = 20
	cardGap   = 16
	rowHeight = 56
	// sidebarWidth is the navigation column's width. It is a Shell option as
	// well; this is the value the app asks for.
	sidebarWidth = 248
)

// viewFn builds the window from the app's state. It runs on the main thread
// for every frame, so it only reads state and asks for actions; the scan that
// fills that state runs elsewhere.
//
// The window itself is a Onevra sidebar Shell (see main.go), which installs
// the theme before this runs and gives the app its navigation column and
// floating header. view_shell.go holds the builders it calls, and the
// standalone composition the tests render.
// applyTheme installs the theme for the current appearance. It is called at
// the top of the view, so a change of appearance takes effect on the next
// frame.
func (a *App) applyTheme(c *ui.Context) {
	c.SetTheme(themeFor(c, a.appearanceOf(c)))
}

// appearanceButton switches between the light and the dark appearance in one
// press, which is what a day-to-day user needs; a menu of three choices costs
// two presses for the same result. The icon shows the appearance in use, so
// the button reads as a state rather than only as an action.
//
// It lives beside the app's mark rather than in the page, because it is about
// the window and not about the disk, and it must not scroll away.
func (a *App) appearanceButton(c *ui.Context) {
	dark := a.themeInit && a.theme
	label := "Switch to the dark appearance"
	icon := lucide.Moon()
	if dark {
		label = "Switch to the light appearance"
		icon = lucide.Sun()
	}
	b := components.IconButton(c, icon, label, components.IconButtonPlain)
	if b.Clicked() {
		a.ToggleTheme()
	}
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

// sidebar is the navigation column: the app's mark, one item per page with
// the space it would reclaim, and the disk's own capacity at the bottom.
//
// It is the Shell's Left region, so it is built below the window's drag strip
// and above the resize handle.
func (a *App) sidebar(c *ui.Context) {
	t := c.Theme()
	// The Shell leaves its sidebar transparent when the window has a native
	// material behind it, so this column paints its own surface. Without it
	// the page's rows show through wherever the navigation does not paint,
	// which reads as the window being broken.
	ui.Column(c).FillWidth().FillHeight().Background(t.Surface).Children(func() {
		// The mark sits under the window controls, which the Shell keeps
		// clear above this region.
		ui.Row(c).FillWidth().Padding(sidebar.HeaderHeight(c)+4, 16, 12, 16).
			Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
			a.mark(c)
			ui.Column(c).Gap(0).Grow(1).Shrink(1).MinWidth(0).Children(func() {
				components.Text(c, "Sweep").FontSize(15).Bold().SingleLine()
				components.Text(c, "Clean & purge").FontSize(11).TextColor(t.TextMuted).SingleLine()
			})
			a.appearanceButton(c)
		})
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).FillWidth().Padding(0, t.Space(2.5), t.Space(2), t.Space(2.5)).
				Gap(t.Space(0.5)).Children(func() {
				a.navItem(c, PageMap, "Disk space map", lucide.Grid2x2(), 0)
				a.navItem(c, PageArtifacts, "Build artifacts", lucide.Boxes(), a.totalShown())
				a.navItem(c, PageCaches, "Package caches", lucide.Database(), a.cacheBytes())
				a.navItem(c, PageSettings, "Settings & rules", lucide.Settings(), 0)
			})
		})
		a.storageCard(c)
	})
}

// mark is the app's icon: a rounded square in the accent with a brush on it.
// It is drawn rather than loaded so it inherits the theme's accent.
func (a *App) mark(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).Size(34, 34).Shrink(0).Radius(9).Center().Background(t.Accent).Children(func() {
		components.Icon(c, lucide.BrushCleaning()).Size(18, 18).TextColor(t.AccentText)
	})
}

// navItem is one row of the navigation column. The chosen page is a filled
// capsule in the accent, which is the one strong colour in the chrome; the
// others are quiet until hovered.
//
// The trailing count is the page's own answer to "how much": the space it
// would reclaim, not a number of rows, because that is what decides whether
// the page is worth opening.
func (a *App) navItem(c *ui.Context, page Page, label string, icon *ui.SVG, bytes int64) {
	t := c.Theme()
	chosen := a.page == page
	row := ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(t.Space(2.5)).
		Padding(t.Space(2), t.Space(2.5)).Radius(t.Radius).Cursor(ui.CursorPointer)
	switch {
	case chosen:
		row.Background(t.Accent)
	case row.Hovered():
		row.Background(t.Text.Alpha(0.06))
	}
	row.OnClick(func() { a.goTo(page) })
	row.Children(func() {
		iconColor := t.TextMuted
		textColor := t.Text
		if chosen {
			iconColor, textColor = t.AccentText, t.AccentText
		}
		components.Icon(c, icon).Size(17, 17).Shrink(0).TextColor(iconColor)
		components.Text(c, label).FontSize(13).Grow(1).Shrink(1).MinWidth(0).
			SingleLine().Ellipsis("…").TextColor(textColor)
		if bytes > 0 {
			// The count is the page's own total, in a pill whose face is
			// derived from the row's: on the chosen row it is a lighter wash
			// of the accent text, so the capsule stays one object.
			bg, fg := t.Text.Alpha(0.07), t.TextMuted
			if chosen {
				bg, fg = t.AccentText.Alpha(0.20), t.AccentText
			}
			ui.Box(c).Shrink(0).Padding(1, 7).Radius(999).Background(bg).Children(func() {
				components.Text(c, scan.HumanBytes(bytes)).FontSize(11).FontWeight(500).
					SingleLine().TextColor(fg)
			})
		}
	})
	row.Label(label)
}

// goTo changes page. It is the only place the page changes, so a page's own
// controls can never leave the sidebar showing the wrong item.
func (a *App) goTo(page Page) {
	if a.page == page {
		return
	}
	a.page = page
	a.saveSettings()
}

// storageCard is the disk's capacity, at the bottom of the column where it is
// always visible: the app exists because the disk is full, so the number that
// says how full belongs in the chrome rather than behind a page.
func (a *App) storageCard(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).FillWidth().Padding(t.Space(2.5), pagePad, pagePad, pagePad).Children(func() {
		ui.Column(c).FillWidth().Padding(t.Space(2.5)).Gap(t.Space(1.5)).
			Radius(radiusCard).Background(t.Text.Alpha(0.04)).Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(t.Space(2)).Children(func() {
				components.Icon(c, lucide.HardDrive()).Size(15, 15).Shrink(0).TextColor(t.TextMuted)
				components.Text(c, a.diskLabel()).FontSize(12).FontWeight(600).
					Grow(1).Shrink(1).MinWidth(0).SingleLine().Ellipsis("…")
				if b := components.Button(c, "", components.ButtonGhost); false {
					_ = b
				}
			})
			if !a.haveDisk {
				components.Text(c, "Capacity unavailable").FontSize(11).TextColor(t.TextMuted)
				return
			}
			used := float64(a.volume.UsedRatio())
			track := ui.Box(c).FillWidth().Height(6).Radius(3).Background(t.Text.Alpha(0.10)).
				Clip()
			track.Draw(func(p *ui.Painter, r ui.Rect) {
				if r.W <= 0 {
					return
				}
				p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * float32(used), H: r.H}, a.capacityColor(c, used), 3)
			})
			ui.Textf(c, "%s free of %s",
				scan.HumanBytes(a.volume.Free), scan.HumanBytes(a.volume.Total)).
				FontSize(11).TextColor(t.TextMuted).SingleLine()
		})
	})
}

// diskLabel names the disk the scan is on, falling back to the scan target
// when the volume is unknown.
func (a *App) diskLabel() string {
	if a.haveDisk && a.volume.Label != "" {
		return a.volume.Label
	}
	return a.rootLabel()
}

// capacityColor is the meter's colour: success while there is room, warning
// past 80%, danger past 95%. It is the same signal the system's own storage
// pane gives, so the colour is never the only thing that says "full".
func (a *App) capacityColor(c *ui.Context, ratio float64) ui.Color {
	t := c.Theme()
	switch {
	case ratio >= 0.95:
		return t.Danger
	case ratio >= 0.80:
		return t.Warning
	default:
		return t.Success
	}
}

// cacheBytes is the space the cache page would reclaim, for its badge. It
// counts every cache the scan found, not the filtered subset: the badge is a
// property of the disk, and a page whose badge said 0 B while its empty state
// offered to reset a filter would be describing two different things.
func (a *App) cacheBytes() int64 { return scan.Total(scan.Caches(a.found)) }

// cacheEmptyNote explains an empty cache page in the terms that are true: the
// scan may have found no caches at all, or the size filter may be hiding them,
// and those are different answers.
func (a *App) cacheEmptyNote() string {
	if len(scan.Caches(a.found)) > 0 {
		return "The caches on this disk are below the minimum size. Lower it to see them."
	}
	return "Nothing cached: every artifact on this disk is build output."
}

// content is the page area: a fixed title row, the page itself, and a fixed
// action bar. Only the middle scrolls, so the title and the primary action
// never leave the screen while a long list does.
// content is the page area: a fixed title row, the page itself, and a fixed
// action bar. Only the middle scrolls, so the title and the primary action
// never leave the screen while a long list does.
func (a *App) content(c *ui.Context) {
	t := c.Theme()
	// The Shell's floating header overlays this region, so the page owns its
	// top inset. HeaderHeight is the clearance the Shell itself uses, which
	// keeps the title below the window controls on every platform.
	top := sidebar.HeaderHeight(c) + 10
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).FillWidth().Padding(top, pagePad, 0, pagePad).Children(func() {
			a.pageHeader(c)
		})
		ui.Column(c).FillWidth().Grow(1).MinHeight(0).Padding(0, pagePad, 0, pagePad).
			Children(func() {
				a.pageBody(c)
			})
		a.footerBar(c)
	})
}

// pageHeader is the page's title, its description, and the actions that act
// on the whole page. It is the same shape on every page, so the primary
// action is always in the same place.
func (a *App) pageHeader(c *ui.Context) {
	t := c.Theme()
	title, subtitle := a.pageTitle()
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(cardGap).Padding(0, 0, cardGap, 0).
		Shrink(0).Children(func() {
		ui.Column(c).Gap(2).Grow(1).Shrink(1).MinWidth(0).Children(func() {
			components.Text(c, title).FontSize(24).Bold().SingleLine().Ellipsis("…")
			components.Text(c, subtitle).FontSize(12).TextColor(t.TextMuted).
				SingleLine().Ellipsis("…")
		})
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Shrink(0).Children(func() {
			a.viewSwitch(c)
			a.primaryAction(c)
		})
	})
}

// footerBar is the window's action bar, at the bottom of the page area. It is
// outside the scroll so the selection total and the action that uses it are
// always visible, which is what makes a long list safe to read.
func (a *App) footerBar(c *ui.Context) {
	if a.removing {
		a.removingFooter(c)
		return
	}
	t := c.Theme()
	chosen, total := a.selection()
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(t.Space(3)).
		Padding(t.Space(3), pagePad).Shrink(0).
		BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(t.Space(2)).Grow(1).Shrink(1).MinWidth(0).
			Children(func() {
				icon, color := lucide.ShieldCheck(), t.Success
				if len(chosen) > 0 {
					icon, color = lucide.Eraser(), t.TextMuted
				}
				components.Icon(c, icon).Size(15, 15).Shrink(0).TextColor(color)
				components.Text(c, a.footNote(len(chosen), total)).
					FontSize(12).TextColor(t.TextMuted).SingleLine().Ellipsis("…")
			})
		if reveal, ok := a.revealTarget(); ok {
			if b := components.Button(c, "Reveal", components.ButtonSecondary); b.Clicked() {
				a.Reveal(reveal)
			}
		}
		b := components.Button(c, a.reclaimLabel(len(chosen), total), components.ButtonPrimary)
		b.Disabled(len(chosen) == 0 || a.removing)
		b.Tooltip(cleanTooltip(len(chosen), total))
		if b.Clicked() && len(chosen) > 0 {
			a.confirm = true
		}
	})
	a.confirmDialog(c, chosen, total)
}

// removingFooter is the footer while a removal runs. It replaces the actions
// with what the removal is doing, so the window cannot be asked to start a
// second one, and the progress is a count rather than a spinner alone:
// deleting several gigabytes takes long enough that "which one is it on" is
// the question the user actually has.
func (a *App) removingFooter(c *ui.Context) {
	t := c.Theme()
	done, total := a.removed, a.total
	ui.Column(c).FillWidth().Gap(t.Space(1.5)).
		Padding(t.Space(3), pagePad).Shrink(0).
		BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
		ui.Textf(c, "Removing %d of %d…", done, total).
			FontSize(12).FontWeight(600).SingleLine().Ellipsis("…")
		if total > 0 {
			removalDots(c, done, total)
		}
	})
}

// pageTitle is what each page is called and what it answers.
func (a *App) pageTitle() (string, string) {
	switch a.page {
	case PageMap:
		return "Disk space map", a.scanNote("area is size, colour is build system")
	case PageCaches:
		return "Package caches", a.scanNote("regenerated by the next run, not the next build")
	case PageSettings:
		return "Settings & rules", "What Sweep looks for, and where it looks"
	default:
		return "Build artifacts", a.scanNote("largest first, grouped by build system")
	}
}

// scanNote joins a page's own description to the state of the scan, so every
// page says where its numbers come from.
func (a *App) scanNote(what string) string {
	switch {
	case a.scanning:
		return fmt.Sprintf("Scanning %s · %s folders read", a.rootLabel(), withCommas(a.visited))
	case a.lastErr != nil:
		return "Scan failed: " + a.lastErr.Error()
	}
	return what
}

// viewSwitch is the map/list control. It is only meaningful on the pages that
// present the same data two ways, so it is absent elsewhere rather than
// disabled: a control that cannot act is noise.
func (a *App) viewSwitch(c *ui.Context) {
	if a.page != PageMap && a.page != PageArtifacts {
		return
	}
	t := c.Theme()
	segments := []struct {
		page  Page
		label string
		icon  *ui.SVG
	}{
		{PageMap, "Map", lucide.Grid2x2()},
		{PageArtifacts, "List", lucide.Rows3()},
	}
	ui.Row(c).Padding(3).Gap(2).Radius(t.Radius + 2).Background(t.Text.Alpha(0.06)).
		Children(func() {
			for _, seg := range segments {
				on := a.page == seg.page
				b := ui.ButtonBase(c).Padding(t.Space(1), t.Space(2)).Radius(t.Radius).
					Gap(t.Space(1.5)).AlignItems(ui.Center).Cursor(ui.CursorPointer)
				switch {
				case on:
					b.Background(t.Surface).TextColor(t.Text)
				default:
					b.TextColor(t.TextMuted)
					if b.Hovered() {
						b.TextColor(t.Text)
					}
				}
				b.Label(seg.label)
				b.OnClick(func() { a.goTo(seg.page) })
				b.Children(func() {
					components.Icon(c, seg.icon).Size(14, 14).Shrink(0)
					components.Text(c, seg.label).FontSize(12).FontWeight(500).SingleLine()
				})
			}
		})
}

// primaryAction is the page's one main button. On the artifact pages it
// chooses everything the page shows and opens the confirmation, which is what
// "clean suggested" means: the suggestion is the current filter.
func (a *App) primaryAction(c *ui.Context) {
	switch a.page {
	case PageSettings:
		return
	case PageMap, PageArtifacts, PageCaches:
		shown := a.pageArtifacts()
		label := "Clean suggested"
		if bytes := scan.Total(shown); bytes > 0 {
			label = "Clean suggested · " + scan.HumanBytes(bytes)
		}
		// A narrow window drops the total from the label rather than letting
		// the button push the title off screen; the footer still names the
		// exact amount the action will free.
		width, _ := c.Size()
		if width-sidebarWidth-2*pagePad < 620 {
			label = "Clean"
		}
		b := components.Button(c, label, components.ButtonPrimary)
		b.Disabled(len(shown) == 0 || a.removing)
		b.Tooltip(fmt.Sprintf("Choose all %d folders on this page and review them before removing",
			len(shown)))
		if b.Clicked() {
			a.suggest()
		}
	}
}

// pageArtifacts is the artifacts the current page presents. It is the one
// definition the header, the footer and the page bodies all read, so the
// button's total cannot disagree with the rows.
func (a *App) pageArtifacts() []scan.Artifact {
	if a.page == PageCaches {
		return scan.Caches(a.shown)
	}
	return a.shown
}

// suggest chooses everything on the page and opens the confirmation, so one
// press is a complete action rather than a selection the user then has to
// finish.
func (a *App) suggest() {
	shown := a.pageArtifacts()
	if len(shown) == 0 {
		return
	}
	a.selected = map[string]bool{}
	for _, art := range shown {
		a.selected[art.Path] = true
	}
	a.confirm = true
}

// pageBody dispatches to the page. Each page is a separate file so the layout
// of one cannot quietly become the layout of another.
//
// The body fills the room the header and the footer leave. The artifacts page
// does not scroll as a whole: its table scrolls inside its own card, and a
// scroll inside a scroll is unusable. The settings page scrolls as a page. The
// map scrolls in neither direction — it is sized to fit, because the map and the
// project it describes have to be visible together.
func (a *App) pageBody(c *ui.Context) {
	ui.Column(c).FillWidth().FillHeight().Padding(0, 0, pagePad, 0).Clip().Children(func() {
		switch a.page {
		case PageMap:
			a.mapView(c)
		case PageCaches:
			a.artifactsPage(c, scan.Caches(a.shown), "Package caches", a.cacheEmptyNote())
		case PageSettings:
			a.scrollBody(c, func() { a.settingsPage(c) })
		default:
			a.artifactsPage(c, a.shown, "Build artifacts",
				"Nothing found here: raise the minimum size or scan somewhere else.")
		}
	})
}

// scrollBody wraps a page that is read from top to bottom.
func (a *App) scrollBody(c *ui.Context, build func()) {
	ui.Scroll(c).FillWidth().FillHeight().Children(func() {
		ui.Column(c).FillWidth().Gap(cardGap).Children(build)
	})
}
