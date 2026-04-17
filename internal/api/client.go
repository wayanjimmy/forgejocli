package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// Client wraps the Forgejo API
type Client struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewClient creates a new Forgejo API client
func NewClient(baseURL, token, socksProxy string) (*Client, error) {
	var httpClient *http.Client

	if socksProxy != "" {
		u, err := url.Parse(socksProxy)
		if err != nil {
			return nil, fmt.Errorf("parsing proxy URL %q: %w", socksProxy, err)
		}
		dialer, err := proxy.SOCKS5("tcp", u.Host, nil, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("creating SOCKS5 dialer: %w", err)
		}
		transport := &http.Transport{
			Dial: dialer.Dial,
		}
		httpClient = &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		}
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/") + "/api/v1",
		token:   token,
		client:  httpClient,
	}, nil
}

func (c *Client) doRequest(method, path string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	return data, resp.StatusCode, nil
}

func (c *Client) get(path string) ([]byte, error) {
	data, status, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("API %d: %s", status, string(data))
	}
	return data, nil
}

func (c *Client) post(path string, body interface{}) ([]byte, error) {
	data, status, err := c.doRequest("POST", path, body)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("API %d: %s", status, string(data))
	}
	return data, nil
}

func (c *Client) patch(path string, body interface{}) ([]byte, error) {
	data, status, err := c.doRequest("PATCH", path, body)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("API %d: %s", status, string(data))
	}
	return data, nil
}

func (c *Client) delete(path string) error {
	data, status, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("API %d: %s", status, string(data))
	}
	return nil
}

// ---- Types ----

type User struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

type Repository struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	Fork        bool   `json:"fork"`
	Stars       int    `json:"stars_count"`
	Forks       int    `json:"forks_count"`
	OpenIssues  int    `json:"open_issues_count"`
	DefaultBranch string `json:"default_branch"`
	HTMLURL     string `json:"html_url"`
	SSHURL      string `json:"ssh_url"`
	CloneURL    string `json:"clone_url"`
	Owner       *User  `json:"owner"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Issue struct {
	ID        int       `json:"id"`
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
	HTMLURL   string    `json:"html_url"`
	User      *User     `json:"user"`
	Assignee  *User     `json:"assignee"`
	Labels    []Label   `json:"labels"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Label struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Color string `json:"color"`
}

// Comment represents a comment on an issue or PR
type Comment struct {
	ID        int       `json:"id"`
	HTMLURL   string    `json:"html_url"`
	Body      string    `json:"body"`
	User      *User     `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IssueWithComments wraps an issue with its comments
type IssueWithComments struct {
	Issue    *Issue    `json:"issue"`
	Comments []Comment `json:"comments"`
}

// CreateCommentOption for creating comments
type CreateCommentOption struct {
	Body string `json:"body"`
}

type PullRequest struct {
	ID        int       `json:"id"`
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
	HTMLURL   string    `json:"html_url"`
	User      *User     `json:"user"`
	Mergeable bool      `json:"mergeable"`
	Merged    bool      `json:"merged"`
	Base      PRBranch  `json:"base"`
	Head      PRBranch  `json:"head"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PRBranch struct {
	Label string `json:"label"`
	Ref   string `json:"ref"`
	SHA   string `json:"sha"`
}

type CreateIssueOption struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type EditIssueOption struct {
	Title *string `json:"title,omitempty"`
	Body  *string `json:"body,omitempty"`
	State *string `json:"state,omitempty"`
}

type CreatePROption struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
}

type MergePROption struct {
	Do        string `json:"Do"`
	MergeTitle string `json:"MergeTitleField,omitempty"`
	CommitMsg  string `json:"MergeMessageField,omitempty"`
}

type CreateRepoOption struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Private     bool   `json:"private"`
	AutoInit    bool   `json:"auto_init"`
}

// ---- Repo API ----

func (c *Client) ListRepos(owner string, page, limit int) ([]Repository, error) {
	path := fmt.Sprintf("/repos/search?limit=%d&page=%d", limit, page)
	if owner != "" {
		path += "&user=" + url.QueryEscape(owner)
	}
	data, err := c.get(path)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []Repository `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing repos: %w", err)
	}
	return result.Data, nil
}

func (c *Client) GetRepo(owner, repo string) (*Repository, error) {
	data, err := c.get(fmt.Sprintf("/repos/%s/%s", owner, repo))
	if err != nil {
		return nil, err
	}
	var r Repository
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parsing repo: %w", err)
	}
	return &r, nil
}

func (c *Client) CreateRepo(opt CreateRepoOption) (*Repository, error) {
	data, err := c.post("/user/repos", opt)
	if err != nil {
		return nil, err
	}
	var r Repository
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parsing created repo: %w", err)
	}
	return &r, nil
}

func (c *Client) DeleteRepo(owner, repo string) error {
	return c.delete(fmt.Sprintf("/repos/%s/%s", owner, repo))
}

// ---- Issues API ----

func (c *Client) ListIssues(owner, repo string, page, limit int, state string) ([]Issue, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues?limit=%d&page=%d&type=issues",
		owner, repo, limit, page)
	if state != "" {
		path += "&state=" + state
	}
	data, err := c.get(path)
	if err != nil {
		return nil, err
	}
	var issues []Issue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, fmt.Errorf("parsing issues: %w", err)
	}
	return issues, nil
}

func (c *Client) GetIssue(owner, repo string, index int) (*Issue, error) {
	data, err := c.get(fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, index))
	if err != nil {
		return nil, err
	}
	var issue Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("parsing issue: %w", err)
	}
	return &issue, nil
}

func (c *Client) CreateIssue(owner, repo string, opt CreateIssueOption) (*Issue, error) {
	data, err := c.post(fmt.Sprintf("/repos/%s/%s/issues", owner, repo), opt)
	if err != nil {
		return nil, err
	}
	var issue Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("parsing created issue: %w", err)
	}
	return &issue, nil
}

func (c *Client) EditIssue(owner, repo string, index int, opt EditIssueOption) (*Issue, error) {
	data, err := c.patch(fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, index), opt)
	if err != nil {
		return nil, err
	}
	var issue Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("parsing edited issue: %w", err)
	}
	return &issue, nil
}

func (c *Client) CloseIssue(owner, repo string, index int) (*Issue, error) {
	state := "closed"
	return c.EditIssue(owner, repo, index, EditIssueOption{State: &state})
}

func (c *Client) ReopenIssue(owner, repo string, index int) (*Issue, error) {
	state := "open"
	return c.EditIssue(owner, repo, index, EditIssueOption{State: &state})
}

// ---- Comments API ----

// fetchCommentsPage is a DRY helper for fetching a single page of comments
func (c *Client) fetchCommentsPage(owner, repo string, index, page, limit int) ([]Comment, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments?limit=%d&page=%d",
		owner, repo, index, limit, page)
	data, err := c.get(path)
	if err != nil {
		return nil, err
	}
	var comments []Comment
	if err := json.Unmarshal(data, &comments); err != nil {
		return nil, fmt.Errorf("parsing comments: %w", err)
	}
	return comments, nil
}

// GetIssueComments retrieves comments for an issue with optional pagination.
// limit=0 means fetch all comments (auto-pagination).
// Uses DRY helper function to avoid code duplication.
func (c *Client) GetIssueComments(owner, repo string, index int, page, limit int) ([]Comment, error) {
	// If limit is 0, use auto-pagination with default page size
	if limit == 0 {
		limit = 50 // Default for auto-pagination
		var allComments []Comment
		for p := 1; ; p++ {
			comments, err := c.fetchCommentsPage(owner, repo, index, p, limit)
			if err != nil {
				return nil, err
			}
			allComments = append(allComments, comments...)
			if len(comments) < limit {
				break // Last page reached
			}
		}
		return allComments, nil
	}

	// Single page request
	return c.fetchCommentsPage(owner, repo, index, page, limit)
}

// GetIssueWithComments retrieves issue and comments with optional pagination
// commentLimit=0 fetches all comments; >0 limits to specific count
func (c *Client) GetIssueWithComments(owner, repo string, index int, commentLimit int) (*IssueWithComments, error) {
	issue, err := c.GetIssue(owner, repo, index)
	if err != nil {
		return nil, err
	}

	// If limit specified, fetch just first page with that limit
	// Otherwise fetch all with auto-pagination
	page := 1
	comments, err := c.GetIssueComments(owner, repo, index, page, commentLimit)
	if err != nil {
		return nil, err
	}

	return &IssueWithComments{
		Issue:    issue,
		Comments: comments,
	}, nil
}

// CreateComment adds a comment to an issue
func (c *Client) CreateComment(owner, repo string, index int, body string) (*Comment, error) {
	opt := CreateCommentOption{Body: body}
	data, err := c.post(fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, index), opt)
	if err != nil {
		return nil, err
	}
	var comment Comment
	if err := json.Unmarshal(data, &comment); err != nil {
		return nil, fmt.Errorf("parsing comment: %w", err)
	}
	return &comment, nil
}

// GetComment retrieves a single comment by ID
func (c *Client) GetComment(owner, repo string, commentID int) (*Comment, error) {
	data, err := c.get(fmt.Sprintf("/repos/%s/%s/issues/comments/%d", owner, repo, commentID))
	if err != nil {
		return nil, err
	}
	var comment Comment
	if err := json.Unmarshal(data, &comment); err != nil {
		return nil, fmt.Errorf("parsing comment: %w", err)
	}
	return &comment, nil
}

// DeleteComment removes a comment
func (c *Client) DeleteComment(owner, repo string, commentID int) error {
	return c.delete(fmt.Sprintf("/repos/%s/%s/issues/comments/%d", owner, repo, commentID))
}

// ---- Pull Requests API ----

func (c *Client) ListPRs(owner, repo string, page, limit int, state string) ([]PullRequest, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls?limit=%d&page=%d",
		owner, repo, limit, page)
	if state != "" {
		path += "&state=" + state
	}
	data, err := c.get(path)
	if err != nil {
		return nil, err
	}
	var prs []PullRequest
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, fmt.Errorf("parsing PRs: %w", err)
	}
	return prs, nil
}

func (c *Client) GetPR(owner, repo string, index int) (*PullRequest, error) {
	data, err := c.get(fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, index))
	if err != nil {
		return nil, err
	}
	var pr PullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("parsing PR: %w", err)
	}
	return &pr, nil
}

func (c *Client) CreatePR(owner, repo string, opt CreatePROption) (*PullRequest, error) {
	data, err := c.post(fmt.Sprintf("/repos/%s/%s/pulls", owner, repo), opt)
	if err != nil {
		return nil, err
	}
	var pr PullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("parsing created PR: %w", err)
	}
	return &pr, nil
}

func (c *Client) MergePR(owner, repo string, index int, commitMsg string) error {
	opt := MergePROption{
		Do:        "merge",
		CommitMsg: commitMsg,
	}
	_, err := c.post(fmt.Sprintf("/repos/%s/%s/pulls/%d/merge", owner, repo, index), opt)
	return err
}

func (c *Client) ClosePR(owner, repo string, index int) (*PullRequest, error) {
	state := "closed"
	data, err := c.patch(fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, index),
		map[string]string{"state": state})
	if err != nil {
		return nil, err
	}
	var pr PullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("parsing closed PR: %w", err)
	}
	return &pr, nil
}

func (c *Client) ReopenPR(owner, repo string, index int) (*PullRequest, error) {
	state := "open"
	data, err := c.patch(fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, index),
		map[string]string{"state": state})
	if err != nil {
		return nil, err
	}
	var pr PullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("parsing reopened PR: %w", err)
	}
	return &pr, nil
}

// ---- Attachments API ----

type Attachment struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Size          int64     `json:"size"`
	DownloadCount int       `json:"download_count"`
	Created       time.Time `json:"created_at"`
	UUID          string    `json:"uuid"`
	BrowserURL    string    `json:"browser_download_url"`
}

// UploadIssueAsset uploads a file as an attachment to an issue/PR.
// Returns the attachment with download URL.
func (c *Client) UploadIssueAsset(owner, repo string, index int, filePath string) (*Attachment, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %w", filePath, err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("attachment", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("creating multipart form: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("copying file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	apiURL := fmt.Sprintf("%s/repos/%s/%s/issues/%d/assets?name=%s",
		c.baseURL, owner, repo, index, url.QueryEscape(filepath.Base(filePath)))

	req, err := http.NewRequest("POST", apiURL, &body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API %d: %s", resp.StatusCode, string(respData))
	}

	var att Attachment
	if err := json.Unmarshal(respData, &att); err != nil {
		return nil, fmt.Errorf("parsing attachment response: %w", err)
	}

	return &att, nil
}

// helper for string pointer
func strPtr(s string) *string { return &s }

// ---- Actions API ----

type ActionRun struct {
	ID           int                    `json:"id"`
	Title        string                 `json:"title"`
	Status       string                 `json:"status"`
	Event        string                 `json:"event"`
	CommitSHA    string                 `json:"commit_sha"`
	PrettyRef    string                 `json:"prettyref"`
	HTMLURL      string                 `json:"html_url"`
	WorkflowID   string                 `json:"workflow_id"`
	IndexInRepo  int                    `json:"index_in_repo"`
	TriggerUser  *ActionUser            `json:"trigger_user"`
	TriggerEvent string                 `json:"trigger_event"`
	Created      time.Time              `json:"created"`
	Started      time.Time              `json:"started"`
	Stopped      time.Time              `json:"stopped"`
	Updated      time.Time              `json:"updated"`
	Duration     int64                  `json:"duration"`
	Repository   *ActionRunRepository   `json:"repository"`
}

type ActionRunRepository struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

type ActionUser struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	FullName string `json:"full_name"`
}

type ListActionRunsResponse struct {
	TotalCount   int         `json:"total_count"`
	WorkflowRuns []ActionRun `json:"workflow_runs"`
}

func (c *Client) ListActionRuns(owner, repo string, page, limit int, status string) (*ListActionRunsResponse, error) {
	path := fmt.Sprintf("/repos/%s/%s/actions/runs?limit=%d&page=%d",
		url.PathEscape(owner), url.PathEscape(repo), limit, page)
	if status != "" {
		path += "&status=" + url.QueryEscape(status)
	}
	data, err := c.get(path)
	if err != nil {
		return nil, err
	}
	var result ListActionRunsResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing action runs: %w", err)
	}
	return &result, nil
}

func (c *Client) GetActionRun(owner, repo string, runID int) (*ActionRun, error) {
	data, err := c.get(fmt.Sprintf("/repos/%s/%s/actions/runs/%d",
		url.PathEscape(owner), url.PathEscape(repo), runID))
	if err != nil {
		return nil, err
	}
	var run ActionRun
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("parsing action run: %w", err)
	}
	return &run, nil
}
