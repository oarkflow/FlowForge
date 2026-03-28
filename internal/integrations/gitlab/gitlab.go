package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/oarkflow/deploy/backend/internal/integrations"
)

const defaultBaseURL = "https://gitlab.com/api/v4"

// Client communicates with the GitLab REST API (v4).
type Client struct {
	httpClient  *http.Client
	accessToken string // personal access token or OAuth token
	baseURL     string
}

// NewClient returns a GitLab API client authenticated with the given token.
func NewClient(accessToken string) *Client {
	return &Client{
		httpClient:  &http.Client{},
		accessToken: accessToken,
		baseURL:     defaultBaseURL,
	}
}

// NewClientWithBase is like NewClient but allows overriding the API base URL
// (useful for self-managed GitLab instances or testing).
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
			return nil, fmt.Errorf("gitlab: marshal body: %w", err)
		}
		buf = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c *Client) do(req *http.Request, target any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gitlab: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitlab: API error %d: %s", resp.StatusCode, string(b))
	}
	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("gitlab: decode response: %w", err)
		}
	}
	return nil
}

// ---- API response types (internal) ----------------------------------------

type glProject struct {
	ID                int64  `json:"id"`
	PathWithNamespace string `json:"path_with_namespace"`
	Description       string `json:"description"`
	HTTPURLToRepo     string `json:"http_url_to_repo"`
	SSHURLToRepo      string `json:"ssh_url_to_repo"`
	DefaultBranch     string `json:"default_branch"`
	Visibility        string `json:"visibility"` // private, internal, public
	LastActivityAt    string `json:"last_activity_at"`
}

func (p *glProject) toRepoInfo() integrations.RepoInfo {
	return integrations.RepoInfo{
		ID:            strconv.FormatInt(p.ID, 10),
		FullName:      p.PathWithNamespace,
		Description:   p.Description,
		CloneURL:      p.HTTPURLToRepo,
		SSHURL:        p.SSHURLToRepo,
		DefaultBranch: p.DefaultBranch,
		Private:       p.Visibility == "private",
		UpdatedAt:     p.LastActivityAt,
	}
}

type glHook struct {
	ID int64 `json:"id"`
}

// ---- SCMProvider implementation --------------------------------------------

// ListRepos (ListProjects) returns projects the authenticated user has
// access to, with optional search and pagination.
func (c *Client) ListRepos(ctx context.Context, opts integrations.ListReposOptions) ([]integrations.RepoInfo, int, error) {
	page := opts.Page
	if page <= 0 {
		page = 1
	}
	perPage := opts.PerPage
	if perPage <= 0 || perPage > 100 {
		perPage = 30
	}

	path := fmt.Sprintf("/projects?membership=true&per_page=%d&page=%d&order_by=updated_at", perPage, page)
	if opts.Search != "" {
		path += "&search=" + opts.Search
	}

	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("gitlab: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("gitlab: API error %d: %s", resp.StatusCode, string(b))
	}

	var projects []glProject
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, 0, fmt.Errorf("gitlab: decode response: %w", err)
	}

	total := 0
	if h := resp.Header.Get("X-Total"); h != "" {
		if n, err := strconv.Atoi(h); err == nil {
			total = n
		}
	}

	repos := make([]integrations.RepoInfo, len(projects))
	for i := range projects {
		repos[i] = projects[i].toRepoInfo()
	}
	return repos, total, nil
}

// GetRepo returns information about a single project.
// owner and repo are joined with "/" to form the URL-encoded path.
func (c *Client) GetRepo(ctx context.Context, owner, repo string) (*integrations.RepoInfo, error) {
	// GitLab uses URL-encoded namespace/project as the ID in the path.
	encoded := strings.ReplaceAll(owner+"/"+repo, "/", "%2F")
	path := fmt.Sprintf("/projects/%s", encoded)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var p glProject
	if err := c.do(req, &p); err != nil {
		return nil, err
	}
	info := p.toRepoInfo()
	return &info, nil
}

// CreateWebhook creates a project webhook and returns the hook ID as a string.
func (c *Client) CreateWebhook(ctx context.Context, owner, repo, url, secret string, events []string) (string, error) {
	encoded := strings.ReplaceAll(owner+"/"+repo, "/", "%2F")
	path := fmt.Sprintf("/projects/%s/hooks", encoded)

	body := map[string]any{
		"url":                   url,
		"token":                 secret,
		"enable_ssl_verification": true,
	}

	// Map generic event names to GitLab hook booleans.
	for _, e := range events {
		switch e {
		case "push":
			body["push_events"] = true
		case "pull_request", "merge_request":
			body["merge_requests_events"] = true
		case "tag", "tag_push":
			body["tag_push_events"] = true
		case "release":
			body["releases_events"] = true
		case "pipeline":
			body["pipeline_events"] = true
		case "note":
			body["note_events"] = true
		}
	}

	req, err := c.newRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return "", err
	}
	var hook glHook
	if err := c.do(req, &hook); err != nil {
		return "", err
	}
	return strconv.FormatInt(hook.ID, 10), nil
}

// DeleteWebhook removes a webhook from the specified project.
func (c *Client) DeleteWebhook(ctx context.Context, owner, repo, hookID string) error {
	encoded := strings.ReplaceAll(owner+"/"+repo, "/", "%2F")
	path := fmt.Sprintf("/projects/%s/hooks/%s", encoded, hookID)
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gitlab: delete webhook: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitlab: delete webhook %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

// SetCommitStatus creates a commit status on the given SHA.
// state must be one of: pending, running, success, failed, canceled.
func (c *Client) SetCommitStatus(ctx context.Context, owner, repo, sha, state, targetURL, description, ctxName string) error {
	encoded := strings.ReplaceAll(owner+"/"+repo, "/", "%2F")
	path := fmt.Sprintf("/projects/%s/statuses/%s", encoded, sha)
	body := map[string]string{
		"state":       state,
		"target_url":  targetURL,
		"description": description,
		"name":        ctxName,
	}
	req, err := c.newRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// Compile-time check that Client implements SCMProvider.
var _ integrations.SCMProvider = (*Client)(nil)

// ---- Webhook validation & parsing ------------------------------------------

// ValidateWebhookToken verifies the X-Gitlab-Token header value against the
// expected secret. GitLab sends the secret token as a plain string (not HMAC).
func ValidateWebhookToken(token, expectedToken string) bool {
	if expectedToken == "" || token == "" {
		return false
	}
	return token == expectedToken
}

// WebhookEvent is the parsed representation of a GitLab webhook delivery.
type WebhookEvent struct {
	Type          string // push, pull_request (merge_request), tag
	Branch        string
	CommitSHA     string
	CommitMessage string
	Author        string
	PRNumber      int // merge request IID
	PRAction      string
	Tag           string
	RepoFullName  string
}

// ToTriggerEvent converts a WebhookEvent to the provider-agnostic TriggerEvent.
func (e *WebhookEvent) ToTriggerEvent() integrations.TriggerEvent {
	return integrations.TriggerEvent{
		Provider:      "gitlab",
		EventType:     e.Type,
		Branch:        e.Branch,
		CommitSHA:     e.CommitSHA,
		CommitMessage: e.CommitMessage,
		Author:        e.Author,
		RepoFullName:  e.RepoFullName,
		PRNumber:      e.PRNumber,
		PRAction:      e.PRAction,
		Tag:           e.Tag,
	}
}

// ---- internal payload structs ----------------------------------------------

type glPushPayload struct {
	ObjectKind string `json:"object_kind"` // "push" or "tag_push"
	Ref        string `json:"ref"`
	After      string `json:"after"`
	CheckoutSHA string `json:"checkout_sha"`
	UserUsername string `json:"user_username"`
	Commits     []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  struct {
			Name string `json:"name"`
		} `json:"author"`
	} `json:"commits"`
	Project struct {
		PathWithNamespace string `json:"path_with_namespace"`
	} `json:"project"`
}

type glMRPayload struct {
	ObjectKind       string `json:"object_kind"` // "merge_request"
	ObjectAttributes struct {
		IID          int    `json:"iid"`
		Action       string `json:"action"` // open, close, reopen, update, merge
		Title        string `json:"title"`
		SourceBranch string `json:"source_branch"`
		TargetBranch string `json:"target_branch"`
		LastCommit   struct {
			ID string `json:"id"`
		} `json:"last_commit"`
	} `json:"object_attributes"`
	User struct {
		Username string `json:"username"`
	} `json:"user"`
	Project struct {
		PathWithNamespace string `json:"path_with_namespace"`
	} `json:"project"`
}

type glTagPayload struct {
	ObjectKind string `json:"object_kind"` // "tag_push"
	Ref        string `json:"ref"`
	After      string `json:"after"`
	CheckoutSHA string `json:"checkout_sha"`
	UserUsername string `json:"user_username"`
	Project     struct {
		PathWithNamespace string `json:"path_with_namespace"`
	} `json:"project"`
}

// ParseWebhookEvent parses a GitLab webhook payload.
//
// eventType is the value of the X-Gitlab-Event header
// (e.g. "Push Hook", "Merge Request Hook", "Tag Push Hook").
func ParseWebhookEvent(eventType string, payload []byte) (*WebhookEvent, error) {
	switch eventType {
	case "Push Hook":
		return parseGLPush(payload)
	case "Tag Push Hook":
		return parseGLTag(payload)
	case "Merge Request Hook":
		return parseGLMergeRequest(payload)
	default:
		return nil, fmt.Errorf("gitlab: unsupported event type: %s", eventType)
	}
}

func parseGLPush(payload []byte) (*WebhookEvent, error) {
	var p glPushPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("gitlab: parse push: %w", err)
	}

	ev := &WebhookEvent{
		Type:         "push",
		Branch:       strings.TrimPrefix(p.Ref, "refs/heads/"),
		CommitSHA:    p.After,
		Author:       p.UserUsername,
		RepoFullName: p.Project.PathWithNamespace,
	}
	if p.CheckoutSHA != "" {
		ev.CommitSHA = p.CheckoutSHA
	}
	if len(p.Commits) > 0 {
		last := p.Commits[len(p.Commits)-1]
		ev.CommitSHA = last.ID
		ev.CommitMessage = last.Message
	}
	return ev, nil
}

func parseGLTag(payload []byte) (*WebhookEvent, error) {
	var p glTagPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("gitlab: parse tag: %w", err)
	}
	return &WebhookEvent{
		Type:         "tag",
		Tag:          strings.TrimPrefix(p.Ref, "refs/tags/"),
		CommitSHA:    p.CheckoutSHA,
		Author:       p.UserUsername,
		RepoFullName: p.Project.PathWithNamespace,
	}, nil
}

func parseGLMergeRequest(payload []byte) (*WebhookEvent, error) {
	var p glMRPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("gitlab: parse merge_request: %w", err)
	}
	return &WebhookEvent{
		Type:          "pull_request",
		Branch:        p.ObjectAttributes.SourceBranch,
		CommitSHA:     p.ObjectAttributes.LastCommit.ID,
		CommitMessage: p.ObjectAttributes.Title,
		Author:        p.User.Username,
		PRNumber:      p.ObjectAttributes.IID,
		PRAction:      p.ObjectAttributes.Action,
		RepoFullName:  p.Project.PathWithNamespace,
	}, nil
}
