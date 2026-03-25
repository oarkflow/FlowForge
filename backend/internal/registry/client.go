package registry

import (
	"context"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// Client interface for all registry types
type Client interface {
	// ListImages returns images/repositories in the registry
	ListImages(ctx context.Context, limit int) ([]models.RegistryImage, error)
	// ListTags returns tags for a specific image
	ListTags(ctx context.Context, imageName string) ([]models.RegistryTag, error)
	// DeleteTag removes a specific tag
	DeleteTag(ctx context.Context, imageName, tag string) error
	// ValidateCredentials checks if the credentials are valid
	ValidateCredentials(ctx context.Context) error
}

// NewClient creates a registry client based on type
func NewClient(reg *models.Registry, password string) (Client, error) {
	switch reg.Type {
	case "dockerhub":
		return NewDockerHubClient(reg.Username, password)
	case "ghcr":
		return NewGHCRClient(reg.Username, password)
	case "ecr":
		return NewECRClient(reg.URL, password)
	case "gcr":
		return NewGCRClient(reg.URL, password)
	case "acr":
		return NewACRClient(reg.URL, reg.Username, password)
	case "harbor":
		return NewHarborClient(reg.URL, reg.Username, password)
	default:
		return NewGenericClient(reg.URL, reg.Username, password)
	}
}
