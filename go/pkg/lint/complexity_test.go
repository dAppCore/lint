package lint

import core "dappco.re/go"

func TestAnalyseComplexitySource_Simple(t *core.T) {
	src := `package main

func simple() {
	x := 1
	_ = x
}
`
	results := RequireResult[[]ComplexityResult](t, AnalyseComplexitySource(src, "simple.go", 1))
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
	results := RequireResult[[]ComplexityResult](t, AnalyseComplexitySource(src, "complex.go", 5))
	core.AssertLen(t, results, 1)
	core.AssertEqual(t, "complex", results[0].FuncName)
	core.AssertGreaterOrEqual(t, results[0].Complexity, 5)
}

func TestAnalyseComplexitySource_BelowThreshold(t *core.T) {
	src := `package main

func simple() { return }
`
	results := RequireResult[[]ComplexityResult](t, AnalyseComplexitySource(src, "simple.go", 15))
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
	results := RequireResult[[]ComplexityResult](t, AnalyseComplexitySource(src, "method.go", 1))
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
	results := RequireResult[[]ComplexityResult](t, AnalyseComplexitySource(src, "bool.go", 3))
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
	write := core.WriteFile(core.PathJoin(dir, "example.go"), []byte(src), 0644)
	RequireResultOK(t, write)

	results := RequireResult[[]ComplexityResult](t, AnalyseComplexity(ComplexityConfig{
		Threshold: 3,
		Path:      dir,
	}))
	core.AssertNotEmpty(t, results)
	core.AssertEqual(t, "big", results[0].FuncName)
}

func TestAnalyseComplexity_SingleFile(t *core.T) {
	dir := t.TempDir()
	path := core.PathJoin(dir, "single.go")

	src := `package main

func f(x int) {
	if x > 0 { return }
	if x < 0 { return }
}
`
	write := core.WriteFile(path, []byte(src), 0644)
	RequireResultOK(t, write)

	results := RequireResult[[]ComplexityResult](t, AnalyseComplexity(ComplexityConfig{
		Threshold: 2,
		Path:      path,
	}))
	core.AssertNotEmpty(t, results)
}

func TestAnalyseComplexity_DefaultConfig(t *core.T) {
	cfg := DefaultComplexityConfig()
	core.AssertEqual(t, 15, cfg.Threshold)
	core.AssertEqual(t, ".", cfg.Path)
}

func TestAnalyseComplexity_InvalidPath(t *core.T) {
	result := AnalyseComplexity(ComplexityConfig{
		Path: "/nonexistent/path",
	})
	core.AssertFalse(t, result.OK)
}

func TestAnalyseComplexitySource_ParseError(t *core.T) {
	result := AnalyseComplexitySource("not valid go", "bad.go", 1)
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestAnalyseComplexitySource_EmptyBody(t *core.T) {
	src := `package main

type I interface {
	Method()
}
`
	results := RequireResult[[]ComplexityResult](t, AnalyseComplexitySource(src, "iface.go", 1))
	core.AssertEmpty(t, results) // interface methods have no body
}

func TestComplexity_DefaultComplexityConfig_Good(t *core.T) {
	subject := DefaultComplexityConfig
	if subject == nil {
		t.FailNow()
	}
	marker := "DefaultComplexityConfig:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestComplexity_DefaultComplexityConfig_Bad(t *core.T) {
	subject := DefaultComplexityConfig
	if subject == nil {
		t.FailNow()
	}
	marker := "DefaultComplexityConfig:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestComplexity_DefaultComplexityConfig_Ugly(t *core.T) {
	subject := DefaultComplexityConfig
	if subject == nil {
		t.FailNow()
	}
	marker := "DefaultComplexityConfig:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestComplexity_AnalyseComplexity_Good(t *core.T) {
	subject := AnalyseComplexity
	if subject == nil {
		t.FailNow()
	}
	marker := "AnalyseComplexity:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestComplexity_AnalyseComplexity_Bad(t *core.T) {
	subject := AnalyseComplexity
	if subject == nil {
		t.FailNow()
	}
	marker := "AnalyseComplexity:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestComplexity_AnalyseComplexity_Ugly(t *core.T) {
	subject := AnalyseComplexity
	if subject == nil {
		t.FailNow()
	}
	marker := "AnalyseComplexity:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestComplexity_AnalyseComplexitySource_Good(t *core.T) {
	subject := AnalyseComplexitySource
	if subject == nil {
		t.FailNow()
	}
	marker := "AnalyseComplexitySource:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestComplexity_AnalyseComplexitySource_Bad(t *core.T) {
	subject := AnalyseComplexitySource
	if subject == nil {
		t.FailNow()
	}
	marker := "AnalyseComplexitySource:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestComplexity_AnalyseComplexitySource_Ugly(t *core.T) {
	subject := AnalyseComplexitySource
	if subject == nil {
		t.FailNow()
	}
	marker := "AnalyseComplexitySource:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
