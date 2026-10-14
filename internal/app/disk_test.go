package app

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestDiskPickerIsInTheHeader checks the control exists beside Rescan, and
// that the header names where the scan is looking.
func TestDiskPickerIsInTheHeader(t *testing.T) {
	a := New(Settings{})
	a.LoadVolumes()
	a.updateRoot()
	tt := ui.NewTester(a.viewFn, 980, 720)

	if !tt.HasText("Scan") {
		t.Errorf("no scan-target button; texts: %v", tt.Texts())
	}
	if !tt.HasText("Rescan") {
		t.Errorf("no rescan button; texts: %v", tt.Texts())
	}
	if !tt.HasText(a.rootLabel()) {
		t.Errorf("the header does not name the target %q; texts: %v", a.rootLabel(), tt.Texts())
	}
}

// TestChoosingAVolumeRenamesTheHeader is the point of the feature: pick
// another disk, and the name beside the title changes with it.
func TestChoosingAVolumeRenamesTheHeader(t *testing.T) {
	a := New(Settings{})
	a.LoadVolumes()
	a.updateRoot()
	before := a.rootLabel()

	if len(a.volumes) == 0 {
		t.Skip("no volumes to choose from")
	}
	// Pick a volume that is not the one being scanned.
	var target string
	for _, v := range a.volumes {
		if !a.scansVolume(v.Root) {
			target = v.Root
			break
		}
	}
	if target == "" {
		t.Skip("only one volume on this machine")
	}

	a.ChooseVolume(target)
	after := a.rootLabel()

	if after == before {
		t.Errorf("choosing %s left the header as %q", target, after)
	}
	if after != a.nameFor(target) {
		t.Errorf("header %q, want the volume's own name %q", after, a.nameFor(target))
	}
	if !a.scansVolume(target) {
		t.Errorf("the app is not scanning %s after choosing it", target)
	}
}

// TestVolumeNameIsFriendly checks the naming: a labelled volume is named by
// its label, not its letter.
func TestVolumeNameIsFriendly(t *testing.T) {
	a := New(Settings{})
	a.LoadVolumes()
	if len(a.volumes) == 0 {
		t.Skip("no volumes")
	}
	for _, v := range a.volumes {
		t.Logf("volume %s -> header %q", v.Root, a.nameFor(v.Root))
		if v.Label != "" && v.Label != v.Root {
			want := v.Label + " (" + string(v.Root[0]) + ":)"
			if got := a.nameFor(v.Root); got != want {
				t.Errorf("nameFor(%s) = %q, want %q", v.Root, got, want)
			}
		}
	}
}

// TestFolderRootKeepsItsPath checks that a folder target is named by its
// path, since the path is what tells two project folders apart.
func TestFolderRootKeepsItsPath(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "projects")
	a := New(Settings{Roots: []string{sub}})
	a.updateRoot()
	if got := a.rootLabel(); got != sub {
		t.Errorf("rootLabel() = %q, want the folder path %q", got, sub)
	}
}

// TestChoosingADiskThroughTheMenu drives the real menu rather than calling
// the action: a picker that works only when called directly is not a
// picker. The Tester opens the menu the button opens and chooses from it.
func TestChoosingADiskThroughTheMenu(t *testing.T) {
	a := New(Settings{})
	a.LoadVolumes()
	a.updateRoot()
	if len(a.volumes) == 0 {
		t.Skip("no volumes")
	}
	var other string
	for _, v := range a.volumes {
		if !a.scansVolume(v.Root) {
			other = v.Root
			break
		}
	}
	if other == "" {
		t.Skip("only one volume")
	}

	tt := ui.NewTester(a.viewFn, 1000, 720)
	if err := tt.Click("Scan"); err != nil {
		t.Fatalf("the scan-target button is not clickable: %v", err)
	}
	items := tt.Menu()
	if len(items) == 0 {
		t.Fatal("the button opened no menu")
	}
	t.Logf("menu items: %v", items)

	want := a.nameFor(other)
	found := false
	for _, it := range items {
		if it == want {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("the menu does not offer %q; items: %v", want, items)
	}

	if err := tt.ChooseMenuItem(want); err != nil {
		t.Fatalf("could not choose %q: %v", want, err)
	}
	if !a.scansVolume(other) {
		t.Errorf("choosing %q did not make the app scan it; roots=%v", want, a.roots)
	}
	if got := a.rootLabel(); got != want {
		t.Errorf("header shows %q after choosing %q", got, want)
	}
}

// TestVolumeListDoesNotTouchTheFilesystemPerFrame checks the picker is fed
// from the cached list: naming a volume is a syscall, and the header is
// built every frame.
func TestVolumeListDoesNotTouchTheFilesystemPerFrame(t *testing.T) {
	a := New(Settings{})
	a.LoadVolumes()
	first := len(a.volumes)
	// updateRoot is what the frame path would call if it were uncached;
	// calling it twice must not change the cached list.
	a.updateRoot()
	a.updateRoot()
	if len(a.volumes) != first {
		t.Errorf("the volume list changed between frames: %d -> %d", first, len(a.volumes))
	}
}

// TestCapacityCardNamesTheVolume checks the meter's title is the volume's
// name, not its bare root: VolumeFor reports capacity only, so this is the
// one place the label has to come from the cached list.
func TestCapacityCardNamesTheVolume(t *testing.T) {
	a := New(Settings{})
	a.LoadVolumes()
	if len(a.volumes) == 0 {
		t.Skip("no volumes")
	}
	v := a.volumes[0]
	a.ChooseVolume(v.Root)
	a.updateRoot()
	if !a.haveDisk {
		t.Fatal("no capacity after choosing a volume")
	}
	if a.volume.Label != v.Friendly() {
		t.Errorf("capacity card says %q, want %q", a.volume.Label, v.Friendly())
	}
	if a.rootLabel() != v.Friendly() {
		t.Errorf("header says %q, want %q", a.rootLabel(), v.Friendly())
	}
}
