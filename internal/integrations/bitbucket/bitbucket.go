package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/oarkflow/deploy/backend/internal/integrations"
)

const defaultBaseURL = "https://api.bitbucket.org/2.0"

// Client communicates with the Bitbucket Cloud REST API (v2).
// Authentication uses an App Password with the username:app_password scheme
// (HTTP Basic Auth).
type Client struct {
	httpClient *http.Client
	username   string
	appPass    string // app password
	baseURL    string
}

// NewClient returns a Bitbucket Cloud API client.
// accessToken should be in the format "username:app_password".
func NewClient(accessToken string) *Client {
	user, pass, _ := strings.Cut(accessToken, ":")
	return &Client{
		httpClient: &http.Client{},
		username:   user,
		appPass:    pass,
		baseURL:    defaultBaseURL,
	}
}

// NewClientWithBase is like NewClient but allows overriding the API base URL
// (useful for testing).
func NewClientWithBase(accessToken, baseURL string) *Client {
	user, pass, _ := strings.Cut(accessToken, ":")
	return &Client{
		httpClient: &http.Client{},
		username:   user,
		appPass:    pass,
		baseURL:    baseURL,
	}
}

// ---- helpers ---------------------------------------------------------------

func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	url := c.baseURL + path

	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("bitbucket: marshal body: %w", err)
		}
		buf = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, buf)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.username, c.appPass)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c *Client) do(req *http.Request, target any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("bitbucket: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bitbucket: API error %d: %s", resp.StatusCode, string(b))
	}
	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("bitbucket: decode response: %w", err)
		}
	}
	return nil
}

// ---- API response types (internal) ----------------------------------------

type bbRepo struct {
	UUID      string `json:"uuid"`
	FullName  string `json:"full_name"`
	IsPrivate bool   `json:"is_private"`
	Mainbranch *struct {
		Name string `json:"name"`
	} `json:"mainbranch"`
	Links struct {
		Clone []struct {
			Name string `json:"name"` // https or ssh
			Href string `json:"href"`
		} `json:"clone"`
	} `json:"links"`
}

func (r *bbRepo) toRepoInfo() integrations.RepoInfo {
	info := integrations.RepoInfo{
		ID:       r.UUID,
		FullName: r.FullName,
		Private:  r.IsPrivate,
	}
	if r.Mainbranch != nil {
		info.DefaultBranch = r.Mainbranch.Name
	} else {
		info.DefaultBranch = "main"
	}
	for _, link := range r.Links.Clone {
		switch link.Name {
		case "https":
			info.CloneURL = link.Href
		case "ssh":
			info.SSHURL = link.Href
		}
	}
	return info
}

type bbHook struct {
	UUID string `json:"uuid"`
}

// ---- SCMProvider implementation --------------------------------------------

// ListRepos returns repositories the authenticated user has access to,
// with optional search and pagination.
func (c *Client) ListRepos(ctx context.Context, opts integrations.ListReposOptions) ([]integrations.RepoInfo, int, error) {
	page := opts.Page
	if page <= 0 {
		page = 1
	}
	perPage := opts.PerPage
	if perPage <= 0 || perPage > 100 {
		perPage = 30
	}

	path := fmt.Sprintf("/repositories/%s?pagelen=%d&page=%d", c.username, perPage, page)
	if opts.Search != "" {
		path += "&q=name~%22" + opts.Search + "%22"
	}

	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.SetBasicAuth(c.username, c.appPass)

	var paged struct {
		Values []bbRepo `json:"values"`
		Size   int      `json:"size"`
	}
	if err := c.do(req, &paged); err != nil {
		return nil, 0, err
	}

	repos := make([]integrations.RepoInfo, len(paged.Values))
	for i := range paged.Values {
		repos[i] = paged.Values[i].toRepoInfo()
	}
	return repos, paged.Size, nil
}

// GetRepo returns information about a single repository.
// owner is the workspace slug; repo is the repository slug.
func (c *Client) GetRepo(ctx context.Context, owner, repo string) (*integrations.RepoInfo, error) {
	path := fmt.Sprintf("/repositories/%s/%s", owner, repo)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var r bbRepo
	if err := c.do(req, &r); err != nil {
		return nil, err
	}
	info := r.toRepoInfo()
	return &info, nil
}

// CreateWebhook creates a webhook on the specified repository.
// The secret parameter is set as part of the webhook description because
// Bitbucket Cloud does not support shared-secret HMAC signing for webhooks.
// Instead, use IP allowlisting or other verification on your receiver.
// events should use Bitbucket event keys like "repo:push", "pullrequest:created", etc.
func (c *Client) CreateWebhook(ctx context.Context, owner, repo, url, secret string, events []string) (string, error) {
	path := fmt.Sprintf("/repositories/%s/%s/hooks", owner, repo)
	body := map[string]any{
		"description": "FlowForge CI/CD webhook",
		"url":         url,
		"active":      true,
		"events":      events,
		"secret":      secret,
	}
	req, err := c.newRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return "", err
	}
	var hook bbHook
	if err := c.do(req, &hook); err != nil {
		return "", err
	}
	// Bitbucket UUIDs are wrapped in braces, e.g. {uuid}
	return hook.UUID, nil
}

// DeleteWebhook removes a webhook from the specified repository.
func (c *Client) DeleteWebhook(ctx context.Context, owner, repo, hookID string) error {
	path := fmt.Sprintf("/repositories/%s/%s/hooks/%s", owner, repo, hookID)
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("bitbucket: delete webhook: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bitbucket: delete webhook %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

// SetCommitStatus creates a build status on a commit.
// state must be one of: INPROGRESS, SUCCESSFUL, FAILED, STOPPED.
func (c *Client) SetCommitStatus(ctx context.Context, owner, repo, sha, state, targetURL, description, ctxName string) error {
	path := fmt.Sprintf("/repositories/%s/%s/commit/%s/statuses/build", owner, repo, sha)
	body := map[string]string{
		"state":       mapStateToBitbucket(state),
		"key":         ctxName,
		"url":         targetURL,
		"description": description,
	}
	req, err := c.newRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// mapStateToBitbucket converts generic status states to Bitbucket's expected values.
func mapStateToBitbucket(state string) string {
	switch strings.ToLower(state) {
	case "pending", "running":
		return "INPROGRESS"
	case "success":
		return "SUCCESSFUL"
	case "failure", "error":
		return "FAILED"
	case "cancelled":
		return "STOPPED"
	default:
		return strings.ToUpper(state)
	}
}

// Compile-time check that Client implements SCMProvider.
var _ integrations.SCMProvider = (*Client)(nil)

// ---- Webhook validation & parsing ------------------------------------------

// WebhookEvent is the parsed representation of a Bitbucket webhook delivery.
type WebhookEvent struct {
	Type          string // push, pull_request, tag
	Branch        string
	CommitSHA     string
	CommitMessage string
	Author        string
	PRNumber      int
	PRAction      string
	Tag           string
	RepoFullName  string
}

// ToTriggerEvent converts a WebhookEvent to the provider-agnostic TriggerEvent.
func (e *WebhookEvent) ToTriggerEvent() integrations.TriggerEvent {
	return integrations.TriggerEvent{
		Provider:      "bitbucket",
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

type bbPushPayload struct {
	Push struct {
		Changes []struct {
			New *struct {
				Type   string `json:"type"` // branch or tag
				Name   string `json:"name"`
				Target struct {
					Hash    string `json:"hash"`
					Message string `json:"message"`
					Author  struct {
						User struct {
							DisplayName string `json:"display_name"`
							Nickname    string `json:"nickname"`
						} `json:"user"`
					} `json:"author"`
				} `json:"target"`
			} `json:"new"`
			Old *struct {
				Type string `json:"type"`
				Name string `json:"name"`
			} `json:"old"`
		} `json:"changes"`
	} `json:"push"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Actor struct {
		DisplayName string `json:"display_name"`
		Nickname    string `json:"nickname"`
	} `json:"actor"`
}

type bbPRPayload struct {
	PullRequest struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		State string `json:"state"` // OPEN, MERGED, DECLINED, SUPERSEDED
		Source struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
			Commit struct {
				Hash string `json:"hash"`
			} `json:"commit"`
		} `json:"source"`
		Destination struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
		} `json:"destination"`
		Author struct {
			DisplayName string `json:"display_name"`
			Nickname    string `json:"nickname"`
		} `json:"author"`
	} `json:"pullrequest"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

// ParseWebhookEvent parses a Bitbucket webhook payload.
//
// eventType is the value of the X-Event-Key header
// (e.g. "repo:push", "pullrequest:created", "pullrequest:updated").
func ParseWebhookEvent(eventType string, payload []byte) (*WebhookEvent, error) {
	switch {
	case eventType == "repo:push":
		return parseBBPush(payload)
	case strings.HasPrefix(eventType, "pullrequest:"):
		return parseBBPullRequest(eventType, payload)
	default:
		return nil, fmt.Errorf("bitbucket: unsupported event type: %s", eventType)
	}
}

func parseBBPush(payload []byte) (*WebhookEvent, error) {
	var p bbPushPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("bitbucket: parse push: %w", err)
	}

	ev := &WebhookEvent{
		RepoFullName: p.Repository.FullName,
		Author:       p.Actor.Nickname,
	}
	if ev.Author == "" {
		ev.Author = p.Actor.DisplayName
	}

	// Look at the first change to determine push type.
	if len(p.Push.Changes) == 0 {
		return nil, fmt.Errorf("bitbucket: push event with no changes")
	}

	change := p.Push.Changes[0]
	if change.New == nil {
		// Deletion event — nothing to trigger on.
		return nil, fmt.Errorf("bitbucket: push event is a deletion")
	}

	target := change.New.Target
	ev.CommitSHA = target.Hash
	ev.CommitMessage = target.Message
	if target.Author.User.Nickname != "" {
		ev.Author = target.Author.User.Nickname
	}

	switch change.New.Type {
	case "tag":
		ev.Type = "tag"
		ev.Tag = change.New.Name
	default:
		ev.Type = "push"
		ev.Branch = change.New.Name
	}

	return ev, nil
}

func parseBBPullRequest(eventType string, payload []byte) (*WebhookEvent, error) {
	var p bbPRPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("bitbucket: parse pullrequest: %w", err)
	}

	// Extract action from event key: "pullrequest:created" → "created"
	action := strings.TrimPrefix(eventType, "pullrequest:")

	author := p.PullRequest.Author.Nickname
	if author == "" {
		author = p.PullRequest.Author.DisplayName
	}

	return &WebhookEvent{
		Type:          "pull_request",
		Branch:        p.PullRequest.Source.Branch.Name,
		CommitSHA:     p.PullRequest.Source.Commit.Hash,
		CommitMessage: p.PullRequest.Title,
		Author:        author,
		PRNumber:      p.PullRequest.ID,
		PRAction:      action,
		RepoFullName:  p.Repository.FullName,
	}, nil
}
