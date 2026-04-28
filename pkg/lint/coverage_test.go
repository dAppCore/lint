package lint

import (
	core "dappco.re/go"
	"io/fs"
	"path/filepath"
)

func TestParseCoverProfile(t *core.T) {
	data := `mode: set
example.com/pkg/foo/bar.go:10.2,15.16 3 1
example.com/pkg/foo/bar.go:15.16,17.3 1 0
example.com/pkg/foo/baz.go:5.2,8.10 2 1
example.com/other/x.go:1.2,5.10 4 4
`
	snap, err := ParseCoverProfile(data)
	core.RequireNoError(t, err)
	core.AssertNotEmpty(t, snap.Packages)
	core.AssertContains(t, snap.Packages, "example.com/pkg/foo")
	core.AssertContains(t, snap.Packages, "example.com/other")
	core.AssertGreater(t, snap.Total, 0.0)
}

func TestParseCoverProfile_Empty(t *core.T) {
	snap, err := ParseCoverProfile("mode: set\n")
	core.RequireNoError(t, err)
	core.AssertEmpty(t, snap.Packages)
	core.AssertEqual(t, 0.0, snap.Total)
}

func TestParseCoverOutput(t *core.T) {
	output := `?   	example.com/skipped	[no test files]
ok  	example.com/pkg1	0.5s	coverage: 85.0% of statements
ok  	example.com/pkg2	0.2s	coverage: 100.0% of statements
`
	snap, err := ParseCoverOutput(output)
	core.RequireNoError(t, err)
	core.AssertLen(t, snap.Packages, 2)
	core.AssertEqual(t, 85.0, snap.Packages["example.com/pkg1"])
	core.AssertEqual(t, 100.0, snap.Packages["example.com/pkg2"])
	core.AssertInDelta(t, 92.5, snap.Total, 0.1)
}

func TestParseCoverOutput_Empty(t *core.T) {
	snap, err := ParseCoverOutput("FAIL\texample.com/broken [build failed]\n")
	core.RequireNoError(t, err)
	core.AssertEmpty(t, snap.Packages)
	core.AssertEqual(t, 0.0, snap.Total)
}

func TestCompareCoverage(t *core.T) {
	prev := CoverageSnapshot{
		Packages: map[string]float64{
			"pkg/a": 80.0,
			"pkg/b": 90.0,
			"pkg/c": 50.0,
		},
		Total: 73.3,
	}
	curr := CoverageSnapshot{
		Packages: map[string]float64{
			"pkg/a": 85.0, // improved
			"pkg/b": 85.0, // regressed
			"pkg/d": 70.0, // new
		},
		Total: 80.0,
	}

	comp := CompareCoverage(prev, curr)
	core.AssertLen(t, comp.Improvements, 1)
	core.AssertEqual(t, "pkg/a", comp.Improvements[0].Package)
	core.AssertLen(t, comp.Regressions, 1)
	core.AssertEqual(t, "pkg/b", comp.Regressions[0].Package)
	core.AssertContains(t, comp.NewPackages, "pkg/d")
	core.AssertContains(t, comp.Removed, "pkg/c")
	core.AssertInDelta(t, 6.7, comp.TotalDelta, 0.1)
}

func TestCompareCoverage_SortsResultSlices(t *core.T) {
	prev := CoverageSnapshot{
		Packages: map[string]float64{
			"pkg/z": 90.0,
			"pkg/b": 60.0,
			"pkg/a": 80.0,
			"pkg/c": 50.0,
		},
		Total: 70.0,
	}
	curr := CoverageSnapshot{
		Packages: map[string]float64{
			"pkg/b": 55.0,
			"pkg/a": 70.0,
			"pkg/c": 60.0,
			"pkg/y": 40.0,
		},
		Total: 55.0,
	}

	comp := CompareCoverage(prev, curr)

	core.AssertEqual(t, []string{"pkg/y"}, comp.NewPackages)
	core.AssertEqual(t, []string{"pkg/z"}, comp.Removed)
	RequireLen(t, comp.Regressions, 2)
	core.AssertEqual(t, "pkg/a", comp.Regressions[0].Package)
	core.AssertEqual(t, "pkg/b", comp.Regressions[1].Package)
	RequireLen(t, comp.Improvements, 1)
	core.AssertEqual(t, "pkg/c", comp.Improvements[0].Package)
}

func TestCompareCoverage_NoChange(t *core.T) {
	snap := CoverageSnapshot{
		Packages: map[string]float64{"pkg/a": 80.0},
		Total:    80.0,
	}
	comp := CompareCoverage(snap, snap)
	core.AssertEmpty(t, comp.Improvements)
	core.AssertEmpty(t, comp.Regressions)
	core.AssertEmpty(t, comp.NewPackages)
	core.AssertEmpty(t, comp.Removed)
	core.AssertEqual(t, 0.0, comp.TotalDelta)
}

func TestCoverageStore_AppendAndLoad(t *core.T) {
	path := filepath.Join(t.TempDir(), "coverage.json")
	store := NewCoverageStore(path)

	snap := CoverageSnapshot{
		Packages: map[string]float64{"pkg/a": 80.0},
		Total:    80.0,
	}

	err := store.Append(snap)
	core.RequireNoError(t, err)

	snapshots, err := store.Load()
	core.RequireNoError(t, err)
	core.AssertLen(t, snapshots, 1)
	core.AssertEqual(t, 80.0, snapshots[0].Total)
}

func TestCoverageStore_Latest(t *core.T) {
	path := filepath.Join(t.TempDir(), "coverage.json")
	store := NewCoverageStore(path)

	// Empty store returns nil
	latest, err := store.Latest()
	core.RequireNoError(t, err)
	core.AssertNil(t, latest)

	snap1 := CoverageSnapshot{
		Packages: map[string]float64{"pkg/a": 80.0},
		Total:    80.0,
	}
	snap2 := CoverageSnapshot{
		Packages: map[string]float64{"pkg/a": 90.0},
		Total:    90.0,
	}

	core.RequireNoError(t, store.Append(snap1))
	core.RequireNoError(t, store.Append(snap2))

	latest, err = store.Latest()
	core.RequireNoError(t, err)
	RequireNotNil(t, latest)
}

func TestCoverageStore_LoadNotExist(t *core.T) {
	store := NewCoverageStore("/nonexistent/path.json")
	_, err := store.Load()
	core.AssertErrorIs(t, err, fs.ErrNotExist)
}
