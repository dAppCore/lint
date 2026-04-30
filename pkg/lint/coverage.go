package lint

import (
	"bufio"
	"cmp"
	"math"
	"regexp"
	"slices"
	"strconv"
	"time"

	"dappco.re/go"
	coreio "dappco.re/go/io"
)

const (
	coverageCoveragestoreAppend1a31c1 = "CoverageStore.Append"
)

// CoverageSnapshot represents a point-in-time coverage measurement.
type CoverageSnapshot struct {
	Timestamp time.Time          `json:"timestamp"`
	Packages  map[string]float64 `json:"packages"`       // package → coverage %
	Total     float64            `json:"total"`          // overall coverage %
	Meta      map[string]string  `json:"meta,omitempty"` // optional metadata (commit, branch, etc.)
}

type coveragePkgStats struct {
	covered int
	total   int
}

// CoverageRegression flags a package whose coverage changed between runs.
type CoverageRegression struct {
	Package  string  `json:"package"`
	Previous float64 `json:"previous"`
	Current  float64 `json:"current"`
	Delta    float64 `json:"delta"` // Negative means regression
}

// CoverageComparison holds the result of comparing two snapshots.
type CoverageComparison struct {
	Regressions  []CoverageRegression `json:"regressions,omitempty"`
	Improvements []CoverageRegression `json:"improvements,omitempty"`
	NewPackages  []string             `json:"new_packages,omitempty"`
	Removed      []string             `json:"removed,omitempty"`
	TotalDelta   float64              `json:"total_delta"`
}

// CoverageStore persists coverage snapshots to a JSON file.
type CoverageStore struct {
	Path string // File path for JSON storage
}

// NewCoverageStore creates a store backed by the given file path.
func NewCoverageStore(path string) *CoverageStore {
	return &CoverageStore{Path: path}
}

// Append adds a snapshot to the store.
func (s *CoverageStore) Append(snap CoverageSnapshot) error {
	snapshots, err := s.Load()
	if err != nil && !isNotExistError(err) {
		return core.E(coverageCoveragestoreAppend1a31c1, "load snapshots", err)
	}

	snapshots = append(snapshots, snap)

	r := core.JSONMarshal(snapshots)
	if !r.OK {
		return core.E(coverageCoveragestoreAppend1a31c1, "marshal snapshots", r.Value.(error))
	}
	data := r.Value.([]byte)

	if err := coreio.Local.Write(s.Path, string(data)); err != nil {
		return core.E(coverageCoveragestoreAppend1a31c1, "write "+s.Path, err)
	}
	return nil
}

// Load reads all snapshots from the store.
func (s *CoverageStore) Load() ([]CoverageSnapshot, error) {
	raw, err := coreio.Local.Read(s.Path)
	if err != nil {
		return nil, err
	}

	var snapshots []CoverageSnapshot
	if r := core.JSONUnmarshal([]byte(raw), &snapshots); !r.OK {
		return nil, core.E("CoverageStore.Load", "parse "+s.Path, r.Value.(error))
	}
	return snapshots, nil
}

// Latest returns the most recent snapshot, or nil if the store is empty.
func (s *CoverageStore) Latest() (*CoverageSnapshot, error) {
	snapshots, err := s.Load()
	if err != nil {
		if isNotExistError(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(snapshots) == 0 {
		return nil, nil
	}

	latest := &snapshots[0]
	for i := range snapshots {
		if snapshots[i].Timestamp.After(latest.Timestamp) {
			latest = &snapshots[i]
		}
	}
	return latest, nil
}

// ParseCoverProfile parses output from `go test -coverprofile=cover.out` format.
func ParseCoverProfile(data string) (CoverageSnapshot, error) {
	snap := CoverageSnapshot{
		Timestamp: time.Now(),
		Packages:  make(map[string]float64),
	}

	packages := make(map[string]*coveragePkgStats)

	scanner := bufio.NewScanner(core.NewReader(data))
	for scanner.Scan() {
		pkg, stmts, count, ok := parseCoverProfileLine(scanner.Text())
		if !ok {
			continue
		}
		addCoverageStats(packages, pkg, stmts, count)
	}

	setCoveragePercentages(&snap, packages)
	return snap, nil
}

func parseCoverProfileLine(line string) (string, int, int, bool) {
	if core.HasPrefix(line, "mode:") {
		return "", 0, 0, false
	}
	parts := coverageFields(line)
	if len(parts) != 3 {
		return "", 0, 0, false
	}
	fileParts := core.Split(parts[0], ":")
	if len(fileParts) < 2 {
		return "", 0, 0, false
	}
	stmts, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, 0, false
	}
	count, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", 0, 0, false
	}
	return coveragePackage(fileParts[0]), stmts, count, true
}

func coveragePackage(file string) string {
	fileSegments := core.Split(file, "/")
	if len(fileSegments) <= 1 {
		return file
	}
	return core.Join("/", fileSegments[:len(fileSegments)-1]...)
}

func addCoverageStats(packages map[string]*coveragePkgStats, pkg string, stmts int, count int) {
	if _, ok := packages[pkg]; !ok {
		packages[pkg] = &coveragePkgStats{}
	}
	packages[pkg].total += stmts
	if count > 0 {
		packages[pkg].covered += stmts
	}
}

func setCoveragePercentages(snap *CoverageSnapshot, packages map[string]*coveragePkgStats) {
	totalCovered := 0
	totalStmts := 0
	for pkg, stats := range packages {
		snap.Packages[pkg] = coveragePercent(stats.covered, stats.total)
		totalCovered += stats.covered
		totalStmts += stats.total
	}
	if totalStmts > 0 {
		snap.Total = coveragePercent(totalCovered, totalStmts)
	}
}

func coveragePercent(covered int, total int) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(float64(covered)/float64(total)*1000) / 10
}

// ParseCoverOutput parses the human-readable `go test -cover ./...` output.
func ParseCoverOutput(output string) (CoverageSnapshot, error) {
	snap := CoverageSnapshot{
		Timestamp: time.Now(),
		Packages:  make(map[string]float64),
	}

	re := regexp.MustCompile(`ok\s+(\S+)\s+.*coverage:\s+([\d.]+)%`)
	scanner := bufio.NewScanner(core.NewReader(output))

	totalPct := 0.0
	count := 0

	for scanner.Scan() {
		matches := re.FindStringSubmatch(scanner.Text())
		if len(matches) == 3 {
			pct, _ := strconv.ParseFloat(matches[2], 64)
			snap.Packages[matches[1]] = pct
			totalPct += pct
			count++
		}
	}

	if count > 0 {
		snap.Total = math.Round(totalPct/float64(count)*10) / 10
	}

	return snap, nil
}

func coverageFields(line string) []string {
	scanner := bufio.NewScanner(core.NewReader(line))
	scanner.Split(bufio.ScanWords)

	fields := make([]string, 0, 3)
	for scanner.Scan() {
		fields = append(fields, scanner.Text())
	}
	return fields
}

// CompareCoverage computes the difference between two snapshots.
func CompareCoverage(previous, current CoverageSnapshot) CoverageComparison {
	comp := CoverageComparison{
		TotalDelta: math.Round((current.Total-previous.Total)*10) / 10,
	}

	for pkg, curPct := range current.Packages {
		prevPct, existed := previous.Packages[pkg]
		if !existed {
			comp.NewPackages = append(comp.NewPackages, pkg)
			continue
		}

		delta := math.Round((curPct-prevPct)*10) / 10
		if delta < 0 {
			comp.Regressions = append(comp.Regressions, CoverageRegression{
				Package:  pkg,
				Previous: prevPct,
				Current:  curPct,
				Delta:    delta,
			})
		} else if delta > 0 {
			comp.Improvements = append(comp.Improvements, CoverageRegression{
				Package:  pkg,
				Previous: prevPct,
				Current:  curPct,
				Delta:    delta,
			})
		}
	}

	for pkg := range previous.Packages {
		if _, exists := current.Packages[pkg]; !exists {
			comp.Removed = append(comp.Removed, pkg)
		}
	}

	slices.Sort(comp.NewPackages)
	slices.Sort(comp.Removed)
	slices.SortFunc(comp.Regressions, func(a, b CoverageRegression) int {
		return cmp.Or(
			cmp.Compare(a.Package, b.Package),
			cmp.Compare(a.Previous, b.Previous),
			cmp.Compare(a.Current, b.Current),
			cmp.Compare(a.Delta, b.Delta),
		)
	})
	slices.SortFunc(comp.Improvements, func(a, b CoverageRegression) int {
		return cmp.Or(
			cmp.Compare(a.Package, b.Package),
			cmp.Compare(a.Previous, b.Previous),
			cmp.Compare(a.Current, b.Current),
			cmp.Compare(a.Delta, b.Delta),
		)
	})

	return comp
}
