---
name: edcontrols
description: Manage EdControls projects, tickets, audits, and templates via the ec CLI
metadata:
  {
    "openclaw":
      {
        "requires": { "bins": ["ec"], "env": ["EDCONTROLS_ACCESS_TOKEN"] },
        "primaryEnv": "EDCONTROLS_ACCESS_TOKEN"
      }
  }
---

# EdControls

This skill allows you to interact with the EdControls platform - a project management and collaboration tool for construction and inspection workflows.

## What You Can Do

- **Projects**: List and view project details
- **Tickets**: List, view, create, update, assign, complete, archive, and delete tickets
- **Audits**: List, view, and create audits from templates
- **Templates**: List, view, create, update, publish/unpublish audit templates
- **Maps**: List, view, upload, and manage project drawings
- **Files**: List, view, upload, download, and manage project files

## Common Commands

### Check Authentication
```bash
ec whoami
```

### List Projects
```bash
ec projects list
```

### Work with Tickets

List tickets for a project:
```bash
ec tickets list <project-id>
```

Filter by status (created, started, completed):
```bash
ec tickets list <project-id> -s started
```

Get ticket details:
```bash
ec tickets get <ticket-id>
```

Update a ticket:
```bash
ec tickets update <project-id> <ticket-id> -t "New title" -d "Description" -r user@example.com
```

Add a comment:
```bash
ec tickets update <project-id> <ticket-id> -m "This is a comment"
```

Mark as completed:
```bash
ec tickets update <project-id> <ticket-id> --complete
```

Assign a ticket:
```bash
ec tickets assign <project-id> <ticket-id> user@example.com
```

### Work with Audits

List audits:
```bash
ec audits list <project-id>
```

Filter by status (started, In Progress, completed):
```bash
ec audits list <project-id> -s completed
```

Create an audit from a template:
```bash
ec audits create <project-id> <template-id> -n "Audit Name" -r auditor@example.com
```

### Work with Templates

List templates:
```bash
ec templates list <project-id>
```

Publish a template:
```bash
ec templates publish <project-id> <template-id>
```

### JSON Output

All commands support `-j` flag for JSON output, useful for processing:
```bash
ec tickets list <project-id> -j
ec audits get <audit-id> -j
```

## Example Tasks

### "Summarize all open tickets for project X"
1. List tickets with status filter: `ec tickets list <project-id> -s started -j`
2. Parse the JSON output
3. Generate a summary of titles, assignees, and due dates

### "Assign all unassigned tickets to John"
1. List tickets: `ec tickets list <project-id> -s created -j`
2. For each ticket without a responsible person:
   `ec tickets assign <project-id> <ticket-id> john@example.com`

### "Tag all completed audits with 'reviewed'"
1. List completed audits: `ec audits list <project-id> -s completed -j`
2. For each audit, update tags via the API

### "Create weekly inspection audits"
1. Find the template: `ec templates list <project-id> -j`
2. Create audit: `ec audits create <project-id> <template-id> -n "Week 7 Inspection" -r inspector@example.com -d "2026-02-21T17:00:00Z"`

### "Generate a status report"
1. Get project info: `ec projects get <project-id> -j`
2. Count tickets by status: `ec tickets list <project-id> -j`
3. Count audits by status: `ec audits list <project-id> -j`
4. Compile into a formatted report

## Tips

- Use human IDs (6 characters like `CC455B`) instead of full CouchDB IDs for tickets and audits
- Omit the project-id argument to search across all projects
- Use `-l` to limit results and `-p` for pagination
- Pipe JSON output to `jq` for filtering: `ec tickets list <project-id> -j | jq '.[] | select(.status == "started")'`
