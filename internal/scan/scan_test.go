package scan

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// TestMatchGenericNamesNeedProof is the test that matters most in this
// package: a directory called "build" or "obj" is common and usually not
// an artifact, and matching one would delete work.
func TestMatchGenericNamesNeedProof(t *testing.T) {
	dir := t.TempDir()

	// A bare "build" directory, with no Gradle file, must not match.
	plain := filepath.Join(dir, "build")
	if err := os.Mkdir(plain, 0o755); err != nil {
		t.Fatal(err)
	}
	if r, ok := Match(plain, "build"); ok {
		t.Errorf("bare build matched %q, want no match", r.Kind)
	}

	// The same directory, once it holds a Gradle file, is an artifact.
	if err := os.WriteFile(filepath.Join(plain, "build.gradle"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	r, ok := Match(plain, "build")
	if !ok {
		t.Fatal("build with build.gradle did not match")
	}
	if r.Kind != JVM {
		t.Errorf("kind %q, want %q", r.Kind, JVM)
	}
}

// TestMatchRustTargetNeedsCachedirTag covers the same rule for Rust: a
// directory named "target" is only Cargo's when it carries the tag Cargo
// writes.
func TestMatchRustTargetNeedsCachedirTag(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok := Match(target, "target"); ok {
		t.Error("target without CACHEDIR.TAG matched, want no match")
	}
	if err := os.WriteFile(filepath.Join(target, "CACHEDIR.TAG"), []byte("Signature: 8a477f597d28d172789f06886806bc55"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, ok := Match(target, "target")
	if !ok {
		t.Fatal("target with CACHEDIR.TAG did not match")
	}
	if r.Kind != Rust || !r.Costly {
		t.Errorf("got kind %q costly %v, want Rust and costly", r.Kind, r.Costly)
	}
}

// TestMatchUnambiguousNames checks the names that need no proof.
func TestMatchUnambiguousNames(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind Kind
	}{
		{"node_modules", Node},
		{"__pycache__", Python},
		{"DerivedData", Xcode},
		{".pytest_cache", Python},
		{"cmake-build-debug", CMake},
		{"cmake-build-release", CMake},
	} {
		r, ok := Match(filepath.Join("C:/projects/app", tc.name), tc.name)
		if !ok {
			t.Errorf("%s did not match", tc.name)
			continue
		}
		if r.Kind != tc.kind {
			t.Errorf("%s: kind %q, want %q", tc.name, r.Kind, tc.kind)
		}
	}
}

// TestMatchUnrelatedNames checks that ordinary directories are left alone.
func TestMatchUnrelatedNames(t *testing.T) {
	for _, name := range []string{"src", "internal", "docs", "assets", "cmd", "target-notes"} {
		if r, ok := Match(filepath.Join("C:/projects/app", name), name); ok {
			t.Errorf("%s matched %q, want no match", name, r.Kind)
		}
	}
}

// TestWalkFindsArtifacts builds a small tree with one artifact of each
// shape and checks that the walk finds exactly those.
func TestWalkFindsArtifacts(t *testing.T) {
	root := t.TempDir()
	write := func(rel string, size int) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A Rust target, tagged as Cargo tags it.
	write("proj-a/target/debug/app.exe", 4096)
	if err := os.WriteFile(filepath.Join(root, "proj-a", "target", "CACHEDIR.TAG"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A Node install.
	write("proj-b/node_modules/pkg/index.js", 1024)
	// A plain source tree that must not be reported.
	write("proj-c/src/main.go", 512)
	// A bare "build" directory that must not be reported.
	write("proj-d/build/notes.txt", 64)

	got := Walk(context.Background(), Options{Roots: []string{root}, Workers: 4})

	byPath := map[string]Artifact{}
	for _, a := range got {
		byPath[filepath.Base(a.Path)] = a
	}
	if _, ok := byPath["target"]; !ok {
		t.Error("target was not found")
	}
	if _, ok := byPath["node_modules"]; !ok {
		t.Error("node_modules was not found")
	}
	if _, ok := byPath["build"]; ok {
		t.Error("bare build was reported as an artifact")
	}
	if len(got) != 2 {
		t.Errorf("found %d artifacts, want 2: %v", len(got), got)
	}
	// The Rust target holds one 4096-byte file plus the tag.
	for _, a := range got {
		if a.Name == "target" && a.Bytes < 4096 {
			t.Errorf("target measured %d bytes, want at least 4096", a.Bytes)
		}
	}
}

// TestWalkDoesNotDescendIntoArtifacts checks the nesting rule: a build
// directory inside a build directory belongs to the outer one, so its
// bytes are counted once, not twice.
func TestWalkDoesNotDescendIntoArtifacts(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "app", "node_modules", "dep", "node_modules")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inner, "index.js"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	got := Walk(context.Background(), Options{Roots: []string{root}, Workers: 2})
	if len(got) != 1 {
		t.Fatalf("found %d artifacts, want 1: %v", len(got), got)
	}
	if got[0].Bytes != 2048 {
		t.Errorf("measured %d bytes, want 2048 (the nested tree counted once)", got[0].Bytes)
	}
}

// TestWalkStopsOnCancel checks that a cancelled scan returns promptly
// rather than walking the whole tree.
func TestWalkStopsOnCancel(t *testing.T) {
	root := t.TempDir()
	for i := range 200 {
		dir := filepath.Join(root, "d", "x", strconv.Itoa(i))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() {
		Walk(ctx, Options{Roots: []string{root}, Workers: 4})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a cancelled walk did not return")
	}
}

// TestWalkSkipsJunctionsOnWindows is covered by the platform build tags of
// isLink; here the name filter is what is checked.
func TestWalkSkipsSystemDirectories(t *testing.T) {
	root := t.TempDir()
	// "Windows" is skipped by name, so a "node_modules" inside it is not
	// reported.
	deep := filepath.Join(root, "Windows", "System32", "node_modules")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "x.js"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Walk(context.Background(), Options{Roots: []string{root}, Workers: 2})
	if len(got) != 0 {
		t.Errorf("found %v, want nothing under a skipped directory", got)
	}
}

// TestMatchXcodeAndAndroidTrees covers the rules that need a sibling
// check or a specific name: a Pods directory is only CocoaPods' when the
// Podfile sits beside it, and an Android build tree is recorded whole
// rather than as the many CMakeFiles directories inside it.
func TestMatchXcodeAndAndroidTrees(t *testing.T) {
	root := t.TempDir()

	pods := filepath.Join(root, "ios", "Pods")
	if err := os.MkdirAll(pods, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok := Match(pods, "Pods"); ok {
		t.Error("Pods without a sibling Podfile matched")
	}
	if err := os.WriteFile(filepath.Join(root, "ios", "Podfile"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if r, ok := Match(pods, "Pods"); !ok || r.Kind != Xcode {
		t.Errorf("Pods with a sibling Podfile gave %q, %v", r.Kind, ok)
	}

	cxx := filepath.Join(root, "android", "app", ".cxx")
	if r, ok := Match(cxx, ".cxx"); !ok || r.Kind != CMake {
		t.Errorf(".cxx gave %q, %v, want CMake", r.Kind, ok)
	}
}

// TestWalkRecordsAndroidTreeOnce checks that an Android build tree is one
// artifact rather than dozens: the NDK writes a CMakeFiles directory per
// ABI, and reporting each separately buries the real space users.
func TestWalkRecordsAndroidTreeOnce(t *testing.T) {
	root := t.TempDir()
	cxx := filepath.Join(root, "android", "app", ".cxx", "Debug", "abc", "arm64-v8a", "lib", "CMakeFiles")
	if err := os.MkdirAll(cxx, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cxx, "build.o"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	got := Walk(context.Background(), Options{Roots: []string{root}, Workers: 2})
	if len(got) != 1 {
		t.Fatalf("found %d artifacts, want 1: %v", len(got), got)
	}
	if got[0].Name != ".cxx" {
		t.Errorf("recorded %q, want the whole .cxx tree", got[0].Name)
	}
	if got[0].Bytes != 4096 {
		t.Errorf("measured %d bytes, want 4096", got[0].Bytes)
	}
}

// TestWalkIsConcurrencySafe runs many walks at once over the same tree,
// which is what exercises the work queue's locking and its closing rule.
// Run with -race where a C compiler is available.
func TestWalkIsConcurrencySafe(t *testing.T) {
	root := t.TempDir()
	for i := range 30 {
		dir := filepath.Join(root, "p"+strconv.Itoa(i), "node_modules")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "f.js"), make([]byte, 128), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	const runs = 8
	results := make([]int, runs)
	done := make(chan int, runs)
	for range runs {
		go func() {
			done <- len(Walk(context.Background(), Options{Roots: []string{root}, Workers: 4}))
		}()
	}
	for r := range runs {
		results[r] = <-done
	}
	for r, n := range results {
		if n != 30 {
			t.Errorf("run %d found %d artifacts, want 30", r, n)
		}
	}
}

// TestWalkWithManyWorkers checks that a worker count larger than the work
// does not deadlock the queue, which is the failure mode of a bounded
// work channel.
func TestWalkWithManyWorkers(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "app", "node_modules")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	done := make(chan int, 1)
	go func() {
		done <- len(Walk(context.Background(), Options{Roots: []string{root}, Workers: 64}))
	}()
	select {
	case n := <-done:
		if n != 1 {
			t.Errorf("found %d artifacts, want 1", n)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("a walk with more workers than work did not finish")
	}
}

// TestHumanBytes checks the sizes the list shows.
func TestHumanBytes(t *testing.T) {
	for _, tc := range []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1 KB"},
		{1536, "1.5 KB"},
		{1048576, "1 MB"},
		{1205862, "1.1 MB"},
		{1073741824, "1 GB"},
		{3221225472, "3 GB"},
		{47500000000, "44.2 GB"},
	} {
		if got := HumanBytes(tc.in); got != tc.want {
			t.Errorf("HumanBytes(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestGroupByKindKeepsKindOrder(t *testing.T) {
	arts := []Artifact{
		{Kind: Node, Bytes: 10, Path: "n"},
		{Kind: Rust, Bytes: 100, Path: "r"},
		{Kind: Node, Bytes: 50, Path: "n2"},
	}
	groups := GroupByKind(arts)
	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(groups))
	}
	if groups[0].Kind != Rust {
		t.Errorf("first group %q, want Rust (kind order, not size order)", groups[0].Kind)
	}
	if groups[1].Bytes != 60 {
		t.Errorf("Node total %d, want 60", groups[1].Bytes)
	}
	if groups[1].Artifacts[0].Path != "n2" {
		t.Error("artifacts within a group are not largest first")
	}
}
