package queries

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type UserRepo struct {
	db *sqlx.DB
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	user := &models.User{}
	err := r.db.GetContext(ctx, user, "SELECT * FROM users WHERE id = ? AND deleted_at IS NULL", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return user, err
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	err := r.db.GetContext(ctx, user, "SELECT * FROM users WHERE email = ? AND deleted_at IS NULL", email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return user, err
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	user := &models.User{}
	err := r.db.GetContext(ctx, user, "SELECT * FROM users WHERE username = ? AND deleted_at IS NULL", username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return user, err
}

func (r *UserRepo) List(ctx context.Context, limit, offset int) ([]models.User, error) {
	users := []models.User{}
	err := r.db.SelectContext(ctx, &users, "SELECT * FROM users WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT ? OFFSET ?", limit, offset)
	return users, err
}

func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO users (id, email, username, password_hash, display_name, avatar_url, role, totp_secret, totp_enabled, is_active, created_at, updated_at)
		VALUES (:id, :email, :username, :password_hash, :display_name, :avatar_url, :role, :totp_secret, :totp_enabled, :is_active, :created_at, :updated_at)`,
		user)
	return err
}

func (r *UserRepo) Update(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE users SET email=:email, username=:username, display_name=:display_name, avatar_url=:avatar_url, role=:role, updated_at=:updated_at WHERE id=:id`,
		user)
	return err
}

func (r *UserRepo) SoftDelete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET deleted_at = ? WHERE id = ?", time.Now(), id)
	return err
}
