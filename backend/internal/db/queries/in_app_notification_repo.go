package queries

import (
	"github.com/jmoiron/sqlx"
	"github.com/oarkflow/deploy/backend/internal/models"
)

type InAppNotificationRepo struct {
	db *sqlx.DB
}

func (r *InAppNotificationRepo) ListByUser(userID string, limit, offset int) ([]models.InAppNotification, error) {
	var notifs []models.InAppNotification
	err := r.db.Select(&notifs,
		"SELECT * FROM in_app_notifications WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return notifs, nil
}

func (r *InAppNotificationRepo) CountUnread(userID string) (int, error) {
	var count int
	err := r.db.Get(&count,
		"SELECT COUNT(*) FROM in_app_notifications WHERE user_id = ? AND is_read = 0", userID)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *InAppNotificationRepo) Create(notif *models.InAppNotification) error {
	_, err := r.db.Exec(`
		INSERT INTO in_app_notifications (user_id, title, message, type, category, link)
		VALUES (?, ?, ?, ?, ?, ?)`,
		notif.UserID, notif.Title, notif.Message, notif.Type, notif.Category, notif.Link)
	if err != nil {
		return err
	}
	return r.db.Get(notif, "SELECT * FROM in_app_notifications WHERE rowid = last_insert_rowid()")
}

func (r *InAppNotificationRepo) MarkRead(id, userID string) error {
	_, err := r.db.Exec(
		"UPDATE in_app_notifications SET is_read = 1 WHERE id = ? AND user_id = ?",
		id, userID)
	return err
}

func (r *InAppNotificationRepo) MarkAllRead(userID string) error {
	_, err := r.db.Exec(
		"UPDATE in_app_notifications SET is_read = 1 WHERE user_id = ? AND is_read = 0",
		userID)
	return err
}

func (r *InAppNotificationRepo) Delete(id, userID string) error {
	_, err := r.db.Exec(
		"DELETE FROM in_app_notifications WHERE id = ? AND user_id = ?",
		id, userID)
	return err
}

func (r *InAppNotificationRepo) DeleteOlderThan(days int) error {
	_, err := r.db.Exec(
		"DELETE FROM in_app_notifications WHERE created_at < datetime('now', '-' || ? || ' days')",
		days)
	return err
}
