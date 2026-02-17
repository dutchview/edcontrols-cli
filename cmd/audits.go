package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/dutchview/edcontrols-cli/internal/api"
)

type AuditsCmd struct {
	List   AuditsListCmd   `cmd:"" help:"List audits"`
	Get    AuditsGetCmd    `cmd:"" help:"Get audit details"`
	Create AuditsCreateCmd `cmd:"" help:"Create an audit from a template"`
}

type AuditsListCmd struct {
	Database    string `arg:"" name:"project-id" optional:"" help:"Project ID (omit to search all active projects)"`
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
	AuditID  string `arg:"" help:"Audit ID (human ID like '708739' or full CouchDB ID)"`
	Database string `short:"p" name:"project" help:"Project ID (optional, will search if not provided)"`
	JSON     bool   `short:"j" help:"Output as JSON"`
}

func (c *AuditsGetCmd) Run(client *api.Client) error {
	database := c.Database
	auditID := c.AuditID

	// If the audit ID looks like a human ID (6 chars or less), search for it
	isHumanID := len(c.AuditID) <= 6

	if isHumanID {
		// Search for the audit by human ID
		var searchDB string
		if c.Database != "" {
			searchDB = c.Database
		}
		foundDB, foundID, err := findAuditByHumanID(client, c.AuditID, searchDB)
		if err != nil {
			return err
		}
		database = foundDB
		auditID = foundID
	}

	if c.JSON {
		// Return raw securedata document for JSON output
		doc, err := client.GetDocument(database, auditID)
		if err != nil {
			return err
		}
		return printJSON(doc)
	}

	audit, err := client.GetAudit(database, auditID)
	if err != nil {
		return err
	}

	fmt.Printf("Audit: %s\n", audit.Name)
	fmt.Printf("ID: %s (%s)\n", humanID(auditID), auditID)

	// Fetch project name
	project, err := client.GetProject(database)
	if err == nil && project.ProjectName != "" {
		fmt.Printf("Project: %s (%s)\n", project.ProjectName, database)
	} else {
		fmt.Printf("Project: %s\n", database)
	}

	// Fetch template name and template group name
	if audit.Template != "" {
		template, err := client.GetAuditTemplate(database, audit.Template)
		if err == nil {
			fmt.Printf("Template: %s\n", template.Name)

			// Fetch template group name if available
			if template.GroupID != "" {
				templateGroup, err := client.GetMapGroup(database, template.GroupID)
				if err == nil && templateGroup.Name != "" {
					fmt.Printf("Template Group: %s\n", templateGroup.Name)
				}
			}
		} else {
			fmt.Printf("Template: %s\n", audit.Template)
		}
	}

	fmt.Printf("Status: %s\n", statusString(audit.Status))

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

	// Display questions and answers
	if len(audit.Questions) > 0 {
		fmt.Printf("\n--- Questions & Answers ---\n")
		for _, category := range audit.Questions {
			fmt.Printf("\n[%s]\n", category.CategoryName)
			for _, q := range category.Questions {
				answerStr := formatAnswer(q.Answer)
				if answerStr != "" {
					fmt.Printf("  Q: %s\n", q.Question)
					fmt.Printf("  A: %s\n", answerStr)
				} else {
					fmt.Printf("  Q: %s\n", q.Question)
					fmt.Printf("  A: (no answer)\n")
				}
			}
		}
	}

	return nil
}

// formatAnswer converts an answer slice to a readable string
func formatAnswer(answer []interface{}) string {
	if len(answer) == 0 {
		return ""
	}

	var parts []string
	for _, a := range answer {
		switch v := a.(type) {
		case string:
			if v != "" {
				parts = append(parts, v)
			}
		case map[string]interface{}:
			// Handle complex answer objects (e.g., dates, files)
			if text, ok := v["text"].(string); ok && text != "" {
				parts = append(parts, text)
			} else if value, ok := v["value"].(string); ok && value != "" {
				parts = append(parts, value)
			} else if date, ok := v["date"].(string); ok && date != "" {
				parts = append(parts, date)
			}
		case bool:
			if v {
				parts = append(parts, "Yes")
			} else {
				parts = append(parts, "No")
			}
		case float64:
			parts = append(parts, fmt.Sprintf("%.0f", v))
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}

// findAuditByHumanID searches for an audit matching the given human ID.
// If limitToDatabase is not empty, only searches that database.
// Returns the database name and full CouchDB ID of the audit.
func findAuditByHumanID(client *api.Client, searchID string, limitToDatabase string) (string, string, error) {
	searchID = strings.ToUpper(searchID)

	var projectIDs []string

	if limitToDatabase != "" {
		projectIDs = []string{limitToDatabase}
	} else {
		projects, _, err := client.ListProjects(api.ListProjectsOptions{})
		if err != nil {
			return "", "", err
		}
		for _, project := range projects {
			// Skip glacier projects and inactive projects for faster search
			if project.ProjectID == "glacier_project_documents" || !project.IsActive {
				continue
			}
			projectIDs = append(projectIDs, project.ProjectID)
		}
	}

	// Use the POST search endpoint which supports searchById across multiple projects
	audits, err := client.SearchAuditsByID(projectIDs, strings.ToLower(searchID))
	if err != nil {
		return "", "", err
	}

	// Verify the human ID matches and extract the database from the audit
	for _, audit := range audits {
		if humanID(audit.CouchDbID) == searchID {
			// Extract database from the audit's ID field (format: database|couchDbId)
			if audit.ID != "" && strings.Contains(audit.ID, "|") {
				parts := strings.SplitN(audit.ID, "|", 2)
				return parts[0], audit.CouchDbID, nil
			}
			// Fallback: search each project to find where this audit exists
			for _, projectID := range projectIDs {
				_, err := client.GetAudit(projectID, audit.CouchDbID)
				if err == nil {
					return projectID, audit.CouchDbID, nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("audit with ID %s not found", searchID)
}

type AuditsCreateCmd struct {
	Database    string   `arg:"" name:"project-id" help:"Project ID"`
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
