package secrets

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/models"
	"github.com/oarkflow/deploy/backend/pkg/crypto"
)

// DecryptedSecret is a secret with its plaintext value available.
type DecryptedSecret struct {
	ID        string    `json:"id"`
	ProjectID *string   `json:"project_id,omitempty"`
	OrgID     *string   `json:"org_id,omitempty"`
	Scope     string    `json:"scope"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Masked    int       `json:"masked"`
	CreatedBy *string   `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SecretMetadata is a secret listing entry without the decrypted value.
type SecretMetadata struct {
	ID        string    `json:"id"`
	ProjectID *string   `json:"project_id,omitempty"`
	OrgID     *string   `json:"org_id,omitempty"`
	Scope     string    `json:"scope"`
	Key       string    `json:"key"`
	Masked    int       `json:"masked"`
	IsEmpty   bool      `json:"is_empty"`
	CreatedBy *string   `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SecretStore wraps the secret repository and handles encryption/decryption
// transparently using AES-256-GCM.
type SecretStore struct {
	repos         *queries.Repositories
	encryptionKey []byte
}

// NewSecretStore creates a new SecretStore. The encryptionKey must be exactly
// 32 bytes for AES-256.
func NewSecretStore(repos *queries.Repositories, encryptionKey []byte) *SecretStore {
	return &SecretStore{
		repos:         repos,
		encryptionKey: encryptionKey,
	}
}

// Create encrypts the plaintext value and stores a new secret scoped to the
// given project.
func (s *SecretStore) Create(ctx context.Context, projectID, key, value, createdBy string) error {
	if key == "" {
		return errors.New("secret key must not be empty")
	}
	if value == "" {
		return errors.New("secret value must not be empty")
	}

	encrypted, err := crypto.Encrypt(s.encryptionKey, value)
	if err != nil {
		return fmt.Errorf("encrypting secret: %w", err)
	}

	secret := &models.Secret{
		ProjectID: &projectID,
		Scope:     "project",
		Key:       key,
		ValueEnc:  encrypted,
		Masked:    1,
		CreatedBy: &createdBy,
	}

	if err := s.repos.Secrets.Create(ctx, secret); err != nil {
		return fmt.Errorf("storing secret: %w", err)
	}
	return nil
}

// CreateEmpty creates a secret placeholder with an empty encrypted value.
// This is used during project import to pre-populate secret keys that
// the pipeline references so the user can fill them in later.
func (s *SecretStore) CreateEmpty(ctx context.Context, projectID, key, createdBy string) error {
	if key == "" {
		return errors.New("secret key must not be empty")
	}

	// Encrypt an empty string as a placeholder.
	encrypted, err := crypto.Encrypt(s.encryptionKey, "")
	if err != nil {
		return fmt.Errorf("encrypting empty secret: %w", err)
	}

	secret := &models.Secret{
		ProjectID: &projectID,
		Scope:     "project",
		Key:       key,
		ValueEnc:  encrypted,
		Masked:    1,
		CreatedBy: &createdBy,
	}

	if err := s.repos.Secrets.Create(ctx, secret); err != nil {
		return fmt.Errorf("storing secret: %w", err)
	}
	return nil
}

// Get retrieves a secret by ID and decrypts its value.
func (s *SecretStore) Get(ctx context.Context, id string) (*DecryptedSecret, error) {
	secret, err := s.repos.Secrets.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching secret: %w", err)
	}

	plaintext, err := crypto.Decrypt(s.encryptionKey, secret.ValueEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypting secret: %w", err)
	}

	return &DecryptedSecret{
		ID:        secret.ID,
		ProjectID: secret.ProjectID,
		OrgID:     secret.OrgID,
		Scope:     secret.Scope,
		Key:       secret.Key,
		Value:     plaintext,
		Masked:    secret.Masked,
		CreatedBy: secret.CreatedBy,
		CreatedAt: secret.CreatedAt,
		UpdatedAt: secret.UpdatedAt,
	}, nil
}

// List returns secret metadata (without decrypted values) for a project.
func (s *SecretStore) List(ctx context.Context, projectID string, limit, offset int) ([]SecretMetadata, error) {
	secrets, err := s.repos.Secrets.ListByProject(ctx, projectID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing secrets: %w", err)
	}

	result := make([]SecretMetadata, len(secrets))
	for i, sec := range secrets {
		// Check if the secret value is empty (placeholder from import).
		isEmpty := false
		if plaintext, err := crypto.Decrypt(s.encryptionKey, sec.ValueEnc); err == nil {
			isEmpty = plaintext == ""
		}
		result[i] = SecretMetadata{
			ID:        sec.ID,
			ProjectID: sec.ProjectID,
			OrgID:     sec.OrgID,
			Scope:     sec.Scope,
			Key:       sec.Key,
			Masked:    sec.Masked,
			IsEmpty:   isEmpty,
			CreatedBy: sec.CreatedBy,
			CreatedAt: sec.CreatedAt,
			UpdatedAt: sec.UpdatedAt,
		}
	}
	return result, nil
}

// Update re-encrypts a secret with a new plaintext value.
func (s *SecretStore) Update(ctx context.Context, id, newValue string) error {
	if newValue == "" {
		return errors.New("secret value must not be empty")
	}

	secret, err := s.repos.Secrets.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching secret for update: %w", err)
	}

	encrypted, err := crypto.Encrypt(s.encryptionKey, newValue)
	if err != nil {
		return fmt.Errorf("encrypting secret: %w", err)
	}

	secret.ValueEnc = encrypted
	if err := s.repos.Secrets.Update(ctx, secret); err != nil {
		return fmt.Errorf("updating secret: %w", err)
	}
	return nil
}

// Delete removes a secret by ID.
func (s *SecretStore) Delete(ctx context.Context, id string) error {
	if err := s.repos.Secrets.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting secret: %w", err)
	}
	return nil
}

// GetForInjection retrieves all secrets for a project and returns them as a
// key=value map suitable for environment variable injection. Values are
// decrypted.
func (s *SecretStore) GetForInjection(ctx context.Context, projectID string) (map[string]string, error) {
	// Fetch all secrets for the project (large limit to get them all).
	secrets, err := s.repos.Secrets.ListByProject(ctx, projectID, 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("listing secrets for injection: %w", err)
	}

	result := make(map[string]string, len(secrets))
	for _, sec := range secrets {
		plaintext, err := crypto.Decrypt(s.encryptionKey, sec.ValueEnc)
		if err != nil {
			return nil, fmt.Errorf("decrypting secret %q: %w", sec.Key, err)
		}
		result[sec.Key] = plaintext
	}
	return result, nil
}
