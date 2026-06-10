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
- **Stats**: Count tickets/audits grouped by responsible, status, project, tag, template, or time period — without listing them
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

### Count and Break Down Tickets (preferred for "how many" questions)

For any counting or breakdown question ("how many tickets per responsible/status/project?"), use `stats` instead of listing tickets — it returns compact counts instead of full documents.

Open tickets per responsible across all projects:
```bash
ec tickets stats --group-by responsible -s started
```

Tickets per project and status (portfolio overview):
```bash
ec tickets stats --group-by project,status
```

Overdue tickets per responsible:
```bash
ec tickets stats --group-by responsible --overdue
```

Tickets created per month, this year:
```bash
ec tickets stats --group-by created:month --created-after 1y
```

Every row shows COUNT and OVERDUE; a TOTAL row is always included. All `tickets list` filters work (`-s`, `-r`, `-t`, `-g`, `--search`, `--created-after`, ...). Dimensions (max two, comma-separated): `responsible`, `status`, `project`, `tag`, `author`, `created:week`, `created:month`, `completed:week`, `completed:month`. Add `-j` for JSON. Note: grouping by `tag` counts a ticket once per tag, so the column sum can exceed TOTAL.

Audits work the same way, with `template` and `auditor` instead of `responsible`:
```bash
ec audits stats --group-by template,status
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
2. Count tickets by status: `ec tickets stats <project-id> --group-by status -j`
3. Count audits by status: `ec audits stats <project-id> --group-by status -j`
4. Compile into a formatted report

### "How many open tickets does each subcontractor have?"
1. One command, no listing needed: `ec tickets stats --group-by responsible -s started`
2. The OVERDUE column shows who is sitting on overdue work

## Tips

- Use human IDs (6 characters like `CC455B`) instead of full CouchDB IDs for tickets and audits
- Omit the project-id argument to search across all projects
- Prefer `stats` over `list -j` whenever the question is about counts or breakdowns — it is far cheaper than parsing ticket dumps
- Use `-l` to limit results and `-p` for pagination
- Pipe JSON output to `jq` for filtering: `ec tickets list <project-id> -j | jq '.[] | select(.status == "started")'`
