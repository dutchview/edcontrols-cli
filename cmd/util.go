package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func statusString(status string) string {
	if status == "" {
		return "-"
	}
	return status
}

// humanID converts a CouchDB ID to a human-readable format.
// Takes the last 6 characters, reverses them, and converts to uppercase.
// Example: e4fcf23e74fe3a9c74dec23350b554cc -> CC455B
func humanID(couchDbID string) string {
	if len(couchDbID) < 6 {
		return strings.ToUpper(couchDbID)
	}
	last6 := couchDbID[len(couchDbID)-6:]
	// Reverse the string
	runes := []rune(last6)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return strings.ToUpper(string(runes))
}

// isFieldSet checks if an interface{} field is set (not nil, not empty, not false).
// Used for fields like Archived/Deleted that can be null, bool, or datetime string.
func isFieldSet(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != ""
	default:
		return true
	}
}
