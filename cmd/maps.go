package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/mauricejumelet/edcontrols-cli/internal/api"
)

type MapsCmd struct {
	List   MapsListCmd   `cmd:"" help:"List maps (drawings)"`
	Get    MapsGetCmd    `cmd:"" help:"Get map details"`
	Add    MapsAddCmd    `cmd:"" help:"Add a new map (upload and convert PDF/image)"`
	Delete MapsDeleteCmd `cmd:"" help:"Delete a map"`
	Groups MapGroupsCmd  `cmd:"" help:"Manage map groups"`
}

type MapGroupsCmd struct {
	List MapGroupsListCmd `cmd:"" help:"List map groups"`
}

type MapGroupsListCmd struct {
	Database string `arg:"" help:"Project database name (required)"`
	Search   string `short:"s" help:"Search by name"`
	Archived bool   `short:"a" help:"Include archived groups"`
	Limit    int    `short:"l" default:"50" help:"Maximum number of groups to return"`
	Page     int    `short:"p" default:"0" help:"Page number (0-based)"`
	JSON     bool   `short:"j" help:"Output as JSON"`
}

func (c *MapGroupsListCmd) Run(client *api.Client) error {
	opts := api.ListGroupsOptions{
		Database:   c.Database,
		SearchName: c.Search,
		Archived:   c.Archived,
		Size:       c.Limit,
		Page:       c.Page,
	}

	groups, total, err := client.ListMapGroups(opts)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(groups)
	}

	if len(groups) == 0 {
		fmt.Println("No map groups found.")
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
	fmt.Printf("\nTotal: %d map groups\n", total)

	return nil
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
	fmt.Fprintln(w, "ID\tNAME\tGROUP\tSTATUS\tCREATED\tMODIFIED")
	fmt.Fprintln(w, "--\t----\t-----\t------\t-------\t--------")

	for _, m := range maps {
		mapID := m.CouchDbID
		if mapID == "" {
			mapID = m.CouchID
		}

		// Handle special Google Maps entry
		if mapID == "EDGeomapMapID" {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", "<google-maps>", "Google Maps", "-", "-", "-", "-")
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

		// Determine status
		status := "active"
		if isFieldSet(m.Deleted) {
			status = "deleted"
		} else if isFieldSet(m.Archived) {
			status = "archived"
		}

		name := truncate(m.Name, 40)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", mapID, name, groupName, status, created, modified)
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

type MapsAddCmd struct {
	Database    string   `arg:"" help:"Project database name"`
	FileGroupID string   `arg:"" help:"File group ID (where the file will be stored)"`
	File        string   `arg:"" help:"Path to PDF or image file to upload" type:"existingfile"`
	Name        string   `short:"n" help:"Map name (defaults to filename)"`
	Tags        []string `short:"t" help:"Tags to add (can be specified multiple times)"`
}

func (c *MapsAddCmd) Run(client *api.Client) error {
	// Validate file type - only PDF, PNG, JPG allowed for maps
	if !isValidMapFileType(c.File) {
		return fmt.Errorf("invalid file type: only PDF, PNG, and JPG files can be converted to maps")
	}

	// Read the file
	fileData, err := os.ReadFile(c.File)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// Get file info
	fileInfo, err := os.Stat(c.File)
	if err != nil {
		return fmt.Errorf("getting file info: %w", err)
	}

	// Determine display name
	displayName := c.Name
	if displayName == "" {
		displayName = fileInfo.Name()
	}

	// Generate unique upload filename with timestamp
	ext := ""
	if idx := strings.LastIndex(fileInfo.Name(), "."); idx >= 0 {
		ext = fileInfo.Name()[idx:]
	}
	baseName := strings.TrimSuffix(fileInfo.Name(), ext)
	uploadName := fmt.Sprintf("%s-%d%s", baseName, time.Now().UnixMilli(), ext)

	// Determine content type based on extension
	contentType := getContentType(c.File)

	fmt.Printf("Uploading %s (%s)...\n", displayName, formatFileSize(fileInfo.Size()))

	// Step 1: Initiate upload
	initResp, err := client.InitiateUpload(c.Database, uploadName)
	if err != nil {
		return fmt.Errorf("initiating upload: %w", err)
	}

	// Step 2: Upload file data
	if err := client.UploadChunk(initResp.UUID, uploadName, 0, fileData); err != nil {
		return fmt.Errorf("uploading file: %w", err)
	}

	// Step 3: Complete upload
	completeResp, err := client.CompleteUpload(initResp.UUID, uploadName)
	if err != nil {
		return fmt.Errorf("completing upload: %w", err)
	}

	// Step 4: Create the file document
	fileResp, err := client.CreateFile(api.CreateFileOptions{
		Database:     c.Database,
		FileName:     displayName,
		UploadedName: uploadName,
		FileURL:      completeResp.SignedURL,
		FileGroupID:  c.FileGroupID,
		ContentType:  contentType,
		Size:         fileInfo.Size(),
		Tags:         c.Tags,
	})
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}

	if fileResp.Code != 200 {
		return fmt.Errorf("file creation failed: %s", fileResp.Message)
	}

	fmt.Printf("File uploaded. Converting to map...\n")

	// Step 5: Get the file to retrieve its ID and versionId
	// Wait briefly for indexing, then search for recently created files
	time.Sleep(500 * time.Millisecond)

	files, _, err := client.ListFiles(api.ListFilesOptions{
		Database:  c.Database,
		Size:      20,
		SortBy:    "CREATIONDATE",
		SortOrder: "DESC",
	})
	if err != nil {
		return fmt.Errorf("finding uploaded file: %w", err)
	}

	// Find the file we just uploaded by matching the display name
	var uploadedFile *api.File
	for i := range files {
		name := files[i].FileName
		if name == "" {
			name = files[i].Name
		}
		if name == displayName {
			uploadedFile = &files[i]
			break
		}
	}

	if uploadedFile == nil {
		return fmt.Errorf("could not find uploaded file '%s' (searched %d recent files)", displayName, len(files))
	}

	fileID := uploadedFile.CouchDbID
	if fileID == "" {
		fileID = uploadedFile.CouchID
	}
	if fileID == "" {
		// Try extracting from the compound ID field (format: database|couchDbId)
		if uploadedFile.ID != "" && strings.Contains(uploadedFile.ID, "|") {
			parts := strings.SplitN(uploadedFile.ID, "|", 2)
			if len(parts) == 2 {
				fileID = parts[1]
			}
		}
	}

	// Get full file details including versionId
	fullFile, err := client.GetFile(c.Database, fileID)
	if err != nil {
		return fmt.Errorf("getting file details: %w", err)
	}

	if fullFile.VersionID == "" {
		return fmt.Errorf("file has no versionId, cannot convert to map")
	}

	// Get file group name for the tiler
	groupName := ""
	group, err := client.GetFileGroup(c.Database, c.FileGroupID)
	if err == nil && group.Name != "" {
		groupName = group.Name
	}

	// Step 6: Convert file to map
	if err := client.ConvertFileToMap(c.Database, fullFile.CouchDbID, fullFile.VersionID, displayName, groupName); err != nil {
		return fmt.Errorf("converting to map: %w", err)
	}

	fmt.Printf("Map '%s' queued for creation.\n", displayName)
	fmt.Printf("File ID: %s\n", fullFile.CouchDbID)

	return nil
}

type MapsDeleteCmd struct {
	Database string `arg:"" help:"Project database name"`
	MapID    string `arg:"" help:"Map ID (full CouchDB ID)"`
}

func (c *MapsDeleteCmd) Run(client *api.Client) error {
	if err := client.DeleteLibraryItems(c.Database, nil, []string{c.MapID}); err != nil {
		return fmt.Errorf("deleting map: %w", err)
	}

	fmt.Printf("Map %s deleted successfully.\n", c.MapID)
	return nil
}
