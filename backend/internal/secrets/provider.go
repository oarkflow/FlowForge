package secrets

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

// SecretProvider is the common interface for external secret backends.
// Implementations include the local encrypted store (SQLite), HashiCorp Vault,
// AWS Secrets Manager, and GCP Secret Manager.
type SecretProvider interface {
	// Type returns the provider identifier (e.g. "local", "vault", "aws", "gcp").
	Type() string

	// Get retrieves a single secret value by key.
	Get(ctx context.Context, key string) (string, error)

	// Set creates or updates a secret.
	Set(ctx context.Context, key, value string) error

	// Delete removes a secret by key.
	Delete(ctx context.Context, key string) error

	// List returns all secret keys available in this provider.
	List(ctx context.Context) ([]string, error)

	// RotateNotify is called to signal that a secret should be rotated.
	// The provider may trigger an external rotation or simply flag it.
	RotateNotify(ctx context.Context, key string) error

	// Healthy returns true if the provider connection is alive.
	Healthy(ctx context.Context) bool
}

// ProviderRegistry manages registered secret providers and implements a
// fallback chain: it first attempts the configured external provider, then
// falls back to the local encrypted store.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]SecretProvider
	// order defines the fallback resolution order (first match wins).
	order []string
}

// NewProviderRegistry creates an empty registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]SecretProvider),
	}
}

// Register adds a provider to the registry. Providers registered earlier have
// higher priority in the fallback chain.
func (r *ProviderRegistry) Register(p SecretProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Type()] = p
	r.order = append(r.order, p.Type())
	log.Info().Str("provider", p.Type()).Msg("secret provider registered")
}

// GetProvider returns a provider by type name, or nil if not found.
func (r *ProviderRegistry) GetProvider(providerType string) SecretProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providers[providerType]
}

// Get attempts to retrieve a secret by walking the provider chain in
// registration order. The first provider that returns a value wins.
func (r *ProviderRegistry) Get(ctx context.Context, key string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, name := range r.order {
		p := r.providers[name]
		val, err := p.Get(ctx, key)
		if err == nil && val != "" {
			return val, nil
		}
		// Log and try next provider on error.
		if err != nil {
			log.Debug().Err(err).Str("provider", name).Str("key", key).Msg("secret not found in provider, trying next")
		}
	}
	return "", fmt.Errorf("secret %q not found in any provider", key)
}

// Set writes a secret to the specified provider (or the first available).
func (r *ProviderRegistry) Set(ctx context.Context, providerType, key, value string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if providerType != "" {
		p, ok := r.providers[providerType]
		if !ok {
			return fmt.Errorf("unknown provider %q", providerType)
		}
		return p.Set(ctx, key, value)
	}

	// Fallback: write to the first provider.
	if len(r.order) == 0 {
		return fmt.Errorf("no secret providers registered")
	}
	return r.providers[r.order[0]].Set(ctx, key, value)
}

// ListProviders returns the type names of all registered providers.
func (r *ProviderRegistry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

// HealthCheck returns a map of provider type -> healthy status.
func (r *ProviderRegistry) HealthCheck(ctx context.Context) map[string]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]bool, len(r.providers))
	for name, p := range r.providers {
		result[name] = p.Healthy(ctx)
	}
	return result
}

// LocalProvider wraps the existing SecretStore to satisfy the SecretProvider
// interface, allowing it to participate in the fallback chain.
type LocalProvider struct {
	store *SecretStore
	// projectID scopes local operations to a project context.
	projectID string
}

// NewLocalProvider creates a LocalProvider backed by the existing SecretStore.
func NewLocalProvider(store *SecretStore, projectID string) *LocalProvider {
	return &LocalProvider{store: store, projectID: projectID}
}

func (p *LocalProvider) Type() string { return "local" }

func (p *LocalProvider) Get(ctx context.Context, key string) (string, error) {
	secrets, err := p.store.GetForInjection(ctx, p.projectID)
	if err != nil {
		return "", err
	}
	val, ok := secrets[key]
	if !ok {
		return "", fmt.Errorf("secret %q not found in local store", key)
	}
	return val, nil
}

func (p *LocalProvider) Set(ctx context.Context, key, value string) error {
	return p.store.Create(ctx, p.projectID, key, value, "system")
}

func (p *LocalProvider) Delete(ctx context.Context, key string) error {
	// Local store Delete requires the secret ID, not key.
	// We list and find the matching key.
	secrets, err := p.store.List(ctx, p.projectID, 10000, 0)
	if err != nil {
		return err
	}
	for _, s := range secrets {
		if s.Key == key {
			return p.store.Delete(ctx, s.ID)
		}
	}
	return fmt.Errorf("secret %q not found", key)
}

func (p *LocalProvider) List(ctx context.Context) ([]string, error) {
	secrets, err := p.store.List(ctx, p.projectID, 10000, 0)
	if err != nil {
		return nil, err
	}
	keys := make([]string, len(secrets))
	for i, s := range secrets {
		keys[i] = s.Key
	}
	return keys, nil
}

func (p *LocalProvider) RotateNotify(_ context.Context, _ string) error {
	// Local store does not support external rotation.
	return nil
}

func (p *LocalProvider) Healthy(_ context.Context) bool {
	return true
}
