package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mauricejumelet/edcontrols-cli/internal/config"
)

const baseURL = "https://web.edcontrols.com"

type Client struct {
	httpClient *http.Client
	token      string
	email      string // Cached after first fetch
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{},
		token:      cfg.Token,
	}
}

// UserInfo represents the current user's information from the auth endpoint
type UserInfo struct {
	Email string `json:"email"`
	Name  struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	} `json:"name"`
	CompanyName string   `json:"companyName"`
	Roles       []string `json:"roles"`
	Enabled     bool     `json:"enabled"`
}

// GetCurrentUser fetches the current user's information from the auth endpoint
func (c *Client) GetCurrentUser() (*UserInfo, error) {
	body, err := c.doRequest("GET", "/api/v1/users/me", nil)
	if err != nil {
		return nil, err
	}

	var userInfo UserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("parsing user info: %w", err)
	}

	// Cache the email
	c.email = userInfo.Email

	return &userInfo, nil
}

// Email returns the current user's email, fetching it if not cached
func (c *Client) Email() (string, error) {
	if c.email != "" {
		return c.email, nil
	}

	userInfo, err := c.GetCurrentUser()
	if err != nil {
		return "", fmt.Errorf("fetching user info: %w", err)
	}

	return userInfo.Email, nil
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
	Archived    interface{} `json:"archived"`   // null, datetime string, or bool
	Contract    string      `json:"contract"`   // Contract document ID
	Geomap      bool        `json:"geomap"`     // Whether geomap is enabled
	IsGlacier   bool        `json:"isGlacier"`  // Whether project is in glacier storage
}

// Contract represents an EdControls contract/client
type Contract struct {
	ID           string   `json:"_id,omitempty"`
	Rev          string   `json:"_rev,omitempty"`
	Name         string   `json:"name"`
	Type         string   `json:"type,omitempty"`
	Projects     []string `json:"projects,omitempty"`
	Active       bool     `json:"contractActive,omitempty"`
	IsDemo       bool     `json:"isDemoContract,omitempty"`
	PricePlan    string   `json:"pricePlan,omitempty"`
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
	MapID        string         `json:"map,omitempty"`
	Database     string         `json:"database,omitempty"`
	Participants *Participants  `json:"participants,omitempty"`
}

// Map represents an EdControls map (drawing)
type Map struct {
	ID        string      `json:"id,omitempty"`
	CouchID   string      `json:"_id,omitempty"`
	CouchDbID string      `json:"couchDbId,omitempty"`
	Name      string      `json:"name"`
	GroupID   string      `json:"groupId,omitempty"`
	GroupName string      `json:"groupName,omitempty"`
	Dates     *MapDates   `json:"dates,omitempty"`
	Tags      []string    `json:"tags,omitempty"`
	Archived  interface{} `json:"archived,omitempty"` // null, datetime string, or bool
	Deleted   interface{} `json:"deleted,omitempty"`  // null, datetime string, or bool
}

// MapDates holds date fields for a map
type MapDates struct {
	CreationDate string `json:"creationDate,omitempty"`
	LastModified string `json:"lastModifiedDate,omitempty"`
}

// MapGroup represents an EdControls map group (drawing group)
type MapGroup struct {
	ID        string `json:"id,omitempty"`
	CouchID   string `json:"_id,omitempty"`
	CouchDbID string `json:"couchDbId,omitempty"`
	Name      string `json:"name"`
	Archived  bool   `json:"archived,omitempty"`
}

// File represents an EdControls file (attachment/document)
type File struct {
	ID          string      `json:"id,omitempty"`
	CouchID     string      `json:"_id,omitempty"`
	CouchDbID   string      `json:"couchDbId,omitempty"`
	Name        string      `json:"name"`
	FileName    string      `json:"fileName,omitempty"`
	ContentType string      `json:"contentType,omitempty"`
	Size        interface{} `json:"size,omitempty"` // Can be string or number
	GroupID     string      `json:"groupId,omitempty"`
	FileGroupID string      `json:"fileGroupId,omitempty"` // Used in document (alternative to groupId)
	GroupName   string      `json:"groupName,omitempty"`
	Dates       *FileDates  `json:"dates,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Author      *Person     `json:"author,omitempty"`
	Archived    interface{} `json:"archived,omitempty"` // null, datetime string, or bool
	Deleted     interface{} `json:"deleted,omitempty"`  // null, datetime string, or bool
	VersionID   string      `json:"versionId,omitempty"` // Download token
}

// FileDates holds date fields for a file
type FileDates struct {
	CreationDate string `json:"creationDate,omitempty"`
	LastModified string `json:"lastModifiedDate,omitempty"`
}

// FileGroup represents an EdControls file group
type FileGroup struct {
	ID        string `json:"id,omitempty"`
	CouchID   string `json:"_id,omitempty"`
	CouchDbID string `json:"couchDbId,omitempty"`
	Name      string `json:"name"`
	Archived  bool   `json:"archived,omitempty"`
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
	ID           string           `json:"id"`
	CouchID      string           `json:"_id,omitempty"`
	CouchDbID    string           `json:"couchDbId,omitempty"`
	Name         string           `json:"name"`
	Status       string           `json:"status"`
	Template     string           `json:"template,omitempty"`
	TemplateName string           `json:"templateName,omitempty"`
	TemplateID   string           `json:"templateId,omitempty"`
	Author       *Person          `json:"author,omitempty"`
	Dates        *AuditDates      `json:"dates,omitempty"`
	GroupID      string           `json:"groupId,omitempty"`
	Tags         []string         `json:"tags,omitempty"`
	Database     string           `json:"database,omitempty"`
	Participants *Participants    `json:"participants,omitempty"`
	Questions    []QuestionCategory `json:"questions,omitempty"`
}

// QuestionCategory represents a category of questions in an audit
type QuestionCategory struct {
	CategoryName string     `json:"categoryName"`
	Questions    []Question `json:"questions,omitempty"`
}

// Question represents a single question in an audit
type Question struct {
	Question    string           `json:"question"`
	Description string           `json:"description,omitempty"`
	Answer      []interface{}    `json:"answer,omitempty"`
	Settings    *QuestionSettings `json:"settings,omitempty"`
}

// QuestionSettings holds settings for a question
type QuestionSettings struct {
	AnswerType string   `json:"answertype,omitempty"`
	Choice     string   `json:"choice,omitempty"`
	Answer     []string `json:"answer,omitempty"` // Predefined options for multiplechoice
}

// TemplateDates holds date fields for a template
type TemplateDates struct {
	CreationDate string `json:"creationDate,omitempty"`
	LastModified string `json:"lastModifiedDate,omitempty"`
}

// AuditTemplate represents an EdControls audit template
type AuditTemplate struct {
	ID          string         `json:"id"`
	CouchID     string         `json:"_id,omitempty"`
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
	email, err := c.Email()
	if err != nil {
		return nil, 0, fmt.Errorf("getting user email: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/licenseserver/user/%s/projects", url.PathEscape(email))
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
	// Use the user's project list to find the project
	projects, _, err := c.ListProjects(ListProjectsOptions{})
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		if p.ProjectID == database {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("project %s not found", database)
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

// SearchAuditsByID searches for audits by ID across multiple projects using the POST search endpoint
func (c *Client) SearchAuditsByID(projectIDs []string, searchID string) ([]Audit, error) {
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

	endpoint := "/api/v2/data/audits/search?size=10&page=0"
	body, err := c.doRequest("POST", endpoint, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	var audits []Audit
	if err := json.Unmarshal(result.Results, &audits); err != nil {
		return nil, fmt.Errorf("parsing audits: %w", err)
	}

	return audits, nil
}

// SearchMapsByID searches for maps by ID across multiple projects using the POST search endpoint
func (c *Client) SearchMapsByID(projectIDs []string, searchID string) ([]Map, error) {
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

	endpoint := "/api/v2/data/maps/search?size=10&page=0"
	body, err := c.doRequest("POST", endpoint, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	var maps []Map
	if err := json.Unmarshal(result.Results, &maps); err != nil {
		return nil, fmt.Errorf("parsing maps: %w", err)
	}

	return maps, nil
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

// UpdateTicketFieldsOptions contains options for updating ticket fields with operation tracking
type UpdateTicketFieldsOptions struct {
	Title            *string
	Description      *string
	DueDate          *string
	ClearDue         bool
	Responsible      *string // Email of the responsible person
	ClearResponsible bool    // Clear the responsible (sets status back to created)
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

// GetAudit returns a single audit via the securedata endpoint
func (c *Client) GetAudit(database, auditID string) (*Audit, error) {
	endpoint := fmt.Sprintf("/api/v1/securedata/%s/%s", url.PathEscape(database), url.PathEscape(auditID))
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var audit Audit
	if err := json.Unmarshal(body, &audit); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	// Set CouchDbID from CouchID (_id) if not set
	if audit.CouchDbID == "" && audit.CouchID != "" {
		audit.CouchDbID = audit.CouchID
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
	endpoint := fmt.Sprintf("/api/v1/securedata/%s/%s", url.PathEscape(database), url.PathEscape(templateID))
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var template AuditTemplate
	if err := json.Unmarshal(body, &template); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	// Set CouchDbID from CouchID (_id) if not set
	if template.CouchDbID == "" && template.CouchID != "" {
		template.CouchDbID = template.CouchID
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

// GetMap returns a map (drawing) by ID
func (c *Client) GetMap(database, mapID string) (*Map, error) {
	endpoint := fmt.Sprintf("/api/v1/securedata/%s/%s", url.PathEscape(database), url.PathEscape(mapID))
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var m Map
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &m, nil
}

// GetMapGroup returns a map group (drawing group) by ID
func (c *Client) GetMapGroup(database, groupID string) (*MapGroup, error) {
	endpoint := fmt.Sprintf("/api/v1/securedata/%s/%s", url.PathEscape(database), url.PathEscape(groupID))
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var mg MapGroup
	if err := json.Unmarshal(body, &mg); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &mg, nil
}

// ListMapsOptions contains options for listing maps
type ListMapsOptions struct {
	Database   string // Required
	GroupID    string
	SearchName string
	SearchByID string
	Tag        string
	Archived   bool
	AllMaps    bool
	SortBy     string
	SortOrder  string
	Page       int
	Size       int
}

// ListMaps returns maps for a project
func (c *Client) ListMaps(opts ListMapsOptions) ([]Map, int, error) {
	params := url.Values{}
	params.Set("database", opts.Database)

	if opts.GroupID != "" {
		params.Set("groupid", opts.GroupID)
	}
	if opts.SearchName != "" {
		params.Set("searchByName", opts.SearchName)
	}
	if opts.SearchByID != "" {
		params.Set("searchById", opts.SearchByID)
	}
	if opts.Tag != "" {
		params.Set("tag", opts.Tag)
	}
	if opts.Archived {
		params.Set("archived", "true")
	}
	if opts.AllMaps {
		params.Set("allMaps", "true")
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

	endpoint := "/api/v2/data/maps?" + params.Encode()
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, fmt.Errorf("parsing response: %w", err)
	}

	var maps []Map
	if err := json.Unmarshal(result.Results, &maps); err != nil {
		return nil, 0, fmt.Errorf("parsing maps: %w", err)
	}

	return maps, result.Hits, nil
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

// ListFilesOptions contains options for listing files
type ListFilesOptions struct {
	Database   string // Required
	GroupID    string
	SearchName string
	SearchByID string
	Tag        string
	Archived   bool
	SortBy     string
	SortOrder  string
	Page       int
	Size       int
}

// ListFiles returns files for a project
func (c *Client) ListFiles(opts ListFilesOptions) ([]File, int, error) {
	params := url.Values{}

	if opts.GroupID != "" {
		params.Set("groupid", opts.GroupID)
	}
	if opts.SearchName != "" {
		params.Set("searchByName", opts.SearchName)
	}
	if opts.SearchByID != "" {
		params.Set("searchById", opts.SearchByID)
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

	endpoint := "/api/v2/data/file/" + url.PathEscape(opts.Database) + "?" + params.Encode()
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, fmt.Errorf("parsing response: %w", err)
	}

	var files []File
	if err := json.Unmarshal(result.Results, &files); err != nil {
		return nil, 0, fmt.Errorf("parsing files: %w", err)
	}

	return files, result.Hits, nil
}

// GetFile returns a single file via the securedata endpoint
func (c *Client) GetFile(database, fileID string) (*File, error) {
	endpoint := fmt.Sprintf("/api/v1/securedata/%s/%s", url.PathEscape(database), url.PathEscape(fileID))
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var file File
	if err := json.Unmarshal(body, &file); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	// Set CouchDbID from CouchID (_id) if not set
	if file.CouchDbID == "" && file.CouchID != "" {
		file.CouchDbID = file.CouchID
	}

	return &file, nil
}

// GetFileGroup returns a file group by ID
func (c *Client) GetFileGroup(database, groupID string) (*FileGroup, error) {
	endpoint := fmt.Sprintf("/api/v1/securedata/%s/%s", url.PathEscape(database), url.PathEscape(groupID))
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var group FileGroup
	if err := json.Unmarshal(body, &group); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &group, nil
}

// SearchFilesByID searches for files by ID across multiple projects using the POST search endpoint
func (c *Client) SearchFilesByID(projectIDs []string, searchID string) ([]File, error) {
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

	endpoint := "/api/v2/data/file/search?size=10&page=0"
	body, err := c.doRequest("POST", endpoint, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	var files []File
	if err := json.Unmarshal(result.Results, &files); err != nil {
		return nil, fmt.Errorf("parsing files: %w", err)
	}

	return files, nil
}

// TemplateGroup represents an EdControls audit template group
type TemplateGroup struct {
	ID        string `json:"id,omitempty"`
	CouchID   string `json:"_id,omitempty"`
	CouchDbID string `json:"couchDbId,omitempty"`
	Name      string `json:"name"`
	Archived  bool   `json:"archived,omitempty"`
}

// GetContract returns a contract by its ID from the clients database
func (c *Client) GetContract(contractID string) (*Contract, error) {
	endpoint := fmt.Sprintf("/api/v1/securedata/clients/%s", url.PathEscape(contractID))
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var contract Contract
	if err := json.Unmarshal(body, &contract); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &contract, nil
}

// ListGroupsOptions contains options for listing groups
type ListGroupsOptions struct {
	Database   string
	SearchName string
	Archived   bool
	Page       int
	Size       int
}

// ListMapGroups returns map groups (drawing groups) for a project
func (c *Client) ListMapGroups(opts ListGroupsOptions) ([]MapGroup, int, error) {
	params := url.Values{}
	params.Set("database", opts.Database)

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

	endpoint := "/api/v2/data/drawingGroups?" + params.Encode()
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, fmt.Errorf("parsing response: %w", err)
	}

	var groups []MapGroup
	if err := json.Unmarshal(result.Results, &groups); err != nil {
		return nil, 0, fmt.Errorf("parsing map groups: %w", err)
	}

	return groups, result.Hits, nil
}

// ListTemplateGroups returns audit template groups for a project
func (c *Client) ListTemplateGroups(opts ListGroupsOptions) ([]TemplateGroup, int, error) {
	params := url.Values{}
	params.Set("database", opts.Database)

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

	endpoint := "/api/v2/data/audits/templategroups?" + params.Encode()
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, fmt.Errorf("parsing response: %w", err)
	}

	var groups []TemplateGroup
	if err := json.Unmarshal(result.Results, &groups); err != nil {
		return nil, 0, fmt.Errorf("parsing template groups: %w", err)
	}

	return groups, result.Hits, nil
}

// ListFileGroups returns file groups for a project
func (c *Client) ListFileGroups(opts ListGroupsOptions) ([]FileGroup, int, error) {
	params := url.Values{}

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

	endpoint := "/api/v2/data/fileGroup/" + url.PathEscape(opts.Database) + "?" + params.Encode()
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, fmt.Errorf("parsing response: %w", err)
	}

	var groups []FileGroup
	if err := json.Unmarshal(result.Results, &groups); err != nil {
		return nil, 0, fmt.Errorf("parsing file groups: %w", err)
	}

	return groups, result.Hits, nil
}

// UploadInitResponse is the response from the initiate upload endpoint
type UploadInitResponse struct {
	UUID string `json:"uuid"`
}

// UploadCompleteResponse is the response from the complete upload endpoint
type UploadCompleteResponse struct {
	SignedURL string `json:"signedUrl"`
}

// InitiateUpload initiates a file upload and returns a UUID for subsequent operations
func (c *Client) InitiateUpload(database, fileName string) (*UploadInitResponse, error) {
	reqBody := map[string]string{
		"fileName": fileName,
		"database": database,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	body, err := c.doRequest("POST", "/api/v1/fileUpload/initiate", strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	var result UploadInitResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &result, nil
}

// UploadChunk uploads a chunk of file data
func (c *Client) UploadChunk(uuid string, fileName string, chunkIndex int, data []byte) error {
	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the file chunk
	part, err := writer.CreateFormFile("chunk", "blob")
	if err != nil {
		return fmt.Errorf("creating form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("writing chunk data: %w", err)
	}

	// Add fileName field
	if err := writer.WriteField("fileName", fileName); err != nil {
		return fmt.Errorf("writing fileName field: %w", err)
	}

	// Add chunkIndex field
	if err := writer.WriteField("chunkIndex", fmt.Sprintf("%d", chunkIndex)); err != nil {
		return fmt.Errorf("writing chunkIndex field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("closing multipart writer: %w", err)
	}

	// Build the request manually since we need custom Content-Type
	reqURL := baseURL + "/api/v1/fileUpload/upload?uuid=" + url.QueryEscape(uuid)
	req, err := http.NewRequest("POST", reqURL, &buf)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload chunk failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// CompleteUpload marks an upload as complete and returns the signed URL
func (c *Client) CompleteUpload(uuid, fileName string) (*UploadCompleteResponse, error) {
	reqBody := map[string]string{
		"uuid":     uuid,
		"fileName": fileName,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	body, err := c.doRequest("POST", "/api/v1/fileUpload/uploadCompleted", strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	var result UploadCompleteResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &result, nil
}

// CreateFileOptions contains options for creating a file document
type CreateFileOptions struct {
	Database     string
	FileName     string   // Display name
	UploadedName string   // Name used during upload (with timestamp)
	FileURL      string   // Signed URL from upload completion
	FileGroupID  string   // File group ID
	ContentType  string   // MIME type
	Size         int64    // File size in bytes
	Tags         []string // Optional tags
}

// CreateFileResponse is the response from creating a file
type CreateFileResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CreateFile creates a file document in EdControls
func (c *Client) CreateFile(opts CreateFileOptions) (*CreateFileResponse, error) {
	// Get project info to get the CouchDB ID
	project, err := c.GetProject(opts.Database)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	// Get the current user's email
	email, err := c.Email()
	if err != nil {
		return nil, fmt.Errorf("getting user email: %w", err)
	}

	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	timeOnly := time.Now().UTC().Format("15:04:05")
	channelID := fmt.Sprintf("%d%s", time.Now().UnixMilli(), project.CouchDbID)

	// Build the file document
	fileDoc := map[string]interface{}{
		"fileName":    opts.FileName,
		"couchDbId":   "",
		"attachments": []interface{}{},
		"project":     project.CouchDbID,
		"type":        "IB.EdBundle.Document.File",
		"content": map[string]string{
			"author":       email,
			"lastModifier": email,
		},
		"archived":    nil,
		"fileGroupID": opts.FileGroupID,
		"contentType": opts.ContentType,
		"thumbnail":   nil,
		"dates": map[string]string{
			"creationDate":     now,
			"lastModifiedDate": now,
		},
		"tags":     opts.Tags,
		"deleted":  nil,
		"size":     fmt.Sprintf("%d", opts.Size),
		"versions": nil,
		"operation": []map[string]interface{}{
			{
				"changedProperties": []string{"file Upload"},
				"oldValues":         []string{""},
				"newValues":         []string{opts.FileName},
				"author":            email,
				"time":              now,
				"summary":           "File uploaded via CLI",
				"actionType":        "created",
				"platform": map[string]string{
					"userInterface":    "cli",
					"interfaceVersion": "1.0.0",
				},
			},
		},
		"channelId": channelID,
		"job":       nil,
		"time":      timeOnly,
	}

	if opts.Tags == nil {
		fileDoc["tags"] = []string{}
	}

	jsonBody, err := json.Marshal(fileDoc)
	if err != nil {
		return nil, fmt.Errorf("marshaling file document: %w", err)
	}

	// Build the endpoint with query parameters
	params := url.Values{}
	params.Set("fileUrl", opts.FileURL)
	params.Set("uploadedName", opts.UploadedName)

	endpoint := fmt.Sprintf("/api/v2/data/file/%s?%s", url.PathEscape(opts.Database), params.Encode())

	body, err := c.doRequest("POST", endpoint, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	var result CreateFileResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &result, nil
}

// UpdateDocumentTags updates the tags on a document (file or map)
func (c *Client) UpdateDocumentTags(database, docID string, tags []string) error {
	// Get the current document
	doc, err := c.GetDocument(database, docID)
	if err != nil {
		return fmt.Errorf("getting document: %w", err)
	}

	// Get user email for operation record
	email, err := c.Email()
	if err != nil {
		return fmt.Errorf("getting user email: %w", err)
	}

	// Get old tags for operation record
	var oldTags []string
	if existingTags, ok := doc["tags"].([]interface{}); ok {
		for _, t := range existingTags {
			if s, ok := t.(string); ok {
				oldTags = append(oldTags, s)
			}
		}
	}

	// Update tags
	doc["tags"] = tags

	// Update dates.lastModifiedDate
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	if dates, ok := doc["dates"].(map[string]interface{}); ok {
		dates["lastModifiedDate"] = now
	}

	// Update content.lastModifier if it exists
	if content, ok := doc["content"].(map[string]interface{}); ok {
		content["lastModifier"] = email
	}

	// Build operation record
	operation := map[string]interface{}{
		"author":            email,
		"changedProperties": []string{"tags"},
		"oldValues":         []interface{}{oldTags},
		"newValues":         []interface{}{tags},
		"time":              now,
		"platform": map[string]string{
			"userInterface":    "cli",
			"interfaceVersion": "1.0.0",
		},
	}

	// Append to operations array
	if ops, ok := doc["operation"].([]interface{}); ok {
		doc["operation"] = append(ops, operation)
	} else {
		doc["operation"] = []interface{}{operation}
	}

	// PUT the updated document
	jsonBody, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshaling document: %w", err)
	}

	endpoint := fmt.Sprintf("/api/v1/securedata/%s/%s", url.PathEscape(database), url.PathEscape(docID))
	_, err = c.doRequest("PUT", endpoint, strings.NewReader(string(jsonBody)))
	return err
}

// UpdateTicketDueDate updates the due date on a ticket
// If dueDate is empty, the due date is cleared
func (c *Client) UpdateTicketDueDate(database, ticketID string, dueDate string) error {
	// Get the current document
	doc, err := c.GetDocument(database, ticketID)
	if err != nil {
		return fmt.Errorf("getting ticket: %w", err)
	}

	// Get user email for operation record
	email, err := c.Email()
	if err != nil {
		return fmt.Errorf("getting user email: %w", err)
	}

	// Get old due date for operation record
	oldDueDate := ""
	if plan, ok := doc["plan"].(map[string]interface{}); ok {
		if dd, ok := plan["dueDate"].(string); ok {
			oldDueDate = dd
		}
	}

	// Update plan.dueDate
	if plan, ok := doc["plan"].(map[string]interface{}); ok {
		if dueDate == "" {
			delete(plan, "dueDate")
		} else {
			plan["dueDate"] = dueDate
		}
	} else if dueDate != "" {
		doc["plan"] = map[string]interface{}{"dueDate": dueDate}
	}

	// Update dates.lastModifiedDate
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	if dates, ok := doc["dates"].(map[string]interface{}); ok {
		dates["lastModifiedDate"] = now
	}

	// Update content.lastmodifier if it exists
	if content, ok := doc["content"].(map[string]interface{}); ok {
		content["lastmodifier"] = map[string]interface{}{
			"type":  "IB.EdBundle.Document.Person",
			"email": email,
		}
	}

	// Build operation record
	actionType := "updated"
	summary := "user updated following fields"
	operation := map[string]interface{}{
		"author":            email,
		"changedProperties": []string{"duedate"},
		"oldValues":         []interface{}{oldDueDate},
		"newValues":         []interface{}{dueDate},
		"time":              now,
		"summary":           summary,
		"actionType":        actionType,
		"platform": map[string]string{
			"userInterface":    "cli",
			"interfaceVersion": "1.0.0",
		},
	}

	// Append to operations array
	if ops, ok := doc["operation"].([]interface{}); ok {
		doc["operation"] = append(ops, operation)
	} else {
		doc["operation"] = []interface{}{operation}
	}

	// PUT the updated document
	jsonBody, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshaling document: %w", err)
	}

	endpoint := fmt.Sprintf("/api/v1/securedata/%s/%s", url.PathEscape(database), url.PathEscape(ticketID))
	_, err = c.doRequest("PUT", endpoint, strings.NewReader(string(jsonBody)))
	return err
}

// GetTicketDueDate returns the current due date of a ticket
func (c *Client) GetTicketDueDate(database, ticketID string) (string, error) {
	doc, err := c.GetDocument(database, ticketID)
	if err != nil {
		return "", fmt.Errorf("getting ticket: %w", err)
	}

	if plan, ok := doc["plan"].(map[string]interface{}); ok {
		if dueDate, ok := plan["dueDate"].(string); ok {
			return dueDate, nil
		}
	}
	return "", nil
}

// UpdateTicketFields updates multiple ticket fields with proper operation tracking
func (c *Client) UpdateTicketFields(database, ticketID string, opts UpdateTicketFieldsOptions) error {
	// Get the current document
	doc, err := c.GetDocument(database, ticketID)
	if err != nil {
		return fmt.Errorf("getting ticket: %w", err)
	}

	// Get user email for operation record
	email, err := c.Email()
	if err != nil {
		return fmt.Errorf("getting user email: %w", err)
	}

	// Track changes for operation record
	var changedProps []string
	var oldValues []interface{}
	var newValues []interface{}

	// Handle title update
	if opts.Title != nil {
		oldTitle := ""
		if content, ok := doc["content"].(map[string]interface{}); ok {
			if t, ok := content["title"].(string); ok {
				oldTitle = t
			}
			content["title"] = *opts.Title
		}
		changedProps = append(changedProps, "title")
		oldValues = append(oldValues, oldTitle)
		newValues = append(newValues, *opts.Title)
	}

	// Handle description update
	if opts.Description != nil {
		oldDesc := ""
		if content, ok := doc["content"].(map[string]interface{}); ok {
			if d, ok := content["body"].(string); ok {
				oldDesc = d
			}
			content["body"] = *opts.Description
		}
		changedProps = append(changedProps, "description")
		oldValues = append(oldValues, oldDesc)
		newValues = append(newValues, *opts.Description)
	}

	// Handle due date update
	if opts.DueDate != nil || opts.ClearDue {
		oldDueDate := ""
		if plan, ok := doc["plan"].(map[string]interface{}); ok {
			if dd, ok := plan["dueDate"].(string); ok {
				oldDueDate = dd
			}
		}

		newDueDate := ""
		if opts.DueDate != nil {
			newDueDate = *opts.DueDate
		}

		// Update plan.dueDate
		if plan, ok := doc["plan"].(map[string]interface{}); ok {
			if opts.ClearDue {
				delete(plan, "dueDate")
			} else if opts.DueDate != nil {
				plan["dueDate"] = *opts.DueDate
			}
		} else if opts.DueDate != nil {
			doc["plan"] = map[string]interface{}{"dueDate": *opts.DueDate}
		}

		changedProps = append(changedProps, "duedate")
		oldValues = append(oldValues, oldDueDate)
		newValues = append(newValues, newDueDate)
	}

	// Handle responsible update
	if opts.Responsible != nil || opts.ClearResponsible {
		// Get old responsible email
		oldResponsible := ""
		if participants, ok := doc["participants"].(map[string]interface{}); ok {
			if resp, ok := participants["responsible"].(map[string]interface{}); ok {
				if email, ok := resp["email"].(string); ok {
					oldResponsible = email
				}
			}
		}

		// Get old status
		oldStatus := ""
		if state, ok := doc["state"].(map[string]interface{}); ok {
			if s, ok := state["state"].(string); ok {
				oldStatus = s
			}
		}

		newResponsible := ""
		newStatus := ""

		if opts.ClearResponsible {
			// Clear responsible and set status to created
			if participants, ok := doc["participants"].(map[string]interface{}); ok {
				delete(participants, "responsible")
			}
			newStatus = "created"
		} else if opts.Responsible != nil {
			// Set responsible and change status to started
			newResponsible = *opts.Responsible
			if participants, ok := doc["participants"].(map[string]interface{}); ok {
				participants["responsible"] = map[string]interface{}{
					"type":  "IB.EdBundle.Document.Person",
					"email": *opts.Responsible,
				}
			} else {
				doc["participants"] = map[string]interface{}{
					"type": "IB.EdBundle.Document.Participants",
					"responsible": map[string]interface{}{
						"type":  "IB.EdBundle.Document.Person",
						"email": *opts.Responsible,
					},
					"consulted": []interface{}{},
					"informed":  []interface{}{},
				}
			}
			newStatus = "started"
		}

		// Update state
		if newStatus != "" && newStatus != oldStatus {
			if state, ok := doc["state"].(map[string]interface{}); ok {
				state["state"] = newStatus
			} else {
				doc["state"] = map[string]interface{}{
					"type":  "IB.EdBundle.Document.State",
					"state": newStatus,
				}
			}

			// Add status change to operation record
			changedProps = append(changedProps, "status")
			oldValues = append(oldValues, oldStatus)
			newValues = append(newValues, newStatus)
		}

		// Add responsible change to operation record
		changedProps = append(changedProps, "responsible")
		oldValues = append(oldValues, oldResponsible)
		newValues = append(newValues, newResponsible)
	}

	// If no changes, return early
	if len(changedProps) == 0 {
		return nil
	}

	// Update dates.lastModifiedDate
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	if dates, ok := doc["dates"].(map[string]interface{}); ok {
		dates["lastModifiedDate"] = now
	}

	// Update content.lastmodifier
	if content, ok := doc["content"].(map[string]interface{}); ok {
		content["lastmodifier"] = map[string]interface{}{
			"type":  "IB.EdBundle.Document.Person",
			"email": email,
		}
	}

	// Build operation record
	operation := map[string]interface{}{
		"author":            email,
		"changedProperties": changedProps,
		"oldValues":         oldValues,
		"newValues":         newValues,
		"time":              now,
		"summary":           "user updated following fields",
		"actionType":        "updated",
		"platform": map[string]string{
			"userInterface":    "cli",
			"interfaceVersion": "1.0.0",
		},
	}

	// Append to operations array
	if ops, ok := doc["operation"].([]interface{}); ok {
		doc["operation"] = append(ops, operation)
	} else {
		doc["operation"] = []interface{}{operation}
	}

	// PUT the updated document
	jsonBody, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshaling document: %w", err)
	}

	endpoint := fmt.Sprintf("/api/v1/securedata/%s/%s", url.PathEscape(database), url.PathEscape(ticketID))
	_, err = c.doRequest("PUT", endpoint, strings.NewReader(string(jsonBody)))
	return err
}

// DeleteLibraryItems deletes files and/or maps from a project
func (c *Client) DeleteLibraryItems(database string, fileIDs, mapIDs []string) error {
	// Get project info for channelId
	project, err := c.GetProject(database)
	if err != nil {
		return fmt.Errorf("getting project: %w", err)
	}

	now := time.Now().UTC()
	channelID := fmt.Sprintf("%d%s", now.UnixMilli(), project.CouchDbID)
	timeOnly := now.Format("15:04:05")

	reqBody := map[string]interface{}{
		"channelId":             channelID,
		"mapList":               mapIDs,
		"fileList":              fileIDs,
		"time":                  timeOnly,
		"isFileDeletionEnabled": true,
		"database":              database,
	}

	if fileIDs == nil {
		reqBody["fileList"] = []string{}
	}
	if mapIDs == nil {
		reqBody["mapList"] = []string{}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	_, err = c.doRequest("POST", "/api/v1/bulk/library", strings.NewReader(string(jsonBody)))
	return err
}

// ConvertFileToMap converts a file to a map (tiled drawing)
func (c *Client) ConvertFileToMap(database, fileID, versionID, fileName, groupName string) error {
	email, err := c.Email()
	if err != nil {
		return fmt.Errorf("getting user email: %w", err)
	}

	// Get project info for channelId
	project, err := c.GetProject(database)
	if err != nil {
		return fmt.Errorf("getting project: %w", err)
	}

	now := time.Now().UTC()
	channelID := fmt.Sprintf("%d%s", now.UnixMilli(), project.CouchDbID)
	timeOnly := now.Format("15:04:05")

	reqBody := map[string]interface{}{
		"sendStatus": map[string]string{
			"channelId": channelID,
			"time":      timeOnly,
			"fileName":  fileName,
		},
		"mapId":        "",
		"fileStackUrl": nil,
		"headers": map[string]string{
			"from":    email,
			"to":      database + "@edcontrols.nl",
			"subject": groupName,
			"date":    now.Format("2006-01-02T15:04:05.000Z"),
		},
		"readyForTiler": true,
		"platform": map[string]string{
			"userInterface":    "cli",
			"interfaceVersion": "1.0.0",
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	endpoint := fmt.Sprintf("/api/v2/data/tiler/%s/%s/tileDocument?versionId=%s",
		url.PathEscape(database),
		url.PathEscape(fileID),
		url.QueryEscape(versionID))

	_, err = c.doRequest("POST", endpoint, strings.NewReader(string(jsonBody)))
	return err
}

// ArchiveFile archives or unarchives files
func (c *Client) ArchiveFile(database string, fileIDs []string, archive bool) error {
	reqBody := map[string]interface{}{
		"id": fileIDs,
		"platform": map[string]string{
			"userInterface":    "cli",
			"interfaceVersion": "1.0.0",
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	endpoint := fmt.Sprintf("/api/v2/data/file/%s/archive?archive=%t",
		url.PathEscape(database), archive)

	_, err = c.doRequest("PUT", endpoint, strings.NewReader(string(jsonBody)))
	return err
}

// DownloadFile downloads a file and returns its contents
func (c *Client) DownloadFile(database, fileID, versionID, fileName string) ([]byte, error) {
	// Build the download URL: /api/v2/data/file/{database}/{fileId}/{versionId}/{fileName}/downloadFile
	endpoint := fmt.Sprintf("/api/v2/data/file/%s/%s/%s/%s/downloadFile",
		url.PathEscape(database),
		url.PathEscape(fileID),
		url.PathEscape(versionID),
		url.PathEscape(fileName))

	reqURL := baseURL + endpoint
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "*/*")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download failed (%d): %s", resp.StatusCode, string(respBody))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return data, nil
}
