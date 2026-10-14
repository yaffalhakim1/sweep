package app

import (
	"time"

	"sweep/internal/disk"
	"sweep/internal/scan"
)

// diskFixture is a plausible full-ish volume for the tests and the
// rendered frames, so the capacity card has something to show.
func diskFixture() disk.Volume {
	const gb = 1 << 30
	return disk.Volume{
		Root:  `C:\`,
		Label: `Windows (C:)`,
		Total: 476 * gb,
		Free:  28 * gb,
	}
}

// realWorld is a scan with the shape a real developer's disk has: a few
// enormous artifacts and a long tail of small ones. It is what the map has
// to stay readable with, so it is what the render test draws.
func realWorld() []scan.Artifact {
	type row struct {
		path  string
		name  string
		kind  scan.Kind
		bytes int64
		files int
	}
	rows := []row{
		{`C:\Users\dev\Documents\Learn\rust\waku\node_modules`, "node_modules", scan.Node, 15_500_000_000, 214_000},
		{`C:\Users\dev\Documents\Learn\rust\kaku-gui\target`, "target", scan.Rust, 5_700_000_000, 38_000},
		{`C:\Users\dev\Documents\Work\lxp-v2\.next`, ".next", scan.Node, 2_300_000_000, 12_400},
		{`C:\Users\dev\Documents\Work\clicky\.next`, ".next", scan.Node, 1_400_000_000, 9_100},
		{`C:\Users\dev\Documents\Work\clicky\node_modules`, "node_modules", scan.Node, 816_000_000, 41_000},
		{`C:\Users\dev\Documents\Work\lxp-v2\node_modules`, "node_modules", scan.Node, 769_000_000, 39_500},
		{`C:\Users\dev\Documents\Learn\rust\waku\android\.cxx`, ".cxx", scan.CMake, 741_000_000, 26_000},
		{`C:\Users\dev\Documents\Work\b2c-v3\node_modules`, "node_modules", scan.Node, 580_000_000, 33_200},
		{`C:\Users\dev\Documents\Freelance\e-menu\.next`, ".next", scan.Node, 474_000_000, 5_800},
		{`C:\Users\dev\Documents\Freelance\e-menu\node_modules`, "node_modules", scan.Node, 460_000_000, 29_400},
		{`C:\Users\dev\Documents\Work\b2c-v2\.next`, ".next", scan.Node, 395_000_000, 4_900},
		{`C:\Users\dev\Documents\Learn\Frontend\docvault\node_modules`, "node_modules", scan.Node, 316_000_000, 24_100},
		{`C:\Users\dev\Documents\Learn\Frontend\portfolio\.next`, ".next", scan.Node, 253_000_000, 3_400},
		{`C:\Users\dev\Documents\Learn\Frontend\medium\.next`, ".next", scan.Node, 98_600_000, 2_100},
		{`C:\Users\dev\Documents\Freelance\e-menu\restaurant\node_modules`, "node_modules", scan.Node, 90_000_000, 18_700},
		{`C:\Users\dev\Documents\Freelance\solutive\.venv`, ".venv", scan.Python, 69_500_000, 6_200},
		{`C:\Users\dev\Documents\Learn\rust\waku\android\.gradle`, ".gradle", scan.JVM, 59_200_000, 1_400},
		{`C:\Users\dev\Documents\Learn\rust\waku\.opencode\node_modules`, "node_modules", scan.Node, 52_400_000, 11_000},
		{`C:\Users\dev\Documents\Work\b2c-v3\apps\client\node_modules`, "node_modules", scan.Node, 50_100_000, 9_800},
		{`C:\Users\dev\Documents\Learn\go\kata\build`, "build", scan.CMake, 34_000_000, 900},
		{`C:\Users\dev\Documents\Learn\go\ports\build`, "build", scan.CMake, 22_000_000, 700},
		{`C:\Users\dev\Documents\Learn\rust\kaku-gui\crates\ui\target`, "target", scan.Rust, 18_400_000, 3_100},
		{`C:\Users\dev\Documents\Learn\Frontend\docvault\.next`, ".next", scan.Node, 12_800_000, 800},
		{`C:\Users\dev\Documents\Freelance\solutive\api\__pycache__`, "__pycache__", scan.Python, 4_200_000, 240},
		{`C:\Users\dev\Documents\Learn\python\ml\.mypy_cache`, ".mypy_cache", scan.Python, 2_600_000, 180},
		{`C:\Users\dev\Documents\Learn\python\ml\.pytest_cache`, ".pytest_cache", scan.Python, 1_100_000, 60},
		{`C:\Users\dev\Documents\Learn\rust\waku\.turbo`, ".turbo", scan.Node, 900_000, 40},
		{`C:\Users\dev\Documents\Learn\Frontend\portfolio\.vite`, ".vite", scan.Node, 400_000, 30},
	}
	now := time.Now()
	arts := make([]scan.Artifact, 0, len(rows))
	for i, r := range rows {
		arts = append(arts, scan.Artifact{
			Path:    r.path,
			Name:    r.name,
			Kind:    r.kind,
			Bytes:   r.bytes,
			Files:   r.files,
			Touched: now.Add(-time.Duration(i+1) * 9 * time.Hour),
			Rule:    scan.Rule{Costly: true, Note: "Build output."},
		})
	}
	return arts
}
