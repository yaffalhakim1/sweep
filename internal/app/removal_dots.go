package app

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// removalDots is the removal's progress drawn as a row of dots that fill from
// the left, in place of a bar: the same information in the language the rest of
// the window already speaks, and a row whose next dot keeps a slow pulse, which
// is what says the removal is still running.
//
// The pulse is the only motion, and the row repaints rather than rebuilds: the
// dots come from the painter, not from the view. A removal deletes gigabytes
// and keeps the window busy the whole time, so it must not add work per frame.
func removalDots(c *ui.Context, done, total int) {
	t := c.Theme()
	still := c.Preferences().ReduceMotion

	box := ui.Box(c).FillWidth().Height(dotSize).
		// A bar of its own tells assistive technology the same thing the
		// component would: a progress indicator from 0 to the total.
		Role(ui.RoleProgress).
		Range(0, float64(total), float64(done)).
		Label("Removal progress")

	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if !still {
			p.AnimationFrame()
		}

		cols := int((r.W-dotSize)/dotPitch) + 1
		if cols < 1 {
			cols = 1
		}
		// The leftover under one pitch is split either side, so the row is
		// centred in the footer rather than ragged at its end.
		left := (r.W - (float32(cols-1)*dotPitch + dotSize)) / 2

		filled := removalFilled(done, total, cols)
		lead := float32(0)
		if !still {
			lead = removalLead(p.Now().UnixMilli(), removalBreath)
		}

		for col := 0; col < cols; col++ {
			color := t.Border
			switch {
			case col < filled:
				color = t.Accent
			case col == filled && !still:
				// The next dot to fill, breathing: this is the waiting.
				color = t.Accent.Alpha(lead)
			}
			p.Fill(ui.Rect{
				X: r.X + left + float32(col)*dotPitch,
				Y: r.Y,
				W: dotSize,
				H: dotSize,
			}, color, dotSize/2)
		}
	})
}

// removalBreath is one breath of the leading dot, in milliseconds: slow enough
// to read as waiting rather than as blinking.
const removalBreath = 1000

// removalFilled is how many of cols dots a removal has filled. A finished
// removal fills all of them and a fresh one none, so the row cannot claim a
// progress it does not have.
func removalFilled(done, total, cols int) int {
	if total <= 0 || cols <= 0 {
		return 0
	}
	if done >= total {
		return cols
	}
	filled := int(math.Round(float64(done) / float64(total) * float64(cols)))
	return max(0, min(filled, cols))
}

// removalLead is the alpha of the next dot at a moment: from 0.45 to 1, once
// per period. It never reaches zero, because a leading edge that vanishes
// would read as a gap in the progress rather than as work.
func removalLead(ms, period int64) float32 {
	if period <= 0 {
		return 1
	}
	phase := float64(ms%period) / float64(period)
	return 0.45 + 0.55*float32((math.Sin(phase*2*math.Pi)+1)/2)
}
