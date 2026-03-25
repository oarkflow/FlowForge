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

const (
	dockerHubAPI    = "https://hub.docker.com/v2"
	dockerHubAuthURL = "https://hub.docker.com/v2/users/login/"
	dockerRegistryV2 = "https://registry.hub.docker.com/v2"
)

// DockerHubClient implements the Client interface for Docker Hub.
type DockerHubClient struct {
	username string
	password string
	token    string
	client   *http.Client
}

// NewDockerHubClient creates a new Docker Hub registry client.
func NewDockerHubClient(username, password string) (*DockerHubClient, error) {
	return &DockerHubClient{
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// authenticate obtains a JWT token from Docker Hub.
func (c *DockerHubClient) authenticate(ctx context.Context) error {
	if c.token != "" {
		return nil
	}

	payload := fmt.Sprintf(`{"username":"%s","password":"%s"}`, c.username, c.password)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dockerHubAuthURL, strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate with Docker Hub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Docker Hub authentication failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var authResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}
	c.token = authResp.Token
	return nil
}

func (c *DockerHubClient) doAuthenticatedRequest(ctx context.Context, method, url string) (*http.Response, error) {
	if err := c.authenticate(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	return c.client.Do(req)
}

// ValidateCredentials checks if the credentials are valid.
func (c *DockerHubClient) ValidateCredentials(ctx context.Context) error {
	c.token = "" // force re-auth
	return c.authenticate(ctx)
}

// ListImages returns repositories from Docker Hub for the authenticated user.
func (c *DockerHubClient) ListImages(ctx context.Context, limit int) ([]models.RegistryImage, error) {
	if limit <= 0 {
		limit = 25
	}
	url := fmt.Sprintf("%s/repositories/%s/?page_size=%d", dockerHubAPI, c.username, limit)
	resp, err := c.doAuthenticatedRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list repositories: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Results []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			PullCount   int64  `json:"pull_count"`
			LastUpdated string `json:"last_updated"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode repositories: %w", err)
	}

	images := make([]models.RegistryImage, 0, len(result.Results))
	for _, repo := range result.Results {
		images = append(images, models.RegistryImage{
			Name:      fmt.Sprintf("%s/%s", c.username, repo.Name),
			PullCount: repo.PullCount,
			PushedAt:  repo.LastUpdated,
		})
	}
	return images, nil
}

// ListTags returns tags for a specific image on Docker Hub.
func (c *DockerHubClient) ListTags(ctx context.Context, imageName string) ([]models.RegistryTag, error) {
	url := fmt.Sprintf("%s/repositories/%s/tags/?page_size=100", dockerHubAPI, imageName)
	resp, err := c.doAuthenticatedRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list tags: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Results []struct {
			Name        string `json:"name"`
			FullSize    int64  `json:"full_size"`
			Digest      string `json:"digest"`
			LastUpdated string `json:"last_updated"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode tags: %w", err)
	}

	tags := make([]models.RegistryTag, 0, len(result.Results))
	for _, t := range result.Results {
		tags = append(tags, models.RegistryTag{
			Name:      t.Name,
			Digest:    t.Digest,
			Size:      t.FullSize,
			CreatedAt: t.LastUpdated,
		})
	}
	return tags, nil
}

// DeleteTag removes a specific tag from Docker Hub.
func (c *DockerHubClient) DeleteTag(ctx context.Context, imageName, tag string) error {
	url := fmt.Sprintf("%s/repositories/%s/tags/%s/", dockerHubAPI, imageName, tag)
	resp, err := c.doAuthenticatedRequest(ctx, http.MethodDelete, url)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete tag %s: HTTP %d: %s", tag, resp.StatusCode, string(body))
	}
	return nil
}
