package lint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCoverProfile(t *testing.T) {
	data := `mode: set
example.com/pkg/foo/bar.go:10.2,15.16 3 1
example.com/pkg/foo/bar.go:15.16,17.3 1 0
example.com/pkg/foo/baz.go:5.2,8.10 2 1
example.com/other/x.go:1.2,5.10 4 4
`
	snap, err := ParseCoverProfile(data)
	require.NoError(t, err)
	assert.NotEmpty(t, snap.Packages)
	assert.Contains(t, snap.Packages, "example.com/pkg/foo")
	assert.Contains(t, snap.Packages, "example.com/other")
	assert.Greater(t, snap.Total, 0.0)
}

func TestParseCoverProfile_Empty(t *testing.T) {
	snap, err := ParseCoverProfile("mode: set\n")
	require.NoError(t, err)
	assert.Empty(t, snap.Packages)
	assert.Equal(t, 0.0, snap.Total)
}

func TestParseCoverOutput(t *testing.T) {
	output := `?   	example.com/skipped	[no test files]
ok  	example.com/pkg1	0.5s	coverage: 85.0% of statements
ok  	example.com/pkg2	0.2s	coverage: 100.0% of statements
`
	snap, err := ParseCoverOutput(output)
	require.NoError(t, err)
	assert.Len(t, snap.Packages, 2)
	assert.Equal(t, 85.0, snap.Packages["example.com/pkg1"])
	assert.Equal(t, 100.0, snap.Packages["example.com/pkg2"])
	assert.InDelta(t, 92.5, snap.Total, 0.1)
}

func TestParseCoverOutput_Empty(t *testing.T) {
	snap, err := ParseCoverOutput("FAIL\texample.com/broken [build failed]\n")
	require.NoError(t, err)
	assert.Empty(t, snap.Packages)
	assert.Equal(t, 0.0, snap.Total)
}

func TestCompareCoverage(t *testing.T) {
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
	assert.Len(t, comp.Improvements, 1)
	assert.Equal(t, "pkg/a", comp.Improvements[0].Package)
	assert.Len(t, comp.Regressions, 1)
	assert.Equal(t, "pkg/b", comp.Regressions[0].Package)
	assert.Contains(t, comp.NewPackages, "pkg/d")
	assert.Contains(t, comp.Removed, "pkg/c")
	assert.InDelta(t, 6.7, comp.TotalDelta, 0.1)
}

func TestCompareCoverage_SortsResultSlices(t *testing.T) {
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

	assert.Equal(t, []string{"pkg/y"}, comp.NewPackages)
	assert.Equal(t, []string{"pkg/z"}, comp.Removed)
	require.Len(t, comp.Regressions, 2)
	assert.Equal(t, "pkg/a", comp.Regressions[0].Package)
	assert.Equal(t, "pkg/b", comp.Regressions[1].Package)
	require.Len(t, comp.Improvements, 1)
	assert.Equal(t, "pkg/c", comp.Improvements[0].Package)
}

func TestCompareCoverage_NoChange(t *testing.T) {
	snap := CoverageSnapshot{
		Packages: map[string]float64{"pkg/a": 80.0},
		Total:    80.0,
	}
	comp := CompareCoverage(snap, snap)
	assert.Empty(t, comp.Improvements)
	assert.Empty(t, comp.Regressions)
	assert.Empty(t, comp.NewPackages)
	assert.Empty(t, comp.Removed)
	assert.Equal(t, 0.0, comp.TotalDelta)
}

func TestCoverageStore_AppendAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coverage.json")
	store := NewCoverageStore(path)

	snap := CoverageSnapshot{
		Packages: map[string]float64{"pkg/a": 80.0},
		Total:    80.0,
	}

	err := store.Append(snap)
	require.NoError(t, err)

	snapshots, err := store.Load()
	require.NoError(t, err)
	assert.Len(t, snapshots, 1)
	assert.Equal(t, 80.0, snapshots[0].Total)
}

func TestCoverageStore_Latest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coverage.json")
	store := NewCoverageStore(path)

	// Empty store returns nil
	latest, err := store.Latest()
	require.NoError(t, err)
	assert.Nil(t, latest)

	snap1 := CoverageSnapshot{
		Packages: map[string]float64{"pkg/a": 80.0},
		Total:    80.0,
	}
	snap2 := CoverageSnapshot{
		Packages: map[string]float64{"pkg/a": 90.0},
		Total:    90.0,
	}

	require.NoError(t, store.Append(snap1))
	require.NoError(t, store.Append(snap2))

	latest, err = store.Latest()
	require.NoError(t, err)
	require.NotNil(t, latest)
}

func TestCoverageStore_LoadNotExist(t *testing.T) {
	store := NewCoverageStore("/nonexistent/path.json")
	_, err := store.Load()
	assert.True(t, os.IsNotExist(err))
}
