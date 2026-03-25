package integrations

import "context"

// RepoInfo holds repository information returned by any SCM provider.
type RepoInfo struct {
	ID            string `json:"id"`
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	CloneURL      string `json:"clone_url"`
	SSHURL        string `json:"ssh_url"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
	UpdatedAt     string `json:"updated_at,omitempty"`
}

// ListReposOptions controls pagination and search for listing repositories.
type ListReposOptions struct {
	Search  string
	Page    int
	PerPage int
}

// TriggerEvent is the normalized event produced from any webhook payload.
type TriggerEvent struct {
	Provider      string `json:"provider"`       // github, gitlab, bitbucket
	EventType     string `json:"event_type"`     // push, pull_request, tag
	Branch        string `json:"branch"`         // branch name (e.g. "main")
	CommitSHA     string `json:"commit_sha"`     // head commit SHA
	CommitMessage string `json:"commit_message"` // head commit message
	Author        string `json:"author"`         // commit or event author
	RepoFullName  string `json:"repo_full_name"` // owner/repo
	PRNumber      int    `json:"pr_number,omitempty"`
	PRAction      string `json:"pr_action,omitempty"` // opened, synchronize, closed, etc.
	Tag           string `json:"tag,omitempty"`
}

// SCMProvider is the interface that every source-code management provider must implement.
type SCMProvider interface {
	// ListRepos returns repositories accessible with the current credentials.
	// It accepts ListReposOptions for search and pagination.
	// Returns the list of repos and the total count (0 if unknown).
	ListRepos(ctx context.Context, opts ListReposOptions) ([]RepoInfo, int, error)

	// GetRepo returns information about a single repository.
	GetRepo(ctx context.Context, owner, repo string) (*RepoInfo, error)

	// CreateWebhook registers a webhook on the repository and returns the hook ID.
	CreateWebhook(ctx context.Context, owner, repo, url, secret string, events []string) (string, error)

	// DeleteWebhook removes a previously registered webhook.
	DeleteWebhook(ctx context.Context, owner, repo, hookID string) error

	// SetCommitStatus posts a commit status (e.g. pending, success, failure) to the SCM.
	SetCommitStatus(ctx context.Context, owner, repo, sha, state, targetURL, description, ctxName string) error
}
