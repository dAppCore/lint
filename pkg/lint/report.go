package lint

import (
	"encoding/json"
	"fmt"
	"io"
)

// Summary holds aggregate counts for a set of findings.
type Summary struct {
	Total      int            `json:"total"`
	BySeverity map[string]int `json:"by_severity"`
}

// Summarise counts findings by severity.
func Summarise(findings []Finding) Summary {
	s := Summary{
		Total:      len(findings),
		BySeverity: make(map[string]int),
	}
	for _, f := range findings {
		s.BySeverity[f.Severity]++
	}
	return s
}

// WriteJSON writes findings as a pretty-printed JSON array.
func WriteJSON(w io.Writer, findings []Finding) error {
	if findings == nil {
		findings = []Finding{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(findings)
}

// WriteJSONL writes findings as newline-delimited JSON (one object per line).
func WriteJSONL(w io.Writer, findings []Finding) error {
	for _, f := range findings {
		data, err := json.Marshal(f)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s\n", data); err != nil {
			return err
		}
	}
	return nil
}

// WriteText writes findings in a human-readable format:
//
//	file:line [severity] title (rule-id)
func WriteText(w io.Writer, findings []Finding) {
	for _, f := range findings {
		fmt.Fprintf(w, "%s:%d [%s] %s (%s)\n", f.File, f.Line, f.Severity, f.Title, f.RuleID)
	}
}
