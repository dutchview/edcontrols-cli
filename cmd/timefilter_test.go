package cmd

import (
	"testing"
	"time"
)

func TestParseRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		input     string
		wantErr   bool
		checkFunc func(time.Time) bool
		desc      string
	}{
		{
			input: "3d",
			desc:  "3 days ago",
			checkFunc: func(got time.Time) bool {
				expected := now.AddDate(0, 0, -3)
				return withinSeconds(got, expected, 2)
			},
		},
		{
			input: "2w",
			desc:  "2 weeks ago",
			checkFunc: func(got time.Time) bool {
				expected := now.AddDate(0, 0, -14)
				return withinSeconds(got, expected, 2)
			},
		},
		{
			input: "1mo",
			desc:  "1 month ago",
			checkFunc: func(got time.Time) bool {
				expected := now.AddDate(0, -1, 0)
				return withinSeconds(got, expected, 2)
			},
		},
		{
			input: "1y",
			desc:  "1 year ago",
			checkFunc: func(got time.Time) bool {
				expected := now.AddDate(-1, 0, 0)
				return withinSeconds(got, expected, 2)
			},
		},
		{
			input: "2026-01-15",
			desc:  "absolute date",
			checkFunc: func(got time.Time) bool {
				expected := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
				return got.Equal(expected)
			},
		},
		{
			input:   "abc",
			desc:    "invalid input",
			wantErr: true,
		},
		{
			input:   "",
			desc:    "empty input",
			wantErr: true,
		},
		{
			input:   "3x",
			desc:    "invalid unit",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := ParseRelativeTime(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseRelativeTime(%q) expected error, got %v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseRelativeTime(%q) unexpected error: %v", tt.input, err)
				return
			}
			if !tt.checkFunc(got) {
				t.Errorf("ParseRelativeTime(%q) = %v, did not pass check", tt.input, got)
			}
		})
	}
}

func TestMatchesDates(t *testing.T) {
	// Fixed reference times for testing
	now := time.Now()
	oneWeekAgo := now.AddDate(0, 0, -7)
	twoWeeksAgo := now.AddDate(0, 0, -14)
	threeWeeksAgo := now.AddDate(0, 0, -21)

	// Format dates as API would return them
	recentDate := now.AddDate(0, 0, -3).Format(time.RFC3339)
	oldDate := threeWeeksAgo.Format(time.RFC3339)
	midDate := now.AddDate(0, 0, -10).Format(time.RFC3339)

	tests := []struct {
		desc     string
		filters  DateFilterSet
		created  string
		modified string
		want     bool
	}{
		{
			desc:    "no filters - matches everything",
			filters: DateFilterSet{},
			created: recentDate,
			want:    true,
		},
		{
			desc:    "created-after: recent date passes",
			filters: DateFilterSet{CreatedAfter: &twoWeeksAgo},
			created: recentDate,
			want:    true,
		},
		{
			desc:    "created-after: old date fails",
			filters: DateFilterSet{CreatedAfter: &oneWeekAgo},
			created: oldDate,
			want:    false,
		},
		{
			desc:    "created-before: old date passes",
			filters: DateFilterSet{CreatedBefore: &oneWeekAgo},
			created: oldDate,
			want:    true,
		},
		{
			desc:    "created-before: recent date fails",
			filters: DateFilterSet{CreatedBefore: &twoWeeksAgo},
			created: recentDate,
			want:    false,
		},
		{
			desc:    "range: mid date within range",
			filters: DateFilterSet{CreatedAfter: &threeWeeksAgo, CreatedBefore: &oneWeekAgo},
			created: midDate,
			want:    true,
		},
		{
			desc:    "range: recent date outside range",
			filters: DateFilterSet{CreatedAfter: &threeWeeksAgo, CreatedBefore: &twoWeeksAgo},
			created: recentDate,
			want:    false,
		},
		{
			desc:    "modified-after: recent modified passes",
			filters: DateFilterSet{ModifiedAfter: &twoWeeksAgo},
			created: oldDate,
			modified: recentDate,
			want:    true,
		},
		{
			desc:    "modified-after: old modified fails",
			filters: DateFilterSet{ModifiedAfter: &oneWeekAgo},
			created: oldDate,
			modified: oldDate,
			want:    false,
		},
		{
			desc:    "empty created date fails when filter active",
			filters: DateFilterSet{CreatedAfter: &twoWeeksAgo},
			created: "",
			want:    false,
		},
		{
			desc:     "empty modified date fails when filter active",
			filters:  DateFilterSet{ModifiedAfter: &twoWeeksAgo},
			created:  recentDate,
			modified: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := tt.filters.MatchesDates(tt.created, tt.modified)
			if got != tt.want {
				t.Errorf("MatchesDates() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasDateFilters(t *testing.T) {
	now := time.Now()

	t.Run("empty filter set", func(t *testing.T) {
		f := DateFilterSet{}
		if f.HasDateFilters() {
			t.Error("expected HasDateFilters() = false for empty set")
		}
	})

	t.Run("with created-after", func(t *testing.T) {
		f := DateFilterSet{CreatedAfter: &now}
		if !f.HasDateFilters() {
			t.Error("expected HasDateFilters() = true")
		}
	})

	t.Run("with modified-before", func(t *testing.T) {
		f := DateFilterSet{ModifiedBefore: &now}
		if !f.HasDateFilters() {
			t.Error("expected HasDateFilters() = true")
		}
	})
}

// withinSeconds checks if two times are within n seconds of each other
func withinSeconds(a, b time.Time, n int) bool {
	diff := a.Sub(b)
	if diff < 0 {
		diff = -diff
	}
	return diff < time.Duration(n)*time.Second
}
