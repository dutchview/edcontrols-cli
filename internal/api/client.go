package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/mauricejumelet/edcontrols-cli/internal/config"
)

const baseURL = "https://web.edcontrols.com"

type Client struct {
	httpClient *http.Client
	token      string
	email      string
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{},
		token:      cfg.Token,
		email:      cfg.Email,
	}
}

func (c *Client) Email() string {
	return c.email
}

func (c *Client) doRequest(method, endpoint string, body io.Reader) ([]byte, error) {
	reqURL := baseURL + endpoint

	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Message != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Message)
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

type ErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Status  string `json:"status,omitempty"`
}

// SearchResult represents a paginated API response
type SearchResult struct {
	Size    int             `json:"size"`
	Page    int             `json:"page"`
	Hits    int             `json:"hits"`
	Results json.RawMessage `json:"results"`
}

// Project represents an EdControls project
type Project struct {
	ProjectID   string      `json:"projectId"`
	ProjectName string      `json:"projectName"`
	CouchDbID   string      `json:"couchDbId"`
	Location    string      `json:"location,omitempty"`
	StartDate   string      `json:"startDate,omitempty"`
	EndDate     string      `json:"endDate,omitempty"`
	IsActive    bool        `json:"isActive"`
	Archived    interface{} `json:"archived"` // null, datetime string, or bool
}

// TicketContent holds the content of a ticket
type TicketContent struct {
	Title  string  `json:"title,omitempty"`
	Body   string  `json:"body,omitempty"`
	Author *Person `json:"author,omitempty"`
}

// TicketState holds the state of a ticket
type TicketState struct {
	State string `json:"state,omitempty"`
}

// TicketDates holds date fields for a ticket
type TicketDates struct {
	DueDate        string `json:"dueDate,omitempty"`
	CreationDate   string `json:"creationDate,omitempty"`
	LastModified   string `json:"lastModifiedDate,omitempty"`
	CompletionDate string `json:"completionDate,omitempty"`
}

// Ticket represents an EdControls ticket
type Ticket struct {
	ID           string         `json:"id"`
	CouchDbID    string         `json:"couchDbId,omitempty"`
	Content      *TicketContent `json:"content,omitempty"`
	State        *TicketState   `json:"state,omitempty"`
	Dates        *TicketDates   `json:"dates,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
	GroupID      string         `json:"groupId,omitempty"`
	Database     string         `json:"database,omitempty"`
	Participants *Participants  `json:"participants,omitempty"`
}

// Person represents a participant person
type Person struct {
	Email string `json:"email,omitempty"`
	Type  string `json:"type,omitempty"`
}

// Participants holds participant info for a ticket or audit
type Participants struct {
	Responsible *Person  `json:"responsible,omitempty"`
	Informed    []Person `json:"informed,omitempty"`
	Consulted   []Person `json:"consulted,omitempty"`
}

// AuditDates holds date fields for an audit
type AuditDates struct {
	DueDate        string `json:"dueDate,omitempty"`
	CreationDate   string `json:"creationDate,omitempty"`
	LastModified   string `json:"lastModifiedDate,omitempty"`
	CompletionDate string `json:"completionDate,omitempty"`
}

// Audit represents an EdControls audit
type Audit struct {
	ID           string        `json:"id"`
	CouchDbID    string        `json:"couchDbId,omitempty"`
	Name         string        `json:"name"`
	Status       string        `json:"status"`
	Template     string        `json:"template,omitempty"`
	TemplateName string        `json:"templateName,omitempty"`
	TemplateID   string        `json:"templateId,omitempty"`
	Author       *Person       `json:"author,omitempty"`
	Dates        *AuditDates   `json:"dates,omitempty"`
	GroupID      string        `json:"groupId,omitempty"`
	Tags         []string      `json:"tags,omitempty"`
	Database     string        `json:"database,omitempty"`
	Participants *Participants `json:"participants,omitempty"`
}

// TemplateDates holds date fields for a template
type TemplateDates struct {
	CreationDate string `json:"creationDate,omitempty"`
	LastModified string `json:"lastModifiedDate,omitempty"`
}

// AuditTemplate represents an EdControls audit template
type AuditTemplate struct {
	ID          string         `json:"id"`
	CouchDbID   string         `json:"couchDbId,omitempty"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	IsPublished bool           `json:"isPublished"`
	Author      *Person        `json:"author,omitempty"`
	Dates       *TemplateDates `json:"dates,omitempty"`
	GroupID     string         `json:"groupId,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Database    string         `json:"database,omitempty"`
}

// CouchDBDocument represents a raw CouchDB document for direct access
type CouchDBDocument struct {
	ID   string `json:"_id"`
	Rev  string `json:"_rev"`
	Data map[string]interface{}
}

// ListProjectsOptions contains options for listing projects
type ListProjectsOptions struct {
	Search string
	Page   int
	Size   int
}

// ListProjects returns all projects accessible to the authenticated user
func (c *Client) ListProjects(opts ListProjectsOptions) ([]Project, int, error) {
	endpoint := fmt.Sprintf("/api/v2/licenseserver/user/%s/projects", url.PathEscape(c.email))
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, err
	}

	// The response is a map with project IDs as keys
	var projectsMap map[string]map[string]Project
	if err := json.Unmarshal(body, &projectsMap); err != nil {
		return nil, 0, fmt.Errorf("parsing response: %w", err)
	}

	var projects []Project
	if projectsData, ok := projectsMap["projects"]; ok {
		for _, p := range projectsData {
			// Apply search filter if provided
			if opts.Search != "" {
				searchLower := strings.ToLower(opts.Search)
				if !strings.Contains(strings.ToLower(p.ProjectName), searchLower) &&
					!strings.Contains(strings.ToLower(p.ProjectID), searchLower) {
					continue
				}
			}
			projects = append(projects, p)
		}
	}

	return projects, len(projects), nil
}

// GetProject returns a single project by database name
func (c *Client) GetProject(database string) (*Project, error) {
	endpoint := fmt.Sprintf("/api/v2/data/projects/%s", url.PathEscape(database))
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var project Project
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &project, nil
}

// ListTicketsOptions contains options for listing tickets
type ListTicketsOptions struct {
	Database    string // Required
	Status      string // Comma-separated: "Open,In Progress,Done"
	SearchTitle string
	SearchByID  string // Search by ticket ID (human ID or partial CouchDB ID)
	Responsible string
	GroupID     string
	MapID       string
	Tag         string
	Archived    bool
	SortBy      string // COMPLETIONDATE, LASTMODIFIEDDATE, CREATIONDATE, AUTHOR, TITLE, DUEDATE
	SortOrder   string // ASC or DESC
	Page        int
	Size        int
}

// ListTickets returns tickets for a project
func (c *Client) ListTickets(opts ListTicketsOptions) ([]Ticket, int, error) {
	params := url.Values{}
	params.Set("database", opts.Database)

	if opts.Status != "" {
		params.Set("status", opts.Status)
	}
	if opts.SearchTitle != "" {
		params.Set("searchByTitle", opts.SearchTitle)
	}
	if opts.SearchByID != "" {
		params.Set("searchById", opts.SearchByID)
	}
	if opts.Responsible != "" {
		params.Set("searchByResponsible", opts.Responsible)
	}
	if opts.GroupID != "" {
		params.Set("groupId", opts.GroupID)
	}
	if opts.MapID != "" {
		params.Set("mapId", opts.MapID)
	}
	if opts.Tag != "" {
		params.Set("tag", opts.Tag)
	}
	if opts.Archived {
		params.Set("archived", "true")
	}
	if opts.SortBy != "" {
		params.Set("sortby", opts.SortBy)
	}
	if opts.SortOrder != "" {
		params.Set("sortOrder", opts.SortOrder)
	}
	if opts.Page > 0 {
		params.Set("page", fmt.Sprintf("%d", opts.Page))
	}
	if opts.Size > 0 {
		params.Set("size", fmt.Sprintf("%d", opts.Size))
	} else {
		params.Set("size", "50")
	}

	endpoint := "/api/v2/data/tickets?" + params.Encode()
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, fmt.Errorf("parsing response: %w", err)
	}

	var tickets []Ticket
	if err := json.Unmarshal(result.Results, &tickets); err != nil {
		return nil, 0, fmt.Errorf("parsing tickets: %w", err)
	}

	return tickets, result.Hits, nil
}

// SearchTicketsByID searches for tickets by ID across multiple projects using the POST search endpoint
func (c *Client) SearchTicketsByID(projectIDs []string, searchID string) ([]Ticket, error) {
	reqBody := map[string]interface{}{
		"projects":      projectIDs,
		"searchById":    searchID,
		"sortOrder":     "DESC",
		"sortby":        "LASTMODIFIEDDATE",
		"includeFields": []string{"couchDbId"},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	endpoint := "/api/v2/data/tickets/search?size=10&page=0"
	body, err := c.doRequest("POST", endpoint, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	var tickets []Ticket
	if err := json.Unmarshal(result.Results, &tickets); err != nil {
		return nil, fmt.Errorf("parsing tickets: %w", err)
	}

	return tickets, nil
}

// GetTicket returns a single ticket
func (c *Client) GetTicket(database, ticketID string) (*Ticket, error) {
	endpoint := fmt.Sprintf("/api/v2/data/tickets/%s/%s", url.PathEscape(database), url.PathEscape(ticketID))
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var ticket Ticket
	if err := json.Unmarshal(body, &ticket); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &ticket, nil
}

// UpdateTicketOptions contains options for updating a ticket
type UpdateTicketOptions struct {
	Status      *string
	Responsible *string
	DueDate     *string
	Tags        []string
}

// UpdateTicket updates a ticket via the securedata endpoint
func (c *Client) UpdateTicket(database, ticketID string, opts UpdateTicketOptions) error {
	// First, get the current document
	getEndpoint := fmt.Sprintf("/api/v1/securedata/%s/%s", url.PathEscape(database), url.PathEscape(ticketID))
	body, err := c.doRequest("GET", getEndpoint, nil)
	if err != nil {
		return fmt.Errorf("fetching ticket: %w", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(body, &doc); err != nil {
		return fmt.Errorf("parsing ticket: %w", err)
	}

	// Apply updates
	if opts.Status != nil {
		doc["status"] = *opts.Status
	}
	if opts.Responsible != nil {
		if participants, ok := doc["participants"].(map[string]interface{}); ok {
			participants["responsible"] = *opts.Responsible
		} else {
			doc["participants"] = map[string]interface{}{"responsible": *opts.Responsible}
		}
	}
	if opts.DueDate != nil {
		if dates, ok := doc["dates"].(map[string]interface{}); ok {
			dates["dueDate"] = *opts.DueDate
		} else {
			doc["dates"] = map[string]interface{}{"dueDate": *opts.DueDate}
		}
	}
	if opts.Tags != nil {
		doc["tags"] = opts.Tags
	}

	// PUT the updated document
	jsonBody, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshaling ticket: %w", err)
	}

	_, err = c.doRequest("PUT", getEndpoint, strings.NewReader(string(jsonBody)))
	return err
}

// ListAuditsOptions contains options for listing audits
type ListAuditsOptions struct {
	Database    string // Required
	Template    string
	Status      string
	SearchTitle string
	Auditor     string
	GroupID     string
	Tag         string
	Archived    bool
	SortBy      string
	SortOrder   string
	Page        int
	Size        int
}

// ListAudits returns audits for a project
func (c *Client) ListAudits(opts ListAuditsOptions) ([]Audit, int, error) {
	params := url.Values{}
	params.Set("database", opts.Database)

	if opts.Template != "" {
		params.Set("template", opts.Template)
	}
	if opts.Status != "" {
		params.Set("status", opts.Status)
	}
	if opts.SearchTitle != "" {
		params.Set("searchByTitle", opts.SearchTitle)
	}
	if opts.Auditor != "" {
		params.Set("searchByAuditor", opts.Auditor)
	}
	if opts.GroupID != "" {
		params.Set("groupid", opts.GroupID)
	}
	if opts.Tag != "" {
		params.Set("tag", opts.Tag)
	}
	if opts.Archived {
		params.Set("archived", "true")
	}
	if opts.SortBy != "" {
		params.Set("sortby", opts.SortBy)
	}
	if opts.SortOrder != "" {
		params.Set("sortOrder", opts.SortOrder)
	}
	if opts.Page > 0 {
		params.Set("page", fmt.Sprintf("%d", opts.Page))
	}
	if opts.Size > 0 {
		params.Set("size", fmt.Sprintf("%d", opts.Size))
	} else {
		params.Set("size", "50")
	}

	endpoint := "/api/v2/data/audits?" + params.Encode()
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, fmt.Errorf("parsing response: %w", err)
	}

	var audits []Audit
	if err := json.Unmarshal(result.Results, &audits); err != nil {
		return nil, 0, fmt.Errorf("parsing audits: %w", err)
	}

	return audits, result.Hits, nil
}

// GetAudit returns a single audit
func (c *Client) GetAudit(database, auditID string) (*Audit, error) {
	endpoint := fmt.Sprintf("/api/v2/data/projects/%s/audits/%s", url.PathEscape(database), url.PathEscape(auditID))
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var audit Audit
	if err := json.Unmarshal(body, &audit); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &audit, nil
}

// CreateAuditOptions contains options for creating an audit from a template
type CreateAuditOptions struct {
	Name        string   `json:"name,omitempty"`
	Responsible string   `json:"responsible,omitempty"`
	DueDate     string   `json:"dueDate,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// CreateAudit creates a new audit from a template
func (c *Client) CreateAudit(database, templateID string, opts CreateAuditOptions) (*Audit, error) {
	jsonBody, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	endpoint := fmt.Sprintf("/api/v2/data/projects/%s/audittemplates/%s/createAudit",
		url.PathEscape(database), url.PathEscape(templateID))
	body, err := c.doRequest("POST", endpoint, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	var audit Audit
	if err := json.Unmarshal(body, &audit); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &audit, nil
}

// ListAuditTemplatesOptions contains options for listing audit templates
type ListAuditTemplatesOptions struct {
	Database    string // Required
	GroupID     string
	IsPublished *bool
	SearchName  string
	Archived    bool
	Page        int
	Size        int
}

// ListAuditTemplates returns audit templates for a project
func (c *Client) ListAuditTemplates(opts ListAuditTemplatesOptions) ([]AuditTemplate, int, error) {
	params := url.Values{}
	params.Set("database", opts.Database)

	if opts.GroupID != "" {
		params.Set("groupid", opts.GroupID)
	}
	if opts.IsPublished != nil {
		params.Set("isPublished", fmt.Sprintf("%t", *opts.IsPublished))
	}
	if opts.SearchName != "" {
		params.Set("searchByName", opts.SearchName)
	}
	if opts.Archived {
		params.Set("archived", "true")
	}
	if opts.Page > 0 {
		params.Set("page", fmt.Sprintf("%d", opts.Page))
	}
	if opts.Size > 0 {
		params.Set("size", fmt.Sprintf("%d", opts.Size))
	} else {
		params.Set("size", "50")
	}

	endpoint := "/api/v2/data/audittemplates?" + params.Encode()
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, fmt.Errorf("parsing response: %w", err)
	}

	var templates []AuditTemplate
	if err := json.Unmarshal(result.Results, &templates); err != nil {
		return nil, 0, fmt.Errorf("parsing templates: %w", err)
	}

	return templates, result.Hits, nil
}

// GetAuditTemplate returns a single audit template
func (c *Client) GetAuditTemplate(database, templateID string) (*AuditTemplate, error) {
	endpoint := fmt.Sprintf("/api/v2/data/projects/%s/audittemplates/%s",
		url.PathEscape(database), url.PathEscape(templateID))
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var template AuditTemplate
	if err := json.Unmarshal(body, &template); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &template, nil
}

// UpdateAuditTemplate updates an audit template via the securedata endpoint
func (c *Client) UpdateAuditTemplate(database, templateID string, updates map[string]interface{}) error {
	// First, get the current document
	getEndpoint := fmt.Sprintf("/api/v1/securedata/%s/%s", url.PathEscape(database), url.PathEscape(templateID))
	body, err := c.doRequest("GET", getEndpoint, nil)
	if err != nil {
		return fmt.Errorf("fetching template: %w", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(body, &doc); err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	// Apply updates
	for k, v := range updates {
		doc[k] = v
	}

	// PUT the updated document
	jsonBody, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshaling template: %w", err)
	}

	_, err = c.doRequest("PUT", getEndpoint, strings.NewReader(string(jsonBody)))
	return err
}

// GetDocument returns a raw CouchDB document
func (c *Client) GetDocument(database, docID string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/api/v1/securedata/%s/%s", url.PathEscape(database), url.PathEscape(docID))
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return doc, nil
}

// UpdateDocument updates a raw CouchDB document
func (c *Client) UpdateDocument(database, docID string, doc map[string]interface{}) error {
	endpoint := fmt.Sprintf("/api/v1/securedata/%s/%s", url.PathEscape(database), url.PathEscape(docID))

	jsonBody, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshaling document: %w", err)
	}

	_, err = c.doRequest("PUT", endpoint, strings.NewReader(string(jsonBody)))
	return err
}
