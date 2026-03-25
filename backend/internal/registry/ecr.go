package registry

import (
	"context"
	"fmt"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// ECRClient is a stub implementation for AWS Elastic Container Registry.
type ECRClient struct {
	url    string
	secret string
}

// NewECRClient creates a new ECR client stub.
func NewECRClient(url, secret string) (*ECRClient, error) {
	return &ECRClient{
		url:    url,
		secret: secret,
	}, nil
}

func (c *ECRClient) errMsg() error {
	return fmt.Errorf("ECR integration requires AWS SDK. Full support coming in a future Cloud Integrations phase. Registry URL: %s", c.url)
}

// ValidateCredentials is a stub that returns an informational error.
func (c *ECRClient) ValidateCredentials(_ context.Context) error {
	return c.errMsg()
}

// ListImages is a stub that returns an informational error.
func (c *ECRClient) ListImages(_ context.Context, _ int) ([]models.RegistryImage, error) {
	return nil, c.errMsg()
}

// ListTags is a stub that returns an informational error.
func (c *ECRClient) ListTags(_ context.Context, _ string) ([]models.RegistryTag, error) {
	return nil, c.errMsg()
}

// DeleteTag is a stub that returns an informational error.
func (c *ECRClient) DeleteTag(_ context.Context, _, _ string) error {
	return c.errMsg()
}
