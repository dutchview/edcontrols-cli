package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/mauricejumelet/edcontrols-cli/internal/api"
)

type AuditsCmd struct {
	List   AuditsListCmd   `cmd:"" help:"List audits"`
	Get    AuditsGetCmd    `cmd:"" help:"Get audit details"`
	Create AuditsCreateCmd `cmd:"" help:"Create an audit from a template"`
}

type AuditsListCmd struct {
	Database    string `arg:"" optional:"" help:"Project database name (omit to search all active projects)"`
	Status      string `short:"s" help:"Filter by status (comma-separated)"`
	Template    string `short:"t" help:"Filter by template ID"`
	Search      string `help:"Search by title"`
	Auditor     string `short:"a" help:"Filter by auditor email"`
	GroupID     string `short:"g" help:"Filter by group ID"`
	Tag         string `help:"Filter by tag"`
	Archived    bool   `help:"Include archived audits"`
	AllProjects bool   `help:"Include inactive projects when searching all"`
	Limit       int    `short:"l" default:"50" help:"Maximum number of audits to return"`
	Page        int    `short:"p" default:"0" help:"Page number (0-based)"`
	Sort        string `short:"o" default:"created" enum:"created,modified" help:"Sort by field (created, modified)"`
	Asc         bool   `help:"Sort in ascending order (oldest first)"`
	JSON        bool   `short:"j" help:"Output as JSON"`
}

func (c *AuditsListCmd) Run(client *api.Client) error {
	var allAudits []api.Audit
	var total int
	var limitReached bool
	var showProject bool
	templateNames := make(map[string]string)
	projectNames := make(map[string]string)
	auditProjects := make(map[string]string)

	// Convert sort option to API values
	sortBy := "CREATIONDATE"
	if c.Sort == "modified" {
		sortBy = "LASTMODIFIEDDATE"
	}
	sortOrder := "DESC"
	if c.Asc {
		sortOrder = "ASC"
	}

	if c.Database != "" {
		// Single project query
		opts := api.ListAuditsOptions{
			Database:    c.Database,
			Status:      c.Status,
			Template:    c.Template,
			SearchTitle: c.Search,
			Auditor:     c.Auditor,
			GroupID:     c.GroupID,
			Tag:         c.Tag,
			Archived:    c.Archived,
			Size:        c.Limit,
			Page:        c.Page,
			SortBy:      sortBy,
			SortOrder:   sortOrder,
		}

		audits, t, err := client.ListAudits(opts)
		if err != nil {
			return err
		}
		allAudits = audits
		total = t
		limitReached = total > c.Limit

		// Fetch templates for this project
		templates, _, err := client.ListAuditTemplates(api.ListAuditTemplatesOptions{
			Database: c.Database,
			Size:     500,
		})
		if err == nil {
			for _, t := range templates {
				templateNames[t.CouchDbID] = t.Name
			}
		}
	} else {
		// Query all active projects
		showProject = true
		projects, _, err := client.ListProjects(api.ListProjectsOptions{})
		if err != nil {
			return err
		}

		for _, project := range projects {
			// Skip glacier projects
			if project.ProjectID == "glacier_project_documents" {
				continue
			}
			// Skip inactive projects unless --all-projects is set
			if !project.IsActive && !c.AllProjects {
				continue
			}

			projectNames[project.ProjectID] = project.ProjectName

			opts := api.ListAuditsOptions{
				Database:    project.ProjectID,
				Status:      c.Status,
				Template:    c.Template,
				SearchTitle: c.Search,
				Auditor:     c.Auditor,
				GroupID:     c.GroupID,
				Tag:         c.Tag,
				Archived:    c.Archived,
				Size:        c.Limit,
				SortBy:      sortBy,
				SortOrder:   sortOrder,
			}

			audits, _, err := client.ListAudits(opts)
			if err != nil {
				continue // Skip projects with errors
			}

			// Track which project each audit belongs to
			for _, a := range audits {
				auditProjects[a.CouchDbID] = project.ProjectID
			}
			allAudits = append(allAudits, audits...)

			// Fetch templates for this project
			templates, _, err := client.ListAuditTemplates(api.ListAuditTemplatesOptions{
				Database: project.ProjectID,
				Size:     500,
			})
			if err == nil {
				for _, t := range templates {
					templateNames[t.CouchDbID] = t.Name
				}
			}

			// Stop if we have enough
			if len(allAudits) >= c.Limit {
				allAudits = allAudits[:c.Limit]
				limitReached = true
				break
			}
		}
		total = len(allAudits)
	}

	if c.JSON {
		return printJSON(allAudits)
	}

	if len(allAudits) == 0 {
		fmt.Println("No audits found.")
		return nil
	}

	audits := allAudits

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if showProject {
		fmt.Fprintln(w, "HUMAN_ID\tPROJECT\tNAME\tSTATUS\tCREATED\tTEMPLATE")
		fmt.Fprintln(w, "--------\t-------\t----\t------\t-------\t--------")
	} else {
		fmt.Fprintln(w, "HUMAN_ID\tNAME\tSTATUS\tASSIGNED\tCREATED\tTEMPLATE")
		fmt.Fprintln(w, "--------\t----\t------\t--------\t-------\t--------")
	}

	for _, audit := range audits {
		assigned := "-"
		if audit.Participants != nil && audit.Participants.Responsible != nil && audit.Participants.Responsible.Email != "" {
			assigned = truncate(audit.Participants.Responsible.Email, 25)
		}

		created := "-"
		if audit.Dates != nil && audit.Dates.CreationDate != "" && len(audit.Dates.CreationDate) >= 10 {
			created = audit.Dates.CreationDate[:10]
		}

		template := "-"
		if name, ok := templateNames[audit.Template]; ok {
			template = truncate(name, 30)
		} else if audit.TemplateName != "" {
			template = truncate(audit.TemplateName, 30)
		} else if audit.Template != "" {
			template = audit.Template
		}

		name := truncate(audit.Name, 40)
		if showProject {
			projectName := truncate(projectNames[auditProjects[audit.CouchDbID]], 25)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", humanID(audit.CouchDbID), projectName, name, statusString(audit.Status), created, template)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", humanID(audit.CouchDbID), name, statusString(audit.Status), assigned, created, template)
		}
	}

	w.Flush()

	if limitReached {
		fmt.Printf("\nShowing %d audits (limit reached). Use -l to show more, e.g.: ec audits list -l 100\n", len(audits))
	} else {
		fmt.Printf("\nTotal: %d audits\n", total)
	}

	return nil
}

type AuditsGetCmd struct {
	Database string `arg:"" help:"Project database name"`
	AuditID  string `arg:"" help:"Audit ID"`
	JSON     bool   `short:"j" help:"Output as JSON"`
}

func (c *AuditsGetCmd) Run(client *api.Client) error {
	audit, err := client.GetAudit(c.Database, c.AuditID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(audit)
	}

	fmt.Printf("Audit: %s\n", audit.Name)
	fmt.Printf("ID: %s\n", audit.CouchDbID)
	fmt.Printf("Status: %s\n", statusString(audit.Status))

	if audit.Template != "" {
		fmt.Printf("Template: %s\n", audit.Template)
	}
	if audit.Participants != nil && audit.Participants.Responsible != nil && audit.Participants.Responsible.Email != "" {
		fmt.Printf("Responsible: %s\n", audit.Participants.Responsible.Email)
	}
	if audit.Author != nil && audit.Author.Email != "" {
		fmt.Printf("Author: %s\n", audit.Author.Email)
	}
	if audit.Dates != nil {
		if audit.Dates.DueDate != "" {
			fmt.Printf("Due: %s\n", audit.Dates.DueDate)
		}
		if audit.Dates.CreationDate != "" {
			fmt.Printf("Created: %s\n", audit.Dates.CreationDate)
		}
		if audit.Dates.LastModified != "" {
			fmt.Printf("Modified: %s\n", audit.Dates.LastModified)
		}
		if audit.Dates.CompletionDate != "" {
			fmt.Printf("Completed: %s\n", audit.Dates.CompletionDate)
		}
	}
	if len(audit.Tags) > 0 {
		fmt.Printf("Tags: %v\n", audit.Tags)
	}

	return nil
}

type AuditsCreateCmd struct {
	Database    string   `arg:"" help:"Project database name"`
	TemplateID  string   `arg:"" help:"Audit template ID to use"`
	Name        string   `short:"n" help:"Audit name (optional, defaults to template name)"`
	Responsible string   `short:"r" help:"Responsible person email"`
	DueDate     string   `short:"d" help:"Due date (ISO 8601 format, e.g., 2025-12-31T23:59:59Z)"`
	Tags        []string `short:"t" help:"Tags to add (can be specified multiple times)"`
	JSON        bool     `short:"j" help:"Output as JSON"`
}

func (c *AuditsCreateCmd) Run(client *api.Client) error {
	opts := api.CreateAuditOptions{
		Name:        c.Name,
		Responsible: c.Responsible,
		DueDate:     c.DueDate,
		Tags:        c.Tags,
	}

	audit, err := client.CreateAudit(c.Database, c.TemplateID, opts)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(audit)
	}

	fmt.Printf("Audit created successfully!\n")
	fmt.Printf("ID: %s\n", audit.ID)
	fmt.Printf("Name: %s\n", audit.Name)
	fmt.Printf("Status: %s\n", statusString(audit.Status))

	return nil
}
