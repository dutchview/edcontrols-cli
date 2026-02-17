package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/mauricejumelet/edcontrols-cli/internal/api"
)

type FilesCmd struct {
	List   FilesListCmd   `cmd:"" help:"List files"`
	Get    FilesGetCmd    `cmd:"" help:"Get file details"`
	Groups FileGroupsCmd  `cmd:"" help:"Manage file groups"`
}

type FileGroupsCmd struct {
	List FileGroupsListCmd `cmd:"" help:"List file groups"`
}

type FileGroupsListCmd struct {
	Database string `arg:"" help:"Project database name (required)"`
	Search   string `short:"s" help:"Search by name"`
	Archived bool   `short:"a" help:"Include archived groups"`
	Limit    int    `short:"l" default:"50" help:"Maximum number of groups to return"`
	Page     int    `short:"p" default:"0" help:"Page number (0-based)"`
	JSON     bool   `short:"j" help:"Output as JSON"`
}

func (c *FileGroupsListCmd) Run(client *api.Client) error {
	opts := api.ListGroupsOptions{
		Database:   c.Database,
		SearchName: c.Search,
		Archived:   c.Archived,
		Size:       c.Limit,
		Page:       c.Page,
	}

	groups, total, err := client.ListFileGroups(opts)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(groups)
	}

	if len(groups) == 0 {
		fmt.Println("No file groups found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME")
	fmt.Fprintln(w, "--\t----")

	for _, g := range groups {
		groupID := g.CouchDbID
		if groupID == "" {
			groupID = g.CouchID
		}
		if groupID == "" {
			groupID = g.ID
		}
		fmt.Fprintf(w, "%s\t%s\n", groupID, g.Name)
	}

	w.Flush()
	fmt.Printf("\nTotal: %d file groups\n", total)

	return nil
}

type FilesListCmd struct {
	Database string `arg:"" help:"Project database name (required)"`
	GroupID  string `short:"g" help:"Filter by file group ID"`
	Search   string `short:"s" help:"Search by name"`
	Tag      string `short:"t" help:"Filter by tag"`
	Archived bool   `short:"a" help:"Include archived files"`
	Limit    int    `short:"l" default:"50" help:"Maximum number of files to return"`
	Page     int    `short:"p" default:"0" help:"Page number (0-based)"`
	Sort     string `short:"o" default:"created" enum:"created,modified,name" help:"Sort by field"`
	Asc      bool   `help:"Sort in ascending order"`
	JSON     bool   `short:"j" help:"Output as JSON"`
}

func (c *FilesListCmd) Run(client *api.Client) error {
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

	opts := api.ListFilesOptions{
		Database:   c.Database,
		GroupID:    c.GroupID,
		SearchName: c.Search,
		Tag:        c.Tag,
		Archived:   c.Archived,
		Size:       c.Limit,
		Page:       c.Page,
		SortBy:     sortBy,
		SortOrder:  sortOrder,
	}

	files, total, err := client.ListFiles(opts)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(files)
	}

	if len(files) == 0 {
		fmt.Println("No files found.")
		return nil
	}

	// Build a map of group names
	groupNames := make(map[string]string)
	for _, f := range files {
		if f.GroupID != "" && groupNames[f.GroupID] == "" {
			if f.GroupName != "" {
				groupNames[f.GroupID] = f.GroupName
			} else {
				// Fetch group name
				group, err := client.GetFileGroup(c.Database, f.GroupID)
				if err == nil && group.Name != "" {
					groupNames[f.GroupID] = group.Name
				}
			}
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tGROUP\tSIZE\tCREATED")
	fmt.Fprintln(w, "--\t----\t-----\t----\t-------")

	for _, f := range files {
		fileID := f.CouchDbID
		if fileID == "" {
			fileID = f.CouchID
		}

		created := "-"
		if f.Dates != nil && f.Dates.CreationDate != "" && len(f.Dates.CreationDate) >= 10 {
			created = f.Dates.CreationDate[:10]
		}

		groupName := "-"
		if name, ok := groupNames[f.GroupID]; ok && name != "" {
			groupName = truncate(name, 25)
		}

		// Format file size
		size := "-"
		if sizeBytes := getFileSize(f.Size); sizeBytes > 0 {
			size = formatFileSize(sizeBytes)
		}

		name := f.Name
		if name == "" {
			name = f.FileName
		}
		name = truncate(name, 40)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", fileID, name, groupName, size, created)
	}

	w.Flush()

	limitReached := total > c.Limit
	if limitReached {
		fmt.Printf("\nShowing %d files (limit reached). Use -l to show more, e.g.: ec files list %s -l 100\n", len(files), c.Database)
	} else {
		fmt.Printf("\nTotal: %d files\n", total)
	}

	return nil
}

type FilesGetCmd struct {
	FileID   string `arg:"" help:"File ID (full CouchDB ID)"`
	Database string `short:"d" help:"Project database name (optional, will search if not provided)"`
	JSON     bool   `short:"j" help:"Output as JSON"`
}

func (c *FilesGetCmd) Run(client *api.Client) error {
	database := c.Database
	fileID := c.FileID

	// If no database provided, search for the file across all projects
	if database == "" {
		foundDB, err := findFileByID(client, fileID)
		if err != nil {
			return err
		}
		database = foundDB
	}

	if c.JSON {
		// Return raw securedata document for JSON output
		doc, err := client.GetDocument(database, fileID)
		if err != nil {
			return err
		}
		return printJSON(doc)
	}

	f, err := client.GetFile(database, fileID)
	if err != nil {
		return err
	}

	name := f.Name
	if name == "" {
		name = f.FileName
	}
	fmt.Printf("File: %s\n", name)
	fmt.Printf("ID: %s\n", fileID)

	if f.FileName != "" && f.FileName != name {
		fmt.Printf("Filename: %s\n", f.FileName)
	}

	// Fetch project name
	project, err := client.GetProject(database)
	if err == nil && project.ProjectName != "" {
		fmt.Printf("Project: %s (%s)\n", project.ProjectName, database)
	} else {
		fmt.Printf("Project: %s\n", database)
	}

	// Fetch file group name
	if f.GroupID != "" {
		group, err := client.GetFileGroup(database, f.GroupID)
		if err == nil && group.Name != "" {
			fmt.Printf("File Group: %s\n", group.Name)
		}
	}

	if f.ContentType != "" {
		fmt.Printf("Type: %s\n", f.ContentType)
	}

	if sizeBytes := getFileSize(f.Size); sizeBytes > 0 {
		fmt.Printf("Size: %s (%d bytes)\n", formatFileSize(sizeBytes), sizeBytes)
	}

	if f.Dates != nil {
		if f.Dates.CreationDate != "" {
			fmt.Printf("Created: %s\n", f.Dates.CreationDate)
		}
		if f.Dates.LastModified != "" {
			fmt.Printf("Modified: %s\n", f.Dates.LastModified)
		}
	}

	if f.Author != nil && f.Author.Email != "" {
		fmt.Printf("Author: %s\n", f.Author.Email)
	}

	if len(f.Tags) > 0 {
		fmt.Printf("Tags: %v\n", f.Tags)
	}

	return nil
}

// findFileByID searches for a file by its full CouchDB ID across all active projects.
// Returns the database name where the file was found.
func findFileByID(client *api.Client, fileID string) (string, error) {
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
	files, err := client.SearchFilesByID(projectIDs, fileID)
	if err == nil && len(files) > 0 {
		for _, f := range files {
			fID := f.CouchDbID
			if fID == "" {
				fID = f.CouchID
			}
			if fID == fileID {
				// Extract database from the file's ID field (format: database|couchDbId)
				if f.ID != "" && strings.Contains(f.ID, "|") {
					parts := strings.SplitN(f.ID, "|", 2)
					return parts[0], nil
				}
			}
		}
	}

	// Fallback: search each project directly
	for _, projectID := range projectIDs {
		_, err := client.GetFile(projectID, fileID)
		if err == nil {
			return projectID, nil
		}
	}

	return "", fmt.Errorf("file with ID %s not found", fileID)
}

// getFileSize extracts the file size from an interface{} value
func getFileSize(size interface{}) int64 {
	if size == nil {
		return 0
	}
	switch v := size.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		var n int64
		fmt.Sscanf(v, "%d", &n)
		return n
	}
	return 0
}

// formatFileSize formats a file size in bytes to a human-readable string
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
