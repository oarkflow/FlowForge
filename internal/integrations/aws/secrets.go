package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// SecretsManagerClient is the abstraction over the AWS Secrets Manager API.
// In production, wire a real AWS SDK client; for tests, provide a mock.
type SecretsManagerClient interface {
	// GetSecretValue retrieves the current secret value.
	GetSecretValue(ctx context.Context, secretID string) (string, string, error) // value, versionID, error

	// CreateSecret creates a new secret.
	CreateSecret(ctx context.Context, name, value string) (string, error) // ARN, error

	// PutSecretValue updates an existing secret with a new value.
	PutSecretValue(ctx context.Context, secretID, value string) (string, error) // versionID, error

	// DeleteSecret marks a secret for deletion.
	DeleteSecret(ctx context.Context, secretID string, forceDelete bool) error

	// ListSecrets returns secret names matching the prefix.
	ListSecrets(ctx context.Context, prefix string, maxResults int) ([]SecretEntry, error)

	// Healthy returns true if the service is reachable.
	Healthy(ctx context.Context) bool
}

// SecretEntry is a listing entry from AWS Secrets Manager.
type SecretEntry struct {
	Name        string    `json:"name"`
	ARN         string    `json:"arn"`
	VersionID   string    `json:"version_id"`
	LastChanged time.Time `json:"last_changed"`
}

// Config holds settings for the AWS Secrets Manager provider.
type Config struct {
	// Region is the AWS region (e.g. "us-east-1").
	Region string

	// AccessKeyID and SecretAccessKey are optional static credentials.
	// If empty, the provider falls back to the default AWS credential chain
	// (env vars, shared credentials file, IAM role).
	AccessKeyID     string
	SecretAccessKey string

	// Prefix is prepended to secret names for namespacing.
	// Example: "flowforge/" results in keys like "flowforge/MY_SECRET".
	Prefix string
}

// AWSSecretsProvider implements secrets.SecretProvider backed by AWS Secrets Manager.
type AWSSecretsProvider struct {
	client Client
	cfg    Config

	// versionCache tracks the latest version ID per key.
	mu           sync.RWMutex
	versionCache map[string]string
}

// Client is an alias kept for backward compatibility.
type Client = SecretsManagerClient

// NewAWSSecretsProvider creates the provider.
func NewAWSSecretsProvider(client Client, cfg Config) *AWSSecretsProvider {
	if cfg.Prefix != "" && !strings.HasSuffix(cfg.Prefix, "/") {
		cfg.Prefix += "/"
	}
	return &AWSSecretsProvider{
		client:       client,
		cfg:          cfg,
		versionCache: make(map[string]string),
	}
}

func (a *AWSSecretsProvider) Type() string { return "aws" }

// fullName returns the namespaced AWS secret name.
func (a *AWSSecretsProvider) fullName(key string) string {
	return a.cfg.Prefix + key
}

// Get retrieves a secret value from AWS Secrets Manager.
func (a *AWSSecretsProvider) Get(ctx context.Context, key string) (string, error) {
	value, versionID, err := a.client.GetSecretValue(ctx, a.fullName(key))
	if err != nil {
		return "", fmt.Errorf("aws get %q: %w", key, err)
	}

	a.mu.Lock()
	a.versionCache[key] = versionID
	a.mu.Unlock()

	// If the value looks like JSON with a single "value" field, unwrap it.
	var wrapped map[string]string
	if json.Unmarshal([]byte(value), &wrapped) == nil {
		if v, ok := wrapped["value"]; ok {
			return v, nil
		}
	}
	return value, nil
}

// Set creates or updates a secret in AWS Secrets Manager.
func (a *AWSSecretsProvider) Set(ctx context.Context, key, value string) error {
	name := a.fullName(key)

	// Try to update first; if the secret does not exist, create it.
	versionID, err := a.client.PutSecretValue(ctx, name, value)
	if err != nil {
		// Attempt create.
		_, createErr := a.client.CreateSecret(ctx, name, value)
		if createErr != nil {
			return fmt.Errorf("aws set %q: put failed (%v), create failed (%v)", key, err, createErr)
		}
		log.Info().Str("key", key).Msg("aws secret created")
		return nil
	}

	a.mu.Lock()
	a.versionCache[key] = versionID
	a.mu.Unlock()

	log.Info().Str("key", key).Str("version", versionID).Msg("aws secret updated")
	return nil
}

// Delete marks a secret for deletion.
func (a *AWSSecretsProvider) Delete(ctx context.Context, key string) error {
	if err := a.client.DeleteSecret(ctx, a.fullName(key), false); err != nil {
		return fmt.Errorf("aws delete %q: %w", key, err)
	}

	a.mu.Lock()
	delete(a.versionCache, key)
	a.mu.Unlock()

	return nil
}

// List returns all secret keys under the configured prefix.
func (a *AWSSecretsProvider) List(ctx context.Context) ([]string, error) {
	entries, err := a.client.ListSecrets(ctx, a.cfg.Prefix, 1000)
	if err != nil {
		return nil, fmt.Errorf("aws list: %w", err)
	}
	keys := make([]string, 0, len(entries))
	for _, e := range entries {
		key := strings.TrimPrefix(e.Name, a.cfg.Prefix)
		keys = append(keys, key)
	}
	return keys, nil
}

// RotateNotify logs a rotation request. AWS-native rotation is typically
// configured via a Lambda function in the Secrets Manager console.
func (a *AWSSecretsProvider) RotateNotify(ctx context.Context, key string) error {
	log.Warn().Str("provider", "aws").Str("key", key).Msg("secret rotation requested — configure rotation via AWS Lambda")
	return nil
}

// Healthy checks connectivity to AWS Secrets Manager.
func (a *AWSSecretsProvider) Healthy(ctx context.Context) bool {
	return a.client.Healthy(ctx)
}

// GetVersion returns the cached version ID for a key, if known.
func (a *AWSSecretsProvider) GetVersion(key string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.versionCache[key]
}
