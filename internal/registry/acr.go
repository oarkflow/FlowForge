package registry

import (
	"context"
	"fmt"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// ACRClient is a stub implementation for Azure Container Registry.
type ACRClient struct {
	url      string
	username string
	password string
}

// NewACRClient creates a new ACR client stub.
func NewACRClient(url, username, password string) (*ACRClient, error) {
	return &ACRClient{
		url:      url,
		username: username,
		password: password,
	}, nil
}

func (c *ACRClient) errMsg() error {
	return fmt.Errorf("ACR integration requires Azure SDK. Full support coming in a future Cloud Integrations phase. Registry URL: %s", c.url)
}

// ValidateCredentials is a stub that returns an informational error.
func (c *ACRClient) ValidateCredentials(_ context.Context) error {
	return c.errMsg()
}

// ListImages is a stub that returns an informational error.
func (c *ACRClient) ListImages(_ context.Context, _ int) ([]models.RegistryImage, error) {
	return nil, c.errMsg()
}

// ListTags is a stub that returns an informational error.
func (c *ACRClient) ListTags(_ context.Context, _ string) ([]models.RegistryTag, error) {
	return nil, c.errMsg()
}

// DeleteTag is a stub that returns an informational error.
func (c *ACRClient) DeleteTag(_ context.Context, _, _ string) error {
	return c.errMsg()
}
