package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/mauricejumelet/edcontrols-cli/internal/api"
)

type TemplatesCmd struct {
	List      TemplatesListCmd      `cmd:"" help:"List audit templates"`
	Get       TemplatesGetCmd       `cmd:"" help:"Get audit template details"`
	Create    TemplatesCreateCmd    `cmd:"" help:"Create a new audit template"`
	Update    TemplatesUpdateCmd    `cmd:"" help:"Update an audit template"`
	Publish   TemplatesPublishCmd   `cmd:"" help:"Publish an audit template"`
	Unpublish TemplatesUnpublishCmd `cmd:"" help:"Unpublish an audit template"`
	Groups    TemplateGroupsCmd     `cmd:"" help:"Manage template groups"`
}

type TemplateGroupsCmd struct {
	List   TemplateGroupsListCmd   `cmd:"" help:"List template groups"`
	Create TemplateGroupsCreateCmd `cmd:"" help:"Create a new template group"`
}

type TemplateGroupsListCmd struct {
	Database string `arg:"" help:"Project database name (required)"`
	Search   string `short:"s" help:"Search by name"`
	Archived bool   `short:"a" help:"Include archived groups"`
	Limit    int    `short:"l" default:"50" help:"Maximum number of groups to return"`
	Page     int    `short:"p" default:"0" help:"Page number (0-based)"`
	JSON     bool   `short:"j" help:"Output as JSON"`
}

func (c *TemplateGroupsListCmd) Run(client *api.Client) error {
	opts := api.ListGroupsOptions{
		Database:   c.Database,
		SearchName: c.Search,
		Archived:   c.Archived,
		Size:       c.Limit,
		Page:       c.Page,
	}

	groups, total, err := client.ListTemplateGroups(opts)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(groups)
	}

	if len(groups) == 0 {
		fmt.Println("No template groups found.")
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
	fmt.Printf("\nTotal: %d template groups\n", total)

	return nil
}

type TemplateGroupsCreateCmd struct {
	Database string `arg:"" help:"Project database name"`
	Name     string `arg:"" help:"Name for the new template group"`
	JSON     bool   `short:"j" help:"Output as JSON"`
}

func (c *TemplateGroupsCreateCmd) Run(client *api.Client) error {
	groupID, err := client.CreateTemplateGroup(c.Database, c.Name)
	if err != nil {
		return fmt.Errorf("creating template group: %w", err)
	}

	if c.JSON {
		return printJSON(map[string]string{
			"id":   groupID,
			"name": c.Name,
		})
	}

	fmt.Printf("Template group '%s' created.\n", c.Name)
	fmt.Printf("ID: %s\n", groupID)
	return nil
}

type TemplatesListCmd struct {
	Database  string `arg:"" help:"Project database name (required)"`
	Search    string `short:"s" help:"Search by name"`
	GroupID   string `short:"g" help:"Filter by group ID"`
	Published bool   `short:"p" help:"Only show published templates"`
	Archived  bool   `short:"a" help:"Include archived templates"`
	Limit     int    `short:"l" default:"50" help:"Maximum number of templates to return"`
	Page      int    `default:"0" help:"Page number (0-based)"`
	JSON      bool   `short:"j" help:"Output as JSON"`
}

func (c *TemplatesListCmd) Run(client *api.Client) error {
	var isPublished *bool
	if c.Published {
		t := true
		isPublished = &t
	}

	opts := api.ListAuditTemplatesOptions{
		Database:    c.Database,
		SearchName:  c.Search,
		GroupID:     c.GroupID,
		IsPublished: isPublished,
		Archived:    c.Archived,
		Size:        c.Limit,
		Page:        c.Page,
	}

	templates, total, err := client.ListAuditTemplates(opts)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(templates)
	}

	if len(templates) == 0 {
		fmt.Println("No templates found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tPUBLISHED\tMODIFIED")
	fmt.Fprintln(w, "--\t----\t---------\t--------")

	for _, template := range templates {
		published := "No"
		if template.IsPublished {
			published = "Yes"
		}

		modified := "-"
		if template.Dates != nil && template.Dates.LastModified != "" && len(template.Dates.LastModified) >= 10 {
			modified = template.Dates.LastModified[:10]
		}

		name := truncate(template.Name, 50)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", template.CouchDbID, name, published, modified)
	}

	w.Flush()

	if total > c.Limit {
		fmt.Printf("\n(Showing %d of %d templates, use -l to increase limit)\n", len(templates), total)
	} else {
		fmt.Printf("\nTotal: %d templates\n", total)
	}

	return nil
}

type TemplatesGetCmd struct {
	Database   string `arg:"" help:"Project database name"`
	TemplateID string `arg:"" help:"Template ID"`
	JSON       bool   `short:"j" help:"Output as JSON"`
}

func (c *TemplatesGetCmd) Run(client *api.Client) error {
	if c.JSON {
		// Return raw securedata document for JSON output
		doc, err := client.GetDocument(c.Database, c.TemplateID)
		if err != nil {
			return err
		}
		return printJSON(doc)
	}

	template, err := client.GetAuditTemplate(c.Database, c.TemplateID)
	if err != nil {
		return err
	}

	fmt.Printf("Template: %s\n", template.Name)
	fmt.Printf("ID: %s\n", template.CouchDbID)
	fmt.Printf("Published: %t\n", template.IsPublished)

	if template.Description != "" {
		fmt.Printf("Description: %s\n", template.Description)
	}
	if template.Author != nil && template.Author.Email != "" {
		fmt.Printf("Author: %s\n", template.Author.Email)
	}
	if template.Dates != nil {
		if template.Dates.CreationDate != "" {
			fmt.Printf("Created: %s\n", template.Dates.CreationDate)
		}
		if template.Dates.LastModified != "" {
			fmt.Printf("Modified: %s\n", template.Dates.LastModified)
		}
	}
	if len(template.Tags) > 0 {
		fmt.Printf("Tags: %v\n", template.Tags)
	}

	return nil
}

type TemplatesCreateCmd struct {
	Database string   `arg:"" help:"Project database name"`
	GroupID  string   `arg:"" help:"Template group ID"`
	Name     string   `arg:"" help:"Template name"`
	Tags     []string `short:"t" help:"Tags to add (can be specified multiple times)"`
	JSON     bool     `short:"j" help:"Output as JSON"`
}

func (c *TemplatesCreateCmd) Run(client *api.Client) error {
	opts := api.CreateAuditTemplateOptions{
		Database: c.Database,
		GroupID:  c.GroupID,
		Name:     c.Name,
		Tags:     c.Tags,
	}

	templateID, err := client.CreateAuditTemplate(opts)
	if err != nil {
		return fmt.Errorf("creating template: %w", err)
	}

	if c.JSON {
		return printJSON(map[string]string{
			"id":      templateID,
			"name":    c.Name,
			"groupId": c.GroupID,
		})
	}

	fmt.Printf("Template '%s' created.\n", c.Name)
	fmt.Printf("ID: %s\n", templateID)
	return nil
}

type TemplatesUpdateCmd struct {
	Database    string   `arg:"" help:"Project database name"`
	TemplateID  string   `arg:"" help:"Template ID"`
	Name        string   `short:"n" help:"New template name"`
	Description string   `short:"d" help:"New description"`
	Tags        []string `short:"t" help:"Tags to set (replaces existing)"`
}

func (c *TemplatesUpdateCmd) Run(client *api.Client) error {
	updates := make(map[string]interface{})

	if c.Name != "" {
		updates["name"] = c.Name
	}
	if c.Description != "" {
		updates["description"] = c.Description
	}
	if len(c.Tags) > 0 {
		updates["tags"] = c.Tags
	}

	if len(updates) == 0 {
		return fmt.Errorf("no updates specified")
	}

	if err := client.UpdateAuditTemplate(c.Database, c.TemplateID, updates); err != nil {
		return err
	}

	fmt.Printf("Template %s updated successfully\n", c.TemplateID)
	return nil
}

type TemplatesPublishCmd struct {
	Database   string `arg:"" help:"Project database name"`
	TemplateID string `arg:"" help:"Template ID"`
}

func (c *TemplatesPublishCmd) Run(client *api.Client) error {
	if err := client.PublishAuditTemplate(c.Database, c.TemplateID, true); err != nil {
		return fmt.Errorf("publishing template: %w", err)
	}
	fmt.Printf("Template %s published.\n", c.TemplateID)
	return nil
}

type TemplatesUnpublishCmd struct {
	Database   string `arg:"" help:"Project database name"`
	TemplateID string `arg:"" help:"Template ID"`
}

func (c *TemplatesUnpublishCmd) Run(client *api.Client) error {
	if err := client.PublishAuditTemplate(c.Database, c.TemplateID, false); err != nil {
		return fmt.Errorf("unpublishing template: %w", err)
	}
	fmt.Printf("Template %s unpublished.\n", c.TemplateID)
	return nil
}
