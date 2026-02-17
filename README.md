<p align="center">
  <img src="assets/edcontrols-logo.svg" alt="EdControls Logo" width="300">
</p>

# EdControls CLI

A command-line interface for interacting with the EdControls platform. Manage projects, tickets, audits, templates, maps, and files directly from your terminal.

<p align="center">
  <img src="assets/demo.gif" alt="EdControls CLI Demo" width="800">
</p>

## Installation

### macOS (Homebrew)

The easiest way to install on macOS:

```bash
brew tap dutchview/tap
brew install ec
```

To upgrade to the latest version:

```bash
brew upgrade ec
```

### macOS / Linux (Direct Download)

Download the latest release from the [GitHub Releases](https://github.com/dutchview/edcontrols-cli/releases) page.

**macOS (Apple Silicon):**
```bash
curl -Lo ec.tar.gz https://github.com/dutchview/edcontrols-cli/releases/latest/download/edcontrols-cli_$(curl -s https://api.github.com/repos/dutchview/edcontrols-cli/releases/latest | grep tag_name | cut -d '"' -f 4 | tr -d 'v')_darwin_arm64.tar.gz
tar -xzf ec.tar.gz
sudo mv ec /usr/local/bin/
rm ec.tar.gz
```

**macOS (Intel):**
```bash
curl -Lo ec.tar.gz https://github.com/dutchview/edcontrols-cli/releases/latest/download/edcontrols-cli_$(curl -s https://api.github.com/repos/dutchview/edcontrols-cli/releases/latest | grep tag_name | cut -d '"' -f 4 | tr -d 'v')_darwin_amd64.tar.gz
tar -xzf ec.tar.gz
sudo mv ec /usr/local/bin/
rm ec.tar.gz
```

**Linux (x86_64):**
```bash
curl -Lo ec.tar.gz https://github.com/dutchview/edcontrols-cli/releases/latest/download/edcontrols-cli_$(curl -s https://api.github.com/repos/dutchview/edcontrols-cli/releases/latest | grep tag_name | cut -d '"' -f 4 | tr -d 'v')_linux_amd64.tar.gz
tar -xzf ec.tar.gz
sudo mv ec /usr/local/bin/
rm ec.tar.gz
```

**Linux (ARM64):**
```bash
curl -Lo ec.tar.gz https://github.com/dutchview/edcontrols-cli/releases/latest/download/edcontrols-cli_$(curl -s https://api.github.com/repos/dutchview/edcontrols-cli/releases/latest | grep tag_name | cut -d '"' -f 4 | tr -d 'v')_linux_arm64.tar.gz
tar -xzf ec.tar.gz
sudo mv ec /usr/local/bin/
rm ec.tar.gz
```

### Windows

1. Download the latest `.zip` file from [GitHub Releases](https://github.com/dutchview/edcontrols-cli/releases):
   - `edcontrols-cli_X.X.X_windows_amd64.zip` for 64-bit Windows
   - `edcontrols-cli_X.X.X_windows_arm64.zip` for Windows ARM

2. Extract the `ec.exe` file

3. Move it to a directory in your PATH, or add the directory to your PATH environment variable

**PowerShell (run as Administrator):**
```powershell
# Create directory if it doesn't exist
New-Item -ItemType Directory -Force -Path "$env:LOCALAPPDATA\Programs\ec"

# Move ec.exe to the directory (after extracting the zip)
Move-Item -Path ".\ec.exe" -Destination "$env:LOCALAPPDATA\Programs\ec\"

# Add to PATH (permanent, requires restart of terminal)
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$env:LOCALAPPDATA\Programs\ec*") {
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$env:LOCALAPPDATA\Programs\ec", "User")
}
```

### From Source

Requires Go 1.21 or later.

```bash
git clone https://github.com/dutchview/edcontrols-cli.git
cd edcontrols-cli
go build -o ec .
```

Move the binary to a directory in your PATH:

```bash
# macOS/Linux
sudo mv ec /usr/local/bin/

# Or install to user directory (no sudo required)
mv ec ~/.local/bin/
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
EDCONTROLS_ACCESS_TOKEN=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
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

# Filter by status (available: created, started, completed)
ec tickets list nl_company_abc123 -s "created"
ec tickets list nl_company_abc123 -s "started"
ec tickets list nl_company_abc123 -s "completed"

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
| `-s, --status=STRING` | Filter by status (created, started, completed) |
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

# Specify project ID for faster lookup
ec tickets get CC455B -p nl_company_abc123

# Output as JSON
ec tickets get CC455B -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-p, --project=STRING` | Project ID (optional, will search if not provided) |
| `-j, --json` | Output as JSON |

#### tickets update

Update ticket fields including title, description, due date, responsible, status, and comments.

```bash
# View current ticket values (no flags)
ec tickets update nl_company_abc123 ticket-id-here

# Update title
ec tickets update nl_company_abc123 ticket-id-here -t "New Title"

# Update description (supports HTML)
ec tickets update nl_company_abc123 ticket-id-here -d "<p>New description</p>"

# Set due date
ec tickets update nl_company_abc123 ticket-id-here --due-date "2026-03-15T12:00:00.000Z"

# Clear due date
ec tickets update nl_company_abc123 ticket-id-here --clear-due

# Assign responsible (sets status to "started")
ec tickets update nl_company_abc123 ticket-id-here -r user@example.com

# Clear responsible (sets status back to "created")
ec tickets update nl_company_abc123 ticket-id-here --clear-responsible

# Mark as completed (uses existing responsible or current user)
ec tickets update nl_company_abc123 ticket-id-here --complete

# Assign and complete in one command
ec tickets update nl_company_abc123 ticket-id-here -r user@example.com --complete

# Add a comment
ec tickets update nl_company_abc123 ticket-id-here -m "This is a comment"

# Multiple updates at once
ec tickets update nl_company_abc123 ticket-id-here \
  -t "Updated Title" \
  -d "<p>New description</p>" \
  -r user@example.com \
  --due-date "2026-03-15T12:00:00.000Z" \
  -m "Added via CLI"
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-t, --title=STRING` | New title for the ticket |
| `-d, --description=STRING` | New description (supports HTML) |
| `--due-date=STRING` | Due date (ISO 8601 format) |
| `--clear-due` | Clear the due date |
| `-r, --responsible=STRING` | Assign to this email (sets status to started) |
| `--clear-responsible` | Clear the responsible (sets status to created) |
| `--complete` | Mark as completed |
| `-m, --comment=STRING` | Add a comment to the ticket |

**Notes:**
- All updates are tracked in the operation timeline
- HTML in descriptions and comments is sanitized to prevent XSS
- Running without flags shows current values

#### tickets assign

Assign a ticket to someone.

```bash
ec tickets assign nl_company_abc123 ticket-id-here john@example.com
```

#### tickets open

Reopen a ticket (set status to created).

```bash
ec tickets open nl_company_abc123 ticket-id-here
```

#### tickets close

Close a ticket (set status to completed).

```bash
ec tickets close nl_company_abc123 ticket-id-here
```

#### tickets archive

Archive a ticket.

```bash
ec tickets archive nl_company_abc123 ticket-id-here
```

#### tickets unarchive

Unarchive (restore) a ticket.

```bash
ec tickets unarchive nl_company_abc123 ticket-id-here
```

#### tickets delete

Permanently delete a ticket.

```bash
ec tickets delete nl_company_abc123 ticket-id-here
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
| `-s, --status=STRING` | Filter by status (started, In Progress, completed) |
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

# Specify project ID for faster lookup
ec audits get 708739 -p nl_company_abc123

# Output as JSON (includes full audit data with questions/answers)
ec audits get 708739 -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-p, --project=STRING` | Project ID (optional, will search if not provided) |
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

#### templates create

Create a new audit template in a template group.

```bash
# Create a template in a group
ec templates create nl_company_abc123 group-id-here "My New Template"

# Create with tags
ec templates create nl_company_abc123 group-id-here "Safety Checklist" -t "safety" -t "checklist"

# Output as JSON
ec templates create nl_company_abc123 group-id-here "My Template" -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-t, --tags=TAGS,...` | Tags to add (can be specified multiple times) |
| `-j, --json` | Output as JSON |

**Notes:**
- Templates are created with an empty "Questions" category
- New templates start as unpublished
- Add questions via the EdControls web interface

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

#### templates publish

Publish an audit template (makes it available for creating audits).

```bash
ec templates publish nl_company_abc123 template-id-here
```

#### templates unpublish

Unpublish an audit template.

```bash
ec templates unpublish nl_company_abc123 template-id-here
```

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

#### templates groups create

Create a new template group.

```bash
# Create a template group
ec templates groups create nl_company_abc123 "My New Group"

# Output as JSON
ec templates groups create nl_company_abc123 "Safety Templates" -j
```

**Flags:**

| Flag | Description |
|------|-------------|
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

# Specify project ID for faster lookup
ec maps get abc123def456789 -p nl_company_abc123

# Output as JSON
ec maps get abc123def456789 -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-p, --project=STRING` | Project ID (optional, will search if not provided) |
| `-j, --json` | Output as JSON |

#### maps add

Upload a PDF or image file and convert it to a tiled map.

```bash
# Add a map from a PDF file
ec maps add nl_company_abc123 file-group-id-here /path/to/floorplan.pdf

# Add with custom name
ec maps add nl_company_abc123 file-group-id-here /path/to/floorplan.pdf -n "Ground Floor"

# Add with tags
ec maps add nl_company_abc123 file-group-id-here /path/to/floorplan.pdf -t "architecture" -t "floor-1"
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-n, --name=STRING` | Map name (defaults to filename) |
| `-t, --tags=TAGS,...` | Tags to add (can be specified multiple times) |

**Notes:**
- Only PDF, PNG, and JPG files can be converted to maps
- The file is first uploaded to the file group, then converted to a tiled map
- Conversion is queued and may take some time to complete

#### maps delete

Delete a map.

```bash
ec maps delete nl_company_abc123 map-id-here
```

#### maps tags

View or update tags on a map.

```bash
# View current tags
ec maps tags nl_company_abc123 map-id-here

# Set tags (replaces existing)
ec maps tags nl_company_abc123 map-id-here -t tag1 -t tag2

# Clear all tags
ec maps tags nl_company_abc123 map-id-here --clear
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-t, --tags=TAGS,...` | Tags to set (replaces existing tags) |
| `--clear` | Clear all tags from the map |

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
| `-a, --archived` | Include archived files (shown with status "archived" or "deleted") |
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

# Specify project ID for faster lookup
ec files get abc123def456789 -p nl_company_abc123

# Output as JSON
ec files get abc123def456789 -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-p, --project=STRING` | Project ID (optional, will search if not provided) |
| `-j, --json` | Output as JSON |

#### files add

Upload a new file to a project.

```bash
# Upload a file to a file group
ec files add nl_company_abc123 group-id-here /path/to/document.pdf

# Upload with custom name
ec files add nl_company_abc123 group-id-here /path/to/document.pdf -n "Custom Name.pdf"

# Upload with tags
ec files add nl_company_abc123 group-id-here /path/to/document.pdf -t "report" -t "2026"

# Output as JSON
ec files add nl_company_abc123 group-id-here /path/to/document.pdf -j
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-n, --name=STRING` | Custom file name (defaults to original filename) |
| `-t, --tags=TAGS,...` | Tags to add (can be specified multiple times) |
| `-j, --json` | Output as JSON |

#### files download

Download a file.

```bash
# Download file (uses original filename)
ec files download file-id-here

# Download with specific project ID
ec files download file-id-here -p nl_company_abc123

# Download to custom path
ec files download file-id-here -o /path/to/save/file.pdf
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-p, --project=STRING` | Project ID (optional, will search if not provided) |
| `-o, --output=STRING` | Output file path (defaults to original filename) |

#### files archive / unarchive

Archive or unarchive a file.

```bash
# Archive a file
ec files archive nl_company_abc123 file-id-here

# Unarchive a file
ec files unarchive nl_company_abc123 file-id-here
```

#### files delete

Delete a file.

```bash
ec files delete nl_company_abc123 file-id-here
```

#### files tags

View or update tags on a file.

```bash
# View current tags
ec files tags nl_company_abc123 file-id-here

# Set tags (replaces existing)
ec files tags nl_company_abc123 file-id-here -t tag1 -t tag2

# Clear all tags
ec files tags nl_company_abc123 file-id-here --clear
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-t, --tags=TAGS,...` | Tags to set (replaces existing tags) |
| `--clear` | Clear all tags from the file |

#### files to-map

Convert a file to a map (tiled drawing). Only PDF, PNG, and JPG files can be converted.

```bash
ec files to-map nl_company_abc123 file-id-here
```

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

## Status Values

### Ticket Statuses

Tickets have three possible status values:

| Status | Description |
|--------|-------------|
| `created` | New ticket, no responsible person assigned |
| `started` | Ticket has been assigned to someone |
| `completed` | Ticket has been marked as done |

**Status transitions:**
- When a responsible person is assigned, status automatically changes to `started`
- When the responsible is cleared, status reverts to `created`
- The `tickets close` command sets status to `completed`
- The `tickets open` command sets status back to `created`

### Audit Statuses

Audits have three possible status values:

| Status | Description |
|--------|-------------|
| `started` | Audit has been started |
| `In Progress` | Audit is actively being worked on |
| `completed` | Audit has been completed |

Note: Both `started` and `In Progress` indicate an active audit. The difference is historical - newer audits typically use `started`.

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

# List all new tickets and count them
ec tickets list nl_company_abc123 -s "created" -j | jq '.[] | .id' | wc -l

# Export project list to file
ec projects list -j > projects.json
```

---

## Cross-Project Search

Some commands support searching across all accessible projects:

```bash
# Search tickets across all projects (omit project ID argument)
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

# View new/unassigned tickets for a project
ec tickets list nl_company_abc123 -s "created"

# Get details on a specific ticket
ec tickets get CC455B

# Update a ticket with multiple changes
ec tickets update nl_company_abc123 ticket-id-here \
  -t "Updated Title" \
  -r user@example.com \
  -m "Status update from CLI"

# Mark a ticket as completed
ec tickets update nl_company_abc123 ticket-id-here --complete

# Close a completed ticket
ec tickets close nl_company_abc123 ticket-id-here
```

### Managing Tickets

```bash
# Assign a ticket to someone
ec tickets assign nl_company_abc123 ticket-id user@example.com

# Add a comment to a ticket
ec tickets update nl_company_abc123 ticket-id -m "This is my comment"

# Set a due date
ec tickets update nl_company_abc123 ticket-id --due-date "2026-03-15T12:00:00.000Z"

# Archive old tickets
ec tickets archive nl_company_abc123 ticket-id-here

# Restore archived tickets
ec tickets unarchive nl_company_abc123 ticket-id-here

# Permanently delete a ticket
ec tickets delete nl_company_abc123 ticket-id-here
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

# Create audit with tags
ec audits create nl_company_abc123 template-id-here \
  -n "Q1 Review" \
  -t "quarterly" -t "review"
```

### Managing Templates

```bash
# Create a new template group
ec templates groups create nl_company_abc123 "Safety Checklists"

# Create a new template in the group
ec templates create nl_company_abc123 group-id-here "Fire Safety Checklist" \
  -t "safety" -t "fire"

# Update template metadata
ec templates update nl_company_abc123 template-id-here \
  -n "Updated Template Name" \
  -d "New description"
```

### File Management

```bash
# Upload a file
ec files add nl_company_abc123 group-id /path/to/document.pdf

# Download a file
ec files download file-id-here -o ./downloaded.pdf

# Update file tags
ec files tags nl_company_abc123 file-id -t "report" -t "2026"

# Convert a PDF to a map
ec files to-map nl_company_abc123 file-id-here

# Archive old files
ec files archive nl_company_abc123 file-id-here
```

### Map Management

```bash
# Upload and convert a PDF to a tiled map
ec maps add nl_company_abc123 file-group-id /path/to/floorplan.pdf -n "Ground Floor"

# Update map tags
ec maps tags nl_company_abc123 map-id -t "architecture" -t "floor-1"

# Delete a map
ec maps delete nl_company_abc123 map-id-here
```

### Exporting Data

```bash
# Export all audits for a project
ec audits list nl_company_abc123 -l 1000 -j > audits.json

# Export completed audits only
ec audits list nl_company_abc123 -s "completed" -j > completed_audits.json

# Export all tickets with full details
ec tickets list nl_company_abc123 -l 1000 -j > tickets.json

# Export project list
ec projects list -j > projects.json
```

### Scripting with jq

```bash
# Get ticket titles
ec tickets list nl_company_abc123 -j | jq '.[].content.title'

# Count new/unassigned tickets
ec tickets list nl_company_abc123 -s "created" -j | jq 'length'

# Get audit IDs for a specific template
ec audits list nl_company_abc123 -t template-id -j | jq '.[].couchDbId'

# Extract responsible emails from tickets
ec tickets list nl_company_abc123 -j | jq '.[].participants.responsible.email // empty'
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
