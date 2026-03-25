package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// HarborClient implements the Client interface for Harbor registry.
type HarborClient struct {
	baseURL  string
	username string
	password string
	client   *http.Client
}

// NewHarborClient creates a new Harbor registry client.
func NewHarborClient(url, username, password string) (*HarborClient, error) {
	url = strings.TrimRight(url, "/")
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	return &HarborClient{
		baseURL:  url,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *HarborClient) doRequest(ctx context.Context, method, path string) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Accept", "application/json")
	return c.client.Do(req)
}

// ValidateCredentials checks if the Harbor credentials are valid.
func (c *HarborClient) ValidateCredentials(ctx context.Context) error {
	// Try the Harbor API health endpoint or user info
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v2.0/users/current")
	if err != nil {
		return fmt.Errorf("failed to connect to Harbor: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("Harbor authentication failed: invalid credentials (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response from Harbor: HTTP %d", resp.StatusCode)
	}
	return nil
}

// ListImages returns repositories from all Harbor projects.
func (c *HarborClient) ListImages(ctx context.Context, limit int) ([]models.RegistryImage, error) {
	// First list projects
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v2.0/projects?page_size=50")
	if err != nil {
		return nil, fmt.Errorf("failed to list Harbor projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list Harbor projects: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var projects []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("failed to decode projects: %w", err)
	}

	var images []models.RegistryImage
	for _, proj := range projects {
		repoResp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v2.0/projects/%s/repositories?page_size=50", proj.Name))
		if err != nil {
			continue
		}

		if repoResp.StatusCode == http.StatusOK {
			var repos []struct {
				Name          string `json:"name"`
				ArtifactCount int64  `json:"artifact_count"`
				PullCount     int64  `json:"pull_count"`
				UpdateTime    string `json:"update_time"`
			}
			if err := json.NewDecoder(repoResp.Body).Decode(&repos); err == nil {
				for _, repo := range repos {
					images = append(images, models.RegistryImage{
						Name:      repo.Name,
						PullCount: repo.PullCount,
						PushedAt:  repo.UpdateTime,
					})
				}
			}
		}
		repoResp.Body.Close()

		if limit > 0 && len(images) >= limit {
			images = images[:limit]
			break
		}
	}
	return images, nil
}

// ListTags returns tags for a specific repository in Harbor.
func (c *HarborClient) ListTags(ctx context.Context, imageName string) ([]models.RegistryTag, error) {
	// Harbor API uses project_name/repo_name format
	path := fmt.Sprintf("/api/v2.0/projects/%s/artifacts?page_size=100", imageName)
	resp, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list artifacts: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var artifacts []struct {
		Digest string `json:"digest"`
		Size   int64  `json:"size"`
		Tags   []struct {
			Name string `json:"name"`
		} `json:"tags"`
		PushTime string `json:"push_time"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&artifacts); err != nil {
		return nil, fmt.Errorf("failed to decode artifacts: %w", err)
	}

	var tags []models.RegistryTag
	for _, artifact := range artifacts {
		for _, tag := range artifact.Tags {
			tags = append(tags, models.RegistryTag{
				Name:      tag.Name,
				Digest:    artifact.Digest,
				Size:      artifact.Size,
				CreatedAt: artifact.PushTime,
			})
		}
	}
	return tags, nil
}

// DeleteTag removes a specific tag from Harbor.
func (c *HarborClient) DeleteTag(ctx context.Context, imageName, tag string) error {
	// Use the v2 registry API for deletion
	path := fmt.Sprintf("/v2/%s/manifests/%s", imageName, tag)

	// First get the digest
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)
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
		return fmt.Errorf("Harbor did not return a digest for tag %s", tag)
	}

	// Delete by digest
	deletePath := fmt.Sprintf("/v2/%s/manifests/%s", imageName, digest)
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
