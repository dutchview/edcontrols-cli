package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/dutchview/edcontrols-cli/internal/api"
)

type TicketsCmd struct {
	List      TicketsListCmd      `cmd:"" help:"List tickets"`
	Get       TicketsGetCmd       `cmd:"" help:"Get ticket details"`
	Update    TicketsUpdateCmd    `cmd:"" help:"Update ticket fields (-t title, -d description, --due-date, --clear-due, -r responsible, --clear-responsible, --complete, -m comment)"`
	Assign    TicketsAssignCmd    `cmd:"" help:"Assign a ticket to someone"`
	Open      TicketsOpenCmd      `cmd:"" help:"Reopen a ticket (set status to created)"`
	Close     TicketsCloseCmd     `cmd:"" help:"Close a ticket (set status to completed)"`
	Archive   TicketsArchiveCmd   `cmd:"" help:"Archive a ticket"`
	Unarchive TicketsUnarchiveCmd `cmd:"" help:"Unarchive a ticket"`
	Delete    TicketsDeleteCmd    `cmd:"" help:"Delete a ticket"`
}

type TicketsListCmd struct {
	Database       string `arg:"" name:"project-id" optional:"" help:"Project ID (omit to search all active projects)"`
	Status         string `short:"s" enum:"created,started,completed," default:"" help:"Filter by status (created, started, completed)"`
	Search         string `help:"Search by title"`
	Responsible    string `short:"r" help:"Filter by responsible person email"`
	Tag            string `short:"t" help:"Filter by tag"`
	GroupID        string `short:"g" help:"Filter by group ID"`
	Archived       bool   `short:"a" help:"Include archived tickets"`
	AllProjects    bool   `help:"Include inactive projects when searching all"`
	Limit          int    `short:"l" default:"50" help:"Maximum number of tickets to return"`
	Page           int    `short:"p" default:"0" help:"Page number (0-based)"`
	Sort           string `short:"o" default:"created" enum:"created,modified" help:"Sort by field (created, modified)"`
	Asc            bool   `help:"Sort in ascending order (oldest first)"`
	JSON           bool   `short:"j" help:"Output as JSON"`
	CreatedAfter   string `help:"Show tickets created after this time (e.g., 2w, 3d, 1mo, 1y, or 2026-01-15)"`
	CreatedBefore  string `help:"Show tickets created before this time (e.g., 2w, 3d, 1mo, 1y, or 2026-01-15)"`
	ModifiedAfter  string `help:"Show tickets modified after this time (e.g., 2w, 3d, 1mo, 1y, or 2026-01-15)"`
	ModifiedBefore string `help:"Show tickets modified before this time (e.g., 2w, 3d, 1mo, 1y, or 2026-01-15)"`
}

func (c *TicketsListCmd) Run(client *api.Client) error {
	// Parse date filters
	var filters DateFilterSet
	if c.CreatedAfter != "" {
		t, err := ParseRelativeTime(c.CreatedAfter)
		if err != nil {
			return fmt.Errorf("--created-after: %w", err)
		}
		filters.CreatedAfter = &t
	}
	if c.CreatedBefore != "" {
		t, err := ParseRelativeTime(c.CreatedBefore)
		if err != nil {
			return fmt.Errorf("--created-before: %w", err)
		}
		filters.CreatedBefore = &t
	}
	if c.ModifiedAfter != "" {
		t, err := ParseRelativeTime(c.ModifiedAfter)
		if err != nil {
			return fmt.Errorf("--modified-after: %w", err)
		}
		filters.ModifiedAfter = &t
	}
	if c.ModifiedBefore != "" {
		t, err := ParseRelativeTime(c.ModifiedBefore)
		if err != nil {
			return fmt.Errorf("--modified-before: %w", err)
		}
		filters.ModifiedBefore = &t
	}

	hasDateFilters := filters.HasDateFilters()

	var allTickets []api.Ticket
	var total int
	var limitReached bool
	var showProject bool
	projectNames := make(map[string]string)
	ticketProjects := make(map[string]string)

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
		if hasDateFilters {
			// Over-fetch and auto-page to fill the requested limit
			fetchSize := c.Limit * 3
			if fetchSize > 200 {
				fetchSize = 200
			}
			page := 0
			for {
				opts := api.ListTicketsOptions{
					Database:    c.Database,
					Status:      c.Status,
					SearchTitle: c.Search,
					Responsible: c.Responsible,
					Tag:         c.Tag,
					GroupID:     c.GroupID,
					Archived:    c.Archived,
					Size:        fetchSize,
					Page:        page,
					SortBy:      sortBy,
					SortOrder:   sortOrder,
				}
				tickets, _, err := client.ListTickets(opts)
				if err != nil {
					return err
				}
				for _, t := range tickets {
					created := ""
					modified := ""
					if t.Dates != nil {
						created = t.Dates.CreationDate
						modified = t.Dates.LastModified
					}
					if filters.MatchesDates(created, modified) {
						allTickets = append(allTickets, t)
						if len(allTickets) >= c.Limit {
							break
						}
					}
				}
				if len(allTickets) >= c.Limit || len(tickets) < fetchSize {
					break
				}
				page++
			}
			if len(allTickets) > c.Limit {
				allTickets = allTickets[:c.Limit]
			}
			total = len(allTickets)
			limitReached = len(allTickets) >= c.Limit
		} else {
			opts := api.ListTicketsOptions{
				Database:    c.Database,
				Status:      c.Status,
				SearchTitle: c.Search,
				Responsible: c.Responsible,
				Tag:         c.Tag,
				GroupID:     c.GroupID,
				Archived:    c.Archived,
				Size:        c.Limit,
				Page:        c.Page,
				SortBy:      sortBy,
				SortOrder:   sortOrder,
			}

			tickets, t, err := client.ListTickets(opts)
			if err != nil {
				return err
			}
			allTickets = tickets
			total = t
			limitReached = total > c.Limit
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

			opts := api.ListTicketsOptions{
				Database:    project.ProjectID,
				Status:      c.Status,
				SearchTitle: c.Search,
				Responsible: c.Responsible,
				Tag:         c.Tag,
				GroupID:     c.GroupID,
				Archived:    c.Archived,
				Size:        c.Limit,
				SortBy:      sortBy,
				SortOrder:   sortOrder,
			}

			tickets, _, err := client.ListTickets(opts)
			if err != nil {
				continue // Skip projects with errors
			}

			// Track which project each ticket belongs to and apply date filter
			for _, t := range tickets {
				if hasDateFilters {
					created := ""
					modified := ""
					if t.Dates != nil {
						created = t.Dates.CreationDate
						modified = t.Dates.LastModified
					}
					if !filters.MatchesDates(created, modified) {
						continue
					}
				}
				ticketProjects[t.CouchDbID] = project.ProjectID
				allTickets = append(allTickets, t)
			}

			// Stop if we have enough
			if len(allTickets) >= c.Limit {
				allTickets = allTickets[:c.Limit]
				limitReached = true
				break
			}
		}
		total = len(allTickets)
	}

	if c.JSON {
		return printJSON(allTickets)
	}

	if len(allTickets) == 0 {
		fmt.Println("No tickets found.")
		return nil
	}

	tickets := allTickets

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if showProject {
		fmt.Fprintln(w, "HUMAN_ID\tPROJECT\tTITLE\tSTATUS\tASSIGNED\tCREATED")
		fmt.Fprintln(w, "--------\t-------\t-----\t------\t--------\t-------")
	} else {
		fmt.Fprintln(w, "HUMAN_ID\tTITLE\tSTATUS\tASSIGNED\tCREATED\tDUE")
		fmt.Fprintln(w, "--------\t-----\t------\t--------\t-------\t---")
	}

	for _, ticket := range tickets {
		assigned := "-"
		if ticket.Participants != nil && ticket.Participants.Responsible != nil && ticket.Participants.Responsible.Email != "" {
			assigned = truncate(ticket.Participants.Responsible.Email, 25)
		}

		created := "-"
		if ticket.Dates != nil && ticket.Dates.CreationDate != "" && len(ticket.Dates.CreationDate) >= 10 {
			created = ticket.Dates.CreationDate[:10]
		}

		due := "-"
		if ticket.Dates != nil && ticket.Dates.DueDate != "" && len(ticket.Dates.DueDate) >= 10 {
			due = ticket.Dates.DueDate[:10]
		}

		title := "-"
		if ticket.Content != nil && ticket.Content.Title != "" {
			title = truncate(ticket.Content.Title, 40)
		}

		status := "-"
		if ticket.State != nil && ticket.State.State != "" {
			status = ticket.State.State
		}

		if showProject {
			projectName := truncate(projectNames[ticketProjects[ticket.CouchDbID]], 25)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", humanID(ticket.CouchDbID), projectName, title, status, assigned, created)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", humanID(ticket.CouchDbID), title, status, assigned, created, due)
		}
	}

	w.Flush()

	if limitReached {
		fmt.Printf("\nShowing %d tickets (limit reached). Use -l to show more, e.g.: ec tickets list -l 100\n", len(tickets))
	} else {
		fmt.Printf("\nTotal: %d tickets\n", total)
	}

	return nil
}

type TicketsGetCmd struct {
	TicketID string `arg:"" help:"Ticket ID (human ID like 'CC455B' or full CouchDB ID)"`
	Database string `short:"p" name:"project" help:"Project ID (optional, will search if not provided)"`
	JSON     bool   `short:"j" help:"Output as JSON"`
}

func (c *TicketsGetCmd) Run(client *api.Client) error {
	database := c.Database
	ticketID := c.TicketID

	// If the ticket ID looks like a human ID (6 chars or less), search for it
	isHumanID := len(c.TicketID) <= 6

	if isHumanID {
		// Search for the ticket by human ID
		var searchDB string
		if c.Database != "" {
			// Search only in specified database
			searchDB = c.Database
		}
		foundDB, foundID, err := findTicketByHumanID(client, c.TicketID, searchDB)
		if err != nil {
			return err
		}
		database = foundDB
		ticketID = foundID
	}

	if c.JSON {
		// Return raw securedata document for JSON output
		doc, err := client.GetDocument(database, ticketID)
		if err != nil {
			return err
		}
		return printJSON(doc)
	}

	ticket, err := client.GetTicket(database, ticketID)
	if err != nil {
		return err
	}

	title := "-"
	if ticket.Content != nil && ticket.Content.Title != "" {
		title = ticket.Content.Title
	}
	fmt.Printf("Ticket: %s\n", title)
	fmt.Printf("ID: %s (%s)\n", humanID(ticketID), ticketID)

	// Fetch project name
	project, err := client.GetProject(database)
	if err == nil && project.ProjectName != "" {
		fmt.Printf("Project: %s (%s)\n", project.ProjectName, database)
	} else {
		fmt.Printf("Project: %s\n", database)
	}

	// Fetch map and map group names if available
	if ticket.MapID != "" {
		if ticket.MapID == "EDGeomapMapID" {
			fmt.Printf("Map: Google Maps\n")
		} else {
			mapDoc, err := client.GetMap(database, ticket.MapID)
			if err == nil && mapDoc.Name != "" {
				fmt.Printf("Map: %s\n", mapDoc.Name)
			}
		}
	}

	if ticket.GroupID != "" {
		mapGroup, err := client.GetMapGroup(database, ticket.GroupID)
		if err == nil && mapGroup.Name != "" {
			fmt.Printf("Map Group: %s\n", mapGroup.Name)
		}
	}

	if ticket.State != nil && ticket.State.State != "" {
		fmt.Printf("Status: %s\n", ticket.State.State)
	}

	if ticket.Participants != nil && ticket.Participants.Responsible != nil && ticket.Participants.Responsible.Email != "" {
		fmt.Printf("Responsible: %s\n", ticket.Participants.Responsible.Email)
	}

	if ticket.Dates != nil {
		if ticket.Dates.DueDate != "" {
			fmt.Printf("Due: %s\n", ticket.Dates.DueDate)
		}
		if ticket.Dates.CreationDate != "" {
			fmt.Printf("Created: %s\n", ticket.Dates.CreationDate)
		}
		if ticket.Dates.LastModified != "" {
			fmt.Printf("Modified: %s\n", ticket.Dates.LastModified)
		}
	}

	if ticket.Content != nil && ticket.Content.Author != nil && ticket.Content.Author.Email != "" {
		fmt.Printf("Author: %s\n", ticket.Content.Author.Email)
	}

	if len(ticket.Tags) > 0 {
		fmt.Printf("Tags: %v\n", ticket.Tags)
	}

	if ticket.Content != nil && ticket.Content.Body != "" {
		fmt.Printf("\nDescription:\n%s\n", ticket.Content.Body)
	}

	return nil
}

type TicketsAssignCmd struct {
	Database    string `arg:"" name:"project-id" help:"Project ID"`
	TicketID    string `arg:"" help:"Ticket ID"`
	Responsible string `arg:"" help:"Email of the person to assign"`
}

func (c *TicketsAssignCmd) Run(client *api.Client) error {
	opts := api.UpdateTicketOptions{
		Responsible: &c.Responsible,
	}

	if err := client.UpdateTicket(c.Database, c.TicketID, opts); err != nil {
		return err
	}

	fmt.Printf("Ticket %s assigned to %s\n", c.TicketID, c.Responsible)
	return nil
}

type TicketsOpenCmd struct {
	Database string `arg:"" name:"project-id" help:"Project ID"`
	TicketID string `arg:"" help:"Ticket ID"`
}

func (c *TicketsOpenCmd) Run(client *api.Client) error {
	status := "created"
	opts := api.UpdateTicketOptions{
		Status: &status,
	}

	if err := client.UpdateTicket(c.Database, c.TicketID, opts); err != nil {
		return err
	}

	fmt.Printf("Ticket %s reopened (status: created)\n", c.TicketID)
	return nil
}

type TicketsCloseCmd struct {
	Database string `arg:"" name:"project-id" help:"Project ID"`
	TicketID string `arg:"" help:"Ticket ID"`
}

func (c *TicketsCloseCmd) Run(client *api.Client) error {
	status := "completed"
	opts := api.UpdateTicketOptions{
		Status: &status,
	}

	if err := client.UpdateTicket(c.Database, c.TicketID, opts); err != nil {
		return err
	}

	fmt.Printf("Ticket %s closed (status: completed)\n", c.TicketID)
	return nil
}

// findTicketByHumanID searches for a ticket matching the given human ID.
// If limitToDatabase is not empty, only searches that database.
// Returns the database name and full CouchDB ID of the ticket.
func findTicketByHumanID(client *api.Client, searchID string, limitToDatabase string) (string, string, error) {
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
	// The API searches for IDs ending with the value or its reverse (case-insensitive)
	// Human ID is reversed last 6 chars, so we search with lowercase to match
	tickets, err := client.SearchTicketsByID(projectIDs, strings.ToLower(searchID))
	if err != nil {
		return "", "", err
	}

	// Verify the human ID matches and extract the database from the ticket
	for _, ticket := range tickets {
		if humanID(ticket.CouchDbID) == searchID {
			// Extract database from the ticket's ID field (format: database|couchDbId)
			if ticket.ID != "" && strings.Contains(ticket.ID, "|") {
				parts := strings.SplitN(ticket.ID, "|", 2)
				return parts[0], ticket.CouchDbID, nil
			}
			// Fallback: search each project to find where this ticket exists
			for _, projectID := range projectIDs {
				_, err := client.GetTicket(projectID, ticket.CouchDbID)
				if err == nil {
					return projectID, ticket.CouchDbID, nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("ticket with ID %s not found", searchID)
}

type TicketsUpdateCmd struct {
	Database         string `arg:"" name:"project-id" help:"Project ID"`
	TicketID         string `arg:"" help:"Ticket ID"`
	Title            string `short:"t" help:"New title for the ticket"`
	Description      string `short:"d" help:"New description for the ticket"`
	DueDate          string `help:"Due date (ISO 8601 format, e.g., 2026-03-15T12:00:00.000Z)"`
	ClearDue         bool   `help:"Clear the due date"`
	Responsible      string `short:"r" help:"Assign to this email (also sets status to started)"`
	ClearResponsible bool   `help:"Clear the responsible person (sets status back to created)"`
	Complete         bool   `help:"Mark ticket as completed (uses existing responsible or current user)"`
	Comment          string `short:"m" help:"Add a comment to the ticket"`
}

func (c *TicketsUpdateCmd) Run(client *api.Client) error {
	// Build update options
	opts := api.UpdateTicketFieldsOptions{
		ClearDue:         c.ClearDue,
		ClearResponsible: c.ClearResponsible,
		Complete:         c.Complete,
	}

	if c.Title != "" {
		opts.Title = &c.Title
	}
	if c.Description != "" {
		// Sanitize HTML to prevent XSS attacks
		sanitized := sanitizeHTML(c.Description)
		opts.Description = &sanitized
	}
	if c.DueDate != "" {
		opts.DueDate = &c.DueDate
	}
	if c.Responsible != "" {
		opts.Responsible = &c.Responsible
	}
	if c.Comment != "" {
		// Sanitize HTML in comment to prevent XSS
		sanitized := sanitizeHTML(c.Comment)
		opts.Comment = &sanitized
	}

	// If no updates specified, show current values
	if opts.Title == nil && opts.Description == nil && opts.DueDate == nil && !opts.ClearDue && opts.Responsible == nil && !opts.ClearResponsible && !opts.Complete && opts.Comment == nil {
		ticket, err := client.GetTicket(c.Database, c.TicketID)
		if err != nil {
			return fmt.Errorf("getting ticket: %w", err)
		}

		title := "-"
		description := "-"
		dueDate := "-"
		responsible := "-"
		status := "-"

		if ticket.Content != nil && ticket.Content.Title != "" {
			title = ticket.Content.Title
		}
		if ticket.Content != nil && ticket.Content.Body != "" {
			description = ticket.Content.Body
		}
		if ticket.Dates != nil && ticket.Dates.DueDate != "" {
			dueDate = ticket.Dates.DueDate
		} else {
			// Check plan.dueDate via raw document
			dd, _ := client.GetTicketDueDate(c.Database, c.TicketID)
			if dd != "" {
				dueDate = dd
			}
		}
		if ticket.Participants != nil && ticket.Participants.Responsible != nil && ticket.Participants.Responsible.Email != "" {
			responsible = ticket.Participants.Responsible.Email
		}
		if ticket.State != nil && ticket.State.State != "" {
			status = ticket.State.State
		}

		fmt.Printf("Title: %s\n", title)
		fmt.Printf("Description: %s\n", description)
		fmt.Printf("Due date: %s\n", dueDate)
		fmt.Printf("Responsible: %s\n", responsible)
		fmt.Printf("Status: %s\n", status)
		return nil
	}

	if err := client.UpdateTicketFields(c.Database, c.TicketID, opts); err != nil {
		return fmt.Errorf("updating ticket: %w", err)
	}

	// Report what was updated
	var updates []string
	if opts.Title != nil {
		updates = append(updates, fmt.Sprintf("title=%q", *opts.Title))
	}
	if opts.Description != nil {
		updates = append(updates, fmt.Sprintf("description=%q", truncate(*opts.Description, 50)))
	}
	if opts.DueDate != nil {
		updates = append(updates, fmt.Sprintf("due-date=%s", *opts.DueDate))
	}
	if opts.ClearDue {
		updates = append(updates, "due-date cleared")
	}
	if opts.Responsible != nil && opts.Complete {
		updates = append(updates, fmt.Sprintf("responsible=%s (status->completed)", *opts.Responsible))
	} else if opts.Responsible != nil {
		updates = append(updates, fmt.Sprintf("responsible=%s (status->started)", *opts.Responsible))
	} else if opts.Complete {
		updates = append(updates, "status->completed")
	}
	if opts.ClearResponsible {
		updates = append(updates, "responsible cleared (status->created)")
	}
	if opts.Comment != nil {
		updates = append(updates, fmt.Sprintf("comment added: %q", truncate(*opts.Comment, 50)))
	}

	fmt.Printf("Ticket %s updated: %s\n", c.TicketID, strings.Join(updates, ", "))
	return nil
}

type TicketsArchiveCmd struct {
	Database string `arg:"" name:"project-id" help:"Project ID"`
	TicketID string `arg:"" help:"Ticket ID"`
}

func (c *TicketsArchiveCmd) Run(client *api.Client) error {
	if err := client.ArchiveTicket(c.Database, c.TicketID, true); err != nil {
		return fmt.Errorf("archiving ticket: %w", err)
	}
	fmt.Printf("Ticket %s archived.\n", c.TicketID)
	return nil
}

type TicketsUnarchiveCmd struct {
	Database string `arg:"" name:"project-id" help:"Project ID"`
	TicketID string `arg:"" help:"Ticket ID"`
}

func (c *TicketsUnarchiveCmd) Run(client *api.Client) error {
	if err := client.ArchiveTicket(c.Database, c.TicketID, false); err != nil {
		return fmt.Errorf("unarchiving ticket: %w", err)
	}
	fmt.Printf("Ticket %s unarchived.\n", c.TicketID)
	return nil
}

type TicketsDeleteCmd struct {
	Database string `arg:"" name:"project-id" help:"Project ID"`
	TicketID string `arg:"" help:"Ticket ID"`
}

func (c *TicketsDeleteCmd) Run(client *api.Client) error {
	if err := client.DeleteTickets(c.Database, []string{c.TicketID}); err != nil {
		return fmt.Errorf("deleting ticket: %w", err)
	}
	fmt.Printf("Ticket %s deleted.\n", c.TicketID)
	return nil
}
