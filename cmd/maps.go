package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/mauricejumelet/edcontrols-cli/internal/api"
)

type MapsCmd struct {
	List MapsListCmd `cmd:"" help:"List maps (drawings)"`
	Get  MapsGetCmd  `cmd:"" help:"Get map details"`
}

type MapsListCmd struct {
	Database string `arg:"" help:"Project database name (required)"`
	GroupID  string `short:"g" help:"Filter by map group ID"`
	Search   string `short:"s" help:"Search by name"`
	Tag      string `short:"t" help:"Filter by tag"`
	Archived bool   `short:"a" help:"Include archived maps"`
	AllMaps  bool   `help:"Show all maps (bypass role filtering)"`
	Limit    int    `short:"l" default:"50" help:"Maximum number of maps to return"`
	Page     int    `short:"p" default:"0" help:"Page number (0-based)"`
	Sort     string `short:"o" default:"created" enum:"created,modified,name" help:"Sort by field"`
	Asc      bool   `help:"Sort in ascending order"`
	JSON     bool   `short:"j" help:"Output as JSON"`
}

func (c *MapsListCmd) Run(client *api.Client) error {
	// Convert sort option to API values
	sortBy := "CREATIONDATE"
	switch c.Sort {
	case "modified":
		sortBy = "LASTMODIFIEDDATE"
	case "name":
		sortBy = "name"
	}
	sortOrder := "DESC"
	if c.Asc {
		sortOrder = "ASC"
	}

	opts := api.ListMapsOptions{
		Database:   c.Database,
		GroupID:    c.GroupID,
		SearchName: c.Search,
		Tag:        c.Tag,
		Archived:   c.Archived,
		AllMaps:    c.AllMaps,
		Size:       c.Limit,
		Page:       c.Page,
		SortBy:     sortBy,
		SortOrder:  sortOrder,
	}

	maps, total, err := client.ListMaps(opts)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(maps)
	}

	if len(maps) == 0 {
		fmt.Println("No maps found.")
		return nil
	}

	// Build a map of group names
	groupNames := make(map[string]string)
	for _, m := range maps {
		if m.GroupID != "" && groupNames[m.GroupID] == "" {
			if m.GroupName != "" {
				groupNames[m.GroupID] = m.GroupName
			} else {
				// Fetch group name
				group, err := client.GetMapGroup(c.Database, m.GroupID)
				if err == nil && group.Name != "" {
					groupNames[m.GroupID] = group.Name
				}
			}
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tGROUP\tCREATED\tMODIFIED")
	fmt.Fprintln(w, "--\t----\t-----\t-------\t--------")

	for _, m := range maps {
		mapID := m.CouchDbID
		if mapID == "" {
			mapID = m.CouchID
		}

		// Handle special Google Maps entry
		if mapID == "EDGeomapMapID" {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", "<google-maps>", "Google Maps", "-", "-", "-")
			continue
		}

		created := "-"
		if m.Dates != nil && m.Dates.CreationDate != "" && len(m.Dates.CreationDate) >= 10 {
			created = m.Dates.CreationDate[:10]
		}

		modified := "-"
		if m.Dates != nil && m.Dates.LastModified != "" && len(m.Dates.LastModified) >= 10 {
			modified = m.Dates.LastModified[:10]
		}

		groupName := "-"
		if name, ok := groupNames[m.GroupID]; ok && name != "" {
			groupName = truncate(name, 25)
		}

		name := truncate(m.Name, 40)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", mapID, name, groupName, created, modified)
	}

	w.Flush()

	limitReached := total > c.Limit
	if limitReached {
		fmt.Printf("\nShowing %d maps (limit reached). Use -l to show more, e.g.: ec maps list %s -l 100\n", len(maps), c.Database)
	} else {
		fmt.Printf("\nTotal: %d maps\n", total)
	}

	return nil
}

type MapsGetCmd struct {
	MapID    string `arg:"" help:"Map ID (full CouchDB ID)"`
	Database string `short:"d" help:"Project database name (optional, will search if not provided)"`
	JSON     bool   `short:"j" help:"Output as JSON"`
}

func (c *MapsGetCmd) Run(client *api.Client) error {
	database := c.Database
	mapID := c.MapID

	// If no database provided, search for the map across all projects
	if database == "" {
		foundDB, err := findMapByID(client, mapID)
		if err != nil {
			return err
		}
		database = foundDB
	}

	if c.JSON {
		// Return raw securedata document for JSON output
		doc, err := client.GetDocument(database, mapID)
		if err != nil {
			return err
		}
		return printJSON(doc)
	}

	m, err := client.GetMap(database, mapID)
	if err != nil {
		return err
	}

	fmt.Printf("Map: %s\n", m.Name)
	fmt.Printf("ID: %s\n", mapID)

	// Fetch project name
	project, err := client.GetProject(database)
	if err == nil && project.ProjectName != "" {
		fmt.Printf("Project: %s (%s)\n", project.ProjectName, database)
	} else {
		fmt.Printf("Project: %s\n", database)
	}

	// Fetch map group name
	if m.GroupID != "" {
		group, err := client.GetMapGroup(database, m.GroupID)
		if err == nil && group.Name != "" {
			fmt.Printf("Map Group: %s\n", group.Name)
		}
	}

	if m.Dates != nil {
		if m.Dates.CreationDate != "" {
			fmt.Printf("Created: %s\n", m.Dates.CreationDate)
		}
		if m.Dates.LastModified != "" {
			fmt.Printf("Modified: %s\n", m.Dates.LastModified)
		}
	}

	if len(m.Tags) > 0 {
		fmt.Printf("Tags: %v\n", m.Tags)
	}

	return nil
}

// findMapByID searches for a map by its full CouchDB ID across all active projects.
// Returns the database name where the map was found.
func findMapByID(client *api.Client, mapID string) (string, error) {
	projects, _, err := client.ListProjects(api.ListProjectsOptions{})
	if err != nil {
		return "", err
	}

	var projectIDs []string
	for _, project := range projects {
		if project.ProjectID == "glacier_project_documents" || !project.IsActive {
			continue
		}
		projectIDs = append(projectIDs, project.ProjectID)
	}

	// Try the POST search endpoint first
	maps, err := client.SearchMapsByID(projectIDs, mapID)
	if err == nil && len(maps) > 0 {
		for _, m := range maps {
			mID := m.CouchDbID
			if mID == "" {
				mID = m.CouchID
			}
			if mID == mapID {
				// Extract database from the map's ID field (format: database|couchDbId)
				if m.ID != "" && strings.Contains(m.ID, "|") {
					parts := strings.SplitN(m.ID, "|", 2)
					return parts[0], nil
				}
			}
		}
	}

	// Fallback: search each project directly
	for _, projectID := range projectIDs {
		_, err := client.GetMap(projectID, mapID)
		if err == nil {
			return projectID, nil
		}
	}

	return "", fmt.Errorf("map with ID %s not found", mapID)
}
