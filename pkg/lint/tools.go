package lint

import (
	"bufio"
	"os"
	"os/exec" // Note: AX-6 — Toolkit.Run needs split stdout/stderr and exit codes; core.Process().Run returns combined output only.
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	core "dappco.re/go/core"
	coreerr "dappco.re/go/core/log"
)

// ToolFinding represents a single issue found by an external tool (e.g. go vet).
// Distinct from Finding, which represents a catalog rule match.
type ToolFinding struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Message string `json:"message"`
	Tool    string `json:"tool"`
}

// CoverageReport holds the test coverage percentage for a package.
type CoverageReport struct {
	Package    string  `json:"package"`
	Percentage float64 `json:"percentage"`
}

// RaceCondition represents a data race detected by the Go race detector.
type RaceCondition struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Desc string `json:"desc"`
}

// TrackedComment represents a tracked code comment like TODO, FIXME, or HACK.
type TrackedComment struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

// TODO is kept for compatibility with the older API name.
type TODO = TrackedComment

// Vulnerability represents a dependency vulnerability from govulncheck text output.
type Vulnerability struct {
	ID          string `json:"id"`
	Package     string `json:"package"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// SecretLeak represents a potential secret found in the codebase.
type SecretLeak struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	RuleID string `json:"rule_id"`
	Match  string `json:"match"`
}

// PermIssue represents a file permission issue.
type PermIssue struct {
	File       string `json:"file"`
	Permission string `json:"permission"`
	Issue      string `json:"issue"`
}

// DiffSummary provides a summary of changes.
type DiffSummary struct {
	FilesChanged int `json:"files_changed"`
	Insertions   int `json:"insertions"`
	Deletions    int `json:"deletions"`
}

// Commit represents a single git commit.
type Commit struct {
	Hash    string    `json:"hash"`
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
	Message string    `json:"message"`
}

// BuildResult holds the outcome of a single build target.
type BuildResult struct {
	Target string `json:"target"`
	Path   string `json:"path"`
	Error  error  `json:"-"`
}

// Graph represents a dependency graph.
type Graph struct {
	Nodes []string            `json:"nodes"`
	Edges map[string][]string `json:"edges"`
}

// ComplexFunc represents a function with its cyclomatic complexity score
// as reported by the gocyclo subprocess. For native AST analysis, use ComplexityResult.
type ComplexFunc struct {
	Package  string `json:"package"`
	FuncName string `json:"func_name"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Score    int    `json:"score"`
}

// Toolkit wraps common dev automation commands into structured Go APIs.
type Toolkit struct {
	Dir string // Working directory for commands
}

// NewToolkit creates a Toolkit rooted at the given directory.
func NewToolkit(dir string) *Toolkit {
	return &Toolkit{Dir: dir}
}

// Run executes a command and captures stdout, stderr, and exit code.
func (t *Toolkit) Run(name string, args ...string) (stdout, stderr string, exitCode int, err error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = t.Dir
	stdoutBuf := core.NewBuilder()
	stderrBuf := core.NewBuilder()
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	err = cmd.Run()
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return
}

// FindTrackedComments greps for TODO/FIXME/HACK comments within a directory.
//
//	comments, err := lint.NewToolkit(".").FindTrackedComments("pkg/lint")
func (t *Toolkit) FindTrackedComments(dir string) ([]TrackedComment, error) {
	pattern := `\b(TODO|FIXME|HACK)\b(\(.*\))?:`
	stdout, stderr, exitCode, err := t.Run("git", "grep", "--line-number", "-E", pattern, "--", dir)

	if exitCode == 1 && stdout == "" {
		return nil, nil
	}
	if err != nil && exitCode != 1 {
		return nil, coreerr.E("Toolkit.FindTrackedComments", core.Sprintf("git grep failed (exit %d):\n%s", exitCode, stderr), err)
	}

	var comments []TrackedComment
	re := regexp.MustCompile(pattern)

	for line := range strings.SplitSeq(strings.TrimSpace(stdout), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		lineNum, _ := strconv.Atoi(parts[1])
		match := re.FindStringSubmatch(parts[2])
		todoType := ""
		if len(match) > 1 {
			todoType = match[1]
		}
		msg := strings.TrimSpace(re.Split(parts[2], 2)[1])

		comments = append(comments, TrackedComment{
			File:    parts[0],
			Line:    lineNum,
			Type:    todoType,
			Message: msg,
		})
	}
	return comments, nil
}

// FindTODOs is kept for compatibility with the older API name.
func (t *Toolkit) FindTODOs(dir string) ([]TODO, error) {
	return t.FindTrackedComments(dir)
}

// AuditDeps runs govulncheck to find dependency vulnerabilities (text output).
func (t *Toolkit) AuditDeps() ([]Vulnerability, error) {
	stdout, stderr, exitCode, err := t.Run("govulncheck", "./...")
	if err != nil && exitCode != 0 && !strings.Contains(stdout, "Vulnerability") {
		return nil, coreerr.E("Toolkit.AuditDeps", core.Sprintf("govulncheck failed (exit %d):\n%s", exitCode, stderr), err)
	}

	var vulns []Vulnerability
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	var cur Vulnerability
	inBlock := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Vulnerability #") {
			if cur.ID != "" {
				vulns = append(vulns, cur)
			}
			fields := strings.Fields(line)
			cur = Vulnerability{}
			if len(fields) > 1 {
				cur.ID = fields[1]
			}
			inBlock = true
		} else if inBlock {
			switch {
			case strings.Contains(line, "Package:"):
				cur.Package = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			case strings.Contains(line, "Found in version:"):
				cur.Version = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			case line == "":
				if cur.ID != "" {
					vulns = append(vulns, cur)
					cur = Vulnerability{}
				}
				inBlock = false
			default:
				if !strings.HasPrefix(line, "  ") && cur.Description == "" {
					cur.Description = strings.TrimSpace(line)
				}
			}
		}
	}
	if cur.ID != "" {
		vulns = append(vulns, cur)
	}
	return vulns, nil
}

// DiffStat returns a summary of uncommitted changes.
func (t *Toolkit) DiffStat() (DiffSummary, error) {
	stdout, stderr, exitCode, err := t.Run("git", "diff", "--stat")
	if err != nil && exitCode != 0 {
		return DiffSummary{}, coreerr.E("Toolkit.DiffStat", core.Sprintf("git diff failed (exit %d):\n%s", exitCode, stderr), err)
	}

	var s DiffSummary
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return s, nil
	}

	last := lines[len(lines)-1]
	for _, part := range strings.Split(last, ",") {
		part = strings.TrimSpace(part)
		fields := strings.Fields(part)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.Atoi(fields[0])
		switch {
		case strings.Contains(part, "file"):
			s.FilesChanged = val
		case strings.Contains(part, "insertion"):
			s.Insertions = val
		case strings.Contains(part, "deletion"):
			s.Deletions = val
		}
	}
	return s, nil
}

// UncommittedFiles returns paths of files with uncommitted changes.
func (t *Toolkit) UncommittedFiles() ([]string, error) {
	stdout, stderr, exitCode, err := t.Run("git", "status", "--porcelain")
	if err != nil && exitCode != 0 {
		return nil, coreerr.E("Toolkit.UncommittedFiles", "git status failed:\n"+stderr, err)
	}
	var files []string
	for line := range strings.SplitSeq(strings.TrimSpace(stdout), "\n") {
		if len(line) > 3 {
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}
	return files, nil
}

// Lint runs go vet on the given package pattern.
func (t *Toolkit) Lint(pkg string) ([]ToolFinding, error) {
	_, stderr, exitCode, err := t.Run("go", "vet", pkg)
	if exitCode == 0 {
		return nil, nil
	}
	if err != nil && exitCode != 2 {
		return nil, coreerr.E("Toolkit.Lint", "go vet failed", err)
	}

	var findings []ToolFinding
	for line := range strings.SplitSeq(strings.TrimSpace(stderr), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 4 {
			continue
		}
		lineNum, _ := strconv.Atoi(parts[1])
		findings = append(findings, ToolFinding{
			File:    parts[0],
			Line:    lineNum,
			Message: strings.TrimSpace(parts[3]),
			Tool:    "go vet",
		})
	}
	return findings, nil
}

// ScanSecrets runs gitleaks to find potential secret leaks.
func (t *Toolkit) ScanSecrets(dir string) ([]SecretLeak, error) {
	stdout, _, exitCode, err := t.Run("gitleaks", "detect", "--source", dir, "--report-format", "csv", "--no-git")
	if exitCode == 0 {
		return nil, nil
	}
	if err != nil && exitCode != 1 {
		return nil, coreerr.E("Toolkit.ScanSecrets", "gitleaks failed", err)
	}

	var leaks []SecretLeak
	for line := range strings.SplitSeq(strings.TrimSpace(stdout), "\n") {
		if line == "" || strings.HasPrefix(line, "RuleID") {
			continue
		}
		parts := strings.SplitN(line, ",", 4)
		if len(parts) < 4 {
			continue
		}
		lineNum, _ := strconv.Atoi(parts[2])
		leaks = append(leaks, SecretLeak{
			RuleID: parts[0],
			File:   parts[1],
			Line:   lineNum,
			Match:  parts[3],
		})
	}
	return leaks, nil
}

// ModTidy runs go mod tidy.
func (t *Toolkit) ModTidy() error {
	_, stderr, exitCode, err := t.Run("go", "mod", "tidy")
	if err != nil && exitCode != 0 {
		return coreerr.E("Toolkit.ModTidy", "go mod tidy failed: "+strings.TrimSpace(stderr), nil)
	}
	return nil
}

// Build compiles the given targets.
func (t *Toolkit) Build(targets ...string) ([]BuildResult, error) {
	var results []BuildResult
	for _, target := range targets {
		_, stderr, _, err := t.Run("go", "build", "-o", "/dev/null", target)
		r := BuildResult{Target: target}
		if err != nil {
			r.Error = coreerr.E("Toolkit.Build", strings.TrimSpace(stderr), nil)
		}
		results = append(results, r)
	}
	return results, nil
}

// TestCount returns the number of test functions in a package.
func (t *Toolkit) TestCount(pkg string) (int, error) {
	stdout, stderr, exitCode, err := t.Run("go", "test", "-list", ".*", pkg)
	if err != nil && exitCode != 0 {
		return 0, coreerr.E("Toolkit.TestCount", core.Sprintf("go test -list failed:\n%s", stderr), err)
	}
	count := 0
	for line := range strings.SplitSeq(strings.TrimSpace(stdout), "\n") {
		if strings.HasPrefix(line, "Test") || strings.HasPrefix(line, "Benchmark") {
			count++
		}
	}
	return count, nil
}

// Coverage runs go test -cover and parses per-package coverage percentages.
func (t *Toolkit) Coverage(pkg string) ([]CoverageReport, error) {
	if pkg == "" {
		pkg = "./..."
	}
	stdout, stderr, exitCode, err := t.Run("go", "test", "-cover", pkg)
	if err != nil && exitCode != 0 && !strings.Contains(stdout, "coverage:") {
		return nil, coreerr.E("Toolkit.Coverage", core.Sprintf("go test -cover failed (exit %d):\n%s", exitCode, stderr), err)
	}

	var reports []CoverageReport
	re := regexp.MustCompile(`ok\s+(\S+)\s+.*coverage:\s+([\d.]+)%`)
	scanner := bufio.NewScanner(strings.NewReader(stdout))

	for scanner.Scan() {
		matches := re.FindStringSubmatch(scanner.Text())
		if len(matches) == 3 {
			pct, _ := strconv.ParseFloat(matches[2], 64)
			reports = append(reports, CoverageReport{
				Package:    matches[1],
				Percentage: pct,
			})
		}
	}
	return reports, nil
}

// RaceDetect runs go test -race and parses data race warnings.
func (t *Toolkit) RaceDetect(pkg string) ([]RaceCondition, error) {
	if pkg == "" {
		pkg = "./..."
	}
	_, stderr, _, err := t.Run("go", "test", "-race", pkg)
	if err != nil && !strings.Contains(stderr, "WARNING: DATA RACE") {
		return nil, coreerr.E("Toolkit.RaceDetect", "go test -race failed", err)
	}

	var races []RaceCondition
	lines := strings.Split(stderr, "\n")
	reFile := regexp.MustCompile(`\s+(.*\.go):(\d+)`)

	for i, line := range lines {
		if strings.Contains(line, "WARNING: DATA RACE") {
			rc := RaceCondition{Desc: "Data race detected"}
			for j := i + 1; j < len(lines) && j < i+15; j++ {
				if match := reFile.FindStringSubmatch(lines[j]); len(match) == 3 {
					rc.File = strings.TrimSpace(match[1])
					rc.Line, _ = strconv.Atoi(match[2])
					break
				}
			}
			races = append(races, rc)
		}
	}
	return races, nil
}

// GocycloComplexity runs gocyclo and returns functions exceeding the threshold.
// For native AST analysis without external tools, use AnalyseComplexity instead.
func (t *Toolkit) GocycloComplexity(threshold int) ([]ComplexFunc, error) {
	stdout, stderr, exitCode, err := t.Run("gocyclo", "-over", strconv.Itoa(threshold), ".")
	if err != nil && exitCode == -1 {
		return nil, coreerr.E("Toolkit.GocycloComplexity", "gocyclo not available:\n"+stderr, err)
	}

	var funcs []ComplexFunc
	scanner := bufio.NewScanner(strings.NewReader(stdout))

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		score, _ := strconv.Atoi(fields[0])
		fileParts := strings.Split(fields[3], ":")
		line := 0
		if len(fileParts) > 1 {
			line, _ = strconv.Atoi(fileParts[1])
		}

		funcs = append(funcs, ComplexFunc{
			Score:    score,
			Package:  fields[1],
			FuncName: fields[2],
			File:     fileParts[0],
			Line:     line,
		})
	}
	return funcs, nil
}

// DepGraph runs go mod graph and builds a dependency graph.
func (t *Toolkit) DepGraph(pkg string) (*Graph, error) {
	stdout, stderr, exitCode, err := t.Run("go", "mod", "graph")
	if err != nil && exitCode != 0 {
		return nil, coreerr.E("Toolkit.DepGraph", core.Sprintf("go mod graph failed (exit %d):\n%s", exitCode, stderr), err)
	}

	graph := &Graph{Edges: make(map[string][]string)}
	nodes := make(map[string]struct{})
	scanner := bufio.NewScanner(strings.NewReader(stdout))

	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) >= 2 {
			src, dst := parts[0], parts[1]
			graph.Edges[src] = append(graph.Edges[src], dst)
			nodes[src] = struct{}{}
			nodes[dst] = struct{}{}
		}
	}

	for node := range nodes {
		graph.Nodes = append(graph.Nodes, node)
	}
	slices.Sort(graph.Nodes)
	for src := range graph.Edges {
		slices.Sort(graph.Edges[src])
	}
	return graph, nil
}

// GitLog returns the last n commits from git history.
func (t *Toolkit) GitLog(n int) ([]Commit, error) {
	stdout, stderr, exitCode, err := t.Run("git", "log", core.Sprintf("-n%d", n), "--format=%H|%an|%aI|%s")
	if err != nil && exitCode != 0 {
		return nil, coreerr.E("Toolkit.GitLog", core.Sprintf("git log failed (exit %d):\n%s", exitCode, stderr), err)
	}

	var commits []Commit
	scanner := bufio.NewScanner(strings.NewReader(stdout))

	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "|", 4)
		if len(parts) < 4 {
			continue
		}
		date, _ := time.Parse(time.RFC3339, parts[2])
		commits = append(commits, Commit{
			Hash:    parts[0],
			Author:  parts[1],
			Date:    date,
			Message: parts[3],
		})
	}
	return commits, nil
}

// CheckPerms walks a directory and flags files with overly permissive modes.
func (t *Toolkit) CheckPerms(dir string) ([]PermIssue, error) {
	var issues []PermIssue
	err := filepath.Walk(filepath.Join(t.Dir, dir), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		mode := info.Mode().Perm()
		if mode&0o020 != 0 && mode&0o002 != 0 {
			issues = append(issues, PermIssue{
				File:       path,
				Permission: core.Sprintf("%04o", mode),
				Issue:      "Group and world-writable",
			})
		} else if mode&0o002 != 0 {
			issues = append(issues, PermIssue{
				File:       path,
				Permission: core.Sprintf("%04o", mode),
				Issue:      "World-writable",
			})
		}
		return nil
	})
	if err != nil {
		return nil, coreerr.E("Toolkit.CheckPerms", "walk failed", err)
	}
	return issues, nil
}
