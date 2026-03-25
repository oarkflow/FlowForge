package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"

	"github.com/oarkflow/deploy/backend/internal/approval"
	"github.com/oarkflow/deploy/backend/internal/models"
)

// =========================================================================
// APPROVAL HANDLERS
// =========================================================================

// ListPendingApprovals returns pending approvals where the current user is an approver.
func (h *Handler) ListPendingApprovals(c fiber.Ctx) error {
	userID := getUserID(c)
	approvals, err := h.repo.Approvals.ListPending(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list pending approvals")
	}
	return c.JSON(approvals)
}

// ListProjectApprovals returns all approvals for a project.
func (h *Handler) ListProjectApprovals(c fiber.Ctx) error {
	projectID := c.Params("id")

	// Verify project exists
	if _, err := h.repo.Projects.GetByID(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	approvals, err := h.repo.Approvals.ListByProject(c.Context(), projectID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list project approvals")
	}
	return c.JSON(approvals)
}

// GetApproval returns approval details with its responses.
func (h *Handler) GetApproval(c fiber.Ctx) error {
	approvalID := c.Params("aid")

	appr, err := h.repo.Approvals.GetByID(c.Context(), approvalID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "approval not found")
	}

	responses, err := h.repo.ApprovalResponses.ListByApproval(c.Context(), approvalID)
	if err != nil {
		responses = []models.ApprovalResponse{}
	}

	type approvalDetail struct {
		*models.Approval
		Responses []models.ApprovalResponse `json:"responses"`
	}

	return c.JSON(approvalDetail{
		Approval:  appr,
		Responses: responses,
	})
}

type approveInput struct {
	Comment string `json:"comment"`
}

// ApproveApproval records an approval response.
func (h *Handler) ApproveApproval(c fiber.Ctx) error {
	approvalID := c.Params("aid")
	userID := getUserID(c)

	var input approveInput
	_ = c.Bind().JSON(&input)

	// Get user display name
	approverName := userID
	if user, err := h.repo.Users.GetByID(c.Context(), userID); err == nil {
		if user.DisplayName != nil && *user.DisplayName != "" {
			approverName = *user.DisplayName
		} else {
			approverName = user.Username
		}
	}

	svc := approval.NewService(h.repo)
	updated, err := svc.Respond(c.Context(), approvalID, userID, approverName, "approve", input.Comment)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "approve", "approval", approvalID,
		fiber.Map{"comment": input.Comment})

	return c.JSON(updated)
}

type rejectInput struct {
	Comment string `json:"comment"`
}

// RejectApproval records a rejection response.
func (h *Handler) RejectApproval(c fiber.Ctx) error {
	approvalID := c.Params("aid")
	userID := getUserID(c)

	var input rejectInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Comment == "" {
		return fiber.NewError(fiber.StatusBadRequest, "comment is required when rejecting")
	}

	// Get user display name
	approverName := userID
	if user, err := h.repo.Users.GetByID(c.Context(), userID); err == nil {
		if user.DisplayName != nil && *user.DisplayName != "" {
			approverName = *user.DisplayName
		} else {
			approverName = user.Username
		}
	}

	svc := approval.NewService(h.repo)
	updated, err := svc.Respond(c.Context(), approvalID, userID, approverName, "reject", input.Comment)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "reject", "approval", approvalID,
		fiber.Map{"comment": input.Comment})

	return c.JSON(updated)
}

// CancelApproval cancels a pending approval (only by the requestor).
func (h *Handler) CancelApproval(c fiber.Ctx) error {
	approvalID := c.Params("aid")
	userID := getUserID(c)

	appr, err := h.repo.Approvals.GetByID(c.Context(), approvalID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "approval not found")
	}

	if appr.Status != "pending" {
		return fiber.NewError(fiber.StatusBadRequest, "only pending approvals can be cancelled")
	}

	if appr.RequestedBy != userID {
		return fiber.NewError(fiber.StatusForbidden, "only the requestor can cancel an approval")
	}

	if err := h.repo.Approvals.Cancel(c.Context(), approvalID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to cancel approval")
	}

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "cancel", "approval", approvalID, nil)

	return c.JSON(fiber.Map{"message": "approval cancelled"})
}

// GetApprovalResponses returns all responses for an approval.
func (h *Handler) GetApprovalResponses(c fiber.Ctx) error {
	approvalID := c.Params("aid")

	// Verify approval exists
	if _, err := h.repo.Approvals.GetByID(c.Context(), approvalID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "approval not found")
	}

	responses, err := h.repo.ApprovalResponses.ListByApproval(c.Context(), approvalID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list approval responses")
	}
	return c.JSON(responses)
}

// UpdateProtectionRules updates an environment's protection rules (approval settings).
func (h *Handler) UpdateProtectionRules(c fiber.Ctx) error {
	envID := c.Params("eid")

	var input struct {
		RequireApproval   bool     `json:"require_approval"`
		MinApprovals      int      `json:"min_approvals"`
		RequiredApprovers []string `json:"required_approvers"`
	}
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	env, err := h.repo.Environments.GetByID(c.Context(), envID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "environment not found")
	}

	// Update protection rules
	rules := approval.ProtectionRules{
		RequireApproval: input.RequireApproval,
		MinApprovals:    input.MinApprovals,
	}
	if rules.MinApprovals < 1 {
		rules.MinApprovals = 1
	}

	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to serialize protection rules")
	}
	env.ProtectionRules = string(rulesJSON)

	// Update required approvers
	approversJSON, err := json.Marshal(input.RequiredApprovers)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to serialize approvers")
	}
	env.RequiredApprovers = string(approversJSON)

	if err := h.repo.Environments.Update(c.Context(), env); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update environment protection rules")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update_protection", "environment", envID, input)

	return c.JSON(env)
}
