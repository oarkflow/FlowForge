package queries

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/oarkflow/deploy/backend/internal/models"
)

type DashboardPrefRepo struct {
	db *sqlx.DB
}

func (r *DashboardPrefRepo) GetByUser(userID string) (*models.DashboardPreference, error) {
	var pref models.DashboardPreference
	err := r.db.Get(&pref, "SELECT * FROM dashboard_preferences WHERE user_id = ?", userID)
	if err == sql.ErrNoRows {
		// Return default preferences if none exist
		return &models.DashboardPreference{
			UserID: userID,
			Layout: "[]",
			Theme:  "default",
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &pref, nil
}

func (r *DashboardPrefRepo) Upsert(pref *models.DashboardPreference) error {
	_, err := r.db.Exec(`
		INSERT INTO dashboard_preferences (user_id, layout, theme)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			layout = excluded.layout,
			theme = excluded.theme,
			updated_at = CURRENT_TIMESTAMP`,
		pref.UserID, pref.Layout, pref.Theme)
	return err
}
