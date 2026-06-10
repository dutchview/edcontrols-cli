// Package stats aggregates tickets and audits into grouped counts
// client-side; see docs/adr/0001-client-side-aggregation-for-stats.md.
package stats

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Record is the neutral representation of a ticket or audit that the
// aggregation engine operates on.
type Record struct {
	Status    string
	Assignee  string // ticket responsible or audit auditor
	Project   string
	Template  string
	Author    string
	GroupID   string
	Tags      []string
	Created   string // ISO 8601 strings as returned by the API
	Modified  string
	Completed string
	Due       string
}

// Entity selects which group-by dimensions are valid.
type Entity int

const (
	Tickets Entity = iota
	Audits
)

// Dimension is one axis of a group-by.
type Dimension struct {
	Name   string // as written by the user, e.g. "created:month"
	field  string
	bucket string // "week" or "month" for date fields, else empty
}

// noneKey is the bucket for records missing a value along a dimension.
const noneKey = "(none)"

// values returns the bucket key(s) a record contributes to along this
// dimension.
func (d Dimension) values(r Record) []string {
	var v string
	switch d.field {
	case "status":
		v = r.Status
	case "responsible", "auditor":
		v = r.Assignee
	case "project":
		v = r.Project
	case "template":
		v = r.Template
	case "author":
		v = r.Author
	case "tag":
		if len(r.Tags) == 0 {
			return []string{noneKey}
		}
		return r.Tags
	case "created":
		v = formatBucket(r.Created, d.bucket)
	case "completed":
		v = formatBucket(r.Completed, d.bucket)
	}
	if v == "" {
		v = noneKey
	}
	return []string{v}
}

// formatBucket renders a date string as a week ("2026-W01") or month
// ("2026-01") bucket key, or "" when the date is missing or unparseable.
func formatBucket(date, bucket string) string {
	if date == "" {
		return ""
	}
	t, err := parseDate(date)
	if err != nil {
		return ""
	}
	if bucket == "week" {
		y, w := t.ISOWeek()
		return fmt.Sprintf("%d-W%02d", y, w)
	}
	return t.Format("2006-01")
}

// fieldsFor lists the valid group-by fields per entity. Date fields
// (created, completed) additionally require a :week or :month bucket.
func fieldsFor(entity Entity) map[string]bool {
	fields := map[string]bool{
		"status": true, "project": true, "tag": true, "author": true,
		"created": true, "completed": true,
	}
	if entity == Audits {
		fields["auditor"] = true
		fields["template"] = true
	} else {
		fields["responsible"] = true
	}
	return fields
}

// ParseGroupBy parses a comma-separated group-by spec into at most two
// dimensions, validating names against the entity's field set.
func ParseGroupBy(spec string, entity Entity) ([]Dimension, error) {
	valid := fieldsFor(entity)
	var dims []Dimension
	for _, name := range strings.Split(spec, ",") {
		name = strings.TrimSpace(name)
		d := Dimension{Name: name, field: name}
		if field, bucket, ok := strings.Cut(name, ":"); ok {
			d.field = field
			d.bucket = bucket
		}
		if !valid[d.field] {
			return nil, fmt.Errorf("unknown group-by dimension %q (valid: %s)", name, strings.Join(validNames(entity), ", "))
		}
		isDateField := d.field == "created" || d.field == "completed"
		if isDateField && d.bucket != "week" && d.bucket != "month" {
			return nil, fmt.Errorf("dimension %q requires a :week or :month bucket (e.g. %s:month)", name, d.field)
		}
		if !isDateField && d.bucket != "" {
			return nil, fmt.Errorf("dimension %q does not take a bucket", name)
		}
		dims = append(dims, d)
	}
	if len(dims) > 2 {
		return nil, fmt.Errorf("at most two group-by dimensions are supported, got %d", len(dims))
	}
	return dims, nil
}

// validNames lists the user-facing dimension names for error messages.
func validNames(entity Entity) []string {
	names := []string{"status", "project", "tag", "author", "created:week", "created:month", "completed:week", "completed:month"}
	if entity == Audits {
		return append([]string{"template", "auditor"}, names...)
	}
	return append([]string{"responsible"}, names...)
}

// Filter selects which records take part in the aggregation. It is
// applied client-side even when the server filters too, so results stay
// correct regardless of which filters the search endpoint honors.
type Filter struct {
	Status      string
	Assignee    string // ticket responsible or audit auditor
	Tag         string
	GroupID     string
	OverdueOnly bool
}

// Matches reports whether a record passes the filter.
func (f Filter) Matches(r Record, now time.Time) bool {
	if f.Status != "" && r.Status != f.Status {
		return false
	}
	if f.Assignee != "" && !strings.EqualFold(r.Assignee, f.Assignee) {
		return false
	}
	if f.Tag != "" {
		found := false
		for _, t := range r.Tags {
			if t == f.Tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if f.GroupID != "" && r.GroupID != f.GroupID {
		return false
	}
	if f.OverdueOnly && !r.IsOverdue(now) {
		return false
	}
	return true
}

// Group is one bucket of the aggregation.
type Group struct {
	Keys    map[string]string `json:"keys"`
	Count   int               `json:"count"`
	Overdue int               `json:"overdue"`
}

// Result is the aggregation outcome.
type Result struct {
	Total   int      `json:"total"`
	Overdue int      `json:"overdue"`
	GroupBy []string `json:"groupBy"`
	Groups  []Group  `json:"groups"`
}

// IsOverdue reports whether a record's due date has passed while the
// record is not completed.
func (r Record) IsOverdue(now time.Time) bool {
	if r.Due == "" || r.Status == "completed" {
		return false
	}
	due, err := parseDate(r.Due)
	if err != nil {
		return false
	}
	return due.Before(now)
}

// parseDate parses ISO 8601 date strings as returned by the EdControls API.
func parseDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	var err error
	for _, layout := range formats {
		var t time.Time
		t, err = time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, err
}

// Aggregate groups records along the given dimensions, counting totals
// and overdue records per bucket.
func Aggregate(records []Record, dims []Dimension, now time.Time) Result {
	res := Result{Total: len(records)}
	for _, d := range dims {
		res.GroupBy = append(res.GroupBy, d.Name)
	}

	buckets := make(map[string]*Group)
	for _, r := range records {
		overdue := r.IsOverdue(now)
		if overdue {
			res.Overdue++
		}
		if len(dims) == 0 {
			continue // no grouping requested, totals only
		}
		// A record can land in multiple buckets (multi-valued
		// dimensions like tag), so build the cross product of its
		// per-dimension values.
		combos := [][]string{{}}
		for _, d := range dims {
			var next [][]string
			for _, prefix := range combos {
				for _, v := range d.values(r) {
					next = append(next, append(append([]string{}, prefix...), v))
				}
			}
			combos = next
		}
		for _, combo := range combos {
			key := strings.Join(combo, "\x00")
			g, ok := buckets[key]
			if !ok {
				g = &Group{Keys: map[string]string{}}
				for i, d := range dims {
					g.Keys[d.Name] = combo[i]
				}
				buckets[key] = g
			}
			g.Count++
			if overdue {
				g.Overdue++
			}
		}
	}

	for _, g := range buckets {
		res.Groups = append(res.Groups, *g)
	}
	sort.Slice(res.Groups, func(i, j int) bool {
		a, b := res.Groups[i], res.Groups[j]
		// Time buckets order chronologically; everything else by
		// count descending.
		for _, d := range dims {
			if d.bucket == "" {
				continue
			}
			av, bv := a.Keys[d.Name], b.Keys[d.Name]
			if av != bv {
				// Records without the date sort last.
				if av == noneKey || bv == noneKey {
					return bv == noneKey
				}
				return av < bv
			}
		}
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		return groupKey(a, dims) < groupKey(b, dims)
	})
	return res
}

// Collect drains a paginated fetch until every hit is retrieved. It
// aborts — rather than silently truncating — when the result set
// exceeds cap, so a statistic is either complete or absent.
func Collect(fetchPage func(page int) ([]Record, int, error), cap int) ([]Record, error) {
	var records []Record
	for page := 0; ; page++ {
		batch, hits, err := fetchPage(page)
		if err != nil {
			return nil, err
		}
		if hits > cap {
			return nil, fmt.Errorf("result set has %d records, exceeding the %d limit — narrow the selection (e.g. --status, --created-after, or a project ID)", hits, cap)
		}
		records = append(records, batch...)
		if len(records) >= hits || len(batch) == 0 {
			return records, nil
		}
	}
}

// groupKey gives a deterministic identity for a group, used for
// tie-breaking equal counts.
func groupKey(g Group, dims []Dimension) string {
	parts := make([]string, len(dims))
	for i, d := range dims {
		parts[i] = g.Keys[d.Name]
	}
	return strings.Join(parts, "\x00")
}
