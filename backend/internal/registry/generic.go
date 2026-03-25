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

// GenericClient implements the Client interface for any OCI/Docker Registry HTTP API v2 compliant registry.
type GenericClient struct {
	baseURL  string
	username string
	password string
	client   *http.Client
}

// NewGenericClient creates a new generic OCI registry client.
func NewGenericClient(url, username, password string) (*GenericClient, error) {
	url = strings.TrimRight(url, "/")
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	return &GenericClient{
		baseURL:  url,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *GenericClient) doRequest(ctx context.Context, method, path string) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	return c.client.Do(req)
}

// ValidateCredentials checks if the credentials are valid by calling GET /v2/.
func (c *GenericClient) ValidateCredentials(ctx context.Context) error {
	resp, err := c.doRequest(ctx, http.MethodGet, "/v2/")
	if err != nil {
		return fmt.Errorf("failed to connect to registry: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("authentication failed: invalid credentials (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response from registry: HTTP %d", resp.StatusCode)
	}
	return nil
}

// ListImages returns images/repositories using GET /v2/_catalog.
func (c *GenericClient) ListImages(ctx context.Context, limit int) ([]models.RegistryImage, error) {
	path := "/v2/_catalog"
	if limit > 0 {
		path = fmt.Sprintf("/v2/_catalog?n=%d", limit)
	}
	resp, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list images: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var catalog struct {
		Repositories []string `json:"repositories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		return nil, fmt.Errorf("failed to decode catalog response: %w", err)
	}

	images := make([]models.RegistryImage, 0, len(catalog.Repositories))
	for _, repo := range catalog.Repositories {
		images = append(images, models.RegistryImage{
			Name: repo,
		})
	}
	return images, nil
}

// ListTags returns tags for a specific image using GET /v2/{name}/tags/list.
func (c *GenericClient) ListTags(ctx context.Context, imageName string) ([]models.RegistryTag, error) {
	path := fmt.Sprintf("/v2/%s/tags/list", imageName)
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

// DeleteTag removes a specific tag using DELETE /v2/{name}/manifests/{reference}.
// First fetches the digest via HEAD request, then deletes by digest.
func (c *GenericClient) DeleteTag(ctx context.Context, imageName, tag string) error {
	// First, get the digest for the tag
	digestPath := fmt.Sprintf("/v2/%s/manifests/%s", imageName, tag)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.baseURL+digestPath, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
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
		return fmt.Errorf("registry did not return a digest for tag %s", tag)
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
