package lint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ComplexityConfig controls cyclomatic complexity analysis.
type ComplexityConfig struct {
	Threshold int    // Minimum complexity to report (default 15)
	Path      string // Directory or file path to analyse
}

// ComplexityResult represents a single function with its cyclomatic complexity.
type ComplexityResult struct {
	FuncName   string `json:"func_name"`
	Package    string `json:"package"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Complexity int    `json:"complexity"`
}

// DefaultComplexityConfig returns a config with sensible defaults.
func DefaultComplexityConfig() ComplexityConfig {
	return ComplexityConfig{
		Threshold: 15,
		Path:      ".",
	}
}

// AnalyseComplexity walks Go source files and returns functions exceeding the
// configured complexity threshold. Uses native go/ast parsing — no external tools.
func AnalyseComplexity(cfg ComplexityConfig) ([]ComplexityResult, error) {
	if cfg.Threshold <= 0 {
		cfg.Threshold = 15
	}
	if cfg.Path == "" {
		cfg.Path = "."
	}

	var results []ComplexityResult

	info, err := os.Stat(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", cfg.Path, err)
	}

	if !info.IsDir() {
		fileResults, err := analyseFile(cfg.Path, cfg.Threshold)
		if err != nil {
			return nil, err
		}
		results = append(results, fileResults...)
		return results, nil
	}

	err = filepath.Walk(cfg.Path, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			name := fi.Name()
			if name == "vendor" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fileResults, err := analyseFile(path, cfg.Threshold)
		if err != nil {
			return nil // Skip files that fail to parse
		}
		results = append(results, fileResults...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", cfg.Path, err)
	}

	return results, nil
}

// AnalyseComplexitySource parses Go source code from a string and returns
// complexity results. Useful for testing without file I/O.
func AnalyseComplexitySource(src string, filename string, threshold int) ([]ComplexityResult, error) {
	if threshold <= 0 {
		threshold = 15
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filename, err)
	}

	var results []ComplexityResult
	pkgName := f.Name.Name

	ast.Inspect(f, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			complexity := calculateComplexity(fn)
			if complexity >= threshold {
				pos := fset.Position(fn.Pos())
				funcName := fn.Name.Name
				if fn.Recv != nil && len(fn.Recv.List) > 0 {
					funcName = receiverType(fn.Recv.List[0].Type) + "." + funcName
				}
				results = append(results, ComplexityResult{
					FuncName:   funcName,
					Package:    pkgName,
					File:       pos.Filename,
					Line:       pos.Line,
					Complexity: complexity,
				})
			}
		}
		return true
	})

	return results, nil
}

// analyseFile parses a single Go file and returns functions exceeding the threshold.
func analyseFile(path string, threshold int) ([]ComplexityResult, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return AnalyseComplexitySource(string(src), path, threshold)
}

// calculateComplexity computes the cyclomatic complexity of a function.
// Starts at 1, increments for each branching construct.
func calculateComplexity(fn *ast.FuncDecl) int {
	if fn.Body == nil {
		return 1
	}

	complexity := 1
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt:
			complexity++
		case *ast.ForStmt:
			complexity++
		case *ast.RangeStmt:
			complexity++
		case *ast.CaseClause:
			if node.List != nil {
				complexity++
			}
		case *ast.CommClause:
			if node.Comm != nil {
				complexity++
			}
		case *ast.BinaryExpr:
			if node.Op == token.LAND || node.Op == token.LOR {
				complexity++
			}
		case *ast.TypeSwitchStmt:
			complexity++
		case *ast.SelectStmt:
			complexity++
		}
		return true
	})

	return complexity
}

// receiverType extracts the type name from a method receiver.
func receiverType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverType(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return receiverType(t.X)
	default:
		return "?"
	}
}
