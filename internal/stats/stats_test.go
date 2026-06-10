package stats

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

var testNow = time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

func TestAggregateTwoDimensions(t *testing.T) {
	records := []Record{
		{Project: "Noordtoren", Status: "created"},
		{Project: "Noordtoren", Status: "created"},
		{Project: "Noordtoren", Status: "started"},
		{Project: "Stationsplein", Status: "created"},
	}
	dims, err := ParseGroupBy("project,status", Tickets)
	if err != nil {
		t.Fatalf("ParseGroupBy: %v", err)
	}

	res := Aggregate(records, dims, testNow)

	if want := []string{"project", "status"}; res.GroupBy[0] != want[0] || res.GroupBy[1] != want[1] {
		t.Errorf("GroupBy = %v, want %v", res.GroupBy, want)
	}
	if len(res.Groups) != 3 {
		t.Fatalf("got %d groups, want 3", len(res.Groups))
	}
	g := res.Groups[0]
	if g.Keys["project"] != "Noordtoren" || g.Keys["status"] != "created" || g.Count != 2 {
		t.Errorf("top group = %v count %d, want Noordtoren/created count 2", g.Keys, g.Count)
	}
}

func TestAggregateByTagCountsOncePerTag(t *testing.T) {
	records := []Record{
		{Tags: []string{"brandveiligheid", "E-installatie"}},
		{Tags: nil}, // untagged -> (none)
		{Tags: []string{"brandveiligheid"}},
	}
	dims, _ := ParseGroupBy("tag", Tickets)

	res := Aggregate(records, dims, testNow)

	// Total counts records, not tag occurrences.
	if res.Total != 3 {
		t.Errorf("Total = %d, want 3", res.Total)
	}
	counts := map[string]int{}
	for _, g := range res.Groups {
		counts[g.Keys["tag"]] = g.Count
	}
	want := map[string]int{"brandveiligheid": 2, "E-installatie": 1, "(none)": 1}
	for k, n := range want {
		if counts[k] != n {
			t.Errorf("tag %q count = %d, want %d", k, counts[k], n)
		}
	}
}

func TestAggregateMissingValueBecomesNone(t *testing.T) {
	records := []Record{{Assignee: ""}}
	dims, _ := ParseGroupBy("responsible", Tickets)

	res := Aggregate(records, dims, testNow)

	if res.Groups[0].Keys["responsible"] != "(none)" {
		t.Errorf("key = %q, want (none)", res.Groups[0].Keys["responsible"])
	}
}

func TestAggregateByCreatedMonthSortsChronologically(t *testing.T) {
	records := []Record{
		{Created: "2026-05-03T10:00:00.000Z"},
		{Created: "2026-05-20T10:00:00.000Z"},
		{Created: "2026-03-15T10:00:00.000Z"},
	}
	dims, _ := ParseGroupBy("created:month", Tickets)

	res := Aggregate(records, dims, testNow)

	if len(res.Groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(res.Groups))
	}
	// Chronological, even though 2026-05 has the higher count.
	if res.Groups[0].Keys["created:month"] != "2026-03" || res.Groups[0].Count != 1 {
		t.Errorf("first group = %v count %d, want 2026-03 count 1", res.Groups[0].Keys, res.Groups[0].Count)
	}
	if res.Groups[1].Keys["created:month"] != "2026-05" || res.Groups[1].Count != 2 {
		t.Errorf("second group = %v count %d, want 2026-05 count 2", res.Groups[1].Keys, res.Groups[1].Count)
	}
}

func TestAggregateByCompletedWeekUsesISOWeek(t *testing.T) {
	records := []Record{
		// 2026-01-01 falls in ISO week 2026-W01
		{Completed: "2026-01-01T10:00:00.000Z"},
		{Completed: ""}, // never completed -> (none)
	}
	dims, _ := ParseGroupBy("completed:week", Tickets)

	res := Aggregate(records, dims, testNow)

	counts := map[string]int{}
	for _, g := range res.Groups {
		counts[g.Keys["completed:week"]] = g.Count
	}
	if counts["2026-W01"] != 1 {
		t.Errorf("2026-W01 count = %d, want 1 (got %v)", counts["2026-W01"], counts)
	}
	if counts["(none)"] != 1 {
		t.Errorf("(none) count = %d, want 1", counts["(none)"])
	}
}

func TestParseGroupByValidation(t *testing.T) {
	cases := []struct {
		spec    string
		entity  Entity
		wantErr bool
	}{
		{"status", Tickets, false},
		{"responsible", Tickets, false},
		{"tag,author", Tickets, false},
		{"created:week", Tickets, false},
		{"completed:month", Tickets, false},
		{"banana", Tickets, true},            // unknown dimension
		{"status,tag,author", Tickets, true}, // more than two dimensions
		{"created:day", Tickets, true},       // unsupported bucket
		{"created", Tickets, true},           // date field requires a bucket
		{"template", Tickets, true},          // audit-only dimension
		{"auditor", Tickets, true},           // audit-only dimension
		{"template,status", Audits, false},
		{"auditor", Audits, false},
		{"responsible", Audits, true}, // ticket-only dimension
		{"", Tickets, true},           // empty spec
	}
	for _, c := range cases {
		_, err := ParseGroupBy(c.spec, c.entity)
		if (err != nil) != c.wantErr {
			t.Errorf("ParseGroupBy(%q, %v) err = %v, wantErr %v", c.spec, c.entity, err, c.wantErr)
		}
	}
}

func TestFilterMatches(t *testing.T) {
	r := Record{
		Status:   "created",
		Assignee: "Jan@Bouw.nl",
		Tags:     []string{"brandveiligheid"},
		Due:      "2026-05-01T12:00:00.000Z", // past due -> overdue
	}
	cases := []struct {
		name   string
		filter Filter
		want   bool
	}{
		{"empty filter matches", Filter{}, true},
		{"status match", Filter{Status: "created"}, true},
		{"status mismatch", Filter{Status: "completed"}, false},
		{"assignee case-insensitive", Filter{Assignee: "jan@bouw.nl"}, true},
		{"assignee mismatch", Filter{Assignee: "piet@bouw.nl"}, false},
		{"tag match", Filter{Tag: "brandveiligheid"}, true},
		{"tag mismatch", Filter{Tag: "E-installatie"}, false},
		{"overdue only matches", Filter{OverdueOnly: true}, true},
	}
	for _, c := range cases {
		if got := c.filter.Matches(r, testNow); got != c.want {
			t.Errorf("%s: Matches = %v, want %v", c.name, got, c.want)
		}
	}

	completed := Record{Status: "completed", Due: "2026-05-01T12:00:00.000Z"}
	if (Filter{OverdueOnly: true}).Matches(completed, testNow) {
		t.Error("completed record should not match overdue-only filter")
	}

	grouped := Record{GroupID: "g-1"}
	if !(Filter{GroupID: "g-1"}).Matches(grouped, testNow) {
		t.Error("matching group ID should pass")
	}
	if (Filter{GroupID: "g-2"}).Matches(grouped, testNow) {
		t.Error("different group ID should not pass")
	}
}

func TestCollectPaginatesUntilAllHitsFetched(t *testing.T) {
	// 5 records served in pages of 2.
	all := []Record{{Status: "a"}, {Status: "b"}, {Status: "c"}, {Status: "d"}, {Status: "e"}}
	var pagesFetched []int
	fetch := func(page int) ([]Record, int, error) {
		pagesFetched = append(pagesFetched, page)
		start := page * 2
		end := start + 2
		if end > len(all) {
			end = len(all)
		}
		return all[start:end], len(all), nil
	}

	got, err := Collect(fetch, 50000)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(got) != 5 {
		t.Errorf("collected %d records, want 5", len(got))
	}
	if len(pagesFetched) != 3 {
		t.Errorf("fetched pages %v, want 3 pages", pagesFetched)
	}
}

func TestCollectAbortsWhenCapExceeded(t *testing.T) {
	fetch := func(page int) ([]Record, int, error) {
		return []Record{{}, {}}, 100, nil // claims 100 total hits
	}

	_, err := Collect(fetch, 10)
	if err == nil {
		t.Fatal("Collect should error when hits exceed cap, got nil")
	}
	if !strings.Contains(err.Error(), "narrow") {
		t.Errorf("error should suggest narrowing the selection, got: %v", err)
	}
}

func TestRenderTable(t *testing.T) {
	records := []Record{
		{Assignee: "jan@bouw.nl", Status: "created", Due: "2026-05-01T12:00:00.000Z"},
		{Assignee: "jan@bouw.nl", Status: "created"},
		{Assignee: "piet@bouw.nl", Status: "completed"},
	}
	dims, _ := ParseGroupBy("responsible", Tickets)
	res := Aggregate(records, dims, testNow)

	var buf bytes.Buffer
	RenderTable(&buf, res)
	out := buf.String()

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 4 {
		t.Fatalf("got %d lines, want 4 (header, 2 groups, total):\n%s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "RESPONSIBLE") || !strings.Contains(lines[0], "COUNT") || !strings.Contains(lines[0], "OVERDUE") {
		t.Errorf("header = %q", lines[0])
	}
	if f := strings.Fields(lines[1]); len(f) != 3 || f[0] != "jan@bouw.nl" || f[1] != "2" || f[2] != "1" {
		t.Errorf("first row = %q, want jan@bouw.nl 2 1", lines[1])
	}
	if f := strings.Fields(lines[3]); f[0] != "TOTAL" || f[1] != "3" || f[2] != "1" {
		t.Errorf("total row = %q, want TOTAL 3 1", lines[3])
	}
}

func TestRenderTableNoDimensionsShowsOnlyTotals(t *testing.T) {
	res := Aggregate([]Record{{}, {}}, nil, testNow)

	var buf bytes.Buffer
	RenderTable(&buf, res)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")

	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2 (header, total):\n%s", len(lines), buf.String())
	}
	if f := strings.Fields(lines[1]); f[0] != "TOTAL" || f[1] != "2" {
		t.Errorf("total row = %q, want TOTAL 2 0", lines[1])
	}
}

func TestResultJSONShape(t *testing.T) {
	records := []Record{{Status: "created", Project: "Noordtoren"}}
	dims, _ := ParseGroupBy("project,status", Tickets)
	res := Aggregate(records, dims, testNow)

	data, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"total":1,"overdue":0,"groupBy":["project","status"],"groups":[{"keys":{"project":"Noordtoren","status":"created"},"count":1,"overdue":0}]}`
	if string(data) != want {
		t.Errorf("JSON = %s\nwant   %s", data, want)
	}
}

func TestAggregateCountsOverdue(t *testing.T) {
	records := []Record{
		// past due and not completed: overdue
		{Status: "created", Due: "2026-05-01T12:00:00.000Z"},
		// past due but completed: not overdue
		{Status: "completed", Due: "2026-05-01T12:00:00.000Z"},
		// future due: not overdue
		{Status: "created", Due: "2026-12-01T12:00:00.000Z"},
		// no due date: not overdue
		{Status: "created"},
	}
	dims, _ := ParseGroupBy("status", Tickets)

	res := Aggregate(records, dims, testNow)

	if res.Overdue != 1 {
		t.Errorf("Overdue = %d, want 1", res.Overdue)
	}
	if res.Groups[0].Keys["status"] != "created" || res.Groups[0].Overdue != 1 {
		t.Errorf("created group overdue = %d, want 1", res.Groups[0].Overdue)
	}
	if res.Groups[1].Overdue != 0 {
		t.Errorf("completed group overdue = %d, want 0", res.Groups[1].Overdue)
	}
}

func TestAggregateByStatus(t *testing.T) {
	records := []Record{
		{Status: "created"},
		{Status: "created"},
		{Status: "completed"},
	}
	dims, err := ParseGroupBy("status", Tickets)
	if err != nil {
		t.Fatalf("ParseGroupBy: %v", err)
	}

	res := Aggregate(records, dims, testNow)

	if res.Total != 3 {
		t.Errorf("Total = %d, want 3", res.Total)
	}
	if len(res.Groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(res.Groups))
	}
	if res.Groups[0].Keys["status"] != "created" || res.Groups[0].Count != 2 {
		t.Errorf("first group = %v count %d, want created count 2", res.Groups[0].Keys, res.Groups[0].Count)
	}
	if res.Groups[1].Keys["status"] != "completed" || res.Groups[1].Count != 1 {
		t.Errorf("second group = %v count %d, want completed count 1", res.Groups[1].Keys, res.Groups[1].Count)
	}
}
