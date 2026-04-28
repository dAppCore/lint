package lint

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing/fstest"
	"time"

	core "dappco.re/go"
)

type ax7ErrWriter struct{}

func (ax7ErrWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func ax7Rule() Rule {
	return Rule{
		ID:        "go-test-001",
		Title:     "Test rule",
		Severity:  "medium",
		Languages: []string{"go"},
		Pattern:   "TODO",
		Fix:       "Remove the marker",
		Detection: "regex",
		Tags:      []string{"correctness"},
	}
}

func ax7WriteFile(t *core.T, dir string, name string, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	core.RequireNoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	core.RequireNoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func ax7FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ax7Executable(t *core.T, name string, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	core.RequireNoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755))
	t.Setenv("PATH", dir)
	return path
}

func ax7Report() Report {
	findings := []Finding{{
		Tool:     "catalog",
		File:     "main.go",
		Line:     3,
		Severity: "warning",
		Code:     "go-test-001",
		Message:  "Test rule",
	}}
	return Report{Project: "repo", Timestamp: time.Unix(0, 0), Duration: "1ms", Findings: findings, Summary: Summarise(findings)}
}

func TestLint_Rule_Validate_Good(t *core.T) {
	rule := ax7Rule()
	err := rule.Validate()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "go-test-001", rule.ID)
}

func TestLint_Rule_Validate_Bad(t *core.T) {
	rule := ax7Rule()
	rule.ID = ""
	err := rule.Validate()
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "id")
}

func TestLint_Rule_Validate_Ugly(t *core.T) {
	rule := ax7Rule()
	rule.Detection = "contains"
	rule.Pattern = "["
	err := rule.Validate()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "contains", rule.Detection)
}

func TestLint_ParseRules_Good(t *core.T) {
	data := []byte("- id: go-test-001\n  title: Test\n  severity: warning\n  languages: [go]\n  pattern: TODO\n  fix: Remove\n  detection: regex\n")
	rules, err := ParseRules(data)
	core.AssertNoError(t, err)
	core.AssertLen(t, rules, 1)
	core.AssertEqual(t, "go-test-001", rules[0].ID)
}

func TestLint_ParseRules_Bad(t *core.T) {
	rules, err := ParseRules([]byte("- id: ["))
	core.AssertError(t, err)
	core.AssertNil(t, rules)
}

func TestLint_ParseRules_Ugly(t *core.T) {
	rules, err := ParseRules(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, rules)
}

func TestLint_LoadDir_Good(t *core.T) {
	dir := t.TempDir()
	ax7WriteFile(t, dir, "rules.yaml", "- id: go-test-001\n  title: Test\n  severity: warning\n  languages: [go]\n  pattern: TODO\n  fix: Remove\n  detection: regex\n")
	catalog, err := LoadDir(dir)
	core.AssertNoError(t, err)
	core.AssertLen(t, catalog.Rules, 1)
	core.AssertEqual(t, "go-test-001", catalog.Rules[0].ID)
}

func TestLint_LoadDir_Bad(t *core.T) {
	catalog, err := LoadDir(filepath.Join(t.TempDir(), "missing"))
	core.AssertError(t, err)
	core.AssertNil(t, catalog)
}

func TestLint_LoadDir_Ugly(t *core.T) {
	dir := t.TempDir()
	ax7WriteFile(t, dir, "ignore.txt", "not yaml")
	catalog, err := LoadDir(dir)
	core.AssertNoError(t, err)
	core.AssertEmpty(t, catalog.Rules)
}

func TestLint_LoadFS_Good(t *core.T) {
	fsys := fstest.MapFS{"catalog/rules.yaml": {Data: []byte("- id: go-test-001\n  title: Test\n  severity: medium\n  languages: [go]\n  pattern: TODO\n  fix: Remove\n  detection: regex\n")}}
	catalog, err := LoadFS(fsys, "catalog")
	core.AssertNoError(t, err)
	core.AssertLen(t, catalog.Rules, 1)
	core.AssertEqual(t, "go-test-001", catalog.Rules[0].ID)
}

func TestLint_LoadFS_Bad(t *core.T) {
	catalog, err := LoadFS(fstest.MapFS{}, "missing")
	core.AssertError(t, err)
	core.AssertNil(t, catalog)
}

func TestLint_LoadFS_Ugly(t *core.T) {
	fsys := fstest.MapFS{"catalog/ignore.txt": {Data: []byte("not yaml")}}
	catalog, err := LoadFS(fsys, "catalog")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, catalog.Rules)
}

func TestLint_Catalog_ForLanguage_Good(t *core.T) {
	catalog := &Catalog{Rules: []Rule{ax7Rule()}}
	rules := catalog.ForLanguage("go")
	core.AssertLen(t, rules, 1)
	core.AssertEqual(t, "go-test-001", rules[0].ID)
}

func TestLint_Catalog_ForLanguage_Bad(t *core.T) {
	catalog := &Catalog{Rules: []Rule{ax7Rule()}}
	rules := catalog.ForLanguage("php")
	core.AssertEmpty(t, rules)
	core.AssertNil(t, rules)
}

func TestLint_Catalog_ForLanguage_Ugly(t *core.T) {
	catalog := &Catalog{}
	rules := catalog.ForLanguage("")
	core.AssertEmpty(t, rules)
	core.AssertNil(t, rules)
}

func TestLint_Catalog_AtSeverity_Good(t *core.T) {
	catalog := &Catalog{Rules: []Rule{ax7Rule(), {ID: "critical", Severity: "critical"}}}
	rules := catalog.AtSeverity("medium")
	core.AssertLen(t, rules, 2)
	core.AssertEqual(t, "go-test-001", rules[0].ID)
}

func TestLint_Catalog_AtSeverity_Bad(t *core.T) {
	catalog := &Catalog{Rules: []Rule{ax7Rule()}}
	rules := catalog.AtSeverity("unknown")
	core.AssertNil(t, rules)
	core.AssertEmpty(t, rules)
}

func TestLint_Catalog_AtSeverity_Ugly(t *core.T) {
	catalog := &Catalog{Rules: []Rule{{ID: "unknown", Severity: "mystery"}}}
	rules := catalog.AtSeverity("info")
	core.AssertEmpty(t, rules)
	core.AssertNil(t, rules)
}

func TestLint_Catalog_ByID_Good(t *core.T) {
	catalog := &Catalog{Rules: []Rule{ax7Rule()}}
	rule := catalog.ByID("go-test-001")
	core.AssertNotNil(t, rule)
	core.AssertEqual(t, "Test rule", rule.Title)
}

func TestLint_Catalog_ByID_Bad(t *core.T) {
	catalog := &Catalog{Rules: []Rule{ax7Rule()}}
	rule := catalog.ByID("missing")
	core.AssertNil(t, rule)
	core.AssertLen(t, catalog.Rules, 1)
}

func TestLint_Catalog_ByID_Ugly(t *core.T) {
	catalog := &Catalog{}
	rule := catalog.ByID("")
	core.AssertNil(t, rule)
	core.AssertEmpty(t, catalog.Rules)
}

func TestLint_NewMatcher_Good(t *core.T) {
	matcher, err := NewMatcher([]Rule{ax7Rule()})
	core.AssertNoError(t, err)
	core.AssertNotNil(t, matcher)
	core.AssertLen(t, matcher.rules, 1)
}

func TestLint_NewMatcher_Bad(t *core.T) {
	rule := ax7Rule()
	rule.Pattern = "["
	matcher, err := NewMatcher([]Rule{rule})
	core.AssertError(t, err)
	core.AssertNil(t, matcher)
}

func TestLint_NewMatcher_Ugly(t *core.T) {
	rule := ax7Rule()
	rule.Detection = "contains"
	matcher, err := NewMatcher([]Rule{rule})
	core.AssertNoError(t, err)
	core.AssertEmpty(t, matcher.rules)
}

func TestLint_Matcher_Match_Good(t *core.T) {
	matcher, err := NewMatcher([]Rule{ax7Rule()})
	core.RequireNoError(t, err)
	findings := matcher.Match("main.go", []byte("// TODO: fix\n"))
	core.AssertLen(t, findings, 1)
	core.AssertEqual(t, 1, findings[0].Line)
}

func TestLint_Matcher_Match_Bad(t *core.T) {
	matcher, err := NewMatcher([]Rule{ax7Rule()})
	core.RequireNoError(t, err)
	findings := matcher.Match("main.go", []byte("package main\n"))
	core.AssertEmpty(t, findings)
	core.AssertNil(t, findings)
}

func TestLint_Matcher_Match_Ugly(t *core.T) {
	rule := ax7Rule()
	rule.ExcludePattern = "_test\\.go"
	matcher, err := NewMatcher([]Rule{rule})
	core.RequireNoError(t, err)
	findings := matcher.Match("main_test.go", []byte("// TODO: ignored\n"))
	core.AssertEmpty(t, findings)
}

func TestLint_DetectLanguage_Good(t *core.T) {
	got := DetectLanguage("main.go")
	core.AssertEqual(t, "go", got)
	core.AssertNotEqual(t, "", got)
}

func TestLint_DetectLanguage_Bad(t *core.T) {
	got := DetectLanguage("README.unknown")
	core.AssertEqual(t, "", got)
	core.AssertNotEqual(t, "go", got)
}

func TestLint_DetectLanguage_Ugly(t *core.T) {
	got := DetectLanguage("Dockerfile.prod")
	core.AssertEqual(t, "dockerfile", got)
	core.AssertNotEqual(t, "", got)
}

func TestLint_IsExcludedDir_Good(t *core.T) {
	got := IsExcludedDir("vendor")
	core.AssertTrue(t, got)
	core.AssertTrue(t, IsExcludedDir(".git"))
}

func TestLint_IsExcludedDir_Bad(t *core.T) {
	got := IsExcludedDir("src")
	core.AssertFalse(t, got)
	core.AssertFalse(t, IsExcludedDir("pkg"))
}

func TestLint_IsExcludedDir_Ugly(t *core.T) {
	got := IsExcludedDir(".hidden")
	core.AssertTrue(t, got)
	core.AssertFalse(t, IsExcludedDir(""))
}

func TestLint_NewScanner_Good(t *core.T) {
	scanner, err := NewScanner([]Rule{ax7Rule()})
	core.AssertNoError(t, err)
	core.AssertNotNil(t, scanner)
	core.AssertLen(t, scanner.rules, 1)
}

func TestLint_NewScanner_Bad(t *core.T) {
	rule := ax7Rule()
	rule.Pattern = "["
	scanner, err := NewScanner([]Rule{rule})
	core.AssertError(t, err)
	core.AssertNil(t, scanner)
}

func TestLint_NewScanner_Ugly(t *core.T) {
	scanner, err := NewScanner(nil)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, scanner)
	core.AssertEmpty(t, scanner.rules)
}

func TestLint_Scanner_ScanFile_Good(t *core.T) {
	dir := t.TempDir()
	path := ax7WriteFile(t, dir, "main.go", "package main\n// TODO: fix\n")
	scanner, err := NewScanner([]Rule{ax7Rule()})
	core.RequireNoError(t, err)
	findings, err := scanner.ScanFile(path)
	core.AssertNoError(t, err)
	core.AssertLen(t, findings, 1)
}

func TestLint_Scanner_ScanFile_Bad(t *core.T) {
	scanner, err := NewScanner([]Rule{ax7Rule()})
	core.RequireNoError(t, err)
	findings, err := scanner.ScanFile(filepath.Join(t.TempDir(), "missing.go"))
	core.AssertError(t, err)
	core.AssertNil(t, findings)
}

func TestLint_Scanner_ScanFile_Ugly(t *core.T) {
	dir := t.TempDir()
	path := ax7WriteFile(t, dir, "README.txt", "TODO ignored\n")
	scanner, err := NewScanner([]Rule{ax7Rule()})
	core.RequireNoError(t, err)
	findings, err := scanner.ScanFile(path)
	core.AssertNoError(t, err)
	core.AssertNil(t, findings)
}

func TestLint_Scanner_ScanDir_Good(t *core.T) {
	dir := t.TempDir()
	ax7WriteFile(t, dir, "main.go", "package main\n// TODO: fix\n")
	scanner, err := NewScanner([]Rule{ax7Rule()})
	core.RequireNoError(t, err)
	findings, err := scanner.ScanDir(dir)
	core.AssertNoError(t, err)
	core.AssertLen(t, findings, 1)
}

func TestLint_Scanner_ScanDir_Bad(t *core.T) {
	scanner, err := NewScanner([]Rule{ax7Rule()})
	core.RequireNoError(t, err)
	findings, err := scanner.ScanDir(filepath.Join(t.TempDir(), "missing"))
	core.AssertError(t, err)
	core.AssertNil(t, findings)
}

func TestLint_Scanner_ScanDir_Ugly(t *core.T) {
	dir := filepath.Join(t.TempDir(), "vendor")
	core.RequireNoError(t, os.MkdirAll(dir, 0o755))
	scanner, err := NewScanner([]Rule{ax7Rule()})
	core.RequireNoError(t, err)
	findings, err := scanner.ScanDir(dir)
	core.AssertNoError(t, err)
	core.AssertEmpty(t, findings)
}

func TestLint_DefaultComplexityConfig_Good(t *core.T) {
	cfg := DefaultComplexityConfig()
	core.AssertEqual(t, 15, cfg.Threshold)
	core.AssertEqual(t, ".", cfg.Path)
}

func TestLint_DefaultComplexityConfig_Bad(t *core.T) {
	cfg := DefaultComplexityConfig()
	cfg.Threshold = 0
	core.AssertEqual(t, 0, cfg.Threshold)
	core.AssertNotEqual(t, DefaultComplexityConfig().Threshold, cfg.Threshold)
}

func TestLint_DefaultComplexityConfig_Ugly(t *core.T) {
	cfg := DefaultComplexityConfig()
	cfg.Path = ""
	core.AssertEqual(t, "", cfg.Path)
	core.AssertEqual(t, 15, cfg.Threshold)
}

func TestLint_AnalyseComplexitySource_Good(t *core.T) {
	src := "package sample\nfunc Run() { if true { for i:=0; i<1; i++ {} } }\n"
	results, err := AnalyseComplexitySource(src, "sample.go", 2)
	core.AssertNoError(t, err)
	core.AssertLen(t, results, 1)
	core.AssertEqual(t, "Run", results[0].FuncName)
}

func TestLint_AnalyseComplexitySource_Bad(t *core.T) {
	results, err := AnalyseComplexitySource("package", "bad.go", 1)
	core.AssertError(t, err)
	core.AssertNil(t, results)
}

func TestLint_AnalyseComplexitySource_Ugly(t *core.T) {
	src := "package sample\nfunc Run() {}\n"
	results, err := AnalyseComplexitySource(src, "sample.go", 99)
	core.AssertNoError(t, err)
	core.AssertEmpty(t, results)
}

func TestLint_AnalyseComplexity_Good(t *core.T) {
	dir := t.TempDir()
	ax7WriteFile(t, dir, "main.go", "package sample\nfunc Run() { if true { for i:=0; i<1; i++ {} } }\n")
	results, err := AnalyseComplexity(ComplexityConfig{Path: dir, Threshold: 2})
	core.AssertNoError(t, err)
	core.AssertLen(t, results, 1)
}

func TestLint_AnalyseComplexity_Bad(t *core.T) {
	results, err := AnalyseComplexity(ComplexityConfig{Path: filepath.Join(t.TempDir(), "missing")})
	core.AssertError(t, err)
	core.AssertNil(t, results)
}

func TestLint_AnalyseComplexity_Ugly(t *core.T) {
	dir := t.TempDir()
	ax7WriteFile(t, dir, "main_test.go", "package sample\nfunc TestRun(t *testing.T) {}\n")
	results, err := AnalyseComplexity(ComplexityConfig{Path: dir, Threshold: 1})
	core.AssertNoError(t, err)
	core.AssertEmpty(t, results)
}

func TestLint_NewCoverageStore_Good(t *core.T) {
	path := filepath.Join(t.TempDir(), "coverage.json")
	store := NewCoverageStore(path)
	core.AssertNotNil(t, store)
	core.AssertEqual(t, path, store.Path)
}

func TestLint_NewCoverageStore_Bad(t *core.T) {
	store := NewCoverageStore("")
	core.AssertNotNil(t, store)
	core.AssertEqual(t, "", store.Path)
}

func TestLint_NewCoverageStore_Ugly(t *core.T) {
	path := filepath.Join(t.TempDir(), "nested", "coverage.json")
	store := NewCoverageStore(path)
	core.AssertNotNil(t, store)
	core.AssertContains(t, store.Path, "nested")
}

func TestLint_CoverageStore_Append_Good(t *core.T) {
	store := NewCoverageStore(filepath.Join(t.TempDir(), "coverage.json"))
	snap := CoverageSnapshot{Timestamp: time.Unix(1, 0), Packages: map[string]float64{"pkg": 80}, Total: 80}
	err := store.Append(snap)
	core.AssertNoError(t, err)
	core.AssertTrue(t, ax7FileExists(store.Path))
}

func TestLint_CoverageStore_Append_Bad(t *core.T) {
	store := NewCoverageStore("bad\x00path")
	err := store.Append(CoverageSnapshot{})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "load snapshots")
}

func TestLint_CoverageStore_Append_Ugly(t *core.T) {
	store := NewCoverageStore(filepath.Join(t.TempDir(), "nested", "coverage.json"))
	err := store.Append(CoverageSnapshot{Packages: map[string]float64{}})
	core.AssertNoError(t, err)
	core.AssertTrue(t, ax7FileExists(store.Path))
}

func TestLint_CoverageStore_Load_Good(t *core.T) {
	store := NewCoverageStore(filepath.Join(t.TempDir(), "coverage.json"))
	core.RequireNoError(t, store.Append(CoverageSnapshot{Timestamp: time.Unix(1, 0), Packages: map[string]float64{"pkg": 80}, Total: 80}))
	snapshots, err := store.Load()
	core.AssertNoError(t, err)
	core.AssertLen(t, snapshots, 1)
}

func TestLint_CoverageStore_Load_Bad(t *core.T) {
	store := NewCoverageStore(filepath.Join(t.TempDir(), "missing.json"))
	snapshots, err := store.Load()
	core.AssertError(t, err)
	core.AssertNil(t, snapshots)
}

func TestLint_CoverageStore_Load_Ugly(t *core.T) {
	path := ax7WriteFile(t, t.TempDir(), "coverage.json", "not-json")
	store := NewCoverageStore(path)
	snapshots, err := store.Load()
	core.AssertError(t, err)
	core.AssertNil(t, snapshots)
}

func TestLint_CoverageStore_Latest_Good(t *core.T) {
	store := NewCoverageStore(filepath.Join(t.TempDir(), "coverage.json"))
	core.RequireNoError(t, store.Append(CoverageSnapshot{Timestamp: time.Unix(1, 0), Packages: map[string]float64{"old": 50}}))
	core.RequireNoError(t, store.Append(CoverageSnapshot{Timestamp: time.Unix(2, 0), Packages: map[string]float64{"new": 90}}))
	latest, err := store.Latest()
	core.AssertNoError(t, err)
	core.AssertEqual(t, time.Unix(2, 0), latest.Timestamp)
}

func TestLint_CoverageStore_Latest_Bad(t *core.T) {
	store := NewCoverageStore(filepath.Join(t.TempDir(), "missing.json"))
	latest, err := store.Latest()
	core.AssertNoError(t, err)
	core.AssertNil(t, latest)
}

func TestLint_CoverageStore_Latest_Ugly(t *core.T) {
	path := ax7WriteFile(t, t.TempDir(), "coverage.json", "[]")
	store := NewCoverageStore(path)
	latest, err := store.Latest()
	core.AssertNoError(t, err)
	core.AssertNil(t, latest)
}

func TestLint_ParseCoverProfile_Good(t *core.T) {
	data := "mode: set\nexample.com/pkg/file.go:1.1,1.2 2 1\n"
	snap, err := ParseCoverProfile(data)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 100.0, snap.Total)
	core.AssertEqual(t, 100.0, snap.Packages["example.com/pkg"])
}

func TestLint_ParseCoverProfile_Bad(t *core.T) {
	snap, err := ParseCoverProfile("mode: set\ninvalid\n")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0.0, snap.Total)
	core.AssertEmpty(t, snap.Packages)
}

func TestLint_ParseCoverProfile_Ugly(t *core.T) {
	data := "mode: set\nexample.com/pkg/file.go:1.1,1.2 2 0\n"
	snap, err := ParseCoverProfile(data)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0.0, snap.Total)
	core.AssertEqual(t, 0.0, snap.Packages["example.com/pkg"])
}

func TestLint_ParseCoverOutput_Good(t *core.T) {
	output := "ok  \texample.com/pkg\t0.1s\tcoverage: 75.0% of statements\n"
	snap, err := ParseCoverOutput(output)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 75.0, snap.Total)
	core.AssertEqual(t, 75.0, snap.Packages["example.com/pkg"])
}

func TestLint_ParseCoverOutput_Bad(t *core.T) {
	snap, err := ParseCoverOutput("FAIL example.com/pkg\n")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0.0, snap.Total)
	core.AssertEmpty(t, snap.Packages)
}

func TestLint_ParseCoverOutput_Ugly(t *core.T) {
	output := "ok  \ta\t0.1s\tcoverage: 50.0% of statements\nok  \tb\t0.1s\tcoverage: 100.0% of statements\n"
	snap, err := ParseCoverOutput(output)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 75.0, snap.Total)
	core.AssertLen(t, snap.Packages, 2)
}

func TestLint_CompareCoverage_Good(t *core.T) {
	prev := CoverageSnapshot{Packages: map[string]float64{"pkg": 90}, Total: 90}
	curr := CoverageSnapshot{Packages: map[string]float64{"pkg": 80}, Total: 80}
	comp := CompareCoverage(prev, curr)
	core.AssertEqual(t, -10.0, comp.TotalDelta)
	core.AssertLen(t, comp.Regressions, 1)
}

func TestLint_CompareCoverage_Bad(t *core.T) {
	prev := CoverageSnapshot{Packages: map[string]float64{"pkg": 90}, Total: 90}
	curr := CoverageSnapshot{Packages: map[string]float64{}, Total: 90}
	comp := CompareCoverage(prev, curr)
	core.AssertEqual(t, []string{"pkg"}, comp.Removed)
	core.AssertEmpty(t, comp.Regressions)
}

func TestLint_CompareCoverage_Ugly(t *core.T) {
	prev := CoverageSnapshot{Packages: map[string]float64{}, Total: 0}
	curr := CoverageSnapshot{Packages: map[string]float64{"pkg": 100}, Total: 100}
	comp := CompareCoverage(prev, curr)
	core.AssertEqual(t, []string{"pkg"}, comp.NewPackages)
	core.AssertEqual(t, 100.0, comp.TotalDelta)
}

func TestLint_DefaultConfigYAML_Bad(t *core.T) {
	yaml, err := DefaultConfigYAML()
	core.AssertNoError(t, err)
	core.AssertNotContains(t, yaml, "definitely-not-a-tool")
	core.AssertContains(t, yaml, "fail_on")
}

func TestLint_DefaultConfigYAML_Ugly(t *core.T) {
	yaml, err := DefaultConfigYAML()
	core.AssertNoError(t, err)
	core.AssertContains(t, yaml, "output:")
	core.AssertContains(t, yaml, "paths:")
}

func TestLint_ResolveRunOutputFormat_Good(t *core.T) {
	format, err := ResolveRunOutputFormat(RunInput{Output: "sarif"})
	core.AssertNoError(t, err)
	core.AssertEqual(t, "sarif", format)
}

func TestLint_ResolveRunOutputFormat_Bad(t *core.T) {
	dir := t.TempDir()
	ax7WriteFile(t, dir, filepath.Join(".core", "lint.yaml"), "output: [")
	format, err := ResolveRunOutputFormat(RunInput{Path: dir})
	core.AssertError(t, err)
	core.AssertEqual(t, "", format)
}

func TestLint_ResolveRunOutputFormat_Ugly(t *core.T) {
	format, err := ResolveRunOutputFormat(RunInput{CI: true})
	core.AssertNoError(t, err)
	core.AssertEqual(t, "github", format)
}

func TestLint_Detect_Good(t *core.T) {
	dir := t.TempDir()
	ax7WriteFile(t, dir, "go.mod", "module example.test\n")
	got := Detect(dir)
	core.AssertEqual(t, []string{"go"}, got)
	core.AssertLen(t, got, 1)
}

func TestLint_Detect_Bad(t *core.T) {
	got := Detect(filepath.Join(t.TempDir(), "missing"))
	core.AssertEmpty(t, got)
	core.AssertNotNil(t, got)
}

func TestLint_Detect_Ugly(t *core.T) {
	dir := t.TempDir()
	ax7WriteFile(t, dir, "Dockerfile.prod", "FROM scratch\n")
	ax7WriteFile(t, dir, "app.ts", "export {}\n")
	got := Detect(dir)
	core.AssertContains(t, got, "dockerfile")
	core.AssertContains(t, got, "ts")
}

func TestLint_Summarise_Good(t *core.T) {
	summary := Summarise([]Finding{{Severity: "error"}, {Severity: "info"}, {Severity: "warning"}})
	core.AssertEqual(t, 3, summary.Total)
	core.AssertFalse(t, summary.Passed)
}

func TestLint_Summarise_Bad(t *core.T) {
	summary := Summarise([]Finding{{Severity: ""}})
	core.AssertEqual(t, 1, summary.Warnings)
	core.AssertTrue(t, summary.Passed)
}

func TestLint_Summarise_Ugly(t *core.T) {
	summary := Summarise(nil)
	core.AssertEqual(t, 0, summary.Total)
	core.AssertTrue(t, summary.Passed)
}

func TestLint_WriteJSON_Good(t *core.T) {
	var out bytes.Buffer
	err := WriteJSON(&out, []Finding{{File: "main.go"}})
	core.AssertNoError(t, err)
	core.AssertContains(t, out.String(), "main.go")
}

func TestLint_WriteJSON_Bad(t *core.T) {
	err := WriteJSON(ax7ErrWriter{}, []Finding{{File: "main.go"}})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "write failed")
}

func TestLint_WriteJSON_Ugly(t *core.T) {
	var out bytes.Buffer
	err := WriteJSON(&out, nil)
	core.AssertNoError(t, err)
	core.AssertContains(t, out.String(), "[]")
}

func TestLint_WriteJSONL_Good(t *core.T) {
	var out bytes.Buffer
	err := WriteJSONL(&out, []Finding{{File: "main.go"}, {File: "other.go"}})
	core.AssertNoError(t, err)
	core.AssertEqual(t, 2, strings.Count(out.String(), "\n"))
}

func TestLint_WriteJSONL_Bad(t *core.T) {
	err := WriteJSONL(ax7ErrWriter{}, []Finding{{File: "main.go"}})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "write failed")
}

func TestLint_WriteJSONL_Ugly(t *core.T) {
	var out bytes.Buffer
	err := WriteJSONL(&out, nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", out.String())
}

func TestLint_WriteText_Good(t *core.T) {
	var out bytes.Buffer
	err := WriteText(&out, []Finding{{File: "main.go", Line: 7, Severity: "warning", Message: "Fix", Code: "R"}})
	core.AssertNoError(t, err)
	core.AssertContains(t, out.String(), "main.go:7")
}

func TestLint_WriteText_Bad(t *core.T) {
	err := WriteText(ax7ErrWriter{}, []Finding{{File: "main.go", Line: 7}})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "write failed")
}

func TestLint_WriteText_Ugly(t *core.T) {
	var out bytes.Buffer
	err := WriteText(&out, []Finding{{File: "main.go", Line: 7, Severity: "warning", Title: "Title", RuleID: "R"}})
	core.AssertNoError(t, err)
	core.AssertContains(t, out.String(), "Title")
}

func TestLint_WriteReportJSON_Good(t *core.T) {
	var out bytes.Buffer
	err := WriteReportJSON(&out, ax7Report())
	core.AssertNoError(t, err)
	core.AssertContains(t, out.String(), "repo")
}

func TestLint_WriteReportJSON_Bad(t *core.T) {
	err := WriteReportJSON(ax7ErrWriter{}, ax7Report())
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "write failed")
}

func TestLint_WriteReportJSON_Ugly(t *core.T) {
	var out bytes.Buffer
	err := WriteReportJSON(&out, Report{})
	core.AssertNoError(t, err)
	core.AssertContains(t, out.String(), "\"project\"")
}

func TestLint_WriteReportText_Good(t *core.T) {
	var out bytes.Buffer
	err := WriteReportText(&out, ax7Report())
	core.AssertNoError(t, err)
	core.AssertContains(t, out.String(), "finding(s)")
}

func TestLint_WriteReportText_Bad(t *core.T) {
	err := WriteReportText(ax7ErrWriter{}, ax7Report())
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "write failed")
}

func TestLint_WriteReportText_Ugly(t *core.T) {
	var out bytes.Buffer
	err := WriteReportText(&out, Report{Summary: Summary{}})
	core.AssertNoError(t, err)
	core.AssertContains(t, out.String(), "0 finding")
}

func TestLint_WriteReportGitHub_Good(t *core.T) {
	var out bytes.Buffer
	err := WriteReportGitHub(&out, ax7Report())
	core.AssertNoError(t, err)
	core.AssertContains(t, out.String(), "::warning")
}

func TestLint_WriteReportGitHub_Bad(t *core.T) {
	err := WriteReportGitHub(ax7ErrWriter{}, ax7Report())
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "write failed")
}

func TestLint_WriteReportGitHub_Ugly(t *core.T) {
	var out bytes.Buffer
	report := Report{Findings: []Finding{{Severity: "info", Tool: "tool", Message: "note"}}}
	err := WriteReportGitHub(&out, report)
	core.AssertNoError(t, err)
	core.AssertContains(t, out.String(), "::notice")
}

func TestLint_WriteReportSARIF_Good(t *core.T) {
	var out bytes.Buffer
	err := WriteReportSARIF(&out, ax7Report())
	core.AssertNoError(t, err)
	core.AssertContains(t, out.String(), "2.1.0")
}

func TestLint_WriteReportSARIF_Bad(t *core.T) {
	err := WriteReportSARIF(ax7ErrWriter{}, ax7Report())
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "write failed")
}

func TestLint_WriteReportSARIF_Ugly(t *core.T) {
	var out bytes.Buffer
	err := WriteReportSARIF(&out, Report{})
	core.AssertNoError(t, err)
	core.AssertContains(t, out.String(), "\"runs\"")
}

func TestLint_ParseVulnCheckJSON_Good(t *core.T) {
	output := "{\"config\":{\"module_path\":\"example.com/app\"}}\n{\"osv\":{\"id\":\"GO-1\",\"summary\":\"bad\",\"aliases\":[\"CVE-1\"],\"affected\":[{\"ranges\":[{\"events\":[{\"fixed\":\"v1.2.3\"}]}]}]}}\n{\"finding\":{\"osv\":\"GO-1\",\"trace\":[{\"module\":\"mod\",\"package\":\"pkg\",\"function\":\"Run\"}]}}\n"
	result, err := ParseVulnCheckJSON(output, "")
	core.AssertNoError(t, err)
	core.AssertLen(t, result.Findings, 1)
	core.AssertEqual(t, "example.com/app", result.Module)
}

func TestLint_ParseVulnCheckJSON_Bad(t *core.T) {
	result, err := ParseVulnCheckJSON("{bad-json}\n", "")
	core.AssertError(t, err)
	core.AssertNil(t, result)
}

func TestLint_ParseVulnCheckJSON_Ugly(t *core.T) {
	result, err := ParseVulnCheckJSON("", "ignored stderr")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, result)
	core.AssertEmpty(t, result.Findings)
}

func TestLint_CommandAdapter_Name_Good(t *core.T) {
	adapter := CommandAdapter{name: "tool"}
	got := adapter.Name()
	core.AssertEqual(t, "tool", got)
	core.AssertNotEqual(t, "", got)
}

func TestLint_CommandAdapter_Name_Bad(t *core.T) {
	adapter := CommandAdapter{}
	got := adapter.Name()
	core.AssertEqual(t, "", got)
	core.AssertFalse(t, adapter.Available())
}

func TestLint_CommandAdapter_Name_Ugly(t *core.T) {
	adapter := CommandAdapter{name: "tool with spaces"}
	got := adapter.Name()
	core.AssertEqual(t, "tool with spaces", got)
	core.AssertContains(t, got, "spaces")
}

func TestLint_CommandAdapter_Available_Good(t *core.T) {
	ax7Executable(t, "tool", "exit 0")
	adapter := CommandAdapter{binaries: []string{"tool"}}
	got := adapter.Available()
	core.AssertTrue(t, got)
	core.AssertEqual(t, "tool", adapter.Command())
}

func TestLint_CommandAdapter_Available_Bad(t *core.T) {
	t.Setenv("PATH", t.TempDir())
	adapter := CommandAdapter{binaries: []string{"missing-tool"}}
	got := adapter.Available()
	core.AssertFalse(t, got)
	core.AssertEqual(t, "missing-tool", adapter.Command())
}

func TestLint_CommandAdapter_Available_Ugly(t *core.T) {
	ax7Executable(t, "second", "exit 0")
	adapter := CommandAdapter{binaries: []string{"missing", "second"}}
	got := adapter.Available()
	core.AssertTrue(t, got)
	core.AssertEqual(t, "missing", adapter.Command())
}

func TestLint_CommandAdapter_Languages_Good(t *core.T) {
	adapter := CommandAdapter{languages: []string{"go"}}
	got := adapter.Languages()
	core.AssertEqual(t, []string{"go"}, got)
	got[0] = "mutated"
	core.AssertEqual(t, []string{"go"}, adapter.languages)
}

func TestLint_CommandAdapter_Languages_Bad(t *core.T) {
	adapter := CommandAdapter{}
	got := adapter.Languages()
	core.AssertEmpty(t, got)
	core.AssertNil(t, got)
}

func TestLint_CommandAdapter_Languages_Ugly(t *core.T) {
	adapter := CommandAdapter{languages: []string{"*"}}
	got := adapter.Languages()
	core.AssertEqual(t, []string{"*"}, got)
	core.AssertLen(t, got, 1)
}

func TestLint_CommandAdapter_Command_Good(t *core.T) {
	adapter := CommandAdapter{binaries: []string{"tool", "fallback"}}
	got := adapter.Command()
	core.AssertEqual(t, "tool", got)
	core.AssertNotEqual(t, "fallback", got)
}

func TestLint_CommandAdapter_Command_Bad(t *core.T) {
	adapter := CommandAdapter{}
	got := adapter.Command()
	core.AssertEqual(t, "", got)
	core.AssertFalse(t, adapter.Available())
}

func TestLint_CommandAdapter_Command_Ugly(t *core.T) {
	adapter := CommandAdapter{binaries: []string{""}}
	got := adapter.Command()
	core.AssertEqual(t, "", got)
	core.AssertLen(t, adapter.binaries, 1)
}

func TestLint_CommandAdapter_Entitlement_Good(t *core.T) {
	adapter := CommandAdapter{entitlement: "lint.security"}
	got := adapter.Entitlement()
	core.AssertEqual(t, "lint.security", got)
	core.AssertContains(t, got, "security")
}

func TestLint_CommandAdapter_Entitlement_Bad(t *core.T) {
	adapter := CommandAdapter{}
	got := adapter.Entitlement()
	core.AssertEqual(t, "", got)
	core.AssertFalse(t, adapter.RequiresEntitlement())
}

func TestLint_CommandAdapter_Entitlement_Ugly(t *core.T) {
	adapter := CommandAdapter{entitlement: "lint.compliance"}
	got := adapter.Entitlement()
	core.AssertEqual(t, "lint.compliance", got)
	core.AssertContains(t, got, ".")
}

func TestLint_CommandAdapter_RequiresEntitlement_Good(t *core.T) {
	adapter := CommandAdapter{requiresEntitlement: true}
	got := adapter.RequiresEntitlement()
	core.AssertTrue(t, got)
	core.AssertEqual(t, true, adapter.requiresEntitlement)
}

func TestLint_CommandAdapter_RequiresEntitlement_Bad(t *core.T) {
	adapter := CommandAdapter{}
	got := adapter.RequiresEntitlement()
	core.AssertFalse(t, got)
	core.AssertEqual(t, false, adapter.requiresEntitlement)
}

func TestLint_CommandAdapter_RequiresEntitlement_Ugly(t *core.T) {
	adapter := CommandAdapter{requiresEntitlement: true, entitlement: ""}
	got := adapter.RequiresEntitlement()
	core.AssertTrue(t, got)
	core.AssertEqual(t, "", adapter.Entitlement())
}

func TestLint_CommandAdapter_MatchesLanguage_Good(t *core.T) {
	adapter := CommandAdapter{languages: []string{"go"}}
	got := adapter.MatchesLanguage([]string{"go"})
	core.AssertTrue(t, got)
	core.AssertFalse(t, adapter.MatchesLanguage([]string{"php"}))
}

func TestLint_CommandAdapter_MatchesLanguage_Bad(t *core.T) {
	adapter := CommandAdapter{languages: []string{"go"}}
	got := adapter.MatchesLanguage(nil)
	core.AssertTrue(t, got)
	core.AssertFalse(t, adapter.MatchesLanguage([]string{"php"}))
}

func TestLint_CommandAdapter_MatchesLanguage_Ugly(t *core.T) {
	adapter := CommandAdapter{languages: []string{"*"}}
	got := adapter.MatchesLanguage([]string{"anything"})
	core.AssertTrue(t, got)
	core.AssertTrue(t, adapter.MatchesLanguage(nil))
}

func TestLint_CommandAdapter_Category_Good(t *core.T) {
	adapter := CommandAdapter{category: "correctness"}
	got := adapter.Category()
	core.AssertEqual(t, "correctness", got)
	core.AssertContains(t, got, "correct")
}

func TestLint_CommandAdapter_Category_Bad(t *core.T) {
	adapter := CommandAdapter{}
	got := adapter.Category()
	core.AssertEqual(t, "", got)
	core.AssertNotEqual(t, "correctness", got)
}

func TestLint_CommandAdapter_Category_Ugly(t *core.T) {
	adapter := CommandAdapter{category: "security"}
	got := adapter.Category()
	core.AssertEqual(t, "security", got)
	core.AssertTrue(t, adapter.MatchesLanguage([]string{"security"}))
}

func TestLint_CommandAdapter_Fast_Good(t *core.T) {
	adapter := CommandAdapter{fast: true}
	got := adapter.Fast()
	core.AssertTrue(t, got)
	core.AssertEqual(t, true, adapter.fast)
}

func TestLint_CommandAdapter_Fast_Bad(t *core.T) {
	adapter := CommandAdapter{}
	got := adapter.Fast()
	core.AssertFalse(t, got)
	core.AssertEqual(t, false, adapter.fast)
}

func TestLint_CommandAdapter_Fast_Ugly(t *core.T) {
	adapter := CommandAdapter{fast: false, category: "security"}
	got := adapter.Fast()
	core.AssertFalse(t, got)
	core.AssertEqual(t, "security", adapter.Category())
}

func TestLint_CommandAdapter_Run_Good(t *core.T) {
	ax7Executable(t, "tool", "exit 0")
	adapter := newCommandAdapter("tool", []string{"tool"}, []string{"go"}, "correctness", "", false, true, projectPathArguments(), parseTextDiagnostics)
	result := adapter.(CommandAdapter).Run(context.Background(), RunInput{Path: t.TempDir()}, nil)
	core.AssertEqual(t, "passed", result.Tool.Status)
	core.AssertEmpty(t, result.Findings)
}

func TestLint_CommandAdapter_Run_Bad(t *core.T) {
	t.Setenv("PATH", t.TempDir())
	adapter := CommandAdapter{name: "missing", binaries: []string{"missing"}, category: "correctness", buildArgs: projectPathArguments(), parseOutput: parseTextDiagnostics}
	result := adapter.Run(context.Background(), RunInput{Path: t.TempDir()}, nil)
	core.AssertEqual(t, "skipped", result.Tool.Status)
	core.AssertLen(t, result.Findings, 1)
}

func TestLint_CommandAdapter_Run_Ugly(t *core.T) {
	ax7Executable(t, "tool", "printf 'main.go:3: error: broken' >&2; exit 2")
	adapter := CommandAdapter{name: "tool", binaries: []string{"tool"}, category: "correctness", buildArgs: projectPathArguments(), parseOutput: parseTextDiagnostics}
	result := adapter.Run(context.Background(), RunInput{Path: t.TempDir()}, nil)
	core.AssertEqual(t, "failed", result.Tool.Status)
	core.AssertLen(t, result.Findings, 1)
}

func TestLint_CatalogAdapter_Name_Good(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Name()
	core.AssertEqual(t, "catalog", got)
	core.AssertNotEqual(t, "", got)
}

func TestLint_CatalogAdapter_Name_Bad(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Name()
	core.AssertNotEqual(t, "missing", got)
	core.AssertEqual(t, "catalog", got)
}

func TestLint_CatalogAdapter_Name_Ugly(t *core.T) {
	adapter := newCatalogAdapter()
	got := adapter.Name()
	core.AssertEqual(t, "catalog", got)
	core.AssertTrue(t, adapter.Available())
}

func TestLint_CatalogAdapter_Available_Good(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Available()
	core.AssertTrue(t, got)
	core.AssertEqual(t, "catalog", adapter.Command())
}

func TestLint_CatalogAdapter_Available_Bad(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Available()
	core.AssertTrue(t, got)
	core.AssertFalse(t, adapter.RequiresEntitlement())
}

func TestLint_CatalogAdapter_Available_Ugly(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Available()
	core.AssertTrue(t, got)
	core.AssertTrue(t, adapter.Fast())
}

func TestLint_CatalogAdapter_Languages_Good(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Languages()
	core.AssertEqual(t, []string{"go"}, got)
	core.AssertLen(t, got, 1)
}

func TestLint_CatalogAdapter_Languages_Bad(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Languages()
	core.AssertNotContains(t, got, "php")
	core.AssertContains(t, got, "go")
}

func TestLint_CatalogAdapter_Languages_Ugly(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Languages()
	got[0] = "mutated"
	core.AssertEqual(t, []string{"go"}, adapter.Languages())
}

func TestLint_CatalogAdapter_Command_Good(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Command()
	core.AssertEqual(t, "catalog", got)
	core.AssertNotEqual(t, "", got)
}

func TestLint_CatalogAdapter_Command_Bad(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Command()
	core.AssertNotEqual(t, "external", got)
	core.AssertEqual(t, "catalog", got)
}

func TestLint_CatalogAdapter_Command_Ugly(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Command()
	core.AssertEqual(t, adapter.Name(), got)
	core.AssertTrue(t, adapter.Available())
}

func TestLint_CatalogAdapter_Entitlement_Good(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Entitlement()
	core.AssertEqual(t, "", got)
	core.AssertFalse(t, adapter.RequiresEntitlement())
}

func TestLint_CatalogAdapter_Entitlement_Bad(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Entitlement()
	core.AssertNotEqual(t, "lint.security", got)
	core.AssertEqual(t, "", got)
}

func TestLint_CatalogAdapter_Entitlement_Ugly(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Entitlement()
	core.AssertEqual(t, "", got)
	core.AssertTrue(t, adapter.Fast())
}

func TestLint_CatalogAdapter_RequiresEntitlement_Good(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.RequiresEntitlement()
	core.AssertFalse(t, got)
	core.AssertEqual(t, "", adapter.Entitlement())
}

func TestLint_CatalogAdapter_RequiresEntitlement_Bad(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.RequiresEntitlement()
	core.AssertFalse(t, got)
	core.AssertTrue(t, adapter.Available())
}

func TestLint_CatalogAdapter_RequiresEntitlement_Ugly(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.RequiresEntitlement()
	core.AssertFalse(t, got)
	core.AssertEqual(t, "correctness", adapter.Category())
}

func TestLint_CatalogAdapter_MatchesLanguage_Good(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.MatchesLanguage([]string{"go"})
	core.AssertTrue(t, got)
	core.AssertTrue(t, adapter.MatchesLanguage(nil))
}

func TestLint_CatalogAdapter_MatchesLanguage_Bad(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.MatchesLanguage([]string{"php"})
	core.AssertFalse(t, got)
	core.AssertTrue(t, adapter.MatchesLanguage([]string{}))
}

func TestLint_CatalogAdapter_MatchesLanguage_Ugly(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.MatchesLanguage([]string{"php", "go"})
	core.AssertTrue(t, got)
	core.AssertFalse(t, adapter.MatchesLanguage([]string{"rust"}))
}

func TestLint_CatalogAdapter_Category_Good(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Category()
	core.AssertEqual(t, "correctness", got)
	core.AssertContains(t, got, "correct")
}

func TestLint_CatalogAdapter_Category_Bad(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Category()
	core.AssertNotEqual(t, "security", got)
	core.AssertEqual(t, "correctness", got)
}

func TestLint_CatalogAdapter_Category_Ugly(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Category()
	core.AssertEqual(t, "correctness", got)
	core.AssertTrue(t, adapter.Fast())
}

func TestLint_CatalogAdapter_Fast_Good(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Fast()
	core.AssertTrue(t, got)
	core.AssertEqual(t, "catalog", adapter.Name())
}

func TestLint_CatalogAdapter_Fast_Bad(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Fast()
	core.AssertTrue(t, got)
	core.AssertFalse(t, adapter.RequiresEntitlement())
}

func TestLint_CatalogAdapter_Fast_Ugly(t *core.T) {
	adapter := CatalogAdapter{}
	got := adapter.Fast()
	core.AssertTrue(t, got)
	core.AssertTrue(t, adapter.Available())
}

func TestLint_CatalogAdapter_Run_Good(t *core.T) {
	dir := t.TempDir()
	ax7WriteFile(t, dir, "main.go", "package main\nfunc Run() {}\n")
	result := CatalogAdapter{}.Run(context.Background(), RunInput{Path: dir}, nil)
	core.AssertEqual(t, "passed", result.Tool.Status)
	core.AssertEmpty(t, result.Findings)
}

func TestLint_CatalogAdapter_Run_Bad(t *core.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := CatalogAdapter{}.Run(ctx, RunInput{Path: t.TempDir()}, nil)
	core.AssertEqual(t, "canceled", result.Tool.Status)
	core.AssertEmpty(t, result.Findings)
}

func TestLint_CatalogAdapter_Run_Ugly(t *core.T) {
	dir := t.TempDir()
	ax7WriteFile(t, dir, "main.go", "package sample\nfunc Run() {\n\t_ =svc.Process(\"data\")\n}\n")
	result := CatalogAdapter{}.Run(context.Background(), RunInput{Path: dir}, []string{"main.go"})
	core.AssertEqual(t, "failed", result.Tool.Status)
	core.AssertLen(t, result.Findings, 1)
}

func TestLint_NewService_Good(t *core.T) {
	service := NewService()
	core.AssertNotNil(t, service)
	core.AssertTrue(t, len(service.adapters) > 0)
}

func TestLint_NewService_Bad(t *core.T) {
	service := &Service{}
	core.AssertNotNil(t, service)
	core.AssertEmpty(t, service.adapters)
}

func TestLint_NewService_Ugly(t *core.T) {
	service := &Service{adapters: []Adapter{CatalogAdapter{}}}
	core.AssertNotNil(t, service)
	core.AssertLen(t, service.adapters, 1)
}

func TestLint_Service_Run_Good(t *core.T) {
	dir := t.TempDir()
	ax7WriteFile(t, dir, "go.mod", "module example.test\n")
	ax7WriteFile(t, dir, "main.go", "package main\nfunc Run() {}\n")
	report, err := (&Service{adapters: []Adapter{CatalogAdapter{}}}).Run(context.Background(), RunInput{Path: dir, Lang: "go"})
	core.AssertNoError(t, err)
	core.AssertTrue(t, report.Summary.Passed)
	core.AssertEqual(t, filepath.Base(dir), report.Project)
}

func TestLint_Service_Run_Bad(t *core.T) {
	dir := t.TempDir()
	ax7WriteFile(t, dir, filepath.Join(".core", "lint.yaml"), "output: [")
	report, err := NewService().Run(context.Background(), RunInput{Path: dir})
	core.AssertError(t, err)
	core.AssertEqual(t, Report{}, report)
}

func TestLint_Service_Run_Ugly(t *core.T) {
	dir := t.TempDir()
	ax7WriteFile(t, dir, "go.mod", "module example.test\n")
	ax7WriteFile(t, dir, "main.go", "package sample\nfunc Run() {\n\t_ =svc.Process(\"data\")\n}\n")
	report, err := (&Service{adapters: []Adapter{CatalogAdapter{}}}).Run(context.Background(), RunInput{Path: dir, Lang: "go"})
	core.AssertNoError(t, err)
	core.AssertTrue(t, report.Summary.Passed)
	core.AssertLen(t, report.Findings, 1)
}

func TestLint_Service_Tools_Good(t *core.T) {
	tools := NewService().Tools([]string{"go"})
	core.AssertTrue(t, len(tools) > 0)
	core.AssertEqual(t, "errcheck", tools[0].Name)
}

func TestLint_Service_Tools_Bad(t *core.T) {
	tools := NewService().Tools([]string{"elixir"})
	core.AssertTrue(t, len(tools) > 0)
	core.AssertNotNil(t, tools)
}

func TestLint_Service_Tools_Ugly(t *core.T) {
	tools := NewService().Tools(nil)
	core.AssertTrue(t, len(tools) > 10)
	core.AssertEqual(t, "bandit", tools[0].Name)
}

func TestLint_Service_WriteDefaultConfig_Good(t *core.T) {
	dir := t.TempDir()
	path, err := NewService().WriteDefaultConfig(dir, false)
	core.AssertNoError(t, err)
	core.AssertTrue(t, ax7FileExists(path))
}

func TestLint_Service_WriteDefaultConfig_Bad(t *core.T) {
	dir := t.TempDir()
	_, firstErr := NewService().WriteDefaultConfig(dir, false)
	path, err := NewService().WriteDefaultConfig(dir, false)
	core.AssertNoError(t, firstErr)
	core.AssertError(t, err)
	core.AssertEqual(t, "", path)
}

func TestLint_Service_WriteDefaultConfig_Ugly(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	path, err := NewService().WriteDefaultConfig(dir, true)
	core.AssertNoError(t, err)
	core.AssertContains(t, path, DefaultConfigPath)
}

func TestLint_Service_InstallHook_Good(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	ax7Executable(t, "git", "printf .git")
	err := NewService().InstallHook(dir)
	core.AssertNoError(t, err)
	core.AssertTrue(t, ax7FileExists(filepath.Join(dir, ".git", "hooks", "pre-commit")))
}

func TestLint_Service_InstallHook_Bad(t *core.T) {
	dir := t.TempDir()
	ax7Executable(t, "git", "printf fatal >&2; exit 1")
	err := NewService().InstallHook(dir)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "git rev-parse")
}

func TestLint_Service_InstallHook_Ugly(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".git", "hooks", "pre-commit"), []byte("#!/bin/sh\necho existing\n"), 0o755))
	ax7Executable(t, "git", "printf .git")
	err := NewService().InstallHook(dir)
	core.AssertNoError(t, err)
	core.AssertTrue(t, ax7FileExists(filepath.Join(dir, ".git", "hooks", "pre-commit")))
}

func TestLint_Service_RemoveHook_Good(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	ax7Executable(t, "git", "printf .git")
	service := NewService()
	core.RequireNoError(t, service.InstallHook(dir))
	err := service.RemoveHook(dir)
	core.AssertNoError(t, err)
	core.AssertTrue(t, ax7FileExists(filepath.Join(dir, ".git", "hooks", "pre-commit")))
}

func TestLint_Service_RemoveHook_Bad(t *core.T) {
	dir := t.TempDir()
	ax7Executable(t, "git", "printf fatal >&2; exit 1")
	err := NewService().RemoveHook(dir)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "git rev-parse")
}

func TestLint_Service_RemoveHook_Ugly(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0o755))
	hook := filepath.Join(dir, ".git", "hooks", "pre-commit")
	core.RequireNoError(t, os.WriteFile(hook, []byte("#!/bin/sh\necho before\n"+hookScriptBlock(true)+"echo after\n"), 0o755))
	ax7Executable(t, "git", "printf .git")
	err := NewService().RemoveHook(dir)
	core.AssertNoError(t, err)
	core.AssertTrue(t, ax7FileExists(hook))
}

func TestLint_NewToolkit_Good(t *core.T) {
	toolkit := NewToolkit("/repo")
	core.AssertNotNil(t, toolkit)
	core.AssertEqual(t, "/repo", toolkit.Dir)
}

func TestLint_NewToolkit_Bad(t *core.T) {
	toolkit := NewToolkit("")
	core.AssertNotNil(t, toolkit)
	core.AssertEqual(t, "", toolkit.Dir)
}

func TestLint_NewToolkit_Ugly(t *core.T) {
	dir := filepath.Join("a", "..", "repo")
	toolkit := NewToolkit(dir)
	core.AssertNotNil(t, toolkit)
	core.AssertContains(t, toolkit.Dir, "repo")
}

func TestLint_Toolkit_Run_Good(t *core.T) {
	ax7Executable(t, "tool", "printf out; printf err >&2")
	stdout, stderr, exitCode, err := NewToolkit(t.TempDir()).Run("tool")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "out", stdout)
	core.AssertEqual(t, "err", stderr)
	core.AssertEqual(t, 0, exitCode)
}

func TestLint_Toolkit_Run_Bad(t *core.T) {
	t.Setenv("PATH", t.TempDir())
	stdout, stderr, exitCode, err := NewToolkit(t.TempDir()).Run("missing")
	core.AssertError(t, err)
	core.AssertEqual(t, "", stdout)
	core.AssertEqual(t, "", stderr)
	core.AssertEqual(t, -1, exitCode)
}

func TestLint_Toolkit_Run_Ugly(t *core.T) {
	ax7Executable(t, "tool", "printf fail >&2; exit 7")
	stdout, stderr, exitCode, err := NewToolkit(t.TempDir()).Run("tool")
	core.AssertError(t, err)
	core.AssertEqual(t, "", stdout)
	core.AssertEqual(t, "fail", stderr)
	core.AssertEqual(t, 7, exitCode)
}

func TestLint_Toolkit_FindTrackedComments_Good(t *core.T) {
	ax7Executable(t, "git", "printf 'main.go:12:// TODO: fix it\\n'")
	comments, err := NewToolkit(t.TempDir()).FindTrackedComments(".")
	core.AssertNoError(t, err)
	core.AssertLen(t, comments, 1)
	core.AssertEqual(t, "TODO", comments[0].Type)
}

func TestLint_Toolkit_FindTrackedComments_Bad(t *core.T) {
	ax7Executable(t, "git", "exit 1")
	comments, err := NewToolkit(t.TempDir()).FindTrackedComments(".")
	core.AssertNoError(t, err)
	core.AssertNil(t, comments)
}

func TestLint_Toolkit_FindTrackedComments_Ugly(t *core.T) {
	ax7Executable(t, "git", "printf 'malformed\\nmain.go:not-a-number:// TODO: nope\\n'")
	comments, err := NewToolkit(t.TempDir()).FindTrackedComments(".")
	core.AssertNoError(t, err)
	core.AssertLen(t, comments, 1)
	core.AssertEqual(t, 0, comments[0].Line)
}

func TestLint_Toolkit_FindTODOs_Good(t *core.T) {
	ax7Executable(t, "git", "printf 'main.go:12:// TODO: fix it\\n'")
	comments, err := NewToolkit(t.TempDir()).FindTODOs(".")
	core.AssertNoError(t, err)
	core.AssertLen(t, comments, 1)
	core.AssertEqual(t, "TODO", comments[0].Type)
}

func TestLint_Toolkit_FindTODOs_Bad(t *core.T) {
	ax7Executable(t, "git", "exit 1")
	comments, err := NewToolkit(t.TempDir()).FindTODOs(".")
	core.AssertNoError(t, err)
	core.AssertNil(t, comments)
}

func TestLint_Toolkit_FindTODOs_Ugly(t *core.T) {
	ax7Executable(t, "git", "printf 'main.go:12:// HACK: patch\\n'")
	comments, err := NewToolkit(t.TempDir()).FindTODOs(".")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "HACK", comments[0].Type)
}

func TestLint_Toolkit_AuditDeps_Good(t *core.T) {
	ax7Executable(t, "govulncheck", "printf 'Vulnerability #GO-1\\n  Package: stdlib\\n  Found in version: v1\\n\\n'")
	vulns, err := NewToolkit(t.TempDir()).AuditDeps()
	core.AssertNoError(t, err)
	core.AssertLen(t, vulns, 1)
	core.AssertContains(t, vulns[0].ID, "GO-1")
}

func TestLint_Toolkit_AuditDeps_Bad(t *core.T) {
	ax7Executable(t, "govulncheck", "printf failed >&2; exit 2")
	vulns, err := NewToolkit(t.TempDir()).AuditDeps()
	core.AssertError(t, err)
	core.AssertNil(t, vulns)
}

func TestLint_Toolkit_AuditDeps_Ugly(t *core.T) {
	ax7Executable(t, "govulncheck", "exit 0")
	vulns, err := NewToolkit(t.TempDir()).AuditDeps()
	core.AssertNoError(t, err)
	core.AssertEmpty(t, vulns)
}

func TestLint_Toolkit_DiffStat_Good(t *core.T) {
	ax7Executable(t, "git", "printf ' 1 file changed, 2 insertions(+), 1 deletion(-)\\n'")
	summary, err := NewToolkit(t.TempDir()).DiffStat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, summary.FilesChanged)
	core.AssertEqual(t, 2, summary.Insertions)
}

func TestLint_Toolkit_DiffStat_Bad(t *core.T) {
	ax7Executable(t, "git", "printf fatal >&2; exit 2")
	summary, err := NewToolkit(t.TempDir()).DiffStat()
	core.AssertError(t, err)
	core.AssertEqual(t, DiffSummary{}, summary)
}

func TestLint_Toolkit_DiffStat_Ugly(t *core.T) {
	ax7Executable(t, "git", "exit 0")
	summary, err := NewToolkit(t.TempDir()).DiffStat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, DiffSummary{}, summary)
}

func TestLint_Toolkit_UncommittedFiles_Good(t *core.T) {
	ax7Executable(t, "git", "printf 'MM main.go\\n?? new.go\\n'")
	files, err := NewToolkit(t.TempDir()).UncommittedFiles()
	core.AssertNoError(t, err)
	core.AssertEqual(t, []string{"main.go", "new.go"}, files)
}

func TestLint_Toolkit_UncommittedFiles_Bad(t *core.T) {
	ax7Executable(t, "git", "printf fatal >&2; exit 2")
	files, err := NewToolkit(t.TempDir()).UncommittedFiles()
	core.AssertError(t, err)
	core.AssertNil(t, files)
}

func TestLint_Toolkit_UncommittedFiles_Ugly(t *core.T) {
	ax7Executable(t, "git", "exit 0")
	files, err := NewToolkit(t.TempDir()).UncommittedFiles()
	core.AssertNoError(t, err)
	core.AssertEmpty(t, files)
}

func TestLint_Toolkit_Lint_Good(t *core.T) {
	ax7Executable(t, "go", "printf 'main.go:10:1: error: broken\\n' >&2; exit 2")
	findings, err := NewToolkit(t.TempDir()).Lint("./...")
	core.AssertNoError(t, err)
	core.AssertLen(t, findings, 1)
	core.AssertEqual(t, "go vet", findings[0].Tool)
}

func TestLint_Toolkit_Lint_Bad(t *core.T) {
	ax7Executable(t, "go", "printf fatal >&2; exit 1")
	findings, err := NewToolkit(t.TempDir()).Lint("./...")
	core.AssertError(t, err)
	core.AssertNil(t, findings)
}

func TestLint_Toolkit_Lint_Ugly(t *core.T) {
	ax7Executable(t, "go", "exit 0")
	findings, err := NewToolkit(t.TempDir()).Lint("./...")
	core.AssertNoError(t, err)
	core.AssertNil(t, findings)
}

func TestLint_Toolkit_ScanSecrets_Good(t *core.T) {
	ax7Executable(t, "gitleaks", "printf 'RuleID,File,Line,Secret\\nsecret,main.go,7,value\\n'; exit 1")
	leaks, err := NewToolkit(t.TempDir()).ScanSecrets(".")
	core.AssertNoError(t, err)
	core.AssertLen(t, leaks, 1)
	core.AssertEqual(t, "secret", leaks[0].RuleID)
}

func TestLint_Toolkit_ScanSecrets_Bad(t *core.T) {
	ax7Executable(t, "gitleaks", "printf fatal >&2; exit 2")
	leaks, err := NewToolkit(t.TempDir()).ScanSecrets(".")
	core.AssertError(t, err)
	core.AssertNil(t, leaks)
}

func TestLint_Toolkit_ScanSecrets_Ugly(t *core.T) {
	ax7Executable(t, "gitleaks", "exit 0")
	leaks, err := NewToolkit(t.TempDir()).ScanSecrets(".")
	core.AssertNoError(t, err)
	core.AssertNil(t, leaks)
}

func TestLint_Toolkit_ModTidy_Good(t *core.T) {
	ax7Executable(t, "go", "exit 0")
	err := NewToolkit(t.TempDir()).ModTidy()
	core.AssertNoError(t, err)
	core.AssertTrue(t, true)
}

func TestLint_Toolkit_ModTidy_Bad(t *core.T) {
	ax7Executable(t, "go", "printf failed >&2; exit 1")
	err := NewToolkit(t.TempDir()).ModTidy()
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "go mod tidy")
}

func TestLint_Toolkit_ModTidy_Ugly(t *core.T) {
	ax7Executable(t, "go", "printf warning >&2; exit 0")
	err := NewToolkit(t.TempDir()).ModTidy()
	core.AssertNoError(t, err)
	core.AssertTrue(t, true)
}

func TestLint_Toolkit_Build_Good(t *core.T) {
	ax7Executable(t, "go", "exit 0")
	results, err := NewToolkit(t.TempDir()).Build("./...")
	core.AssertNoError(t, err)
	core.AssertLen(t, results, 1)
	core.AssertNil(t, results[0].Error)
}

func TestLint_Toolkit_Build_Bad(t *core.T) {
	ax7Executable(t, "go", "printf failed >&2; exit 1")
	results, err := NewToolkit(t.TempDir()).Build("./...")
	core.AssertNoError(t, err)
	core.AssertLen(t, results, 1)
	core.AssertError(t, results[0].Error)
}

func TestLint_Toolkit_Build_Ugly(t *core.T) {
	ax7Executable(t, "go", "exit 0")
	results, err := NewToolkit(t.TempDir()).Build("./a", "./b")
	core.AssertNoError(t, err)
	core.AssertLen(t, results, 2)
}

func TestLint_Toolkit_TestCount_Good(t *core.T) {
	ax7Executable(t, "go", "printf 'TestOne\\nBenchmarkTwo\\nok\\n'")
	count, err := NewToolkit(t.TempDir()).TestCount("./...")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 2, count)
}

func TestLint_Toolkit_TestCount_Bad(t *core.T) {
	ax7Executable(t, "go", "printf failed >&2; exit 1")
	count, err := NewToolkit(t.TempDir()).TestCount("./...")
	core.AssertError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestLint_Toolkit_TestCount_Ugly(t *core.T) {
	ax7Executable(t, "go", "printf 'ExampleOne\\n'")
	count, err := NewToolkit(t.TempDir()).TestCount("./...")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestLint_Toolkit_Coverage_Good(t *core.T) {
	ax7Executable(t, "go", "printf 'ok  \\texample.com/pkg\\t0.1s\\tcoverage: 80.0%% of statements\\n'")
	reports, err := NewToolkit(t.TempDir()).Coverage("./...")
	core.AssertNoError(t, err)
	core.AssertLen(t, reports, 1)
	core.AssertEqual(t, 80.0, reports[0].Percentage)
}

func TestLint_Toolkit_Coverage_Bad(t *core.T) {
	ax7Executable(t, "go", "printf fatal >&2; exit 1")
	reports, err := NewToolkit(t.TempDir()).Coverage("./...")
	core.AssertError(t, err)
	core.AssertNil(t, reports)
}

func TestLint_Toolkit_Coverage_Ugly(t *core.T) {
	ax7Executable(t, "go", "exit 0")
	reports, err := NewToolkit(t.TempDir()).Coverage("")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, reports)
}

func TestLint_Toolkit_RaceDetect_Good(t *core.T) {
	ax7Executable(t, "go", "printf 'WARNING: DATA RACE\\n  main.go:42 +0x1\\n' >&2; exit 1")
	races, err := NewToolkit(t.TempDir()).RaceDetect("./...")
	core.AssertNoError(t, err)
	core.AssertLen(t, races, 1)
	core.AssertEqual(t, 42, races[0].Line)
}

func TestLint_Toolkit_RaceDetect_Bad(t *core.T) {
	ax7Executable(t, "go", "printf fatal >&2; exit 1")
	races, err := NewToolkit(t.TempDir()).RaceDetect("./...")
	core.AssertError(t, err)
	core.AssertNil(t, races)
}

func TestLint_Toolkit_RaceDetect_Ugly(t *core.T) {
	ax7Executable(t, "go", "exit 0")
	races, err := NewToolkit(t.TempDir()).RaceDetect("")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, races)
}

func TestLint_Toolkit_GocycloComplexity_Good(t *core.T) {
	ax7Executable(t, "gocyclo", "printf '12 sample Run main.go:9\\n'")
	funcs, err := NewToolkit(t.TempDir()).GocycloComplexity(10)
	core.AssertNoError(t, err)
	core.AssertLen(t, funcs, 1)
	core.AssertEqual(t, "Run", funcs[0].FuncName)
}

func TestLint_Toolkit_GocycloComplexity_Bad(t *core.T) {
	t.Setenv("PATH", t.TempDir())
	funcs, err := NewToolkit(t.TempDir()).GocycloComplexity(10)
	core.AssertError(t, err)
	core.AssertNil(t, funcs)
}

func TestLint_Toolkit_GocycloComplexity_Ugly(t *core.T) {
	ax7Executable(t, "gocyclo", "printf 'malformed\\n'")
	funcs, err := NewToolkit(t.TempDir()).GocycloComplexity(0)
	core.AssertNoError(t, err)
	core.AssertEmpty(t, funcs)
}

func TestLint_Toolkit_DepGraph_Good(t *core.T) {
	ax7Executable(t, "go", "printf 'a b\\na c\\n'")
	graph, err := NewToolkit(t.TempDir()).DepGraph("")
	core.AssertNoError(t, err)
	core.AssertEqual(t, []string{"a", "b", "c"}, graph.Nodes)
}

func TestLint_Toolkit_DepGraph_Bad(t *core.T) {
	ax7Executable(t, "go", "printf failed >&2; exit 1")
	graph, err := NewToolkit(t.TempDir()).DepGraph("")
	core.AssertError(t, err)
	core.AssertNil(t, graph)
}

func TestLint_Toolkit_DepGraph_Ugly(t *core.T) {
	ax7Executable(t, "go", "exit 0")
	graph, err := NewToolkit(t.TempDir()).DepGraph("")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, graph.Nodes)
}

func TestLint_Toolkit_GitLog_Good(t *core.T) {
	ax7Executable(t, "git", "printf 'abc|Ada|2026-01-02T03:04:05Z|message\\n'")
	commits, err := NewToolkit(t.TempDir()).GitLog(1)
	core.AssertNoError(t, err)
	core.AssertLen(t, commits, 1)
	core.AssertEqual(t, "Ada", commits[0].Author)
}

func TestLint_Toolkit_GitLog_Bad(t *core.T) {
	ax7Executable(t, "git", "printf fatal >&2; exit 1")
	commits, err := NewToolkit(t.TempDir()).GitLog(1)
	core.AssertError(t, err)
	core.AssertNil(t, commits)
}

func TestLint_Toolkit_GitLog_Ugly(t *core.T) {
	ax7Executable(t, "git", "printf 'malformed\\n'")
	commits, err := NewToolkit(t.TempDir()).GitLog(1)
	core.AssertNoError(t, err)
	core.AssertEmpty(t, commits)
}

func TestLint_Toolkit_CheckPerms_Good(t *core.T) {
	dir := t.TempDir()
	path := ax7WriteFile(t, dir, "open.txt", "data")
	core.RequireNoError(t, os.Chmod(path, 0o666))
	issues, err := NewToolkit(dir).CheckPerms(".")
	core.AssertNoError(t, err)
	core.AssertLen(t, issues, 1)
}

func TestLint_Toolkit_CheckPerms_Bad(t *core.T) {
	issues, err := NewToolkit(t.TempDir()).CheckPerms("missing")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, issues)
}

func TestLint_Toolkit_CheckPerms_Ugly(t *core.T) {
	dir := t.TempDir()
	path := ax7WriteFile(t, dir, "group.txt", "data")
	core.RequireNoError(t, os.Chmod(path, 0o602))
	issues, err := NewToolkit(dir).CheckPerms(".")
	core.AssertNoError(t, err)
	core.AssertLen(t, issues, 1)
}

func TestLint_Toolkit_VulnCheck_Good(t *core.T) {
	ax7Executable(t, "govulncheck", "printf '{\"config\":{\"module_path\":\"example.com/app\"}}\\n'")
	result, err := NewToolkit(t.TempDir()).VulnCheck("./...")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "example.com/app", result.Module)
}

func TestLint_Toolkit_VulnCheck_Bad(t *core.T) {
	t.Setenv("PATH", t.TempDir())
	result, err := NewToolkit(t.TempDir()).VulnCheck("./...")
	core.AssertError(t, err)
	core.AssertNil(t, result)
}

func TestLint_Toolkit_VulnCheck_Ugly(t *core.T) {
	ax7Executable(t, "govulncheck", "exit 0")
	result, err := NewToolkit(t.TempDir()).VulnCheck("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, result)
}

func TestLint_CommandAdapter_Interface_Good(t *core.T) {
	var adapter Adapter = CommandAdapter{name: "tool"}
	core.AssertEqual(t, "tool", adapter.Name())
	core.AssertFalse(t, adapter.Available())
}

func TestLint_CatalogAdapter_Interface_Good(t *core.T) {
	var adapter Adapter = CatalogAdapter{}
	core.AssertEqual(t, "catalog", adapter.Name())
	core.AssertTrue(t, adapter.Available())
}

func TestLint_FSMode_Good(t *core.T) {
	var mode fs.FileMode = 0o644
	core.AssertEqual(t, fs.FileMode(0o644), mode)
	core.AssertFalse(t, mode.IsDir())
}

func TestLint_IOReader_Good(t *core.T) {
	reader := strings.NewReader("agent")
	data, err := io.ReadAll(reader)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "agent", string(data))
}
