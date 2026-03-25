package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/oarkflow/deploy/backend/internal/models"
)

const (
	ghcrBaseURL = "https://ghcr.io/v2"
)

// GHCRClient implements the Client interface for GitHub Container Registry.
type GHCRClient struct {
	username string
	token    string
	client   *http.Client
}

// NewGHCRClient creates a new GitHub Container Registry client.
func NewGHCRClient(username, token string) (*GHCRClient, error) {
	return &GHCRClient{
		username: username,
		token:    token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *GHCRClient) doRequest(ctx context.Context, method, path string) (*http.Response, error) {
	url := ghcrBaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.username, c.token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	return c.client.Do(req)
}

// ValidateCredentials checks if the GHCR credentials are valid.
func (c *GHCRClient) ValidateCredentials(ctx context.Context) error {
	resp, err := c.doRequest(ctx, http.MethodGet, "/")
	if err != nil {
		return fmt.Errorf("failed to connect to GHCR: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("GHCR authentication failed: invalid token (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response from GHCR: HTTP %d", resp.StatusCode)
	}
	return nil
}

// ListImages returns container images from GHCR for the user/org.
// GHCR does not support the _catalog endpoint, so we try it and fallback gracefully.
func (c *GHCRClient) ListImages(ctx context.Context, limit int) ([]models.RegistryImage, error) {
	// Try the catalog endpoint first
	catalogResp, err := c.doRequest(ctx, http.MethodGet, "/_catalog")
	if err == nil {
		defer catalogResp.Body.Close()
		if catalogResp.StatusCode == http.StatusOK {
			var catalog struct {
				Repositories []string `json:"repositories"`
			}
			if err := json.NewDecoder(catalogResp.Body).Decode(&catalog); err == nil {
				images := make([]models.RegistryImage, 0, len(catalog.Repositories))
				for _, repo := range catalog.Repositories {
					images = append(images, models.RegistryImage{
						Name: repo,
					})
				}
				if limit > 0 && len(images) > limit {
					images = images[:limit]
				}
				return images, nil
			}
		}
	}

	// Fallback: return a helpful message since GHCR doesn't expose catalog API
	return nil, fmt.Errorf("GHCR does not support the catalog API. Use the GitHub API (api.github.com) to list packages, or specify image names directly when listing tags")
}

// ListTags returns tags for a specific image on GHCR.
func (c *GHCRClient) ListTags(ctx context.Context, imageName string) ([]models.RegistryTag, error) {
	path := fmt.Sprintf("/%s/tags/list", imageName)
	resp, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list tags: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tagList struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagList); err != nil {
		return nil, fmt.Errorf("failed to decode tags response: %w", err)
	}

	tags := make([]models.RegistryTag, 0, len(tagList.Tags))
	for _, tag := range tagList.Tags {
		tags = append(tags, models.RegistryTag{
			Name: tag,
		})
	}
	return tags, nil
}

// DeleteTag removes a specific tag from GHCR.
func (c *GHCRClient) DeleteTag(ctx context.Context, imageName, tag string) error {
	// First get the digest
	headPath := fmt.Sprintf("/%s/manifests/%s", imageName, tag)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, ghcrBaseURL+headPath, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(c.username, c.token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get manifest digest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get manifest for tag %s: HTTP %d", tag, resp.StatusCode)
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return fmt.Errorf("GHCR did not return a digest for tag %s", tag)
	}

	// Delete by digest
	deletePath := fmt.Sprintf("/%s/manifests/%s", imageName, digest)
	delResp, err := c.doRequest(ctx, http.MethodDelete, deletePath)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode != http.StatusAccepted && delResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(delResp.Body)
		return fmt.Errorf("failed to delete tag %s: HTTP %d: %s", tag, delResp.StatusCode, string(body))
	}
	return nil
}
