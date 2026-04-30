package lint

import (
	"go/ast"
	"go/parser"
	"go/token"

	core "dappco.re/go"
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

type complexitySkipResult struct {
	Skip      bool
	WalkError error
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
func AnalyseComplexity(cfg ComplexityConfig) core.Result {
	cfg = normaliseComplexityConfig(cfg)
	stat := core.Stat(cfg.Path)
	if !stat.OK {
		err, _ := stat.Value.(error)
		return core.Fail(core.E("AnalyseComplexity", "stat "+cfg.Path, err))
	}
	info := stat.Value.(core.FsFileInfo)
	if !info.IsDir() {
		return analyseFile(cfg.Path, cfg.Threshold)
	}
	return analyseComplexityDir(cfg)
}

func normaliseComplexityConfig(cfg ComplexityConfig) ComplexityConfig {
	if cfg.Threshold <= 0 {
		cfg.Threshold = 15
	}
	if cfg.Path == "" {
		cfg.Path = "."
	}
	return cfg
}

func analyseComplexityDir(cfg ComplexityConfig) core.Result {
	var results []ComplexityResult
	err := core.PathWalkDir(cfg.Path, func(path string, entry core.FsDirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		skip := shouldSkipComplexityPath(path, entry)
		decision := skip.Value.(complexitySkipResult)
		if decision.Skip || decision.WalkError != nil {
			return decision.WalkError
		}
		results = append(results, analyseComplexityFile(path, cfg.Threshold)...)
		return nil
	})
	if err != nil {
		return core.Fail(core.E("AnalyseComplexity", "walk "+cfg.Path, err))
	}

	return core.Ok(results)
}

func shouldSkipComplexityPath(path string, entry core.FsDirEntry) core.Result {
	if entry.IsDir() {
		name := entry.Name()
		if name == "vendor" || core.HasPrefix(name, ".") {
			return core.Ok(complexitySkipResult{Skip: true, WalkError: core.PathSkipDir})
		}
		return core.Ok(complexitySkipResult{Skip: true})
	}
	return core.Ok(complexitySkipResult{Skip: !core.HasSuffix(path, ".go") || core.HasSuffix(path, "_test.go")})
}

func analyseComplexityFile(path string, threshold int) []ComplexityResult {
	fileResults := analyseFile(path, threshold)
	if !fileResults.OK {
		return nil
	}
	return fileResults.Value.([]ComplexityResult)
}

// AnalyseComplexitySource parses Go source code from a string and returns
// complexity results. Useful for testing without file I/O.
func AnalyseComplexitySource(src string, filename string, threshold int) core.Result {
	if threshold <= 0 {
		threshold = 15
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return core.Fail(core.E("AnalyseComplexitySource", "parse "+filename, err))
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

	return core.Ok(results)
}

// analyseFile parses a single Go file and returns functions exceeding the threshold.
func analyseFile(path string, threshold int) core.Result {
	read := core.ReadFile(path)
	if !read.OK {
		err, _ := read.Value.(error)
		return core.Fail(core.E("analyseFile", "read "+path, err))
	}
	return AnalyseComplexitySource(string(read.Value.([]byte)), path, threshold)
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
