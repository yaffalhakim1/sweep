// Command versionbump sets the version in mygo.json, which is the one place
// the build, the installer and the release tag all read it from.
//
//	go run ./internal/tools/versionbump 0.2.0
//
// It rewrites the single "version" field and nothing else, so the diff of a
// version bump is exactly one line.
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// versionPattern is what a version may look like. mygo builds file names
// from it and the release tag is "v" plus it, so a stray character here
// becomes a broken asset name.
var versionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: versionbump <version>   e.g. versionbump 0.2.0")
		os.Exit(2)
	}
	next := strings.TrimPrefix(os.Args[1], "v")
	if !versionPattern.MatchString(next) {
		fmt.Fprintf(os.Stderr, "versionbump: %q is not a version like 0.2.0\n", os.Args[1])
		os.Exit(2)
	}

	const path = "mygo.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "versionbump: %v\n", err)
		os.Exit(1)
	}

	// The version field is replaced in place rather than by re-encoding the
	// JSON: encoding a map sorts its keys, which would turn a one-line
	// version bump into a diff that touches every field.
	field := regexp.MustCompile(`("version"\s*:\s*")([^"]*)(")`)
	match := field.FindSubmatch(raw)
	if match == nil {
		fmt.Fprintf(os.Stderr, "versionbump: no \"version\" field in %s\n", path)
		os.Exit(1)
	}
	was := string(match[2])
	if was == next {
		fmt.Printf("already at %s\n", next)
		return
	}
	out := field.ReplaceAll(raw, []byte("${1}"+next+"${3}"))
	if err := os.WriteFile(path, out, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "versionbump: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s: %s -> %s\n", path, was, next)
	fmt.Printf("next: git add %s && git commit -m \"Bump to %s\" && git push\n", path, next)
}
