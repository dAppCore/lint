// Package detect identifies project types by examining filesystem markers.
package detect

import (
	"path/filepath"

	coreio "dappco.re/go/core/io"
)

// ProjectType identifies a project's language/framework.
type ProjectType string

const (
	Go  ProjectType = "go"
	PHP ProjectType = "php"
)

// IsGoProject returns true if dir contains a go.mod file.
func IsGoProject(dir string) bool {
	return coreio.Local.Exists(filepath.Join(dir, "go.mod"))
}

// IsPHPProject returns true if dir contains a composer.json file.
func IsPHPProject(dir string) bool {
	return coreio.Local.Exists(filepath.Join(dir, "composer.json"))
}

// DetectAll returns all detected project types in the directory.
func DetectAll(dir string) []ProjectType {
	var types []ProjectType
	if IsGoProject(dir) {
		types = append(types, Go)
	}
	if IsPHPProject(dir) {
		types = append(types, PHP)
	}
	return types
}
