package registry

import (
	"context"
	"fmt"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// GCRClient is a stub implementation for Google Container Registry.
type GCRClient struct {
	url    string
	secret string
}

// NewGCRClient creates a new GCR client stub.
func NewGCRClient(url, secret string) (*GCRClient, error) {
	return &GCRClient{
		url:    url,
		secret: secret,
	}, nil
}

func (c *GCRClient) errMsg() error {
	return fmt.Errorf("GCR integration requires GCP SDK. Full support coming in a future Cloud Integrations phase. Registry URL: %s", c.url)
}

// ValidateCredentials is a stub that returns an informational error.
func (c *GCRClient) ValidateCredentials(_ context.Context) error {
	return c.errMsg()
}

// ListImages is a stub that returns an informational error.
func (c *GCRClient) ListImages(_ context.Context, _ int) ([]models.RegistryImage, error) {
	return nil, c.errMsg()
}

// ListTags is a stub that returns an informational error.
func (c *GCRClient) ListTags(_ context.Context, _ string) ([]models.RegistryTag, error) {
	return nil, c.errMsg()
}

// DeleteTag is a stub that returns an informational error.
func (c *GCRClient) DeleteTag(_ context.Context, _, _ string) error {
	return c.errMsg()
}
