package lint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyseComplexitySource_Simple(t *testing.T) {
	src := `package main

func simple() {
	x := 1
	_ = x
}
`
	results, err := AnalyseComplexitySource(src, "simple.go", 1)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "simple", results[0].FuncName)
	assert.Equal(t, 1, results[0].Complexity)
	assert.Equal(t, "main", results[0].Package)
}

func TestAnalyseComplexitySource_Complex(t *testing.T) {
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
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "complex", results[0].FuncName)
	assert.GreaterOrEqual(t, results[0].Complexity, 5)
}

func TestAnalyseComplexitySource_BelowThreshold(t *testing.T) {
	src := `package main

func simple() { return }
`
	results, err := AnalyseComplexitySource(src, "simple.go", 15)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestAnalyseComplexitySource_Method(t *testing.T) {
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
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Foo.Bar", results[0].FuncName)
}

func TestAnalyseComplexitySource_BinaryExpr(t *testing.T) {
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
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.GreaterOrEqual(t, results[0].Complexity, 4)
}

func TestAnalyseComplexity_Directory(t *testing.T) {
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
	require.NoError(t, err)

	results, err := AnalyseComplexity(ComplexityConfig{
		Threshold: 3,
		Path:      dir,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, results)
	assert.Equal(t, "big", results[0].FuncName)
}

func TestAnalyseComplexity_SingleFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "single.go")

	src := `package main

func f(x int) {
	if x > 0 { return }
	if x < 0 { return }
}
`
	err := os.WriteFile(path, []byte(src), 0644)
	require.NoError(t, err)

	results, err := AnalyseComplexity(ComplexityConfig{
		Threshold: 2,
		Path:      path,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestAnalyseComplexity_DefaultConfig(t *testing.T) {
	cfg := DefaultComplexityConfig()
	assert.Equal(t, 15, cfg.Threshold)
	assert.Equal(t, ".", cfg.Path)
}

func TestAnalyseComplexity_InvalidPath(t *testing.T) {
	_, err := AnalyseComplexity(ComplexityConfig{
		Path: "/nonexistent/path",
	})
	assert.Error(t, err)
}

func TestAnalyseComplexitySource_ParseError(t *testing.T) {
	_, err := AnalyseComplexitySource("not valid go", "bad.go", 1)
	assert.Error(t, err)
}

func TestAnalyseComplexitySource_EmptyBody(t *testing.T) {
	src := `package main

type I interface {
	Method()
}
`
	results, err := AnalyseComplexitySource(src, "iface.go", 1)
	require.NoError(t, err)
	assert.Empty(t, results) // interface methods have no body
}
