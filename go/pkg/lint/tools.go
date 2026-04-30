package lint

import (
	"bufio"
	"context"
	"regexp"
	"slices"
	"strconv"
	"time"

	core "dappco.re/go"
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
	Path   string `json:"file_path"`
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

// CommandOutput contains process output captured by Toolkit.Run.
type CommandOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// NewToolkit creates a Toolkit rooted at the given directory.
func NewToolkit(dir string) *Toolkit {
	return &Toolkit{Dir: dir}
}

// Run executes a command and captures stdout, stderr, and exit code.
func (t *Toolkit) Run(name string, args ...string) core.Result {
	return core.Ok(runCoreCommand(context.Background(), t.Dir, name, args))
}

// FindTrackedComments greps for TODO/FIXME/HACK comments within a directory.
//
//	comments, err := lint.NewToolkit(".").FindTrackedComments("pkg/lint")
func (t *Toolkit) FindTrackedComments(dir string) core.Result {
	pattern := `\b(TODO|FIXME|HACK)\b(\(.*\))?:`
	run := t.Run("git", "grep", "--line-number", "-E", pattern, "--", dir).Value.(CommandOutput)

	if run.ExitCode == 1 && run.Stdout == "" {
		return core.Ok([]TrackedComment(nil))
	}
	if run.Err != nil && run.ExitCode != 1 {
		return core.Fail(core.E("Toolkit.FindTrackedComments", core.Sprintf("git grep failed (exit %d):\n%s", run.ExitCode, run.Stderr), run.Err))
	}

	var comments []TrackedComment
	re := regexp.MustCompile(pattern)

	for _, line := range core.Split(core.Trim(run.Stdout), "\n") {
		if line == "" {
			continue
		}
		parts := core.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		lineNum, _ := strconv.Atoi(parts[1])
		match := re.FindStringSubmatch(parts[2])
		todoType := ""
		if len(match) > 1 {
			todoType = match[1]
		}
		msg := core.Trim(re.Split(parts[2], 2)[1])

		comments = append(comments, TrackedComment{
			File:    parts[0],
			Line:    lineNum,
			Type:    todoType,
			Message: msg,
		})
	}
	return core.Ok(comments)
}

// FindTODOs is kept for compatibility with the older API name.
func (t *Toolkit) FindTODOs(dir string) core.Result {
	return t.FindTrackedComments(dir)
}

// AuditDeps runs govulncheck to find dependency vulnerabilities (text output).
func (t *Toolkit) AuditDeps() core.Result {
	run := t.Run("govulncheck", "./...").Value.(CommandOutput)
	if run.Err != nil && run.ExitCode != 0 && !core.Contains(run.Stdout, "Vulnerability") {
		return core.Fail(core.E("Toolkit.AuditDeps", core.Sprintf("govulncheck failed (exit %d):\n%s", run.ExitCode, run.Stderr), run.Err))
	}

	var vulns []Vulnerability
	scanner := bufio.NewScanner(core.NewReader(run.Stdout))
	var cur Vulnerability
	inBlock := false

	for scanner.Scan() {
		parseVulnerabilityLine(scanner.Text(), &cur, &inBlock, &vulns)
	}
	if cur.ID != "" {
		vulns = append(vulns, cur)
	}
	return core.Ok(vulns)
}

func parseVulnerabilityLine(line string, cur *Vulnerability, inBlock *bool, vulns *[]Vulnerability) {
	if core.HasPrefix(line, "Vulnerability #") {
		startVulnerability(line, cur, inBlock, vulns)
		return
	}
	if !*inBlock {
		return
	}
	switch {
	case core.Contains(line, "Package:"):
		cur.Package = vulnerabilityFieldValue(line)
	case core.Contains(line, "Found in version:"):
		cur.Version = vulnerabilityFieldValue(line)
	case line == "":
		finishVulnerability(cur, inBlock, vulns)
	case !core.HasPrefix(line, "  ") && cur.Description == "":
		cur.Description = core.Trim(line)
	}
}

func startVulnerability(line string, cur *Vulnerability, inBlock *bool, vulns *[]Vulnerability) {
	if cur.ID != "" {
		*vulns = append(*vulns, *cur)
	}
	fields := textFields(line)
	*cur = Vulnerability{}
	if len(fields) > 1 {
		cur.ID = fields[1]
	}
	*inBlock = true
}

func vulnerabilityFieldValue(line string) string {
	return core.Trim(core.SplitN(line, ":", 2)[1])
}

func finishVulnerability(cur *Vulnerability, inBlock *bool, vulns *[]Vulnerability) {
	if cur.ID != "" {
		*vulns = append(*vulns, *cur)
		*cur = Vulnerability{}
	}
	*inBlock = false
}

// DiffStat returns a summary of uncommitted changes.
func (t *Toolkit) DiffStat() core.Result {
	run := t.Run("git", "diff", "--stat").Value.(CommandOutput)
	if run.Err != nil && run.ExitCode != 0 {
		return core.Fail(core.E("Toolkit.DiffStat", core.Sprintf("git diff failed (exit %d):\n%s", run.ExitCode, run.Stderr), run.Err))
	}

	var s DiffSummary
	lines := core.Split(core.Trim(run.Stdout), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return core.Ok(s)
	}

	last := lines[len(lines)-1]
	for _, part := range core.Split(last, ",") {
		part = core.Trim(part)
		fields := textFields(part)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.Atoi(fields[0])
		switch {
		case core.Contains(part, "file"):
			s.FilesChanged = val
		case core.Contains(part, "insertion"):
			s.Insertions = val
		case core.Contains(part, "deletion"):
			s.Deletions = val
		}
	}
	return core.Ok(s)
}

// UncommittedFiles returns paths of files with uncommitted changes.
func (t *Toolkit) UncommittedFiles() core.Result {
	run := t.Run("git", "status", "--porcelain").Value.(CommandOutput)
	if run.Err != nil && run.ExitCode != 0 {
		return core.Fail(core.E("Toolkit.UncommittedFiles", "git status failed:\n"+run.Stderr, run.Err))
	}
	var files []string
	for _, line := range core.Split(core.Trim(run.Stdout), "\n") {
		if len(line) > 3 {
			files = append(files, core.Trim(line[3:]))
		}
	}
	return core.Ok(files)
}

// Lint runs go vet on the given package pattern.
func (t *Toolkit) Lint(pkg string) core.Result {
	run := t.Run("go", "vet", pkg).Value.(CommandOutput)
	if run.ExitCode == 0 {
		return core.Ok([]ToolFinding(nil))
	}
	if run.Err != nil && run.ExitCode != 2 {
		return core.Fail(core.E("Toolkit.Lint", "go vet failed", run.Err))
	}

	var findings []ToolFinding
	for _, line := range core.Split(core.Trim(run.Stderr), "\n") {
		if line == "" {
			continue
		}
		parts := core.SplitN(line, ":", 4)
		if len(parts) < 4 {
			continue
		}
		lineNum, _ := strconv.Atoi(parts[1])
		findings = append(findings, ToolFinding{
			File:    parts[0],
			Line:    lineNum,
			Message: core.Trim(parts[3]),
			Tool:    "go vet",
		})
	}
	return core.Ok(findings)
}

// ScanSecrets runs gitleaks to find potential secret leaks.
func (t *Toolkit) ScanSecrets(dir string) core.Result {
	run := t.Run("gitleaks", "detect", "--source", dir, "--report-format", "csv", "--no-git").Value.(CommandOutput)
	if run.ExitCode == 0 {
		return core.Ok([]SecretLeak(nil))
	}
	if run.Err != nil && run.ExitCode != 1 {
		return core.Fail(core.E("Toolkit.ScanSecrets", "gitleaks failed", run.Err))
	}

	var leaks []SecretLeak
	for _, line := range core.Split(core.Trim(run.Stdout), "\n") {
		if line == "" || core.HasPrefix(line, "RuleID") {
			continue
		}
		parts := core.SplitN(line, ",", 4)
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
	return core.Ok(leaks)
}

// ModTidy runs go mod tidy.
func (t *Toolkit) ModTidy() core.Result {
	run := t.Run("go", "mod", "tidy").Value.(CommandOutput)
	if run.Err != nil && run.ExitCode != 0 {
		return core.Fail(core.E("Toolkit.ModTidy", "go mod tidy failed: "+core.Trim(run.Stderr), nil))
	}
	return core.Ok(nil)
}

// Build compiles the given targets.
func (t *Toolkit) Build(targets ...string) core.Result {
	var results []BuildResult
	for _, target := range targets {
		run := t.Run("go", "build", "-o", "/dev/null", target).Value.(CommandOutput)
		r := BuildResult{Target: target}
		if run.Err != nil {
			r.Error = core.E("Toolkit.Build", core.Trim(run.Stderr), nil)
		}
		results = append(results, r)
	}
	return core.Ok(results)
}

// TestCount returns the number of test functions in a package.
func (t *Toolkit) TestCount(pkg string) core.Result {
	run := t.Run("go", "test", "-list", ".*", pkg).Value.(CommandOutput)
	if run.Err != nil && run.ExitCode != 0 {
		return core.Fail(core.E("Toolkit.TestCount", core.Sprintf("go test -list failed:\n%s", run.Stderr), run.Err))
	}
	count := 0
	for _, line := range core.Split(core.Trim(run.Stdout), "\n") {
		if core.HasPrefix(line, "Test") || core.HasPrefix(line, "Benchmark") {
			count++
		}
	}
	return core.Ok(count)
}

// Coverage runs go test -cover and parses per-package coverage percentages.
func (t *Toolkit) Coverage(pkg string) core.Result {
	if pkg == "" {
		pkg = "./..."
	}
	run := t.Run("go", "test", "-cover", pkg).Value.(CommandOutput)
	if run.Err != nil && run.ExitCode != 0 && !core.Contains(run.Stdout, "coverage:") {
		return core.Fail(core.E("Toolkit.Coverage", core.Sprintf("go test -cover failed (exit %d):\n%s", run.ExitCode, run.Stderr), run.Err))
	}

	var reports []CoverageReport
	re := regexp.MustCompile(`ok\s+(\S+)\s+.*coverage:\s+([\d.]+)%`)
	scanner := bufio.NewScanner(core.NewReader(run.Stdout))

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
	return core.Ok(reports)
}

// RaceDetect runs go test -race and parses data race warnings.
func (t *Toolkit) RaceDetect(pkg string) core.Result {
	if pkg == "" {
		pkg = "./..."
	}
	run := t.Run("go", "test", "-race", pkg).Value.(CommandOutput)
	if run.Err != nil && !core.Contains(run.Stderr, "WARNING: DATA RACE") {
		return core.Fail(core.E("Toolkit.RaceDetect", "go test -race failed", run.Err))
	}

	var races []RaceCondition
	lines := core.Split(run.Stderr, "\n")
	reFile := regexp.MustCompile(`\s+(.*\.go):(\d+)`)

	for i, line := range lines {
		if core.Contains(line, "WARNING: DATA RACE") {
			rc := RaceCondition{Desc: "Data race detected"}
			for j := i + 1; j < len(lines) && j < i+15; j++ {
				if match := reFile.FindStringSubmatch(lines[j]); len(match) == 3 {
					rc.File = core.Trim(match[1])
					rc.Line, _ = strconv.Atoi(match[2])
					break
				}
			}
			races = append(races, rc)
		}
	}
	return core.Ok(races)
}

// GocycloComplexity runs gocyclo and returns functions exceeding the threshold.
// For native AST analysis without external tools, use AnalyseComplexity instead.
func (t *Toolkit) GocycloComplexity(threshold int) core.Result {
	run := t.Run("gocyclo", "-over", strconv.Itoa(threshold), ".").Value.(CommandOutput)
	if run.Err != nil && run.ExitCode == -1 {
		return core.Fail(core.E("Toolkit.GocycloComplexity", "gocyclo not available:\n"+run.Stderr, run.Err))
	}

	var funcs []ComplexFunc
	scanner := bufio.NewScanner(core.NewReader(run.Stdout))

	for scanner.Scan() {
		fields := textFields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		score, _ := strconv.Atoi(fields[0])
		fileParts := core.Split(fields[3], ":")
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
	return core.Ok(funcs)
}

// DepGraph runs go mod graph and builds a dependency graph.
func (t *Toolkit) DepGraph(pkg string) core.Result {
	run := t.Run("go", "mod", "graph").Value.(CommandOutput)
	if run.Err != nil && run.ExitCode != 0 {
		return core.Fail(core.E("Toolkit.DepGraph", core.Sprintf("go mod graph failed (exit %d):\n%s", run.ExitCode, run.Stderr), run.Err))
	}

	graph := &Graph{Edges: make(map[string][]string)}
	nodes := make(map[string]struct{})
	scanner := bufio.NewScanner(core.NewReader(run.Stdout))

	for scanner.Scan() {
		parts := textFields(scanner.Text())
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
	return core.Ok(graph)
}

// GitLog returns the last n commits from git history.
func (t *Toolkit) GitLog(n int) core.Result {
	run := t.Run("git", "l"+"og", core.Sprintf("-n%d", n), "--format=%H|%an|%aI|%s").Value.(CommandOutput)
	if run.Err != nil && run.ExitCode != 0 {
		return core.Fail(core.E("Toolkit.GitLog", core.Sprintf("git log failed (exit %d):\n%s", run.ExitCode, run.Stderr), run.Err))
	}

	var commits []Commit
	scanner := bufio.NewScanner(core.NewReader(run.Stdout))

	for scanner.Scan() {
		parts := core.SplitN(scanner.Text(), "|", 4)
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
	return core.Ok(commits)
}

// CheckPerms walks a directory and flags files with overly permissive modes.
func (t *Toolkit) CheckPerms(dir string) core.Result {
	var issues []PermIssue
	err := core.PathWalkDir(core.PathJoin(t.Dir, dir), func(path string, entry core.FsDirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
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
		return core.Fail(core.E("Toolkit.CheckPerms", "walk failed", err))
	}
	return core.Ok(issues)
}

func runCoreCommand(ctx context.Context, workingDir string, name string, args []string) CommandOutput {
	var output CommandOutput
	binary := findExecutable(name)
	if !binary.OK {
		output.ExitCode = -1
		output.Err, _ = binary.Value.(error)
		return output
	}

	stdout := core.NewBuilder()
	stderr := core.NewBuilder()
	cmd := &core.Cmd{
		Path:   binary.Value.(string),
		Args:   append([]string{name}, args...),
		Dir:    workingDir,
		Stdout: stdout,
		Stderr: stderr,
	}

	err := cmd.Start()
	if err == nil {
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()
		select {
		case err = <-done:
		case <-ctx.Done():
			if cmd.Process != nil {
				killErr := cmd.Process.Kill()
				if killErr != nil && err == nil {
					err = killErr
				}
			}
			if err == nil {
				err = ctx.Err()
			}
		}
	}

	output.Stdout = stdout.String()
	output.Stderr = stderr.String()
	output.ExitCode = commandExitCode(err)
	output.Err = err
	return output
}

func commandExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(interface{ ExitCode() int }); ok {
		return exitErr.ExitCode()
	}
	return -1
}

func findExecutable(name string) core.Result {
	if name == "" {
		return core.Fail(core.E("findExecutable", "empty executable name", nil))
	}
	if core.Contains(name, string(core.PathSeparator)) {
		if executablePath(name) {
			return core.Ok(name)
		}
		return core.Fail(core.E("findExecutable", core.Sprintf("%s is not executable", name), nil))
	}
	for _, dir := range core.Split(core.Getenv("PATH"), string(core.PathListSeparator)) {
		if dir == "" {
			dir = "."
		}
		candidate := core.PathJoin(dir, name)
		if executablePath(candidate) {
			return core.Ok(candidate)
		}
	}
	return core.Fail(core.E("findExecutable", core.Sprintf("%s was not found in PATH", name), nil))
}

func executablePath(path string) bool {
	stat := core.Stat(path)
	if !stat.OK {
		return false
	}
	info := stat.Value.(core.FsFileInfo)
	return !info.IsDir() && info.Mode()&0111 != 0
}

func textFields(input string) []string {
	scanner := bufio.NewScanner(core.NewReader(input))
	scanner.Split(bufio.ScanWords)
	var fields []string
	for scanner.Scan() {
		fields = append(fields, scanner.Text())
	}
	return fields
}
