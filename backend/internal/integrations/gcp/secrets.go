package gcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

// SecretManagerClient is the abstraction over the GCP Secret Manager API.
// Provide a real GCP client in production; use a mock for tests.
type SecretManagerClient interface {
	// AccessSecretVersion retrieves the payload of a secret version.
	// version can be "latest" or a numeric version string.
	AccessSecretVersion(ctx context.Context, project, secretID, version string) ([]byte, error)

	// CreateSecret creates a new secret (without a version).
	CreateSecret(ctx context.Context, project, secretID string) error

	// AddSecretVersion adds a new version to an existing secret.
	AddSecretVersion(ctx context.Context, project, secretID string, payload []byte) (string, error) // versionName, error

	// DeleteSecret permanently deletes a secret and all its versions.
	DeleteSecret(ctx context.Context, project, secretID string) error

	// ListSecrets returns all secret IDs in the project matching the prefix.
	ListSecrets(ctx context.Context, project, prefix string) ([]string, error)

	// Healthy returns true if the service is reachable.
	Healthy(ctx context.Context) bool
}

// Config holds settings for the GCP Secret Manager provider.
type Config struct {
	// ProjectID is the GCP project (e.g. "my-project-123").
	ProjectID string

	// CredentialsFile is the path to a service account key JSON file.
	// If empty, application default credentials are used.
	CredentialsFile string

	// Prefix is prepended to secret IDs for namespacing.
	// GCP secret IDs must match [a-zA-Z0-9_-]+ so we use "-" as separator.
	// Example prefix: "flowforge-"
	Prefix string
}

// GCPSecretsProvider implements secrets.SecretProvider backed by GCP Secret Manager.
type GCPSecretsProvider struct {
	client SecretManagerClient
	cfg    Config
}

// NewGCPSecretsProvider creates the provider.
func NewGCPSecretsProvider(client SecretManagerClient, cfg Config) *GCPSecretsProvider {
	if cfg.Prefix != "" && !strings.HasSuffix(cfg.Prefix, "-") {
		cfg.Prefix += "-"
	}
	return &GCPSecretsProvider{
		client: client,
		cfg:    cfg,
	}
}

func (g *GCPSecretsProvider) Type() string { return "gcp" }

// fullID returns the GCP-safe secret ID with the configured prefix.
func (g *GCPSecretsProvider) fullID(key string) string {
	// GCP secret IDs are restricted to [a-zA-Z0-9_-].
	// Replace unsupported characters.
	safe := strings.NewReplacer("/", "-", ".", "-").Replace(key)
	return g.cfg.Prefix + safe
}

// Get retrieves the latest version of a secret from GCP Secret Manager.
func (g *GCPSecretsProvider) Get(ctx context.Context, key string) (string, error) {
	payload, err := g.client.AccessSecretVersion(ctx, g.cfg.ProjectID, g.fullID(key), "latest")
	if err != nil {
		return "", fmt.Errorf("gcp get %q: %w", key, err)
	}
	return string(payload), nil
}

// Set creates or updates a secret. If the secret does not exist it is created
// first, then a new version is added with the provided value.
func (g *GCPSecretsProvider) Set(ctx context.Context, key, value string) error {
	secretID := g.fullID(key)

	// Try adding a version directly (fast path — secret already exists).
	_, err := g.client.AddSecretVersion(ctx, g.cfg.ProjectID, secretID, []byte(value))
	if err != nil {
		// Secret may not exist yet; create it and retry.
		if createErr := g.client.CreateSecret(ctx, g.cfg.ProjectID, secretID); createErr != nil {
			return fmt.Errorf("gcp set %q: add version failed (%v), create failed (%v)", key, err, createErr)
		}
		if _, retryErr := g.client.AddSecretVersion(ctx, g.cfg.ProjectID, secretID, []byte(value)); retryErr != nil {
			return fmt.Errorf("gcp set %q: %w", key, retryErr)
		}
	}

	log.Info().Str("key", key).Msg("gcp secret version added")
	return nil
}

// Delete permanently removes the secret and all its versions.
func (g *GCPSecretsProvider) Delete(ctx context.Context, key string) error {
	if err := g.client.DeleteSecret(ctx, g.cfg.ProjectID, g.fullID(key)); err != nil {
		return fmt.Errorf("gcp delete %q: %w", key, err)
	}
	return nil
}

// List returns all secret keys under the configured prefix.
func (g *GCPSecretsProvider) List(ctx context.Context) ([]string, error) {
	ids, err := g.client.ListSecrets(ctx, g.cfg.ProjectID, g.cfg.Prefix)
	if err != nil {
		return nil, fmt.Errorf("gcp list: %w", err)
	}
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		key := strings.TrimPrefix(id, g.cfg.Prefix)
		keys = append(keys, key)
	}
	return keys, nil
}

// RotateNotify logs a rotation request. GCP rotation should be configured
// via rotation schedules in the Secret Manager console or API.
func (g *GCPSecretsProvider) RotateNotify(ctx context.Context, key string) error {
	log.Warn().Str("provider", "gcp").Str("key", key).Msg("secret rotation requested — configure rotation via GCP Secret Manager")
	return nil
}

// Healthy checks connectivity to GCP Secret Manager.
func (g *GCPSecretsProvider) Healthy(ctx context.Context) bool {
	return g.client.Healthy(ctx)
}
