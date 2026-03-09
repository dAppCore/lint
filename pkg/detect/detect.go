// Package detect identifies project types by examining filesystem markers.
package detect

import "os"

// ProjectType identifies a project's language/framework.
type ProjectType string

const (
	Go  ProjectType = "go"
	PHP ProjectType = "php"
)

// IsGoProject returns true if dir contains a go.mod file.
func IsGoProject(dir string) bool {
	_, err := os.Stat(dir + "/go.mod")
	return err == nil
}

// IsPHPProject returns true if dir contains a composer.json file.
func IsPHPProject(dir string) bool {
	_, err := os.Stat(dir + "/composer.json")
	return err == nil
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
