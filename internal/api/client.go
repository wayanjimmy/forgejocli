package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
func NewClient(baseURL, token, socksProxy string) *Client {
	var httpClient *http.Client

	if socksProxy != "" {
		// Parse socks5://host:port
		u, err := url.Parse(socksProxy)
		if err == nil {
			dialer, err := proxy.SOCKS5("tcp", u.Host, nil, proxy.Direct)
			if err == nil {
				transport := &http.Transport{
					Dial: dialer.Dial,
				}
				httpClient = &http.Client{
					Transport: transport,
					Timeout:   30 * time.Second,
				}
			}
		}
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &Client{
		baseURL: baseURL + "/api/v1",
		token:   token,
		client:  httpClient,
	}
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
	_, status, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("API %d", status)
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

// helper for string pointer
func strPtr(s string) *string { return &s }

// helper for int parsing
func parseInt(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
