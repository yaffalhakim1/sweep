package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"sweep/internal/scan"
)

// footer is the action bar: what is chosen, and what can be done with it.
//
// The left side grows and shrinks so its text truncates in a narrow
// window, while the actions keep their size and stay on screen. A summary
// belongs beside the action it describes, and the header already carries
// the scan's state.
func (a *App) footer(c *ui.Context) {
	t := c.Theme()
	if a.removing {
		a.removingFooter(c)
		return
	}
	chosen, total := a.selection()
	ui.Row(c).AlignItems(ui.Center).Gap(12).Padding(pageMargin, 13).
		Background(t.Background).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).
		Children(func() {
			ui.Column(c).Gap(1).Grow(1).Shrink(1).MinWidth(0).Children(func() {
				if len(chosen) == 0 {
					ui.Text(c, "Nothing chosen yet").FontSize(13).
						TextColor(t.TextMuted).SingleLine().Ellipsis("…")
					return
				}
				ui.Text(c, fmt.Sprintf("%s · %s",
					countLabel(len(chosen)), scan.HumanBytes(total))).
					FontSize(13).FontWeight(600).SingleLine().Ellipsis("…")
				if note := a.selectionNote(chosen); note != "" {
					ui.Text(c, note).FontSize(11).TextColor(t.TextMuted).
						SingleLine().Ellipsis("…")
				}
			})

			// Reveal is offered only for a single choice, because the
			// system's file manager can only show one thing.
			if path, ok := a.revealTarget(); ok {
				b := ui.Button(c, "Reveal").Padding(controlPad, controlPadH)
				b.Tooltip("Show this folder in the file manager")
				if b.Clicked() {
					a.Reveal(path)
				}
			}

			clean := ui.PrimaryButton(c, "Remove…").Padding(controlPad, controlPadH)
			clean.Disabled(len(chosen) == 0)
			clean.Tooltip(cleanTooltip(len(chosen), total))
			if clean.Clicked() && len(chosen) > 0 {
				a.confirm = true
			}
		})

	a.confirmDialog(c, chosen, total)
}

// removingFooter is the footer while a removal runs. It replaces the
// actions with what the removal is doing, so the window cannot be asked to
// start a second one, and the progress is a count rather than a spinner
// alone: deleting several gigabytes takes long enough that "which one is
// it on" is the question the user actually has.
func (a *App) removingFooter(c *ui.Context) {
	t := c.Theme()
	done, total := a.removed, a.total
	ui.Row(c).AlignItems(ui.Center).Gap(12).Padding(pageMargin, 13).
		Background(t.Background).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).
		Children(func() {
			ui.Spinner(c).Size(16, 16).Label("Removing build output")
			ui.Column(c).Gap(3).Grow(1).Shrink(1).MinWidth(0).Children(func() {
				ui.Textf(c, "Removing %d of %d…", done, total).
					FontSize(13).FontWeight(600).SingleLine().Ellipsis("…")
				if total > 0 {
					ui.Progress(c, float64(done)/float64(total)).
						Height(4).FillWidth().Label("Removal progress")
				}
			})
		})
}

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
	case costly == len(chosen):
		return "Rebuilding these will take a while"
	case costly > 0:
		return fmt.Sprintf("%d of them will take a while to rebuild", costly)
	default:
		return ""
	}
}

// confirmDialog asks before removing anything. Removal is permanent, so
// this is the one confirmation in the app: the guard against a slip, not a
// ritual around every action.
func (a *App) confirmDialog(c *ui.Context, chosen []scan.Artifact, total int64) {
	if !a.confirm {
		return
	}
	title := fmt.Sprintf("Remove %s?", countLabel(len(chosen)))
	message := fmt.Sprintf("This permanently deletes %s of build output from %d folders. "+
		"Your source code is not touched, but the next build will take longer.",
		scan.HumanBytes(total), len(chosen))
	choice := ui.AlertDialog(c, &a.confirm, title, message, "Cancel", "Remove")
	if choice == 1 {
		a.Clean()
	}
}
