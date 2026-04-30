package lint

import (
	"io/fs"
	"slices"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

const (
	catalogCatalogLoaddir13dba1 = "Catalog.LoadDir"
	catalogCatalogLoadfs363359  = "Catalog.LoadFS"
)

// severityOrder maps severity names to numeric ranks for threshold comparison.
var severityOrder = map[string]int{
	"info":     0,
	"low":      1,
	"medium":   2,
	"high":     3,
	"critical": 4,
}

// Catalog holds a collection of lint rules loaded from YAML files.
type Catalog struct {
	Rules []Rule
}

// LoadDir reads all .yaml files from the given directory and returns a Catalog.
func LoadDir(dir string) core.Result {
	entries, err := coreio.Local.List(dir)
	if err != nil {
		return core.Fail(core.E(catalogCatalogLoaddir13dba1, "loading catalog from "+dir, err))
	}
	sortDirEntries(entries)

	var rules []Rule
	for _, entry := range entries {
		if entry.IsDir() || !core.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		raw, err := coreio.Local.Read(core.JoinPath(dir, entry.Name()))
		if err != nil {
			return core.Fail(core.E(catalogCatalogLoaddir13dba1, "reading "+entry.Name(), err))
		}
		parsedResult := ParseRules([]byte(raw))
		if !parsedResult.OK {
			err, _ := parsedResult.Value.(error)
			return core.Fail(core.E(catalogCatalogLoaddir13dba1, "parsing "+entry.Name(), err))
		}
		parsed := parsedResult.Value.([]Rule)
		rules = append(rules, parsed...)
	}

	return core.Ok(&Catalog{Rules: rules})
}

// LoadFS reads all .yaml files from the given directory within an fs.FS and returns a Catalog.
func LoadFS(fsys fs.FS, dir string) core.Result {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return core.Fail(core.E(catalogCatalogLoadfs363359, "loading catalog from embedded "+dir, err))
	}
	sortDirEntries(entries)

	var rules []Rule
	for _, entry := range entries {
		if entry.IsDir() || !core.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := fs.ReadFile(fsys, core.JoinPath(dir, entry.Name()))
		if err != nil {
			return core.Fail(core.E(catalogCatalogLoadfs363359, "reading embedded "+entry.Name(), err))
		}
		parsedResult := ParseRules(data)
		if !parsedResult.OK {
			err, _ := parsedResult.Value.(error)
			return core.Fail(core.E(catalogCatalogLoadfs363359, "parsing embedded "+entry.Name(), err))
		}
		parsed := parsedResult.Value.([]Rule)
		rules = append(rules, parsed...)
	}

	return core.Ok(&Catalog{Rules: rules})
}

func sortDirEntries(entries []fs.DirEntry) {
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return core.Compare(a.Name(), b.Name())
	})
}

// ForLanguage returns all rules that apply to the given language.
func (c *Catalog) ForLanguage(lang string) []Rule {
	var result []Rule
	for _, r := range c.Rules {
		if slices.Contains(r.Languages, lang) {
			result = append(result, r)
		}
	}
	return result
}

// AtSeverity returns all rules at or above the given severity threshold.
func (c *Catalog) AtSeverity(threshold string) []Rule {
	minRank, ok := severityOrder[threshold]
	if !ok {
		return nil
	}

	var result []Rule
	for _, r := range c.Rules {
		if rank, ok := severityOrder[r.Severity]; ok && rank >= minRank {
			result = append(result, r)
		}
	}
	return result
}

// ByID returns the rule with the given ID, or nil if not found.
func (c *Catalog) ByID(id string) *Rule {
	for i := range c.Rules {
		if c.Rules[i].ID == id {
			return &c.Rules[i]
		}
	}
	return nil
}
