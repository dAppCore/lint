package lint

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
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
func LoadDir(dir string) (*Catalog, error) {
	entries, err := coreio.Local.List(dir)
	if err != nil {
		return nil, coreerr.E("Catalog.LoadDir", "loading catalog from "+dir, err)
	}
	sortDirEntries(entries)

	var rules []Rule
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		raw, err := coreio.Local.Read(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, coreerr.E("Catalog.LoadDir", "reading "+entry.Name(), err)
		}
		parsed, err := ParseRules([]byte(raw))
		if err != nil {
			return nil, coreerr.E("Catalog.LoadDir", "parsing "+entry.Name(), err)
		}
		rules = append(rules, parsed...)
	}

	return &Catalog{Rules: rules}, nil
}

// LoadFS reads all .yaml files from the given directory within an fs.FS and returns a Catalog.
func LoadFS(fsys fs.FS, dir string) (*Catalog, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, coreerr.E("Catalog.LoadFS", "loading catalog from embedded "+dir, err)
	}
	sortDirEntries(entries)

	var rules []Rule
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := fs.ReadFile(fsys, dir+"/"+entry.Name())
		if err != nil {
			return nil, coreerr.E("Catalog.LoadFS", "reading embedded "+entry.Name(), err)
		}
		parsed, err := ParseRules(data)
		if err != nil {
			return nil, coreerr.E("Catalog.LoadFS", "parsing embedded "+entry.Name(), err)
		}
		rules = append(rules, parsed...)
	}

	return &Catalog{Rules: rules}, nil
}

func sortDirEntries(entries []fs.DirEntry) {
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
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
