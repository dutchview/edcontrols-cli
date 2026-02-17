package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/mauricejumelet/edcontrols-cli/internal/api"
)

type ProjectsCmd struct {
	List ProjectsListCmd `cmd:"" help:"List projects"`
	Get  ProjectsGetCmd  `cmd:"" help:"Get project details"`
}

type ProjectsListCmd struct {
	Search  string `short:"s" help:"Search by project name or ID"`
	Glacier bool   `short:"g" help:"Include glacier (archived to long-term storage) projects"`
	JSON    bool   `short:"j" help:"Output as JSON"`
}

func (c *ProjectsListCmd) Run(client *api.Client) error {
	opts := api.ListProjectsOptions{
		Search: c.Search,
	}

	projects, _, err := client.ListProjects(opts)
	if err != nil {
		return err
	}

	// Filter out glacier projects unless --glacier flag is set
	var filtered []api.Project
	glacierCount := 0
	for _, p := range projects {
		if p.ProjectID == "glacier_project_documents" {
			glacierCount++
			if c.Glacier {
				filtered = append(filtered, p)
			}
		} else {
			filtered = append(filtered, p)
		}
	}

	if c.JSON {
		return printJSON(filtered)
	}

	if len(filtered) == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROJECT_ID\tNAME\tSTATUS")
	fmt.Fprintln(w, "----------\t----\t------")

	for _, project := range filtered {
		// Glacier projects are archived to long-term storage and not directly accessible
		if project.ProjectID == "glacier_project_documents" {
			name := truncate(project.ProjectName, 50)
			fmt.Fprintf(w, "<glacier>\t%s\tarchived\n", name)
			continue
		}

		status := "active"
		if !project.IsActive {
			status = "inactive"
		}
		if project.Archived != nil {
			switch v := project.Archived.(type) {
			case string:
				if v != "" {
					status = "archived"
				}
			case bool:
				if v {
					status = "archived"
				}
			}
		}

		name := truncate(project.ProjectName, 50)
		fmt.Fprintf(w, "%s\t%s\t%s\n", project.ProjectID, name, status)
	}

	w.Flush()
	if glacierCount > 0 && !c.Glacier {
		fmt.Printf("\nTotal: %d projects (%d glacier projects hidden, use -g to show)\n", len(filtered), glacierCount)
	} else {
		fmt.Printf("\nTotal: %d projects\n", len(filtered))
	}
	return nil
}

type ProjectsGetCmd struct {
	Database string `arg:"" name:"project-id" help:"Project ID (projectId)"`
	JSON     bool   `short:"j" help:"Output as JSON"`
}

func (c *ProjectsGetCmd) Run(client *api.Client) error {
	project, err := client.GetProject(c.Database)
	if err != nil {
		return err
	}

	if c.JSON {
		// Return raw securedata document for JSON output
		doc, err := client.GetDocument(c.Database, project.CouchDbID)
		if err != nil {
			return err
		}
		return printJSON(doc)
	}

	fmt.Printf("Project: %s\n", project.ProjectName)
	fmt.Printf("Database: %s\n", project.ProjectID)
	fmt.Printf("CouchDB ID: %s\n", project.CouchDbID)
	fmt.Printf("Location: %s\n", project.Location)
	fmt.Printf("Active: %t\n", project.IsActive)
	if project.Archived != nil {
		switch v := project.Archived.(type) {
		case string:
			if v != "" {
				fmt.Printf("Archived: %s\n", v)
			}
		case bool:
			fmt.Printf("Archived: %t\n", v)
		}
	}
	if project.StartDate != "" {
		fmt.Printf("Start Date: %s\n", project.StartDate)
	}
	if project.EndDate != "" {
		fmt.Printf("End Date: %s\n", project.EndDate)
	}

	return nil
}
