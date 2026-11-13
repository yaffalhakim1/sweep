package app

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// The scanning indicator is a field of dots where a sparse, uneven subset
// brightens and dims, as data lands. It replaces the turning circle because
// the app's own map is a mosaic of blocks: a field of dots says "reading the
// disk" in the same language the result is drawn in.
//
// The field is as wide as the room the card gives it, so the number of
// columns is counted at paint time rather than fixed here.
const (
	dotRows  = 9
	dotPitch = 9
	dotSize  = 4

	// A dot at rest is still visible, so the field reads as a grid that is
	// being read rather than as dots appearing from nowhere.
	dotRest = 0.2
	dotPeak = 0.62
)

// The field borrows Apple's system red, blue and yellow, in the variant each
// appearance needs to stay legible on its surface. This is the one place in
// the app where hue is decoration rather than meaning, and it lasts only as
// long as a scan does.
var (
	dotRedLight    = ui.Hex("#FF3B30")
	dotRedDark     = ui.Hex("#FF453A")
	dotBlueLight   = ui.Hex("#007AFF")
	dotBlueDark    = ui.Hex("#0A84FF")
	dotYellowLight = ui.Hex("#FFCC00")
	dotYellowDark  = ui.Hex("#FFD60A")
)

// dotColor is the colour of a column. The field is three bands rather than
// three colours scattered at random, so the colour reads as a zone and the
// blinking stays the only thing that moves. cols is the field's own width in
// dots, because the field grows to whatever room the card has.
func dotColor(col, cols int, dark bool) ui.Color {
	if cols < 1 {
		cols = 1
	}
	switch col * 3 / cols {
	case 0:
		if dark {
			return dotRedDark
		}
		return dotRedLight
	case 1:
		if dark {
			return dotBlueDark
		}
		return dotBlueLight
	default:
		if dark {
			return dotYellowDark
		}
		return dotYellowLight
	}
}

// dotBlink is a dot's own rhythm: how long one blink takes, where in that
// cycle it starts, and whether it blinks at all. It is a hash of the position
// rather than a random source, so the field is the same on every frame and on
// every run, and a test can predict it.
func dotBlink(col, row int) (period time.Duration, offset float64, quiet bool) {
	h := uint32(col)*0x9E3779B1 ^ uint32(row)*0x85EBCA77
	h ^= h >> 15
	h *= 0x2545F491
	h ^= h >> 13

	// More than half the dots stay dark: a field where everything moves reads
	// as noise instead of as a disk being read.
	quiet = h%100 < 55
	period = time.Duration(900+h%700) * time.Millisecond
	offset = float64((h>>8)%1000) / 1000
	return period, offset, quiet
}

// dotLevel is how bright a dot is at a moment, from 0 to 1. The cube of a
// half sine keeps the peaks short and the rests long, which is what makes the
// field read as data landing rather than as a block that pulses.
func dotLevel(col, row int, now time.Time) float32 {
	period, offset, quiet := dotBlink(col, row)
	if quiet {
		return 0
	}
	phase := float64(now.UnixMilli())/float64(period.Milliseconds()) + offset
	wave := math.Sin(2 * math.Pi * phase)
	if wave <= 0 {
		return 0
	}
	return float32(wave * wave * wave)
}

// scanDots draws the field. It paints only: the frames it asks for repaint
// the last view instead of running it, so a scan that is already busy does
// not pay for the animation.
func scanDots(c *ui.Context) {
	t := c.Theme()
	// A desktop that asks for less motion gets the field at rest. The title
	// and the folder count already say the scan is running, so nothing is
	// lost by holding the dots still.
	still := c.Preferences().ReduceMotion

	height := float32((dotRows-1)*dotPitch) + dotSize

	// On white the same alpha is a haze: the light variants need more of
	// themselves to read as dots at all.
	rest, peak := float32(dotRest), float32(dotPeak)
	if !t.Dark {
		rest, peak = 0.3, 0.88
	}

	ui.Box(c).FillWidth().Height(height).
		// A widget of your own tells assistive technology what it is: a
		// progress indicator whose length is unknown, as the spinner's -1
		// value says. The card's title says the same in words.
		Role(ui.RoleProgress).
		Range(0, 1, -1).
		Label("Scanning").
		Draw(func(p *ui.Painter, r ui.Rect) {
			if !still {
				p.AnimationFrame()
			}
			// The field spans the row. Its columns are counted from the
			// painted width, and the leftover under one pitch is split either
			// side so the ribbon stays centred in the card.
			cols := int((r.W-dotSize)/dotPitch) + 1
			if cols < 1 {
				cols = 1
			}
			left := (r.W - (float32(cols-1)*dotPitch + dotSize)) / 2

			now := p.Now()
			for row := 0; row < dotRows; row++ {
				for col := 0; col < cols; col++ {
					level := float32(0)
					if !still {
						level = dotLevel(col, row, now)
					}
					alpha := rest + (peak-rest)*level
					p.Fill(ui.Rect{
						X: r.X + left + float32(col)*dotPitch,
						Y: r.Y + float32(row)*dotPitch,
						W: dotSize,
						H: dotSize,
					}, dotColor(col, cols, t.Dark).Alpha(alpha), dotSize/2)
				}
			}
		})
}
