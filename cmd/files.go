package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/mauricejumelet/edcontrols-cli/internal/api"
)

type FilesCmd struct {
	List      FilesListCmd      `cmd:"" help:"List files"`
	Get       FilesGetCmd       `cmd:"" help:"Get file details"`
	Add       FilesAddCmd       `cmd:"" help:"Add a new file (upload PDF, image, etc.)"`
	Download  FilesDownloadCmd  `cmd:"" help:"Download a file"`
	Archive   FilesArchiveCmd   `cmd:"" help:"Archive a file"`
	Unarchive FilesUnarchiveCmd `cmd:"" help:"Unarchive a file"`
	Groups    FileGroupsCmd     `cmd:"" help:"Manage file groups"`
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

	// Build a map of group names (check both GroupID and FileGroupID)
	groupNames := make(map[string]string)
	for _, f := range files {
		groupID := f.GroupID
		if groupID == "" {
			groupID = f.FileGroupID
		}
		if groupID != "" && groupNames[groupID] == "" {
			if f.GroupName != "" {
				groupNames[groupID] = f.GroupName
			} else {
				// Fetch group name
				group, err := client.GetFileGroup(c.Database, groupID)
				if err == nil && group.Name != "" {
					groupNames[groupID] = group.Name
				}
			}
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tGROUP\tSIZE\tSTATUS\tCREATED\tMODIFIED")
	fmt.Fprintln(w, "--\t----\t-----\t----\t------\t-------\t--------")

	for _, f := range files {
		fileID := f.CouchDbID
		if fileID == "" {
			fileID = f.CouchID
		}

		created := "-"
		if f.Dates != nil && f.Dates.CreationDate != "" && len(f.Dates.CreationDate) >= 10 {
			created = f.Dates.CreationDate[:10]
		}

		modified := "-"
		if f.Dates != nil && f.Dates.LastModified != "" && len(f.Dates.LastModified) >= 10 {
			modified = f.Dates.LastModified[:10]
		}

		// Get group name (check both GroupID and FileGroupID)
		fileGroupID := f.GroupID
		if fileGroupID == "" {
			fileGroupID = f.FileGroupID
		}
		groupName := "-"
		if name, ok := groupNames[fileGroupID]; ok && name != "" {
			groupName = truncate(name, 25)
		}

		// Format file size
		size := "-"
		if sizeBytes := getFileSize(f.Size); sizeBytes > 0 {
			size = formatFileSize(sizeBytes)
		}

		// Determine status
		status := "active"
		if isFieldSet(f.Deleted) {
			status = "deleted"
		} else if isFieldSet(f.Archived) {
			status = "archived"
		}

		name := f.Name
		if name == "" {
			name = f.FileName
		}
		name = truncate(name, 40)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", fileID, name, groupName, size, status, created, modified)
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

	// Fetch file group name (check both GroupID and FileGroupID)
	groupID := f.GroupID
	if groupID == "" {
		groupID = f.FileGroupID
	}
	if groupID != "" {
		group, err := client.GetFileGroup(database, groupID)
		if err == nil && group.Name != "" {
			fmt.Printf("File Group: %s (%s)\n", group.Name, groupID)
		} else {
			fmt.Printf("File Group ID: %s\n", groupID)
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

type FilesAddCmd struct {
	Database string   `arg:"" help:"Project database name"`
	GroupID  string   `arg:"" help:"File group ID"`
	File     string   `arg:"" help:"Path to file to upload" type:"existingfile"`
	Name     string   `short:"n" help:"File name (defaults to filename)"`
	Tags     []string `short:"t" help:"Tags to add (can be specified multiple times)"`
	JSON     bool     `short:"j" help:"Output as JSON"`
}

func (c *FilesAddCmd) Run(client *api.Client) error {
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

	// Step 2: Upload file data (single chunk for now)
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
		FileGroupID:  c.GroupID,
		ContentType:  contentType,
		Size:         fileInfo.Size(),
		Tags:         c.Tags,
	})
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}

	if c.JSON {
		return printJSON(fileResp)
	}

	fmt.Printf("File uploaded successfully!\n")
	fmt.Printf("Name: %s\n", displayName)

	return nil
}

// getContentType returns the MIME type based on file extension
func getContentType(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".pdf"):
		return "application/pdf"
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lower, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lower, ".bmp"):
		return "image/bmp"
	case strings.HasSuffix(lower, ".tiff"), strings.HasSuffix(lower, ".tif"):
		return "image/tiff"
	case strings.HasSuffix(lower, ".doc"):
		return "application/msword"
	case strings.HasSuffix(lower, ".docx"):
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case strings.HasSuffix(lower, ".xls"):
		return "application/vnd.ms-excel"
	case strings.HasSuffix(lower, ".xlsx"):
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case strings.HasSuffix(lower, ".txt"):
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

type FilesDownloadCmd struct {
	FileID   string `arg:"" help:"File ID (full CouchDB ID)"`
	Database string `short:"d" help:"Project database name (optional, will search if not provided)"`
	Output   string `short:"o" help:"Output file path (defaults to original filename)"`
}

func (c *FilesDownloadCmd) Run(client *api.Client) error {
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

	// Get file details to retrieve versionId and filename
	f, err := client.GetFile(database, fileID)
	if err != nil {
		return fmt.Errorf("getting file details: %w", err)
	}

	if f.VersionID == "" {
		return fmt.Errorf("file has no versionId, cannot download")
	}

	// Determine filename
	fileName := f.FileName
	if fileName == "" {
		fileName = f.Name
	}
	if fileName == "" {
		fileName = "download"
	}

	// Determine output path
	outputPath := c.Output
	if outputPath == "" {
		outputPath = fileName
	}

	fmt.Printf("Downloading %s...\n", fileName)

	// Download the file
	data, err := client.DownloadFile(database, fileID, f.VersionID, fileName)
	if err != nil {
		return fmt.Errorf("downloading file: %w", err)
	}

	// Write to output file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	fmt.Printf("Downloaded to %s (%s)\n", outputPath, formatFileSize(int64(len(data))))

	return nil
}

type FilesArchiveCmd struct {
	Database string `arg:"" help:"Project database name"`
	FileID   string `arg:"" help:"File ID (full CouchDB ID)"`
}

func (c *FilesArchiveCmd) Run(client *api.Client) error {
	if err := client.ArchiveFile(c.Database, []string{c.FileID}, true); err != nil {
		return fmt.Errorf("archiving file: %w", err)
	}

	fmt.Printf("File %s archived successfully.\n", c.FileID)
	return nil
}

type FilesUnarchiveCmd struct {
	Database string `arg:"" help:"Project database name"`
	FileID   string `arg:"" help:"File ID (full CouchDB ID)"`
}

func (c *FilesUnarchiveCmd) Run(client *api.Client) error {
	if err := client.ArchiveFile(c.Database, []string{c.FileID}, false); err != nil {
		return fmt.Errorf("unarchiving file: %w", err)
	}

	fmt.Printf("File %s unarchived successfully.\n", c.FileID)
	return nil
}
