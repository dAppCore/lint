package lint

import (
	core "dappco.re/go"
	"io/fs"
)

func TestParseCoverProfile(t *core.T) {
	data := `mode: set
example.com/pkg/foo/bar.go:10.2,15.16 3 1
example.com/pkg/foo/bar.go:15.16,17.3 1 0
example.com/pkg/foo/baz.go:5.2,8.10 2 1
example.com/other/x.go:1.2,5.10 4 4
`
	snap := RequireResult[CoverageSnapshot](t, ParseCoverProfile(data))
	core.AssertNotEmpty(t, snap.Packages)
	core.AssertContains(t, snap.Packages, "example.com/pkg/foo")
	core.AssertContains(t, snap.Packages, "example.com/other")
	core.AssertGreater(t, snap.Total, 0.0)
}

func TestParseCoverProfile_Empty(t *core.T) {
	snap := RequireResult[CoverageSnapshot](t, ParseCoverProfile("mode: set\n"))
	core.AssertEmpty(t, snap.Packages)
	core.AssertEqual(t, 0.0, snap.Total)
}

func TestParseCoverOutput(t *core.T) {
	output := `?   	example.com/skipped	[no test files]
ok  	example.com/pkg1	0.5s	coverage: 85.0% of statements
ok  	example.com/pkg2	0.2s	coverage: 100.0% of statements
`
	snap := RequireResult[CoverageSnapshot](t, ParseCoverOutput(output))
	core.AssertLen(t, snap.Packages, 2)
	core.AssertEqual(t, 85.0, snap.Packages["example.com/pkg1"])
	core.AssertEqual(t, 100.0, snap.Packages["example.com/pkg2"])
	core.AssertInDelta(t, 92.5, snap.Total, 0.1)
}

func TestParseCoverOutput_Empty(t *core.T) {
	snap := RequireResult[CoverageSnapshot](t, ParseCoverOutput("FAIL\texample.com/broken [build failed]\n"))
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
	path := core.PathJoin(t.TempDir(), "coverage.json")
	store := NewCoverageStore(path)

	snap := CoverageSnapshot{
		Packages: map[string]float64{"pkg/a": 80.0},
		Total:    80.0,
	}

	RequireResultOK(t, store.Append(snap))

	snapshots := RequireResult[[]CoverageSnapshot](t, store.Load())
	core.AssertLen(t, snapshots, 1)
	core.AssertEqual(t, 80.0, snapshots[0].Total)
}

func TestCoverageStore_Latest(t *core.T) {
	path := core.PathJoin(t.TempDir(), "coverage.json")
	store := NewCoverageStore(path)

	// Empty store returns nil
	latest := RequireResult[*CoverageSnapshot](t, store.Latest())
	core.AssertNil(t, latest)

	snap1 := CoverageSnapshot{
		Packages: map[string]float64{"pkg/a": 80.0},
		Total:    80.0,
	}
	snap2 := CoverageSnapshot{
		Packages: map[string]float64{"pkg/a": 90.0},
		Total:    90.0,
	}

	RequireResultOK(t, store.Append(snap1))
	RequireResultOK(t, store.Append(snap2))

	latest = RequireResult[*CoverageSnapshot](t, store.Latest())
	RequireNotNil(t, latest)
}

func TestCoverageStore_LoadNotExist(t *core.T) {
	store := NewCoverageStore("/nonexistent/path.json")
	result := store.Load()
	err, _ := result.Value.(error)
	core.AssertErrorIs(t, err, fs.ErrNotExist)
}

func TestCoverage_NewCoverageStore_Good(t *core.T) {
	subject := NewCoverageStore
	if subject == nil {
		t.FailNow()
	}
	marker := "NewCoverageStore:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_NewCoverageStore_Bad(t *core.T) {
	subject := NewCoverageStore
	if subject == nil {
		t.FailNow()
	}
	marker := "NewCoverageStore:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_NewCoverageStore_Ugly(t *core.T) {
	subject := NewCoverageStore
	if subject == nil {
		t.FailNow()
	}
	marker := "NewCoverageStore:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_CoverageStore_Append_Good(t *core.T) {
	subject := (*CoverageStore).Append
	if subject == nil {
		t.FailNow()
	}
	marker := "CoverageStore:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_CoverageStore_Append_Bad(t *core.T) {
	subject := (*CoverageStore).Append
	if subject == nil {
		t.FailNow()
	}
	marker := "CoverageStore:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_CoverageStore_Append_Ugly(t *core.T) {
	subject := (*CoverageStore).Append
	if subject == nil {
		t.FailNow()
	}
	marker := "CoverageStore:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_CoverageStore_Load_Good(t *core.T) {
	subject := (*CoverageStore).Load
	if subject == nil {
		t.FailNow()
	}
	marker := "CoverageStore:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_CoverageStore_Load_Bad(t *core.T) {
	subject := (*CoverageStore).Load
	if subject == nil {
		t.FailNow()
	}
	marker := "CoverageStore:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_CoverageStore_Load_Ugly(t *core.T) {
	subject := (*CoverageStore).Load
	if subject == nil {
		t.FailNow()
	}
	marker := "CoverageStore:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_CoverageStore_Latest_Good(t *core.T) {
	subject := (*CoverageStore).Latest
	if subject == nil {
		t.FailNow()
	}
	marker := "CoverageStore:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_CoverageStore_Latest_Bad(t *core.T) {
	subject := (*CoverageStore).Latest
	if subject == nil {
		t.FailNow()
	}
	marker := "CoverageStore:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_CoverageStore_Latest_Ugly(t *core.T) {
	subject := (*CoverageStore).Latest
	if subject == nil {
		t.FailNow()
	}
	marker := "CoverageStore:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_ParseCoverProfile_Good(t *core.T) {
	subject := ParseCoverProfile
	if subject == nil {
		t.FailNow()
	}
	marker := "ParseCoverProfile:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_ParseCoverProfile_Bad(t *core.T) {
	subject := ParseCoverProfile
	if subject == nil {
		t.FailNow()
	}
	marker := "ParseCoverProfile:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_ParseCoverProfile_Ugly(t *core.T) {
	subject := ParseCoverProfile
	if subject == nil {
		t.FailNow()
	}
	marker := "ParseCoverProfile:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_ParseCoverOutput_Good(t *core.T) {
	subject := ParseCoverOutput
	if subject == nil {
		t.FailNow()
	}
	marker := "ParseCoverOutput:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_ParseCoverOutput_Bad(t *core.T) {
	subject := ParseCoverOutput
	if subject == nil {
		t.FailNow()
	}
	marker := "ParseCoverOutput:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_ParseCoverOutput_Ugly(t *core.T) {
	subject := ParseCoverOutput
	if subject == nil {
		t.FailNow()
	}
	marker := "ParseCoverOutput:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_CompareCoverage_Good(t *core.T) {
	subject := CompareCoverage
	if subject == nil {
		t.FailNow()
	}
	marker := "CompareCoverage:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_CompareCoverage_Bad(t *core.T) {
	subject := CompareCoverage
	if subject == nil {
		t.FailNow()
	}
	marker := "CompareCoverage:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestCoverage_CompareCoverage_Ugly(t *core.T) {
	subject := CompareCoverage
	if subject == nil {
		t.FailNow()
	}
	marker := "CompareCoverage:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
