package locales

import (
	core "dappco.re/go"
)

// TestEmbedFS_Good_ContainsEnglishLocale asserts the embedded filesystem
// actually carries the en.json translation file (guards against the //go:embed
// glob silently matching nothing).
func TestEmbedFS_Good_ContainsEnglishLocale(t *core.T) {
	data, err := FS.ReadFile("en.json")
	core.RequireNoError(t, err)
	core.AssertNotEmpty(t, data)
}

// TestEmbedFS_Good_EnglishLocaleParsesAsJSON asserts the embedded en.json is
// valid JSON, so a shipped locale is never structurally broken.
func TestEmbedFS_Good_EnglishLocaleParsesAsJSON(t *core.T) {
	data, err := FS.ReadFile("en.json")
	core.RequireNoError(t, err)

	var decoded map[string]any
	result := core.JSONUnmarshal(data, &decoded)
	core.RequireTrue(t, result.OK, result.Error())
	core.AssertNotEmpty(t, decoded)
}

// TestEmbedFS_Bad_MissingFileFails asserts a lookup for a non-existent locale
// returns an error rather than empty success.
func TestEmbedFS_Bad_MissingFileFails(t *core.T) {
	_, err := FS.ReadFile("klingon.json")
	core.AssertNotNil(t, err)
}

// TestEmbedFS_Ugly_ListsAtLeastOneLocale asserts the embedded directory holds at
// least one JSON locale entry.
func TestEmbedFS_Ugly_ListsAtLeastOneLocale(t *core.T) {
	listed := core.ReadDir(FS, ".")
	core.RequireTrue(t, listed.OK, listed.Error())
	entries := listed.Value.([]core.FsDirEntry)
	var jsonCount int
	for _, entry := range entries {
		if core.HasSuffix(entry.Name(), ".json") {
			jsonCount++
		}
	}
	core.AssertGreater(t, jsonCount, 0)
}
