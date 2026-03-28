package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/auth"
)

// SeedDefaultAdmin creates a default admin user if the users table is empty.
// Credentials: admin@flowforge.local / admin / admin123
func SeedDefaultAdmin(database *sqlx.DB) error {
	var count int
	if err := database.Get(&count, "SELECT COUNT(*) FROM users"); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := auth.HashPassword("admin123")
	if err != nil {
		return err
	}

	now := time.Now()
	_, err = database.ExecContext(context.Background(),
		`INSERT INTO users (id, email, username, password_hash, display_name, role, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(),
		"admin@flowforge.local",
		"admin",
		hash,
		"Admin",
		"owner",
		1,
		now,
		now,
	)
	return err
}
