package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// DateFilterSet holds parsed time boundaries for date filtering.
type DateFilterSet struct {
	CreatedAfter   *time.Time
	CreatedBefore  *time.Time
	ModifiedAfter  *time.Time
	ModifiedBefore *time.Time
}

// HasDateFilters returns true if any date filter is set.
func (f *DateFilterSet) HasDateFilters() bool {
	return f.CreatedAfter != nil || f.CreatedBefore != nil ||
		f.ModifiedAfter != nil || f.ModifiedBefore != nil
}

// MatchesDates returns true if the given creation and modification dates
// pass all active filters. Empty date strings are treated as non-matching
// when a filter requires that field.
func (f *DateFilterSet) MatchesDates(createdStr, modifiedStr string) bool {
	if f.CreatedAfter != nil || f.CreatedBefore != nil {
		created, err := parseAPIDate(createdStr)
		if err != nil {
			return false
		}
		if f.CreatedAfter != nil && created.Before(*f.CreatedAfter) {
			return false
		}
		if f.CreatedBefore != nil && created.After(*f.CreatedBefore) {
			return false
		}
	}

	if f.ModifiedAfter != nil || f.ModifiedBefore != nil {
		modified, err := parseAPIDate(modifiedStr)
		if err != nil {
			return false
		}
		if f.ModifiedAfter != nil && modified.Before(*f.ModifiedAfter) {
			return false
		}
		if f.ModifiedBefore != nil && modified.After(*f.ModifiedBefore) {
			return false
		}
	}

	return true
}

var relativeTimeRe = regexp.MustCompile(`^(\d+)(d|w|mo|y)$`)

// ParseRelativeTime parses a relative time expression (e.g., "3d", "2w", "1mo", "1y")
// or an absolute date (e.g., "2026-01-15") and returns the corresponding time.Time.
// Relative times are computed relative to now.
func ParseRelativeTime(s string) (time.Time, error) {
	// Try relative time first
	if m := relativeTimeRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		now := time.Now()
		switch m[2] {
		case "d":
			return now.AddDate(0, 0, -n), nil
		case "w":
			return now.AddDate(0, 0, -n*7), nil
		case "mo":
			return now.AddDate(0, -n, 0), nil
		case "y":
			return now.AddDate(-n, 0, 0), nil
		}
	}

	// Try absolute date (YYYY-MM-DD)
	t, err := time.Parse("2006-01-02", s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid time expression %q (use e.g. 3d, 2w, 1mo, 1y, or 2026-01-15)", s)
}

// parseAPIDate parses date strings from the EdControls API.
// Supports ISO 8601 formats with and without timezone.
func parseAPIDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, layout := range formats {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse date %q", s)
}
