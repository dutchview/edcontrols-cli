package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/mauricejumelet/edcontrols-cli/internal/api"
)

type TicketsCmd struct {
	List   TicketsListCmd   `cmd:"" help:"List tickets"`
	Get    TicketsGetCmd    `cmd:"" help:"Get ticket details"`
	Assign TicketsAssignCmd `cmd:"" help:"Assign a ticket to someone"`
	Open   TicketsOpenCmd   `cmd:"" help:"Open a ticket (set status to Open)"`
	Close  TicketsCloseCmd  `cmd:"" help:"Close a ticket (set status to Done)"`
}

type TicketsListCmd struct {
	Database    string `arg:"" optional:"" help:"Project database name (omit to search all active projects)"`
	Status      string `short:"s" help:"Filter by status (Open, In Progress, Done)"`
	Search      string `help:"Search by title"`
	Responsible string `short:"r" help:"Filter by responsible person email"`
	Tag         string `short:"t" help:"Filter by tag"`
	GroupID     string `short:"g" help:"Filter by group ID"`
	Archived    bool   `short:"a" help:"Include archived tickets"`
	AllProjects bool   `help:"Include inactive projects when searching all"`
	Limit       int    `short:"l" default:"50" help:"Maximum number of tickets to return"`
	Page        int    `short:"p" default:"0" help:"Page number (0-based)"`
	Sort        string `short:"o" default:"created" enum:"created,modified" help:"Sort by field (created, modified)"`
	Asc         bool   `help:"Sort in ascending order (oldest first)"`
	JSON        bool   `short:"j" help:"Output as JSON"`
}

func (c *TicketsListCmd) Run(client *api.Client) error {
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

			// Track which project each ticket belongs to
			for _, t := range tickets {
				ticketProjects[t.CouchDbID] = project.ProjectID
			}
			allTickets = append(allTickets, tickets...)

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
	Database string `short:"d" help:"Project database name (optional, will search if not provided)"`
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

	ticket, err := client.GetTicket(database, ticketID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(ticket)
	}

	title := "-"
	if ticket.Content != nil && ticket.Content.Title != "" {
		title = ticket.Content.Title
	}
	fmt.Printf("Ticket: %s\n", title)
	fmt.Printf("ID: %s (%s)\n", humanID(ticketID), ticketID)
	fmt.Printf("Project: %s\n", database)

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
	Database    string `arg:"" help:"Project database name"`
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
	Database string `arg:"" help:"Project database name"`
	TicketID string `arg:"" help:"Ticket ID"`
}

func (c *TicketsOpenCmd) Run(client *api.Client) error {
	status := "Open"
	opts := api.UpdateTicketOptions{
		Status: &status,
	}

	if err := client.UpdateTicket(c.Database, c.TicketID, opts); err != nil {
		return err
	}

	fmt.Printf("Ticket %s opened\n", c.TicketID)
	return nil
}

type TicketsCloseCmd struct {
	Database string `arg:"" help:"Project database name"`
	TicketID string `arg:"" help:"Ticket ID"`
}

func (c *TicketsCloseCmd) Run(client *api.Client) error {
	status := "Done"
	opts := api.UpdateTicketOptions{
		Status: &status,
	}

	if err := client.UpdateTicket(c.Database, c.TicketID, opts); err != nil {
		return err
	}

	fmt.Printf("Ticket %s closed\n", c.TicketID)
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
