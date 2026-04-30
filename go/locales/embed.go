// Package locales embeds translation files for this module.
package locales

import "embed"

//go:embed *.json
var FS embed.FS
