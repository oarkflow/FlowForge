package importer

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// RepositoryMetadata captures normalized SCM details for an imported source.
type RepositoryMetadata struct {
	Provider      string `json:"provider"`
	FullName      string `json:"full_name,omitempty"`
	CloneURL      string `json:"clone_url,omitempty"`
	SSHURL        string `json:"ssh_url,omitempty"`
	DefaultBranch string `json:"default_branch,omitempty"`
}

type repositoryRef struct {
	Provider string
	Host     string
	FullName string
	Owner    string
	Repo     string
}

func parseRepositoryURL(raw string) (*repositoryRef, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("repository URL is required")
	}

	if strings.Contains(raw, "://") {
		return parseHTTPRepositoryURL(raw)
	}

	if strings.Contains(raw, "@") && strings.Contains(raw, ":") {
		return parseSCPRepositoryURL(raw)
	}

	return nil, fmt.Errorf("unsupported repository URL format")
}

func parseHTTPRepositoryURL(raw string) (*repositoryRef, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	fullName := strings.TrimPrefix(strings.TrimSuffix(u.Path, ".git"), "/")
	ref := &repositoryRef{
		Provider: detectProviderFromHost(u.Hostname()),
		Host:     u.Hostname(),
		FullName: fullName,
	}
	assignOwnerAndRepo(ref)
	return ref, nil
}

func parseSCPRepositoryURL(raw string) (*repositoryRef, error) {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid SCP-style repository URL")
	}
	hostPart := parts[0]
	if at := strings.Index(hostPart, "@"); at >= 0 {
		hostPart = hostPart[at+1:]
	}
	fullName := strings.TrimPrefix(strings.TrimSuffix(parts[1], ".git"), "/")
	ref := &repositoryRef{
		Provider: detectProviderFromHost(hostPart),
		Host:     hostPart,
		FullName: fullName,
	}
	assignOwnerAndRepo(ref)
	return ref, nil
}

func assignOwnerAndRepo(ref *repositoryRef) {
	if ref == nil || ref.FullName == "" {
		return
	}
	parts := strings.Split(ref.FullName, "/")
	if len(parts) == 1 {
		ref.Repo = parts[0]
		return
	}
	ref.Owner = strings.Join(parts[:len(parts)-1], "/")
	ref.Repo = parts[len(parts)-1]
}

func detectProviderFromHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	switch {
	case strings.Contains(host, "github"):
		return "github"
	case strings.Contains(host, "gitlab"):
		return "gitlab"
	case strings.Contains(host, "bitbucket"):
		return "bitbucket"
	default:
		return "git"
	}
}

func injectTokenIntoCloneURL(cloneURL, token, provider string) string {
	if token == "" {
		return cloneURL
	}
	u, err := url.Parse(cloneURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return cloneURL
	}

	switch provider {
	case "github":
		u.User = url.UserPassword("x-access-token", token)
	case "gitlab":
		u.User = url.UserPassword("oauth2", token)
	case "bitbucket":
		if user, pass, ok := strings.Cut(token, ":"); ok && user != "" && pass != "" {
			u.User = url.UserPassword(user, pass)
		} else {
			u.User = url.UserPassword("x-token-auth", token)
		}
	default:
		if user, pass, ok := strings.Cut(token, ":"); ok && user != "" && pass != "" {
			u.User = url.UserPassword(user, pass)
		} else {
			u.User = url.UserPassword("oauth2", token)
		}
	}

	return u.String()
}

func normalizeRepositoryMetadata(cloneURL, branch, sshURL string) RepositoryMetadata {
	meta := RepositoryMetadata{
		Provider:      "git",
		CloneURL:      cloneURL,
		SSHURL:        sshURL,
		DefaultBranch: branch,
	}
	if ref, err := parseRepositoryURL(cloneURL); err == nil {
		meta.Provider = ref.Provider
		meta.FullName = ref.FullName
	}
	if meta.FullName == "" && sshURL != "" {
		if ref, err := parseRepositoryURL(sshURL); err == nil {
			meta.Provider = ref.Provider
			meta.FullName = ref.FullName
		}
	}
	if meta.FullName == "" && cloneURL != "" {
		meta.FullName = strings.TrimSuffix(path.Base(cloneURL), ".git")
	}
	return meta
}
