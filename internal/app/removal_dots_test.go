package app

import "testing"

// removingApp is an app with a removal in flight, which is the only state the
// footer's dots exist in.
func removingApp(done, total int) *App {
	a := testApp()
	a.removing = true
	a.removed, a.total = done, total
	return a
}

// TestRemovalShowsItsCountAndItsDots checks the footer says where the removal
// is, and that the row it draws is alive.
func TestRemovalShowsItsCountAndItsDots(t *testing.T) {
	a := removingApp(3, 12)
	tt := frame(a, 1180, 800)

	if !tt.HasText("Removing 3 of 12…") {
		t.Errorf("the footer does not report the removal; texts: %v", tt.Texts())
	}
	if !tt.FramePending() {
		t.Error("the removal row does not ask for frames, so its leading edge would stand still")
	}
}

// TestRemovalFillsTheRowInProportion pins the reading: none at the start, all
// at the end, half in the middle, whatever the row is wide.
func TestRemovalFillsTheRowInProportion(t *testing.T) {
	for _, tc := range []struct{ done, total, cols, want int }{
		{0, 12, 100, 0},
		{12, 12, 100, 100},
		{6, 12, 100, 50},
		{1, 2, 40, 20},
		{0, 0, 100, 0},   // no total known: nothing to claim
		{99, 12, 40, 40}, // never past the end of the row
	} {
		if got := removalFilled(tc.done, tc.total, tc.cols); got != tc.want {
			t.Errorf("done=%d of %d over %d dots filled %d, want %d",
				tc.done, tc.total, tc.cols, got, tc.want)
		}
	}
}

// TestRemovalLeadBreathesWithoutDisappearing checks the pulse stays visible:
// a leading edge that reached zero would read as a gap in the progress.
func TestRemovalLeadBreathesWithoutDisappearing(t *testing.T) {
	values := map[float32]bool{}
	for ms := int64(0); ms < removalBreath; ms += 10 {
		alpha := removalLead(ms, removalBreath)
		if alpha < 0.4 || alpha > 1 {
			t.Fatalf("at %dms the lead is %.2f, outside 0.4..1", ms, alpha)
		}
		values[alpha] = true
	}
	if len(values) < 20 {
		t.Errorf("the lead only takes %d values over a breath, so it does not move", len(values))
	}
}
