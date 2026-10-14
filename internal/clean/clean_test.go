package clean

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGuardRefusesDangerousPaths is the test that protects the user's
// data. Every case here is a path the app must never remove, whatever
// calls it.
func TestGuardRefusesDangerousPaths(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory to test against")
	}
	for _, tc := range []struct {
		name string
		path string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"relative", "project/node_modules"},
		{"unix root", "/"},
		{"windows root", `C:\`},
		{"home", home},
		{"a volume root", filepath.VolumeName(home) + `\`},
		{"too shallow", filepath.Join(filepath.VolumeName(home)+`\`, "projects")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := Guard(tc.path); err == nil {
				t.Errorf("Guard(%q) allowed the path, want a refusal", tc.path)
			}
		})
	}
}

// TestGuardAllowsArtifactDirectories checks the other side: a real
// artifact path is not refused, so the guards are not so strict that the
// app cannot do its job.
func TestGuardAllowsArtifactDirectories(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "project", "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Guard(target); err != nil {
		t.Errorf("Guard(%q) refused a real artifact: %v", target, err)
	}
}

// TestGuardSetRejectsNestedTargets checks that selecting both a directory
// and something inside it is refused, because removing the parent would
// take the child with it and the reported total would count it twice.
func TestGuardSetRejectsNestedTargets(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "project", "node_modules")
	child := filepath.Join(parent, "dep", "node_modules")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	err := GuardSet([]Target{{Path: parent}, {Path: child}})
	if err == nil {
		t.Error("a parent and its child were both allowed, want a refusal")
	}
}

// TestRemoveDeletesAndReports checks the happy path, and that the reported
// freed bytes are the sizes the scan measured rather than a second walk.
func TestRemoveDeletesAndReports(t *testing.T) {
	dir := t.TempDir()
	one := filepath.Join(dir, "a", "node_modules")
	two := filepath.Join(dir, "b", "target")
	for _, p := range []string{one, two} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, "f"), make([]byte, 100), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	res := Remove([]Target{{Path: one, Bytes: 100}, {Path: two, Bytes: 100}})
	if res.Removed != 2 || res.Freed != 200 {
		t.Errorf("removed %d freed %d, want 2 and 200", res.Removed, res.Freed)
	}
	for _, p := range []string{one, two} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s still exists", p)
		}
	}
}

// TestRemoveSkipsUnsafeTargets checks that a target the guards reject is
// reported as a failure and left alone, while the rest of the run goes on.
func TestRemoveSkipsUnsafeTargets(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "a", "node_modules")
	if err := os.MkdirAll(good, 0o755); err != nil {
		t.Fatal(err)
	}
	res := Remove([]Target{{Path: good, Bytes: 10}, {Path: "/", Bytes: 999}})
	if res.Removed != 1 {
		t.Errorf("removed %d, want 1", res.Removed)
	}
	if len(res.Failures) != 1 {
		t.Fatalf("got %d failures, want 1", len(res.Failures))
	}
	if res.Freed != 10 {
		t.Errorf("freed %d, want only the safe target's 10", res.Freed)
	}
}

// TestRemoveIsIdempotent checks that a path already gone is not a crash,
// which happens when two scans race a cleanup.
func TestRemoveIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	gone := filepath.Join(dir, "a", "node_modules")
	if err := os.MkdirAll(filepath.Dir(gone), 0o755); err != nil {
		t.Fatal(err)
	}
	res := Remove([]Target{{Path: gone, Bytes: 5}})
	if res.Removed != 1 || len(res.Failures) != 0 {
		t.Errorf("removing a path that is already gone reported %+v", res)
	}
}

// TestRemoveClearsReadOnlyFiles covers the case that makes a plain
// RemoveAll fail: build tools leave read-only files behind.
func TestRemoveClearsReadOnlyFiles(t *testing.T) {
	dir := t.TempDir()
	tree := filepath.Join(dir, "a", "target")
	if err := os.MkdirAll(tree, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(tree, "locked.bin")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, 0o444); err != nil {
		t.Fatal(err)
	}
	res := Remove([]Target{{Path: tree, Bytes: 1}})
	if res.Removed != 1 {
		t.Errorf("removing a read-only tree reported %+v", res)
	}
}
