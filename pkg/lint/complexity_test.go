package lint

import (
	core "dappco.re/go"
	"os"
	"path/filepath"
)

func TestAnalyseComplexitySource_Simple(t *core.T) {
	src := `package main

func simple() {
	x := 1
	_ = x
}
`
	results, err := AnalyseComplexitySource(src, "simple.go", 1)
	core.RequireNoError(t, err)
	core.AssertLen(t, results, 1)
	core.AssertEqual(t, "simple", results[0].FuncName)
	core.AssertEqual(t, 1, results[0].Complexity)
	core.AssertEqual(t, "main", results[0].Package)
}

func TestAnalyseComplexitySource_Complex(t *core.T) {
	src := `package main

func complex(x int) string {
	if x > 0 {
		if x > 10 {
			return "big"
		}
		return "small"
	}
	for i := 0; i < x; i++ {
		if i%2 == 0 {
			continue
		}
	}
	switch x {
	case 1:
		return "one"
	case 2:
		return "two"
	case 3:
		return "three"
	default:
		return "other"
	}
}
`
	// Complexity: 1 (base) + 2 (if) + 1 (for) + 1 (if) + 3 (case clauses) = 8
	results, err := AnalyseComplexitySource(src, "complex.go", 5)
	core.RequireNoError(t, err)
	core.AssertLen(t, results, 1)
	core.AssertEqual(t, "complex", results[0].FuncName)
	core.AssertGreaterOrEqual(t, results[0].Complexity, 5)
}

func TestAnalyseComplexitySource_BelowThreshold(t *core.T) {
	src := `package main

func simple() { return }
`
	results, err := AnalyseComplexitySource(src, "simple.go", 15)
	core.RequireNoError(t, err)
	core.AssertEmpty(t, results)
}

func TestAnalyseComplexitySource_Method(t *core.T) {
	src := `package main

type Foo struct{}

func (f *Foo) Bar(x int) {
	if x > 0 {
		if x > 10 {
			return
		}
	}
}
`
	results, err := AnalyseComplexitySource(src, "method.go", 1)
	core.RequireNoError(t, err)
	RequireLen(t, results, 1)
	core.AssertEqual(t, "Foo.Bar", results[0].FuncName)
}

func TestAnalyseComplexitySource_BinaryExpr(t *core.T) {
	src := `package main

func boolHeavy(a, b, c, d bool) bool {
	if a && b || c && d {
		return true
	}
	return false
}
`
	// 1 (base) + 1 (if) + 3 (&&, ||, &&) = 5
	results, err := AnalyseComplexitySource(src, "bool.go", 3)
	core.RequireNoError(t, err)
	RequireLen(t, results, 1)
	core.AssertGreaterOrEqual(t, results[0].Complexity, 4)
}

func TestAnalyseComplexity_Directory(t *core.T) {
	dir := t.TempDir()

	// Write a Go file with a complex function
	src := `package example

func big(x int) {
	if x > 0 {
		for i := range 10 {
			switch i {
			case 1:
				break
			case 2:
				break
			}
		}
	}
}
`
	err := os.WriteFile(filepath.Join(dir, "example.go"), []byte(src), 0644)
	core.RequireNoError(t, err)

	results, err := AnalyseComplexity(ComplexityConfig{
		Threshold: 3,
		Path:      dir,
	})
	core.RequireNoError(t, err)
	core.AssertNotEmpty(t, results)
	core.AssertEqual(t, "big", results[0].FuncName)
}

func TestAnalyseComplexity_SingleFile(t *core.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "single.go")

	src := `package main

func f(x int) {
	if x > 0 { return }
	if x < 0 { return }
}
`
	err := os.WriteFile(path, []byte(src), 0644)
	core.RequireNoError(t, err)

	results, err := AnalyseComplexity(ComplexityConfig{
		Threshold: 2,
		Path:      path,
	})
	core.RequireNoError(t, err)
	core.AssertNotEmpty(t, results)
}

func TestAnalyseComplexity_DefaultConfig(t *core.T) {
	cfg := DefaultComplexityConfig()
	core.AssertEqual(t, 15, cfg.Threshold)
	core.AssertEqual(t, ".", cfg.Path)
}

func TestAnalyseComplexity_InvalidPath(t *core.T) {
	_, err := AnalyseComplexity(ComplexityConfig{
		Path: "/nonexistent/path",
	})
	core.AssertError(t, err)
}

func TestAnalyseComplexitySource_ParseError(t *core.T) {
	results, err := AnalyseComplexitySource("not valid go", "bad.go", 1)
	core.AssertError(t, err)
	core.AssertNil(t, results)
}

func TestAnalyseComplexitySource_EmptyBody(t *core.T) {
	src := `package main

type I interface {
	Method()
}
`
	results, err := AnalyseComplexitySource(src, "iface.go", 1)
	core.RequireNoError(t, err)
	core.AssertEmpty(t, results) // interface methods have no body
}
