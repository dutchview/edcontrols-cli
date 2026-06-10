package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dutchview/edcontrols-cli/internal/api"
	"github.com/dutchview/edcontrols-cli/internal/stats"
)

// statsMaxRecords caps how many documents a stats query may aggregate;
// see docs/adr/0001-client-side-aggregation-for-stats.md.
const statsMaxRecords = 50000

// statsPageSize is the page size used when draining the search endpoint.
const statsPageSize = 500

type TicketsStatsCmd struct {
	Database       string `arg:"" name:"project-id" optional:"" help:"Project ID (omit to aggregate across all active projects)"`
	GroupBy        string `help:"Group counts by up to two dimensions, comma-separated (responsible, status, project, tag, author, created:week, created:month, completed:week, completed:month)"`
	Status         string `short:"s" enum:"created,started,completed," default:"" help:"Filter by status (created, started, completed)"`
	Search         string `help:"Filter by title search"`
	Responsible    string `short:"r" help:"Filter by responsible person email"`
	Tag            string `short:"t" help:"Filter by tag"`
	GroupID        string `short:"g" help:"Filter by group ID"`
	Archived       bool   `short:"a" help:"Include archived tickets"`
	AllProjects    bool   `help:"Include inactive projects when aggregating all"`
	Overdue        bool   `help:"Only count overdue tickets (due date passed, not completed)"`
	JSON           bool   `short:"j" help:"Output as JSON"`
	CreatedAfter   string `help:"Count tickets created after this time (e.g., 2w, 3d, 1mo, 1y, or 2026-01-15)"`
	CreatedBefore  string `help:"Count tickets created before this time (e.g., 2w, 3d, 1mo, 1y, or 2026-01-15)"`
	ModifiedAfter  string `help:"Count tickets modified after this time (e.g., 2w, 3d, 1mo, 1y, or 2026-01-15)"`
	ModifiedBefore string `help:"Count tickets modified before this time (e.g., 2w, 3d, 1mo, 1y, or 2026-01-15)"`
}

func (c *TicketsStatsCmd) Run(client *api.Client) error {
	var dims []stats.Dimension
	if c.GroupBy != "" {
		var err error
		dims, err = stats.ParseGroupBy(c.GroupBy, stats.Tickets)
		if err != nil {
			return err
		}
	}
	dateFilters, err := parseDateFilterSet(c.CreatedAfter, c.CreatedBefore, c.ModifiedAfter, c.ModifiedBefore)
	if err != nil {
		return err
	}
	projectIDs, projectNames, err := resolveStatsProjects(client, c.Database, c.AllProjects)
	if err != nil {
		return err
	}

	searchOpts := api.StatsSearchOptions{
		Projects:    projectIDs,
		Status:      c.Status,
		SearchTitle: c.Search,
		Assignee:    c.Responsible,
		Tag:         c.Tag,
		GroupID:     c.GroupID,
		Archived:    c.Archived,
		Size:        statsPageSize,
	}

	records, err := stats.Collect(func(page int) ([]stats.Record, int, error) {
		opts := searchOpts
		opts.Page = page
		tickets, hits, err := client.SearchTicketsStats(opts)
		if err != nil {
			return nil, 0, err
		}
		reportStatsProgress(page, len(tickets), hits, "tickets")
		recs := make([]stats.Record, 0, len(tickets))
		for _, t := range tickets {
			recs = append(recs, ticketRecord(t, projectNames))
		}
		return recs, hits, nil
	}, statsMaxRecords)
	if err != nil {
		return err
	}

	filter := stats.Filter{
		Status:      c.Status,
		Assignee:    c.Responsible,
		Tag:         c.Tag,
		GroupID:     c.GroupID,
		OverdueOnly: c.Overdue,
	}
	res := stats.Aggregate(filterRecords(records, filter, dateFilters), dims, time.Now())

	if c.JSON {
		return printJSON(res)
	}
	stats.RenderTable(os.Stdout, res)
	return nil
}

type AuditsStatsCmd struct {
	Database       string `arg:"" name:"project-id" optional:"" help:"Project ID (omit to aggregate across all active projects)"`
	GroupBy        string `help:"Group counts by up to two dimensions, comma-separated (template, auditor, status, project, tag, author, created:week, created:month, completed:week, completed:month)"`
	Status         string `short:"s" enum:"started,In Progress,completed," default:"" help:"Filter by status (started, In Progress, completed)"`
	Template       string `short:"t" help:"Filter by template ID"`
	Search         string `help:"Filter by title search"`
	Auditor        string `short:"a" help:"Filter by auditor email"`
	GroupID        string `short:"g" help:"Filter by group ID"`
	Tag            string `help:"Filter by tag"`
	Archived       bool   `help:"Include archived audits"`
	AllProjects    bool   `help:"Include inactive projects when aggregating all"`
	Overdue        bool   `help:"Only count overdue audits (due date passed, not completed)"`
	JSON           bool   `short:"j" help:"Output as JSON"`
	CreatedAfter   string `help:"Count audits created after this time (e.g., 2w, 3d, 1mo, 1y, or 2026-01-15)"`
	CreatedBefore  string `help:"Count audits created before this time (e.g., 2w, 3d, 1mo, 1y, or 2026-01-15)"`
	ModifiedAfter  string `help:"Count audits modified after this time (e.g., 2w, 3d, 1mo, 1y, or 2026-01-15)"`
	ModifiedBefore string `help:"Count audits modified before this time (e.g., 2w, 3d, 1mo, 1y, or 2026-01-15)"`
}

func (c *AuditsStatsCmd) Run(client *api.Client) error {
	var dims []stats.Dimension
	if c.GroupBy != "" {
		var err error
		dims, err = stats.ParseGroupBy(c.GroupBy, stats.Audits)
		if err != nil {
			return err
		}
	}
	dateFilters, err := parseDateFilterSet(c.CreatedAfter, c.CreatedBefore, c.ModifiedAfter, c.ModifiedBefore)
	if err != nil {
		return err
	}
	projectIDs, projectNames, err := resolveStatsProjects(client, c.Database, c.AllProjects)
	if err != nil {
		return err
	}

	searchOpts := api.StatsSearchOptions{
		Projects:    projectIDs,
		Status:      c.Status,
		SearchTitle: c.Search,
		Assignee:    c.Auditor,
		Tag:         c.Tag,
		GroupID:     c.GroupID,
		Archived:    c.Archived,
		Size:        statsPageSize,
	}

	records, err := stats.Collect(func(page int) ([]stats.Record, int, error) {
		opts := searchOpts
		opts.Page = page
		audits, hits, err := client.SearchAuditsStats(opts)
		if err != nil {
			return nil, 0, err
		}
		reportStatsProgress(page, len(audits), hits, "audits")
		recs := make([]stats.Record, 0, len(audits))
		for _, a := range audits {
			r := auditRecord(a, projectNames)
			// The template filter matches IDs, which the record does
			// not carry once resolved to a name, so filter here.
			if c.Template != "" && a.Template != c.Template && a.TemplateID != c.Template {
				continue
			}
			recs = append(recs, r)
		}
		return recs, hits, nil
	}, statsMaxRecords)
	if err != nil {
		return err
	}

	filter := stats.Filter{
		Status:      c.Status,
		Assignee:    c.Auditor,
		Tag:         c.Tag,
		GroupID:     c.GroupID,
		OverdueOnly: c.Overdue,
	}
	res := stats.Aggregate(filterRecords(records, filter, dateFilters), dims, time.Now())

	if c.JSON {
		return printJSON(res)
	}
	stats.RenderTable(os.Stdout, res)
	return nil
}

// parseDateFilterSet parses the four date-filter flags shared by list
// and stats commands.
func parseDateFilterSet(createdAfter, createdBefore, modifiedAfter, modifiedBefore string) (DateFilterSet, error) {
	var filters DateFilterSet
	for _, f := range []struct {
		value string
		flag  string
		dest  **time.Time
	}{
		{createdAfter, "--created-after", &filters.CreatedAfter},
		{createdBefore, "--created-before", &filters.CreatedBefore},
		{modifiedAfter, "--modified-after", &filters.ModifiedAfter},
		{modifiedBefore, "--modified-before", &filters.ModifiedBefore},
	} {
		if f.value == "" {
			continue
		}
		t, err := ParseRelativeTime(f.value)
		if err != nil {
			return filters, fmt.Errorf("%s: %w", f.flag, err)
		}
		*f.dest = &t
	}
	return filters, nil
}

// resolveStatsProjects returns the project IDs to aggregate over and a
// projectID -> name map for the project dimension.
func resolveStatsProjects(client *api.Client, database string, allProjects bool) ([]string, map[string]string, error) {
	projects, _, err := client.ListProjects(api.ListProjectsOptions{})
	if err != nil {
		return nil, nil, err
	}

	names := make(map[string]string, len(projects))
	for _, p := range projects {
		names[p.ProjectID] = p.ProjectName
	}

	if database != "" {
		return []string{database}, names, nil
	}

	var ids []string
	for _, p := range projects {
		if p.ProjectID == "glacier_project_documents" {
			continue
		}
		if !p.IsActive && !allProjects {
			continue
		}
		ids = append(ids, p.ProjectID)
	}
	return ids, names, nil
}

// filterRecords applies the client-side filters and date range filters.
func filterRecords(records []stats.Record, filter stats.Filter, dates DateFilterSet) []stats.Record {
	now := time.Now()
	hasDates := dates.HasDateFilters()
	out := records[:0]
	for _, r := range records {
		if !filter.Matches(r, now) {
			continue
		}
		if hasDates && !dates.MatchesDates(r.Created, r.Modified) {
			continue
		}
		out = append(out, r)
	}
	return out
}

// reportStatsProgress keeps the user informed on stderr during
// multi-page fetches; stdout stays clean for the table or JSON.
func reportStatsProgress(page, batch, hits int, entity string) {
	if hits <= statsPageSize {
		return
	}
	fetched := page*statsPageSize + batch
	if fetched > hits {
		fetched = hits
	}
	fmt.Fprintf(os.Stderr, "Fetched %d/%d %s...\n", fetched, hits, entity)
}

// recordDatabase extracts the project database from a search result,
// whose ID field has the form "database|couchDbId".
func recordDatabase(id, database string) string {
	if database != "" {
		return database
	}
	if db, _, ok := strings.Cut(id, "|"); ok {
		return db
	}
	return ""
}

// projectLabel resolves a database ID to its display name.
func projectLabel(db string, names map[string]string) string {
	if name, ok := names[db]; ok && name != "" {
		return name
	}
	return db
}

// ticketRecord converts an API ticket to a stats record.
func ticketRecord(t api.Ticket, projectNames map[string]string) stats.Record {
	r := stats.Record{
		Project: projectLabel(recordDatabase(t.ID, t.Database), projectNames),
		GroupID: t.GroupID,
		Tags:    t.Tags,
	}
	if t.State != nil {
		r.Status = t.State.State
	}
	if t.Participants != nil && t.Participants.Responsible != nil {
		r.Assignee = t.Participants.Responsible.Email
	}
	if t.Content != nil && t.Content.Author != nil {
		r.Author = t.Content.Author.Email
	}
	if t.Dates != nil {
		r.Created = t.Dates.CreationDate
		r.Modified = t.Dates.LastModified
		r.Completed = t.Dates.CompletionDate
		r.Due = t.Dates.DueDate
	}
	return r
}

// auditRecord converts an API audit to a stats record.
func auditRecord(a api.Audit, projectNames map[string]string) stats.Record {
	r := stats.Record{
		Status:  a.Status,
		Project: projectLabel(recordDatabase(a.ID, a.Database), projectNames),
		GroupID: a.GroupID,
		Tags:    a.Tags,
	}
	switch {
	case a.TemplateName != "":
		r.Template = a.TemplateName
	case a.Template != "":
		r.Template = a.Template
	default:
		r.Template = a.TemplateID
	}
	if a.Participants != nil && a.Participants.Responsible != nil {
		r.Assignee = a.Participants.Responsible.Email
	}
	if a.Author != nil {
		r.Author = a.Author.Email
	}
	if a.Dates != nil {
		r.Created = a.Dates.CreationDate
		r.Modified = a.Dates.LastModified
		r.Completed = a.Dates.CompletionDate
		r.Due = a.Dates.DueDate
	}
	return r
}
