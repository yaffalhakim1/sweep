package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"

	"sweep/internal/scan"
)

// timeLabel is the newest modification inside an artifact as a short phrase:
// it is what tells a project in use from one left behind.
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

// modifiedLabel is the modified column: the relative phrase, and the date
// when the artifact is old enough that "8 months ago" is less useful than the
// month itself.
func modifiedLabel(a scan.Artifact) string {
	if a.Touched.IsZero() {
		return "unknown"
	}
	if time.Since(a.Touched) > 30*24*time.Hour {
		return a.Touched.Format("2 Jan 2006")
	}
	return timeLabel(a.Touched)
}

// artifactLabel is what a tile or row calls an artifact: the folder name,
// which is the part that identifies it.
func artifactLabel(a scan.Artifact) string {
	if a.Name != "" {
		return a.Name
	}
	return a.Path
}

// plural writes a count with its noun.
func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return itoa(n) + " " + many
}

// itoa writes an int without pulling in strconv for a handful of call sites.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// countLabel writes "N folders" for a selection summary.
func countLabel(n int) string { return plural(n, "folder", "folders") }

// cleanTooltip says exactly what the action will do, including that it is
// permanent, so the button is never a surprise.
func cleanTooltip(n int, total int64) string {
	if n == 0 {
		return "Choose something to remove first"
	}
	return fmt.Sprintf("Remove %s permanently, freeing %s",
		countLabel(n), scan.HumanBytes(total))
}

// selectionNote warns about the one consequence a user cannot see from a
// size: a rebuild that will take real time.
func (a *App) selectionNote(chosen []scan.Artifact) string {
	costly := 0
	for _, art := range chosen {
		if art.Rule.Costly {
			costly++
		}
	}
	switch {
	case len(chosen) == 0:
		return ""
	case costly == len(chosen):
		return "Rebuilding these will take a while"
	case costly > 0:
		return fmt.Sprintf("%d of them will take a while to rebuild", costly)
	default:
		return ""
	}
}

// empty is what a page shows when a scan found nothing, which is also where a
// failed scan is reported. It says what was searched and offers a way to
// search somewhere else, rather than only reporting emptiness.
func (a *App) empty(c *ui.Context, note string) {
	t := c.Theme()
	title := "Nothing to clean"
	detail := "No build output was found in " + a.rootLabel() + "."
	if note != "" {
		detail = note
	}
	if a.scanning {
		title = "Scanning…"
		detail = fmt.Sprintf("%s folders read, %d found so far.",
			withCommas(a.visited), a.foundSoFar)
	} else if a.lastErr != nil {
		title = "The scan failed"
		detail = a.lastErr.Error()
	} else if len(a.found) > 0 && len(a.shown) == 0 && note == "" {
		// Only blame the filters when the scan did find something: a first run
		// with a size threshold would otherwise claim a filter is hiding
		// folders that were never there. A page with its own more specific
		// explanation keeps it.
		title = "No folders match the filters"
		detail = "Lower the minimum size, or choose a different ecosystem."
	} else if len(a.found) > 0 {
		// The scan found things, but not of the kind this page shows.
		title = "Nothing on this page"
	}
	// Each state gets the glyph that answers it: a failed search when the
	// scan failed, a field of dots while scanning, and an open box when there
	// is simply nothing to reclaim.
	var icon *ui.SVG
	color := t.Success
	switch {
	case a.scanning:
		// The field is drawn below, in place of the glyph.
	case a.lastErr != nil:
		icon, color = lucide.SearchX(), t.Danger
	default:
		icon = lucide.PackageOpen()
	}
	ui.Column(c).FillWidth().Padding(cardPad, 48).Gap(t.Space(2.5)).Center().
		Radius(radiusCard).Background(t.Surface).Border(1, t.Border).Children(func() {
		if a.scanning {
			scanDots(c)
		} else {
			components.Icon(c, icon).Size(26, 26).TextColor(color)
		}
		components.Text(c, title).FontSize(15).FontWeight(600)
		components.Text(c, detail).FontSize(12).TextColor(t.TextMuted).TextAlign(ui.Center)
		ui.Row(c).Gap(t.Space(2)).Margin(0, t.Space(2), 0, 0).Children(func() {
			// A size filter only hides rows the scan found, so the way back is
			// offered only then; otherwise the way out is another target.
			if len(a.found) > 0 && (a.minSize > 0 || a.kindFilter != "" || a.query != "") {
				if b := components.Button(c, "Reset filters", components.ButtonSecondary); b.Clicked() {
					a.resetFilters()
				}
			}
			if b := components.Button(c, "Choose a folder…", components.ButtonSecondary); b.Clicked() {
				a.ChooseRoot()
			}
			if len(a.roots) > 0 {
				if b := components.Button(c, "Use my home folder", components.ButtonGhost); b.Clicked() {
					a.ResetRoot()
				}
			}
		})
	})
}

// settingsPage is the page that answers "what does this tool actually look
// for": the rule table the walker uses, the scan target, and the appearance.
// It reads the same table the scan reads, so the page cannot describe rules
// that are not in force.
func (a *App) settingsPage(c *ui.Context) {
	t := c.Theme()
	a.targetCard(c)
	a.appearanceCard(c)
	a.rulesCard(c)
	a.safetyCard(c)
	_ = t
}

// targetCard shows and changes where the scan looks.
func (a *App) targetCard(c *ui.Context) {
	t := c.Theme()
	a.panel(c, "Scan target", "Where Sweep looks for build output.", func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(t.Space(2)).Children(func() {
			components.Icon(c, lucide.FolderOpen()).Size(16, 16).Shrink(0).TextColor(t.TextMuted)
			ui.Column(c).Gap(0).Grow(1).Shrink(1).MinWidth(0).Children(func() {
				components.Text(c, a.rootLabel()).FontSize(13).FontWeight(600).
					SingleLine().Ellipsis("…")
				components.Text(c, a.rootPath()).FontSize(11).TextColor(t.TextMuted).
					SingleLine().Ellipsis("…")
			})
			if b := components.Button(c, "Choose folder…", components.ButtonSecondary); b.Clicked() {
				a.ChooseRoot()
			}
			if len(a.roots) > 0 {
				if b := components.Button(c, "Use home", components.ButtonGhost); b.Clicked() {
					a.ResetRoot()
				}
			}
			if b := components.Button(c, "Rescan", components.ButtonSecondary); b.Clicked() {
				a.StartScan()
			}
		})
	})
}

// appearanceCard is the one setting the app has that is not about the disk.
func (a *App) appearanceCard(c *ui.Context) {
	t := c.Theme()
	a.panel(c, "Appearance", "Sweep follows the desktop until you choose.", func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(t.Space(2)).Children(func() {
			dark := a.themeInit && a.theme
			icon := lucide.Sun()
			label := "Light"
			if dark {
				icon, label = lucide.Moon(), "Dark"
			}
			components.Icon(c, icon).Size(16, 16).Shrink(0).TextColor(t.TextMuted)
			components.Text(c, label).FontSize(13).Grow(1).SingleLine()
			if b := components.Button(c, "Switch appearance", components.ButtonSecondary); b.Clicked() {
				a.ToggleTheme()
			}
		})
	})
}

// rulesCard is the rule table. Each row is a directory name, what it belongs
// to, and what proves it — the proof is the part that matters, because a name
// alone is never enough for Sweep to delete anything.
func (a *App) rulesCard(c *ui.Context) {
	t := c.Theme()
	a.panel(c, "Rules", fmt.Sprintf(
		"%d directory names, and what confirms each one.", len(scan.Rules())), func() {
		for _, rule := range scan.Rules() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(t.Space(2)).
				Padding(t.Space(1), 0).Children(func() {
				ui.Box(c).Size(8, 8).Radius(2.5).Shrink(0).
					Background(KindColor(t.Dark, rule.Kind))
				components.Text(c, ruleName(rule)).FontSize(12).FontWeight(500).
					Width(140).Shrink(0).SingleLine().Ellipsis("…")
				components.Text(c, scan.TypeLabel(scan.Artifact{Rule: rule, Kind: rule.Kind})).
					FontSize(11).TextColor(t.TextMuted).Width(140).Shrink(0).
					SingleLine().Ellipsis("…")
				components.Text(c, ruleProof(rule)).FontSize(11).
					TextColor(t.TextMuted).Grow(1).Shrink(1).MinWidth(0).
					SingleLine().Ellipsis("…")
				ui.Box(c).Width(52).Shrink(0).Children(func() {
					if rule.Cache {
						components.Badge(c, "Cache")
					}
				})
			})
		}
	})
}

// ruleName is a rule's directory name, with the wildcard a prefix rule means.
func ruleName(rule scan.Rule) string {
	if rule.Prefix {
		return rule.Name + "*"
	}
	return rule.Name
}

// ruleProof says what confirms a rule. A name that is unambiguous needs no
// proof, and saying so is more honest than an empty cell.
func ruleProof(rule scan.Rule) string {
	if rule.Verify == nil {
		return "The name is unambiguous"
	}
	return "Verified beside a project file"
}

// safetyCard states the guards, which are the app's actual promise.
func (a *App) safetyCard(c *ui.Context) {
	t := c.Theme()
	a.panel(c, "Safety", "What Sweep refuses to do, whatever is selected.", func() {
		for _, line := range []string{
			"A name alone is never enough: a generic folder needs its proof file.",
			"Volume roots, your home folder and shallow paths are refused.",
			"A link or junction is never followed, and never deleted through.",
			"A selection holding a folder and something inside it is refused outright.",
			"Removal is permanent, after a confirmation that names the size.",
		} {
			ui.Row(c).FillWidth().AlignItems(ui.Start).Gap(t.Space(2)).Children(func() {
				components.Icon(c, lucide.CircleCheck()).Size(14, 14).Shrink(0).
					TextColor(t.Success).Margin(1, 0, 0, 0)
				components.Text(c, line).FontSize(12).TextColor(t.TextMuted).
					Grow(1).Shrink(1).MinWidth(0).Wrap()
			})
		}
	})
}

// panel is the shared surface of a settings section: a title, a line saying
// what the section is for, and its content.
func (a *App) panel(c *ui.Context, title, note string, build func()) {
	t := c.Theme()
	ui.Column(c).FillWidth().Padding(cardPad).Gap(t.Space(2)).Radius(radiusCard).
		Background(t.Surface).Border(1, t.Border).Children(func() {
		ui.Column(c).Gap(1).Children(func() {
			components.Text(c, title).FontSize(13).FontWeight(600).SingleLine()
			components.Text(c, note).FontSize(11).TextColor(t.TextMuted).SingleLine().Ellipsis("…")
		})
		build()
	})
}

// confirmDialog asks before removing anything. Removal is permanent, so this
// is the one confirmation in the app: the guard against a slip, not a ritual
// around every action.
func (a *App) confirmDialog(c *ui.Context, chosen []scan.Artifact, total int64) {
	if !a.confirm {
		return
	}
	result := components.AlertDialog(c, &a.confirm, components.AlertDialogOptions{
		Title: fmt.Sprintf("Remove %s?", countLabel(len(chosen))),
		Message: fmt.Sprintf("This permanently deletes %s of build output from %d folders. "+
			"Your source code is not touched, but the next build will take longer.",
			scan.HumanBytes(total), len(chosen)),
		ConfirmLabel: "Remove",
		CancelLabel:  "Cancel",
	})
	if result == components.DialogConfirmed {
		a.Clean()
	}
}
