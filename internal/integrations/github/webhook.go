package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oarkflow/deploy/backend/internal/integrations"
)

// WebhookEvent is the parsed, normalised representation of a GitHub webhook
// delivery. Use ParseWebhookEvent to construct it from the raw payload.
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
		Provider:      "github",
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

// ValidateWebhookSignature verifies the HMAC-SHA256 signature that GitHub
// sends in the X-Hub-Signature-256 header.
//
// signature is the raw header value, e.g. "sha256=abc123...".
func ValidateWebhookSignature(payload []byte, signature, secret string) bool {
	if secret == "" || signature == "" {
		return false
	}

	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return false
	}
	sigHex := signature[len(prefix):]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sigHex), []byte(expected))
}

// ---- internal payload structs ----------------------------------------------

type pushPayload struct {
	Ref        string `json:"ref"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Created    bool   `json:"created"`
	Deleted    bool   `json:"deleted"`
	HeadCommit *struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  struct {
			Name     string `json:"name"`
			Username string `json:"username"`
		} `json:"author"`
	} `json:"head_commit"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Pusher struct {
		Name string `json:"name"`
	} `json:"pusher"`
}

type prPayload struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
		Head struct {
			SHA string `json:"sha"`
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		Title string `json:"title"`
		User  struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"pull_request"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

type createPayload struct {
	RefType    string `json:"ref_type"` // branch or tag
	Ref        string `json:"ref"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
}

// ParseWebhookEvent parses a GitHub webhook payload into a WebhookEvent.
//
// eventType is the value of the X-GitHub-Event header
// (e.g. "push", "pull_request", "create").
func ParseWebhookEvent(eventType string, payload []byte) (*WebhookEvent, error) {
	switch eventType {
	case "push":
		return parsePush(payload)
	case "pull_request":
		return parsePullRequest(payload)
	case "create":
		return parseCreate(payload)
	default:
		return nil, fmt.Errorf("github: unsupported event type: %s", eventType)
	}
}

func parsePush(payload []byte) (*WebhookEvent, error) {
	var p pushPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("github: parse push: %w", err)
	}

	ev := &WebhookEvent{
		RepoFullName: p.Repository.FullName,
	}

	// Tag push: refs/tags/<name>
	if strings.HasPrefix(p.Ref, "refs/tags/") {
		ev.Type = "tag"
		ev.Tag = strings.TrimPrefix(p.Ref, "refs/tags/")
		ev.CommitSHA = p.After
		ev.Author = p.Pusher.Name
		if p.HeadCommit != nil {
			ev.CommitMessage = p.HeadCommit.Message
		}
		return ev, nil
	}

	// Branch push: refs/heads/<name>
	ev.Type = "push"
	ev.Branch = strings.TrimPrefix(p.Ref, "refs/heads/")
	ev.CommitSHA = p.After
	ev.Author = p.Pusher.Name
	if p.HeadCommit != nil {
		ev.CommitSHA = p.HeadCommit.ID
		ev.CommitMessage = p.HeadCommit.Message
		if p.HeadCommit.Author.Username != "" {
			ev.Author = p.HeadCommit.Author.Username
		} else if p.HeadCommit.Author.Name != "" {
			ev.Author = p.HeadCommit.Author.Name
		}
	}
	return ev, nil
}

func parsePullRequest(payload []byte) (*WebhookEvent, error) {
	var p prPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("github: parse pull_request: %w", err)
	}
	return &WebhookEvent{
		Type:          "pull_request",
		Branch:        p.PullRequest.Head.Ref,
		CommitSHA:     p.PullRequest.Head.SHA,
		CommitMessage: p.PullRequest.Title,
		Author:        p.PullRequest.User.Login,
		PRNumber:      p.Number,
		PRAction:      p.Action,
		RepoFullName:  p.Repository.FullName,
	}, nil
}

func parseCreate(payload []byte) (*WebhookEvent, error) {
	var p createPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("github: parse create: %w", err)
	}
	if p.RefType != "tag" {
		return nil, fmt.Errorf("github: create event for %s, not tag", p.RefType)
	}
	return &WebhookEvent{
		Type:         "tag",
		Tag:          p.Ref,
		Author:       p.Sender.Login,
		RepoFullName: p.Repository.FullName,
	}, nil
}
