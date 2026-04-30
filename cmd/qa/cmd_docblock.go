// cmd_docblock.go implements docblock/docstring coverage checking for Go code.
//
// Usage:
//
//	core qa docblock              # Check current directory
//	core qa docblock ./pkg/...    # Check specific packages
//	core qa docblock --threshold=80  # Require 80% coverage
package qa

import (
	"cmp"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/i18n"
	coreio "dappco.re/go/io"
)

// Docblock command flags
var (
	docblockThreshold float64
	docblockVerbose   bool
	docblockJSON      bool
)

// addDocblockCommand adds the 'docblock' command to qa.
func addDocblockCommand(parent *cli.Command) {
	docblockCmd := &cli.Command{
		Use:   "docblock [packages...]",
		Short: i18n.T("cmd.qa.docblock.short"),
		Long:  i18n.T("cmd.qa.docblock.long"),
		RunE: func(cmd *cli.Command, args []string) error {
			paths := args
			if len(paths) == 0 {
				paths = []string{"./..."}
			}
			return RunDocblockCheck(paths, docblockThreshold, docblockVerbose, docblockJSON)
		},
	}

	docblockCmd.Flags().Float64Var(&docblockThreshold, "threshold", 80, i18n.T("cmd.qa.docblock.flag.threshold"))
	docblockCmd.Flags().BoolVarP(&docblockVerbose, "verbose", "v", false, i18n.T("common.flag.verbose"))
	docblockCmd.Flags().BoolVar(&docblockJSON, "json", false, i18n.T("common.flag.json"))

	parent.AddCommand(docblockCmd)
}

// DocblockResult holds the result of a docblock coverage check.
type DocblockResult struct {
	Coverage   float64           `json:"coverage"`
	Threshold  float64           `json:"threshold"`
	Total      int               `json:"total"`
	Documented int               `json:"documented"`
	Missing    []MissingDocblock `json:"missing,omitempty"`
	Warnings   []DocblockWarning `json:"warnings,omitempty"`
	Passed     bool              `json:"passed"`
}

// MissingDocblock represents an exported symbol without documentation.
type MissingDocblock struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Name   string `json:"name"`
	Kind   string `json:"kind"` // func, type, const, var
	Reason string `json:"reason,omitempty"`
}

// DocblockWarning captures a partial parse failure while still preserving
// the successfully parsed files in the same directory.
type DocblockWarning struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// RunDocblockCheck checks docblock coverage for the given packages.
func RunDocblockCheck(paths []string, threshold float64, verbose, jsonOutput bool) error {
	result, err := CheckDocblockCoverage(paths)
	if err != nil {
		return err
	}
	result.Threshold = threshold
	result.Passed = result.Coverage >= threshold

	if jsonOutput {
		return printDocblockJSON(result, threshold)
	}

	printVerboseMissingDocblocks(result, verbose)
	printDocblockWarnings(result)
	return printDocblockSummary(result, threshold)
}

func printDocblockJSON(result *DocblockResult, threshold float64) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	cli.Print("%s\n", string(data))
	if !result.Passed {
		return cli.Err("docblock coverage %.1f%% below threshold %.1f%%", result.Coverage, threshold)
	}
	return nil
}

func printVerboseMissingDocblocks(result *DocblockResult, verbose bool) {
	if !verbose || len(result.Missing) == 0 {
		return
	}
	cli.Print("%s\n\n", i18n.T("cmd.qa.docblock.missing_docs"))
	for _, m := range result.Missing {
		cli.Print("  %s:%d: %s %s\n",
			dimStyle.Render(m.File),
			m.Line,
			dimStyle.Render(m.Kind),
			m.Name,
		)
	}
	cli.Blank()
}

func printDocblockWarnings(result *DocblockResult) {
	if len(result.Warnings) == 0 {
		return
	}
	for _, warning := range result.Warnings {
		cli.Warnf("failed to parse %s: %s", warning.Path, warning.Error)
	}
	cli.Blank()
}

func printDocblockSummary(result *DocblockResult, threshold float64) error {
	coverageStr := fmt.Sprintf("%.1f%%", result.Coverage)
	thresholdStr := fmt.Sprintf("%.1f%%", threshold)

	if result.Passed {
		cli.Print("%s %s %s/%s (%s >= %s)\n",
			successStyle.Render(i18n.T("common.label.success")),
			i18n.T("cmd.qa.docblock.coverage"),
			fmt.Sprintf("%d", result.Documented),
			fmt.Sprintf("%d", result.Total),
			successStyle.Render(coverageStr),
			thresholdStr,
		)
		return nil
	}

	cli.Print("%s %s %s/%s (%s < %s)\n",
		errorStyle.Render(i18n.T("common.label.error")),
		i18n.T("cmd.qa.docblock.coverage"),
		fmt.Sprintf("%d", result.Documented),
		fmt.Sprintf("%d", result.Total),
		errorStyle.Render(coverageStr),
		thresholdStr,
	)

	// Always show compact file:line list when failing (token-efficient for AI agents)
	if len(result.Missing) > 0 {
		cli.Blank()
		for _, m := range result.Missing {
			cli.Print("%s:%d\n", m.File, m.Line)
		}
	}

	return cli.Err("docblock coverage %.1f%% below threshold %.1f%%", result.Coverage, threshold)
}

// CheckDocblockCoverage analyzes Go packages for docblock coverage.
func CheckDocblockCoverage(patterns []string) (*DocblockResult, error) {
	result := &DocblockResult{}

	// Expand patterns to actual directories
	dirs, err := expandPatterns(patterns)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()

	for _, dir := range dirs {
		pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, parser.ParseComments)
		if err != nil {
			// Preserve partial results when a directory contains both valid and
			// invalid files. The caller decides how to present the warning.
			result.Warnings = append(result.Warnings, DocblockWarning{
				Path:  dir,
				Error: err.Error(),
			})
		}

		for _, pkg := range pkgs {
			for filename, file := range pkg.Files {
				checkFile(fset, filename, file, result)
			}
		}
	}

	if result.Total > 0 {
		result.Coverage = float64(result.Documented) / float64(result.Total) * 100
	}

	slices.SortFunc(result.Missing, func(a, b MissingDocblock) int {
		return cmp.Or(
			cmp.Compare(a.File, b.File),
			cmp.Compare(a.Line, b.Line),
			cmp.Compare(a.Kind, b.Kind),
			cmp.Compare(a.Name, b.Name),
		)
	})
	slices.SortFunc(result.Warnings, func(a, b DocblockWarning) int {
		return cmp.Or(
			cmp.Compare(a.Path, b.Path),
			cmp.Compare(a.Error, b.Error),
		)
	})

	return result, nil
}

// expandPatterns expands Go package patterns like ./... to actual directories.
func expandPatterns(patterns []string) ([]string, error) {
	var dirs []string
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		if strings.HasSuffix(pattern, "/...") {
			base := strings.TrimSuffix(pattern, "/...")
			if base == "." {
				base = "."
			}
			if err := expandRecursivePattern(base, seen, &dirs); err != nil {
				return nil, err
			}
			continue
		}
		addDocblockDir(pattern, seen, &dirs)
	}

	return dirs, nil
}

func expandRecursivePattern(base string, seen map[string]bool, dirs *[]string) error {
	return filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		if shouldSkipDocblockDir(info.Name()) {
			return filepath.SkipDir
		}
		addDocblockDir(path, seen, dirs)
		return nil
	})
}

func shouldSkipDocblockDir(name string) bool {
	return name == "vendor" || name == "testdata" || (strings.HasPrefix(name, ".") && name != ".")
}

func addDocblockDir(path string, seen map[string]bool, dirs *[]string) {
	if seen[path] || !hasGoFiles(path) {
		return
	}
	*dirs = append(*dirs, path)
	seen[path] = true
}

// hasGoFiles checks if a directory contains Go files.
func hasGoFiles(dir string) bool {
	entries, err := coreio.Local.List(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			return true
		}
	}
	return false
}

// checkFile analyzes a single file for docblock coverage.
func checkFile(fset *token.FileSet, filename string, file *ast.File, result *DocblockResult) {
	filename = relativeDocblockFilename(filename)

	for _, decl := range file.Decls {
		checkDecl(fset, filename, decl, result)
	}
}

func relativeDocblockFilename(filename string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return filename
	}
	rel, err := filepath.Rel(cwd, filename)
	if err != nil {
		return filename
	}
	return rel
}

func checkDecl(fset *token.FileSet, filename string, decl ast.Decl, result *DocblockResult) {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		checkFuncDecl(fset, filename, d, result)
	case *ast.GenDecl:
		checkGenDecl(fset, filename, d, result)
	}
}

func checkFuncDecl(fset *token.FileSet, filename string, decl *ast.FuncDecl, result *DocblockResult) {
	if !isExportedDocblockFunc(decl) {
		return
	}
	recordDocblock(fset, filename, decl.Pos(), decl.Name.Name, "func", decl.Doc, result)
}

func isExportedDocblockFunc(decl *ast.FuncDecl) bool {
	if !ast.IsExported(decl.Name.Name) {
		return false
	}
	if decl.Recv == nil || len(decl.Recv.List) == 0 {
		return true
	}
	recvType := getReceiverTypeName(decl.Recv.List[0].Type)
	return recvType == "" || ast.IsExported(recvType)
}

func checkGenDecl(fset *token.FileSet, filename string, decl *ast.GenDecl, result *DocblockResult) {
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			checkTypeSpec(fset, filename, decl, s, result)
		case *ast.ValueSpec:
			checkValueSpec(fset, filename, decl, s, result)
		}
	}
}

func checkTypeSpec(
	fset *token.FileSet,
	filename string,
	decl *ast.GenDecl,
	spec *ast.TypeSpec,
	result *DocblockResult,
) {
	if !ast.IsExported(spec.Name.Name) {
		return
	}
	recordDocblock(fset, filename, spec.Pos(), spec.Name.Name, "type", mergedDoc(decl.Doc, spec.Doc), result)
}

func checkValueSpec(
	fset *token.FileSet,
	filename string,
	decl *ast.GenDecl,
	spec *ast.ValueSpec,
	result *DocblockResult,
) {
	for _, name := range spec.Names {
		if !ast.IsExported(name.Name) {
			continue
		}
		recordDocblock(fset, filename, name.Pos(), name.Name, kindFromToken(decl.Tok), mergedDoc(decl.Doc, spec.Doc), result)
	}
}

func mergedDoc(primary *ast.CommentGroup, fallback *ast.CommentGroup) *ast.CommentGroup {
	if primary != nil && len(primary.List) > 0 {
		return primary
	}
	return fallback
}

func recordDocblock(
	fset *token.FileSet,
	filename string,
	pos token.Pos,
	name string,
	kind string,
	doc *ast.CommentGroup,
	result *DocblockResult,
) {
	result.Total++
	if doc != nil && len(doc.List) > 0 {
		result.Documented++
		return
	}
	position := fset.Position(pos)
	result.Missing = append(result.Missing, MissingDocblock{
		File: filename,
		Line: position.Line,
		Name: name,
		Kind: kind,
	})
}

// getReceiverTypeName extracts the type name from a method receiver.
func getReceiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return getReceiverTypeName(t.X)
	}
	return ""
}

// kindFromToken returns a string representation of the token kind.
func kindFromToken(tok token.Token) string {
	switch tok {
	case token.CONST:
		return "const"
	case token.VAR:
		return "var"
	default:
		return "value"
	}
}
