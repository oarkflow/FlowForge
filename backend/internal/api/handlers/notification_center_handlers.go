package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/oarkflow/deploy/backend/internal/models"
)

// ListInAppNotifications returns the current user's in-app notifications (paginated).
func (h *Handler) ListInAppNotifications(c fiber.Ctx) error {
	userID := getUserID(c)
	limit, offset := h.pagination(c)

	notifs, err := h.repo.InAppNotifications.ListByUser(userID, limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list notifications")
	}
	return c.JSON(notifs)
}

// CountUnreadNotifications returns the count of unread notifications for the current user.
func (h *Handler) CountUnreadNotifications(c fiber.Ctx) error {
	userID := getUserID(c)

	count, err := h.repo.InAppNotifications.CountUnread(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to count unread notifications")
	}
	return c.JSON(fiber.Map{"unread_count": count})
}

// MarkNotificationRead marks a single notification as read.
func (h *Handler) MarkNotificationRead(c fiber.Ctx) error {
	userID := getUserID(c)
	nid := c.Params("nid")

	if err := h.repo.InAppNotifications.MarkRead(nid, userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to mark notification as read")
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

// MarkAllNotificationsRead marks all notifications as read for the current user.
func (h *Handler) MarkAllNotificationsRead(c fiber.Ctx) error {
	userID := getUserID(c)

	if err := h.repo.InAppNotifications.MarkAllRead(userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to mark all notifications as read")
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

// DeleteNotification deletes a single notification.
func (h *Handler) DeleteNotification(c fiber.Ctx) error {
	userID := getUserID(c)
	nid := c.Params("nid")

	if err := h.repo.InAppNotifications.Delete(nid, userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete notification")
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

// GetNotificationPreferences returns the current user's notification preferences.
func (h *Handler) GetNotificationPreferences(c fiber.Ctx) error {
	userID := getUserID(c)

	pref, err := h.repo.NotificationPrefs.GetByUser(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get notification preferences")
	}
	return c.JSON(pref)
}

// UpdateNotificationPreferences updates the current user's notification preferences.
func (h *Handler) UpdateNotificationPreferences(c fiber.Ctx) error {
	userID := getUserID(c)

	var input models.NotificationPreference
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	input.UserID = userID

	if err := h.repo.NotificationPrefs.Upsert(&input); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update notification preferences")
	}

	pref, err := h.repo.NotificationPrefs.GetByUser(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get updated preferences")
	}
	return c.JSON(pref)
}

// GetDashboardPreferences returns the current user's dashboard layout preferences.
func (h *Handler) GetDashboardPreferences(c fiber.Ctx) error {
	userID := getUserID(c)

	pref, err := h.repo.DashboardPrefs.GetByUser(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get dashboard preferences")
	}
	return c.JSON(pref)
}

// UpdateDashboardPreferences updates the current user's dashboard layout preferences.
func (h *Handler) UpdateDashboardPreferences(c fiber.Ctx) error {
	userID := getUserID(c)

	var input models.DashboardPreference
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	input.UserID = userID

	if err := h.repo.DashboardPrefs.Upsert(&input); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update dashboard preferences")
	}

	pref, err := h.repo.DashboardPrefs.GetByUser(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get updated preferences")
	}
	return c.JSON(pref)
}
