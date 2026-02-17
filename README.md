# EdControls CLI

A command-line interface for interacting with the EdControls platform. Manage projects, tickets, audits, templates, maps, and files directly from your terminal.

## Installation

### From Source

```bash
git clone https://github.com/mauricejumelet/edcontrols-cli.git
cd edcontrols-cli
go build -o ec .
```

Move the binary to a directory in your PATH:

```bash
mv ec /usr/local/bin/
```

## Configuration

The CLI requires an access token to authenticate with the EdControls API. The token can be provided in several ways (in order of priority):

1. **Command line flag**: `--token=YOUR_TOKEN`
2. **Environment variable**: `EDCONTROLS_ACCESS_TOKEN`
3. **Config file** (`.env` format):
   - `.env` in current directory
   - `~/.config/edcontrols-cli/.env`
   - Custom path via `--config` flag

### Example .env file

```bash
EDCONTROLS_ACCESS_TOKEN=35efc557-d3b6-41da-bf7d-0bc82a285967
```

### Getting Your Token

Get your access token from the EdControls web interface. Tokens are in UUID format (36 characters).

### Verify Configuration

```bash
# Show current user info to verify token works
ec whoami

# Show configuration help
ec configure
```

## Global Flags

These flags are available on all commands:

| Flag | Description |
|------|-------------|
| `-h, --help` | Show help for any command |
| `-c, --config=PATH` | Path to config file (.env format) |
| `--token=STRING` | Access token (overrides config file) |

## Commands

### whoami

Display information about the currently authenticated user.

```bash
# Show user info
ec whoami

# Output as JSON
ec whoami -j
```

**Output:**
```
Email: user@example.com
Name: John Doe
Company: Example Corp
Roles: [ROLE_USER ROLE_ADMIN]
```

---

### contracts

Manage contracts (clients).

#### contracts list

List all contracts accessible to the user.

```bash
# List all contracts
ec contracts list

# Output as JSON
ec contracts list -j
```

**Output:**
```
ID                                NAME                    PROJECTS  ACTIVE  PLAN
--                                ----                    --------  ------  ----
abc123-def456-...                 Example Company         12        Yes     professional
xyz789-uvw012-...                 Another Client          5         Yes     starter
```

#### contracts projects

List all projects for a specific contract.

```bash
# List projects for a contract
ec contracts projects abc123-def456-789

# Output as JSON
ec contracts projects abc123-def456-789 -j
```

---

### projects

Manage projects.

#### projects list

List all projects accessible to the user.

```bash
# List all projects
ec projects list

# Search by name or ID
ec projects list -s "construction"

# Include glacier (archived) projects
ec projects list -g

# Output as JSON
ec projects list -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-s, --search=STRING` | Search by project name or ID |
| `-g, --glacier` | Include glacier (archived to long-term storage) projects |
| `-j, --json` | Output as JSON |

#### projects get

Get details for a specific project.

```bash
# Get project details
ec projects get nl_company_abc123-def456

# Output as JSON
ec projects get nl_company_abc123-def456 -j
```

---

### tickets

Manage tickets (location-based tasks).

#### tickets list

List tickets with various filter options.

```bash
# List tickets for a specific project
ec tickets list nl_company_abc123

# List tickets across ALL active projects (cross-project search)
ec tickets list

# Filter by status
ec tickets list nl_company_abc123 -s "Open"
ec tickets list nl_company_abc123 -s "In Progress"
ec tickets list nl_company_abc123 -s "Done"

# Search by title
ec tickets list nl_company_abc123 --search "foundation"

# Filter by responsible person
ec tickets list nl_company_abc123 -r "john@example.com"

# Filter by tag
ec tickets list nl_company_abc123 -t "urgent"

# Filter by group
ec tickets list nl_company_abc123 -g "group-id-here"

# Include archived tickets
ec tickets list nl_company_abc123 -a

# Pagination
ec tickets list nl_company_abc123 -l 100 -p 0    # First 100 tickets
ec tickets list nl_company_abc123 -l 100 -p 1    # Next 100 tickets

# Sort by modification date (newest first, default)
ec tickets list nl_company_abc123 -o modified

# Sort by creation date (oldest first)
ec tickets list nl_company_abc123 -o created --asc

# Output as JSON
ec tickets list nl_company_abc123 -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-s, --status=STRING` | Filter by status (Open, In Progress, Done) |
| `--search=STRING` | Search by title |
| `-r, --responsible=STRING` | Filter by responsible person email |
| `-t, --tag=STRING` | Filter by tag |
| `-g, --group-id=STRING` | Filter by group ID |
| `-a, --archived` | Include archived tickets |
| `--all-projects` | Include inactive projects when searching all |
| `-l, --limit=50` | Maximum number of tickets to return |
| `-p, --page=0` | Page number (0-based) |
| `-o, --sort="created"` | Sort by field (created, modified) |
| `--asc` | Sort in ascending order (oldest first) |
| `-j, --json` | Output as JSON |

#### tickets get

Get details for a specific ticket. Supports human IDs (last 6 characters of CouchDB ID, reversed and uppercased).

```bash
# Get ticket by human ID (searches across all projects)
ec tickets get CC455B

# Get ticket by full CouchDB ID
ec tickets get b554cc123456789abcdef

# Specify project database for faster lookup
ec tickets get CC455B -d nl_company_abc123

# Output as JSON
ec tickets get CC455B -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-d, --database=STRING` | Project database name (optional, will search if not provided) |
| `-j, --json` | Output as JSON |

#### tickets assign

Assign a ticket to someone.

```bash
ec tickets assign nl_company_abc123 ticket-id-here john@example.com
```

#### tickets open

Reopen a ticket (set status to Open).

```bash
ec tickets open nl_company_abc123 ticket-id-here
```

#### tickets close

Close a ticket (set status to Done).

```bash
ec tickets close nl_company_abc123 ticket-id-here
```

---

### audits

Manage audits.

#### audits list

List audits with various filter options.

```bash
# List audits for a specific project
ec audits list nl_company_abc123

# List audits across ALL active projects (cross-project search)
ec audits list

# Filter by status
ec audits list nl_company_abc123 -s "started"
ec audits list nl_company_abc123 -s "completed"

# Filter by template
ec audits list nl_company_abc123 -t "template-id-here"

# Search by title
ec audits list nl_company_abc123 --search "inspection"

# Filter by auditor
ec audits list nl_company_abc123 -a "auditor@example.com"

# Filter by group
ec audits list nl_company_abc123 -g "group-id-here"

# Filter by tag
ec audits list nl_company_abc123 --tag "safety"

# Include archived audits
ec audits list nl_company_abc123 --archived

# Pagination and sorting
ec audits list nl_company_abc123 -l 100 -p 0
ec audits list nl_company_abc123 -o modified --asc

# Output as JSON
ec audits list nl_company_abc123 -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-s, --status=STRING` | Filter by status (comma-separated) |
| `-t, --template=STRING` | Filter by template ID |
| `--search=STRING` | Search by title |
| `-a, --auditor=STRING` | Filter by auditor email |
| `-g, --group-id=STRING` | Filter by group ID |
| `--tag=STRING` | Filter by tag |
| `--archived` | Include archived audits |
| `--all-projects` | Include inactive projects when searching all |
| `-l, --limit=50` | Maximum number of audits to return |
| `-p, --page=0` | Page number (0-based) |
| `-o, --sort="created"` | Sort by field (created, modified) |
| `--asc` | Sort in ascending order (oldest first) |
| `-j, --json` | Output as JSON |

#### audits get

Get details for a specific audit. Supports human IDs.

```bash
# Get audit by human ID (searches across all projects)
ec audits get 708739

# Get audit by full CouchDB ID
ec audits get d67c1aba0b017a1c9372e726c6512a1f

# Specify project database for faster lookup
ec audits get 708739 -d nl_company_abc123

# Output as JSON (includes full audit data with questions/answers)
ec audits get 708739 -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-d, --database=STRING` | Project database name (optional, will search if not provided) |
| `-j, --json` | Output as JSON |

#### audits create

Create a new audit from a template.

```bash
# Create audit with default name (from template)
ec audits create nl_company_abc123 template-id-here

# Create audit with custom name
ec audits create nl_company_abc123 template-id-here -n "Safety Inspection Q1 2025"

# Assign to responsible person
ec audits create nl_company_abc123 template-id-here -r "inspector@example.com"

# Set due date
ec audits create nl_company_abc123 template-id-here -d "2025-12-31T17:00:00Z"

# Add tags
ec audits create nl_company_abc123 template-id-here -t "safety" -t "Q1"

# Full example
ec audits create nl_company_abc123 template-id-here \
  -n "Safety Inspection Q1 2025" \
  -r "inspector@example.com" \
  -d "2025-03-31T17:00:00Z" \
  -t "safety" -t "quarterly"

# Output as JSON
ec audits create nl_company_abc123 template-id-here -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-n, --name=STRING` | Audit name (optional, defaults to template name) |
| `-r, --responsible=STRING` | Responsible person email |
| `-d, --due-date=STRING` | Due date (ISO 8601 format, e.g., 2025-12-31T23:59:59Z) |
| `-t, --tags=TAGS,...` | Tags to add (can be specified multiple times) |
| `-j, --json` | Output as JSON |

---

### templates

Manage audit templates.

#### templates list

List audit templates for a project.

```bash
# List all templates
ec templates list nl_company_abc123

# Search by name
ec templates list nl_company_abc123 -s "safety"

# Filter by group
ec templates list nl_company_abc123 -g "group-id-here"

# Only show published templates
ec templates list nl_company_abc123 -p

# Include archived templates
ec templates list nl_company_abc123 -a

# Pagination
ec templates list nl_company_abc123 -l 100 --page 0

# Output as JSON
ec templates list nl_company_abc123 -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-s, --search=STRING` | Search by name |
| `-g, --group-id=STRING` | Filter by group ID |
| `-p, --published` | Only show published templates |
| `-a, --archived` | Include archived templates |
| `-l, --limit=50` | Maximum number of templates to return |
| `--page=0` | Page number (0-based) |
| `-j, --json` | Output as JSON |

#### templates get

Get details for a specific template.

```bash
# Get template details
ec templates get nl_company_abc123 template-id-here

# Output as JSON (includes full template structure with questions)
ec templates get nl_company_abc123 template-id-here -j
```

#### templates update

Update an audit template.

```bash
# Update template name
ec templates update nl_company_abc123 template-id-here -n "New Template Name"

# Update description
ec templates update nl_company_abc123 template-id-here -d "New description"

# Set tags (replaces existing tags)
ec templates update nl_company_abc123 template-id-here -t "safety" -t "inspection"

# Update multiple fields
ec templates update nl_company_abc123 template-id-here \
  -n "Safety Inspection Template v2" \
  -d "Updated template for safety inspections" \
  -t "safety" -t "v2"
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-n, --name=STRING` | New template name |
| `-d, --description=STRING` | New description |
| `-t, --tags=TAGS,...` | Tags to set (replaces existing) |

#### templates groups list

List template groups for a project.

```bash
# List all template groups
ec templates groups list nl_company_abc123

# Search by name
ec templates groups list nl_company_abc123 -s "safety"

# Include archived groups
ec templates groups list nl_company_abc123 -a

# Output as JSON
ec templates groups list nl_company_abc123 -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-s, --search=STRING` | Search by name |
| `-a, --archived` | Include archived groups |
| `-l, --limit=50` | Maximum number of groups to return |
| `-p, --page=0` | Page number (0-based) |
| `-j, --json` | Output as JSON |

---

### maps

Manage maps (drawings).

#### maps list

List maps for a project.

```bash
# List all maps
ec maps list nl_company_abc123

# Search by name
ec maps list nl_company_abc123 -s "floor plan"

# Filter by group
ec maps list nl_company_abc123 -g "group-id-here"

# Filter by tag
ec maps list nl_company_abc123 -t "architecture"

# Include archived maps
ec maps list nl_company_abc123 -a

# Show all maps (bypass role filtering)
ec maps list nl_company_abc123 --all-maps

# Pagination and sorting
ec maps list nl_company_abc123 -l 100 -p 0
ec maps list nl_company_abc123 -o modified --asc

# Output as JSON
ec maps list nl_company_abc123 -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-g, --group-id=STRING` | Filter by map group ID |
| `-s, --search=STRING` | Search by name |
| `-t, --tag=STRING` | Filter by tag |
| `-a, --archived` | Include archived maps |
| `--all-maps` | Show all maps (bypass role filtering) |
| `-l, --limit=50` | Maximum number of maps to return |
| `-p, --page=0` | Page number (0-based) |
| `-o, --sort="created"` | Sort by field |
| `--asc` | Sort in ascending order |
| `-j, --json` | Output as JSON |

#### maps get

Get details for a specific map.

```bash
# Get map by ID (searches across all projects)
ec maps get abc123def456789

# Specify project database for faster lookup
ec maps get abc123def456789 -d nl_company_abc123

# Output as JSON
ec maps get abc123def456789 -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-d, --database=STRING` | Project database name (optional, will search if not provided) |
| `-j, --json` | Output as JSON |

#### maps groups list

List map groups for a project.

```bash
# List all map groups
ec maps groups list nl_company_abc123

# Search by name
ec maps groups list nl_company_abc123 -s "floor"

# Include archived groups
ec maps groups list nl_company_abc123 -a

# Output as JSON
ec maps groups list nl_company_abc123 -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-s, --search=STRING` | Search by name |
| `-a, --archived` | Include archived groups |
| `-l, --limit=50` | Maximum number of groups to return |
| `-p, --page=0` | Page number (0-based) |
| `-j, --json` | Output as JSON |

---

### files

Manage files (attachments/documents).

#### files list

List files for a project.

```bash
# List all files
ec files list nl_company_abc123

# Search by name
ec files list nl_company_abc123 -s "report"

# Filter by group
ec files list nl_company_abc123 -g "group-id-here"

# Filter by tag
ec files list nl_company_abc123 -t "documentation"

# Include archived files
ec files list nl_company_abc123 -a

# Pagination and sorting
ec files list nl_company_abc123 -l 100 -p 0
ec files list nl_company_abc123 -o modified --asc
ec files list nl_company_abc123 -o name

# Output as JSON
ec files list nl_company_abc123 -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-g, --group-id=STRING` | Filter by file group ID |
| `-s, --search=STRING` | Search by name |
| `-t, --tag=STRING` | Filter by tag |
| `-a, --archived` | Include archived files |
| `-l, --limit=50` | Maximum number of files to return |
| `-p, --page=0` | Page number (0-based) |
| `-o, --sort="created"` | Sort by field (created, modified, name) |
| `--asc` | Sort in ascending order |
| `-j, --json` | Output as JSON |

#### files get

Get details for a specific file.

```bash
# Get file by ID (searches across all projects)
ec files get abc123def456789

# Specify project database for faster lookup
ec files get abc123def456789 -d nl_company_abc123

# Output as JSON
ec files get abc123def456789 -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-d, --database=STRING` | Project database name (optional, will search if not provided) |
| `-j, --json` | Output as JSON |

#### files groups list

List file groups for a project.

```bash
# List all file groups
ec files groups list nl_company_abc123

# Search by name
ec files groups list nl_company_abc123 -s "reports"

# Include archived groups
ec files groups list nl_company_abc123 -a

# Output as JSON
ec files groups list nl_company_abc123 -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-s, --search=STRING` | Search by name |
| `-a, --archived` | Include archived groups |
| `-l, --limit=50` | Maximum number of groups to return |
| `-p, --page=0` | Page number (0-based) |
| `-j, --json` | Output as JSON |

---

## Human IDs

Tickets and audits support "human IDs" for easier identification. A human ID is derived from the last 6 characters of the CouchDB document ID, reversed and uppercased.

**Example:**
- CouchDB ID: `d67c1aba0b017a1c9372e726c6512a1f`
- Human ID: `F1A215` (last 6 chars `12a1f` → reversed and uppercased)

Human IDs are shown in list views and can be used with `get` commands:

```bash
# These are equivalent
ec tickets get F1A215
ec tickets get d67c1aba0b017a1c9372e726c6512a1f
```

---

## JSON Output

All commands support JSON output with the `-j` or `--json` flag. This is useful for:

- Scripting and automation
- Piping to `jq` for further processing
- Integration with other tools

```bash
# Get ticket data and extract specific field with jq
ec tickets get CC455B -j | jq '.content.title'

# List all open tickets and count them
ec tickets list nl_company_abc123 -s "Open" -j | jq '.[] | .id' | wc -l

# Export project list to file
ec projects list -j > projects.json
```

---

## Cross-Project Search

Some commands support searching across all accessible projects:

```bash
# Search tickets across all projects (omit database argument)
ec tickets list

# Search audits across all projects
ec audits list

# Get ticket/audit by human ID (automatically searches all projects)
ec tickets get CC455B
ec audits get 708739
```

---

## Examples

### Daily Workflow

```bash
# Check who you're logged in as
ec whoami

# List your projects
ec projects list

# View open tickets for a project
ec tickets list nl_company_abc123 -s "Open"

# Get details on a specific ticket
ec tickets get CC455B

# Close a completed ticket
ec tickets close nl_company_abc123 ticket-id-here
```

### Creating Audits

```bash
# Find available templates
ec templates list nl_company_abc123 -p

# Create an audit from a template
ec audits create nl_company_abc123 template-id-here \
  -n "Weekly Safety Inspection" \
  -r "inspector@example.com" \
  -d "2025-02-28T17:00:00Z"
```

### Exporting Data

```bash
# Export all audits for a project
ec audits list nl_company_abc123 -l 1000 -j > audits.json

# Export completed audits only
ec audits list nl_company_abc123 -s "completed" -j > completed_audits.json
```

---

## Version

Check the CLI version:

```bash
ec -v
ec --version
```

---

## License

MIT License
