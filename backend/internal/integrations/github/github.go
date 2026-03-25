package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/oarkflow/deploy/backend/internal/integrations"
)

const defaultBaseURL = "https://api.github.com"

// Client communicates with the GitHub REST API.
type Client struct {
	httpClient  *http.Client
	accessToken string
	baseURL     string
}

// NewClient returns a GitHub API client authenticated with the given personal
// access token (or GitHub App installation token).
func NewClient(accessToken string) *Client {
	return &Client{
		httpClient:  &http.Client{},
		accessToken: accessToken,
		baseURL:     defaultBaseURL,
	}
}

// NewClientWithBase is like NewClient but allows overriding the API base URL
// (useful for GitHub Enterprise or testing).
func NewClientWithBase(accessToken, baseURL string) *Client {
	return &Client{
		httpClient:  &http.Client{},
		accessToken: accessToken,
		baseURL:     baseURL,
	}
}

// ---- helpers ---------------------------------------------------------------

func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	url := c.baseURL + path

	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("github: marshal body: %w", err)
		}
		buf = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c *Client) do(req *http.Request, target any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("github: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github: API error %d: %s", resp.StatusCode, string(b))
	}
	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("github: decode response: %w", err)
		}
	}
	return nil
}

// ---- API response types (internal) ----------------------------------------

type ghRepo struct {
	ID            int64  `json:"id"`
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	CloneURL      string `json:"clone_url"`
	SSHURL        string `json:"ssh_url"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
	UpdatedAt     string `json:"updated_at"`
}

func (r *ghRepo) toRepoInfo() integrations.RepoInfo {
	return integrations.RepoInfo{
		ID:            strconv.FormatInt(r.ID, 10),
		FullName:      r.FullName,
		Description:   r.Description,
		CloneURL:      r.CloneURL,
		SSHURL:        r.SSHURL,
		DefaultBranch: r.DefaultBranch,
		Private:       r.Private,
		UpdatedAt:     r.UpdatedAt,
	}
}

type ghHook struct {
	ID int64 `json:"id"`
}

// ---- SCMProvider implementation --------------------------------------------

// ListRepos returns repositories the authenticated user has access to.
// Supports search (via GitHub search API) and pagination.
func (c *Client) ListRepos(ctx context.Context, opts integrations.ListReposOptions) ([]integrations.RepoInfo, int, error) {
	page := opts.Page
	if page <= 0 {
		page = 1
	}
	perPage := opts.PerPage
	if perPage <= 0 || perPage > 100 {
		perPage = 30
	}

	if opts.Search != "" {
		// Use the search API for query-based lookups.
		path := fmt.Sprintf("/search/repositories?q=%s+in:name+fork:true&per_page=%d&page=%d&sort=updated",
			opts.Search, perPage, page)
		req, err := c.newRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, 0, err
		}
		var result struct {
			TotalCount int      `json:"total_count"`
			Items      []ghRepo `json:"items"`
		}
		if err := c.do(req, &result); err != nil {
			return nil, 0, err
		}
		repos := make([]integrations.RepoInfo, len(result.Items))
		for i := range result.Items {
			repos[i] = result.Items[i].toRepoInfo()
		}
		return repos, result.TotalCount, nil
	}

	// Standard listing with pagination.
	path := fmt.Sprintf("/user/repos?per_page=%d&page=%d&sort=updated", perPage, page)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, 0, err
	}
	var ghRepos []ghRepo
	if err := c.do(req, &ghRepos); err != nil {
		return nil, 0, err
	}
	repos := make([]integrations.RepoInfo, len(ghRepos))
	for i := range ghRepos {
		repos[i] = ghRepos[i].toRepoInfo()
	}
	return repos, 0, nil // total unknown for non-search listing
}

// GetRepo returns information about a single repository.
func (c *Client) GetRepo(ctx context.Context, owner, repo string) (*integrations.RepoInfo, error) {
	path := fmt.Sprintf("/repos/%s/%s", owner, repo)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var r ghRepo
	if err := c.do(req, &r); err != nil {
		return nil, err
	}
	info := r.toRepoInfo()
	return &info, nil
}

// CreateWebhook creates a webhook on the specified repository.
// It returns the webhook ID as a string.
func (c *Client) CreateWebhook(ctx context.Context, owner, repo, url, secret string, events []string) (string, error) {
	path := fmt.Sprintf("/repos/%s/%s/hooks", owner, repo)
	body := map[string]any{
		"name":   "web",
		"active": true,
		"events": events,
		"config": map[string]any{
			"url":          url,
			"content_type": "json",
			"secret":       secret,
			"insecure_ssl": "0",
		},
	}
	req, err := c.newRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return "", err
	}
	var hook ghHook
	if err := c.do(req, &hook); err != nil {
		return "", err
	}
	return strconv.FormatInt(hook.ID, 10), nil
}

// DeleteWebhook removes a webhook from the specified repository.
func (c *Client) DeleteWebhook(ctx context.Context, owner, repo, hookID string) error {
	path := fmt.Sprintf("/repos/%s/%s/hooks/%s", owner, repo, hookID)
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("github: delete webhook: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github: delete webhook %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

// SetCommitStatus creates a commit status on the given SHA.
// state must be one of: error, failure, pending, success.
func (c *Client) SetCommitStatus(ctx context.Context, owner, repo, sha, state, targetURL, description, ctxName string) error {
	path := fmt.Sprintf("/repos/%s/%s/statuses/%s", owner, repo, sha)
	body := map[string]string{
		"state":       state,
		"target_url":  targetURL,
		"description": description,
		"context":     ctxName,
	}
	req, err := c.newRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// Compile-time check that Client implements SCMProvider.
var _ integrations.SCMProvider = (*Client)(nil)
