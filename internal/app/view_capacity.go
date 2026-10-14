package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"sweep/internal/scan"
)

// capacity shows how full the disk is, which is the question the app
// exists to answer: a meter that turns amber at 80% and red at 95% is the
// same signal the system's own storage pane gives.
func (a *App) capacity(c *ui.Context) {
	t := c.Theme()
	if !a.haveDisk {
		return
	}
	v := a.volume
	levels := &ui.MeterLevels{Warning: 0.80, Critical: 0.95}
	card(c, func() {
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Text(c, v.Label).FontSize(13).FontWeight(600).Shrink(0)
			ui.Spacer(c)
			ui.Textf(c, "%s free of %s", scan.HumanBytes(v.Free), scan.HumanBytes(v.Total)).
				FontSize(13).TextColor(t.TextMuted).Shrink(0)
		})
		ui.Meter(c, float64(v.Used()), 0, float64(v.Total), levels).
			Height(8).Fill().Label("Disk in use")
		ui.Text(c, capacityNote(v.UsedRatio())).FontSize(12).TextColor(t.TextMuted)
	})
}

// capacityNote says what the meter means, so the colour is never the only
// signal.
func capacityNote(ratio float64) string {
	switch {
	case ratio >= 0.95:
		return fmt.Sprintf("Critically full at %.0f%% — cleaning now matters", ratio*100)
	case ratio >= 0.80:
		return fmt.Sprintf("%.0f%% full — worth reclaiming space soon", ratio*100)
	default:
		return fmt.Sprintf("%.0f%% full", ratio*100)
	}
}
