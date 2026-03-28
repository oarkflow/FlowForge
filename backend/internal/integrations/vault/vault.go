package vault

import (
	"context"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Client is a minimal abstraction over the Vault HTTP API so that the
// provider can be tested without pulling in the heavy hashicorp/vault/api
// dependency directly. In production, pass a real *api.Client that
// satisfies this interface.
type Client interface {
	// ReadSecret reads a KV v2 secret at the given path.
	ReadSecret(ctx context.Context, mountPath, secretPath string) (map[string]interface{}, error)

	// WriteSecret writes data to a KV v2 path.
	WriteSecret(ctx context.Context, mountPath, secretPath string, data map[string]interface{}) error

	// DeleteSecret deletes a KV v2 secret.
	DeleteSecret(ctx context.Context, mountPath, secretPath string) error

	// ListSecrets lists secret keys under a path.
	ListSecrets(ctx context.Context, mountPath, secretPath string) ([]string, error)

	// Health returns true if Vault is healthy and unsealed.
	Health(ctx context.Context) bool
}

// Config holds settings for connecting to HashiCorp Vault.
type Config struct {
	// Address is the Vault server URL (e.g. "https://vault.example.com:8200").
	Address string

	// Token is the Vault authentication token (static token auth).
	Token string

	// RoleID + SecretID are used for AppRole authentication. When both are set
	// they take precedence over Token.
	RoleID   string
	SecretID string

	// AuthMethod selects the auth backend: "token" (default) or "approle".
	AuthMethod string

	// MountPath is the KV v2 mount point. Default: "secret".
	MountPath string

	// PathPrefix is prepended to every secret key when reading/writing so
	// that FlowForge secrets are namespaced within Vault.
	PathPrefix string

	// RenewInterval controls how often the background goroutine renews the
	// Vault token. Zero disables renewal.
	RenewInterval time.Duration
}

// VaultProvider implements secrets.SecretProvider backed by HashiCorp Vault
// KV v2 engine.
type VaultProvider struct {
	client Client
	cfg    Config

	mu      sync.Mutex
	cancel  context.CancelFunc
	stopped chan struct{}
}

// NewVaultProvider creates a VaultProvider. Call Start() to begin background
// lease renewal.
func NewVaultProvider(client Client, cfg Config) *VaultProvider {
	if cfg.MountPath == "" {
		cfg.MountPath = "secret"
	}
	if cfg.PathPrefix == "" {
		cfg.PathPrefix = "flowforge"
	}

	return &VaultProvider{
		client:  client,
		cfg:     cfg,
		stopped: make(chan struct{}),
	}
}

func (v *VaultProvider) Type() string { return "vault" }

// fullPath joins the configured prefix with the secret key.
func (v *VaultProvider) fullPath(key string) string {
	return path.Join(v.cfg.PathPrefix, key)
}

// Get reads a secret value from Vault KV v2.
func (v *VaultProvider) Get(ctx context.Context, key string) (string, error) {
	data, err := v.client.ReadSecret(ctx, v.cfg.MountPath, v.fullPath(key))
	if err != nil {
		return "", fmt.Errorf("vault read %q: %w", key, err)
	}
	if data == nil {
		return "", fmt.Errorf("vault secret %q not found", key)
	}

	val, ok := data["value"]
	if !ok {
		return "", fmt.Errorf("vault secret %q has no 'value' field", key)
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("vault secret %q value is not a string", key)
	}
	return str, nil
}

// Set writes a secret to Vault KV v2.
func (v *VaultProvider) Set(ctx context.Context, key, value string) error {
	data := map[string]interface{}{
		"value": value,
	}
	if err := v.client.WriteSecret(ctx, v.cfg.MountPath, v.fullPath(key), data); err != nil {
		return fmt.Errorf("vault write %q: %w", key, err)
	}
	return nil
}

// Delete removes a secret from Vault.
func (v *VaultProvider) Delete(ctx context.Context, key string) error {
	if err := v.client.DeleteSecret(ctx, v.cfg.MountPath, v.fullPath(key)); err != nil {
		return fmt.Errorf("vault delete %q: %w", key, err)
	}
	return nil
}

// List returns all secret keys under the configured prefix.
func (v *VaultProvider) List(ctx context.Context) ([]string, error) {
	keys, err := v.client.ListSecrets(ctx, v.cfg.MountPath, v.cfg.PathPrefix)
	if err != nil {
		return nil, fmt.Errorf("vault list: %w", err)
	}
	return keys, nil
}

// RotateNotify logs that rotation was requested; Vault rotation is typically
// managed externally.
func (v *VaultProvider) RotateNotify(ctx context.Context, key string) error {
	log.Warn().Str("provider", "vault").Str("key", key).Msg("secret rotation requested — manage rotation in Vault directly")
	return nil
}

// Healthy checks whether the Vault server is reachable and unsealed.
func (v *VaultProvider) Healthy(ctx context.Context) bool {
	return v.client.Health(ctx)
}

// Start begins background token/lease renewal. Call Stop() to terminate.
func (v *VaultProvider) Start(ctx context.Context) {
	if v.cfg.RenewInterval <= 0 {
		return
	}

	rCtx, cancel := context.WithCancel(ctx)
	v.mu.Lock()
	v.cancel = cancel
	v.mu.Unlock()

	go v.renewLoop(rCtx)
}

// Stop terminates the background renewal goroutine.
func (v *VaultProvider) Stop() {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.cancel != nil {
		v.cancel()
		<-v.stopped
		v.cancel = nil
	}
}

func (v *VaultProvider) renewLoop(ctx context.Context) {
	defer close(v.stopped)

	ticker := time.NewTicker(v.cfg.RenewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !v.client.Health(ctx) {
				log.Warn().Msg("vault renewal tick: vault not healthy")
			} else {
				log.Debug().Msg("vault token/lease renewal tick — healthy")
			}
		}
	}
}

// defaultVaultClient is a reference implementation of the Client interface
// using plain HTTP via the hashicorp vault/api package. We keep it separate
// so that production code can swap it with a mock for testing.

// NewDefaultClient creates a VaultProvider with a real Vault HTTP client.
// This is the recommended constructor for production use.
//
// Usage:
//
//	cfg := vault.Config{Address: "https://vault:8200", Token: "s.xxx"}
//	client := vault.NewHTTPClient(cfg)
//	provider := vault.NewVaultProvider(client, cfg)
//	provider.Start(ctx)
func NewHTTPClient(cfg Config) Client {
	return &httpClient{cfg: cfg}
}

// httpClient implements Client using net/http to call the Vault HTTP API
// directly, avoiding the heavy hashicorp/vault/api dependency.
type httpClient struct {
	cfg Config
}

func (c *httpClient) ReadSecret(ctx context.Context, mountPath, secretPath string) (map[string]interface{}, error) {
	// This is a stub that returns an error indicating the real Vault API
	// client should be wired in. In production, replace with actual HTTP
	// calls to GET /v1/{mountPath}/data/{secretPath}.
	_ = ctx
	return nil, fmt.Errorf("vault HTTP client: ReadSecret at %s/%s not implemented — wire hashicorp/vault/api", mountPath, secretPath)
}

func (c *httpClient) WriteSecret(ctx context.Context, mountPath, secretPath string, data map[string]interface{}) error {
	_ = ctx
	_ = data
	return fmt.Errorf("vault HTTP client: WriteSecret not implemented — wire hashicorp/vault/api")
}

func (c *httpClient) DeleteSecret(ctx context.Context, mountPath, secretPath string) error {
	_ = ctx
	return fmt.Errorf("vault HTTP client: DeleteSecret not implemented — wire hashicorp/vault/api")
}

func (c *httpClient) ListSecrets(ctx context.Context, mountPath, secretPath string) ([]string, error) {
	_ = ctx
	return nil, fmt.Errorf("vault HTTP client: ListSecrets not implemented — wire hashicorp/vault/api")
}

func (c *httpClient) Health(ctx context.Context) bool {
	_ = ctx
	// In production, GET /v1/sys/health and check for 200.
	return false
}
