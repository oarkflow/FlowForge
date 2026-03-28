package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type OrgRepo struct {
	db *sqlx.DB
}

func (r *OrgRepo) GetByID(ctx context.Context, id string) (*models.Organization, error) {
	org := &models.Organization{}
	err := r.db.GetContext(ctx, org, "SELECT * FROM organizations WHERE id = ?", id)
	return org, err
}

func (r *OrgRepo) List(ctx context.Context, limit, offset int) ([]models.Organization, error) {
	orgs := []models.Organization{}
	err := r.db.SelectContext(ctx, &orgs, "SELECT * FROM organizations ORDER BY created_at DESC LIMIT ? OFFSET ?", limit, offset)
	return orgs, err
}

func (r *OrgRepo) Create(ctx context.Context, org *models.Organization) error {
	org.ID = uuid.New().String()
	org.CreatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO organizations (id, name, slug, logo_url, created_at) VALUES (:id, :name, :slug, :logo_url, :created_at)`,
		org)
	return err
}

func (r *OrgRepo) Update(ctx context.Context, org *models.Organization) error {
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE organizations SET name=:name, slug=:slug, logo_url=:logo_url WHERE id=:id`,
		org)
	return err
}

func (r *OrgRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM organizations WHERE id = ?", id)
	return err
}

func (r *OrgRepo) AddMember(ctx context.Context, member *models.OrgMember) error {
	member.JoinedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO org_members (org_id, user_id, role, joined_at) VALUES (:org_id, :user_id, :role, :joined_at)`,
		member)
	return err
}

func (r *OrgRepo) RemoveMember(ctx context.Context, orgID, userID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM org_members WHERE org_id = ? AND user_id = ?", orgID, userID)
	return err
}

func (r *OrgRepo) ListMembers(ctx context.Context, orgID string) ([]models.OrgMember, error) {
	members := []models.OrgMember{}
	err := r.db.SelectContext(ctx, &members, "SELECT * FROM org_members WHERE org_id = ?", orgID)
	return members, err
}
