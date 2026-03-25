package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// =========================================================================
// SCALING POLICY HANDLERS
// =========================================================================

// ListScalingPolicies returns all scaling policies.
// GET /api/v1/scaling/policies
func (h *Handler) ListScalingPolicies(c fiber.Ctx) error {
	policies, err := h.repo.ScalingPolicies.ListPolicies()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list scaling policies")
	}
	return c.JSON(policies)
}

// GetScalingPolicy returns a specific scaling policy.
// GET /api/v1/scaling/policies/:pid
func (h *Handler) GetScalingPolicy(c fiber.Ctx) error {
	policyID := c.Params("pid")
	policy, err := h.repo.ScalingPolicies.GetPolicy(policyID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "scaling policy not found")
	}
	return c.JSON(policy)
}

type createScalingPolicyInput struct {
	Name               string `json:"name"`
	Description        string `json:"description"`
	ExecutorType       string `json:"executor_type"`
	Labels             string `json:"labels"`
	MinAgents          int    `json:"min_agents"`
	MaxAgents          int    `json:"max_agents"`
	ScaleUpThreshold   int    `json:"scale_up_threshold"`
	ScaleDownThreshold int    `json:"scale_down_threshold"`
	ScaleUpStep        int    `json:"scale_up_step"`
	ScaleDownStep      int    `json:"scale_down_step"`
	CooldownSeconds    int    `json:"cooldown_seconds"`
}

// CreateScalingPolicy creates a new scaling policy.
// POST /api/v1/scaling/policies
func (h *Handler) CreateScalingPolicy(c fiber.Ctx) error {
	var input createScalingPolicyInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	executorType := input.ExecutorType
	if executorType == "" {
		executorType = "docker"
	}
	if executorType != "local" && executorType != "docker" && executorType != "kubernetes" {
		return fiber.NewError(fiber.StatusBadRequest, "executor_type must be local, docker, or kubernetes")
	}

	minAgents := input.MinAgents
	if minAgents < 0 {
		minAgents = 0
	}
	maxAgents := input.MaxAgents
	if maxAgents <= 0 {
		maxAgents = 10
	}
	if minAgents > maxAgents {
		return fiber.NewError(fiber.StatusBadRequest, "min_agents cannot exceed max_agents")
	}

	scaleUpThreshold := input.ScaleUpThreshold
	if scaleUpThreshold <= 0 {
		scaleUpThreshold = 5
	}
	scaleDownThreshold := input.ScaleDownThreshold
	if scaleDownThreshold < 0 {
		scaleDownThreshold = 0
	}

	scaleUpStep := input.ScaleUpStep
	if scaleUpStep <= 0 {
		scaleUpStep = 1
	}
	scaleDownStep := input.ScaleDownStep
	if scaleDownStep <= 0 {
		scaleDownStep = 1
	}

	cooldownSeconds := input.CooldownSeconds
	if cooldownSeconds <= 0 {
		cooldownSeconds = 300
	}

	policy := &models.ScalingPolicy{
		Name:               input.Name,
		Description:        input.Description,
		Enabled:            true,
		ExecutorType:       executorType,
		Labels:             input.Labels,
		MinAgents:          minAgents,
		MaxAgents:          maxAgents,
		DesiredAgents:      minAgents,
		ScaleUpThreshold:   scaleUpThreshold,
		ScaleDownThreshold: scaleDownThreshold,
		ScaleUpStep:        scaleUpStep,
		ScaleDownStep:      scaleDownStep,
		CooldownSeconds:    cooldownSeconds,
	}

	if err := h.repo.ScalingPolicies.CreatePolicy(policy); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create scaling policy: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "create", "scaling_policy", policy.ID,
		fiber.Map{"name": input.Name, "executor_type": executorType})

	return c.Status(fiber.StatusCreated).JSON(policy)
}

type updateScalingPolicyInput struct {
	Name               *string `json:"name"`
	Description        *string `json:"description"`
	ExecutorType       *string `json:"executor_type"`
	Labels             *string `json:"labels"`
	MinAgents          *int    `json:"min_agents"`
	MaxAgents          *int    `json:"max_agents"`
	ScaleUpThreshold   *int    `json:"scale_up_threshold"`
	ScaleDownThreshold *int    `json:"scale_down_threshold"`
	ScaleUpStep        *int    `json:"scale_up_step"`
	ScaleDownStep      *int    `json:"scale_down_step"`
	CooldownSeconds    *int    `json:"cooldown_seconds"`
}

// UpdateScalingPolicy updates an existing scaling policy.
// PUT /api/v1/scaling/policies/:pid
func (h *Handler) UpdateScalingPolicy(c fiber.Ctx) error {
	policyID := c.Params("pid")

	existing, err := h.repo.ScalingPolicies.GetPolicy(policyID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "scaling policy not found")
	}

	var input updateScalingPolicyInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if input.Name != nil && *input.Name != "" {
		existing.Name = *input.Name
	}
	if input.Description != nil {
		existing.Description = *input.Description
	}
	if input.ExecutorType != nil && *input.ExecutorType != "" {
		et := *input.ExecutorType
		if et != "local" && et != "docker" && et != "kubernetes" {
			return fiber.NewError(fiber.StatusBadRequest, "executor_type must be local, docker, or kubernetes")
		}
		existing.ExecutorType = et
	}
	if input.Labels != nil {
		existing.Labels = *input.Labels
	}
	if input.MinAgents != nil {
		existing.MinAgents = *input.MinAgents
	}
	if input.MaxAgents != nil {
		existing.MaxAgents = *input.MaxAgents
	}
	if existing.MinAgents > existing.MaxAgents {
		return fiber.NewError(fiber.StatusBadRequest, "min_agents cannot exceed max_agents")
	}
	if input.ScaleUpThreshold != nil {
		existing.ScaleUpThreshold = *input.ScaleUpThreshold
	}
	if input.ScaleDownThreshold != nil {
		existing.ScaleDownThreshold = *input.ScaleDownThreshold
	}
	if input.ScaleUpStep != nil {
		existing.ScaleUpStep = *input.ScaleUpStep
	}
	if input.ScaleDownStep != nil {
		existing.ScaleDownStep = *input.ScaleDownStep
	}
	if input.CooldownSeconds != nil {
		existing.CooldownSeconds = *input.CooldownSeconds
	}

	if err := h.repo.ScalingPolicies.UpdatePolicy(existing); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update scaling policy")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "scaling_policy", policyID, input)

	return c.JSON(existing)
}

// DeleteScalingPolicy deletes a scaling policy and its events.
// DELETE /api/v1/scaling/policies/:pid
func (h *Handler) DeleteScalingPolicy(c fiber.Ctx) error {
	policyID := c.Params("pid")
	if _, err := h.repo.ScalingPolicies.GetPolicy(policyID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "scaling policy not found")
	}

	if err := h.repo.ScalingPolicies.DeletePolicy(policyID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete scaling policy")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete", "scaling_policy", policyID, nil)

	return c.JSON(fiber.Map{"message": "scaling policy deleted"})
}

// EnableScalingPolicy enables a scaling policy.
// POST /api/v1/scaling/policies/:pid/enable
func (h *Handler) EnableScalingPolicy(c fiber.Ctx) error {
	policyID := c.Params("pid")
	if _, err := h.repo.ScalingPolicies.GetPolicy(policyID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "scaling policy not found")
	}

	if err := h.repo.ScalingPolicies.SetEnabled(policyID, true); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to enable scaling policy")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "enable", "scaling_policy", policyID, nil)

	policy, err := h.repo.ScalingPolicies.GetPolicy(policyID)
	if err != nil {
		return c.JSON(fiber.Map{"message": "scaling policy enabled"})
	}
	return c.JSON(policy)
}

// DisableScalingPolicy disables a scaling policy.
// POST /api/v1/scaling/policies/:pid/disable
func (h *Handler) DisableScalingPolicy(c fiber.Ctx) error {
	policyID := c.Params("pid")
	if _, err := h.repo.ScalingPolicies.GetPolicy(policyID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "scaling policy not found")
	}

	if err := h.repo.ScalingPolicies.SetEnabled(policyID, false); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to disable scaling policy")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "disable", "scaling_policy", policyID, nil)

	policy, err := h.repo.ScalingPolicies.GetPolicy(policyID)
	if err != nil {
		return c.JSON(fiber.Map{"message": "scaling policy disabled"})
	}
	return c.JSON(policy)
}

// =========================================================================
// SCALING EVENT HANDLERS
// =========================================================================

// ListScalingEvents returns events for a specific policy.
// GET /api/v1/scaling/policies/:pid/events
func (h *Handler) ListScalingEvents(c fiber.Ctx) error {
	policyID := c.Params("pid")

	// Verify policy exists
	if _, err := h.repo.ScalingPolicies.GetPolicy(policyID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "scaling policy not found")
	}

	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	events, err := h.repo.ScalingEvents.ListByPolicy(policyID, limit)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list scaling events")
	}
	return c.JSON(events)
}

// ListRecentScalingEvents returns recent scaling events across all policies.
// GET /api/v1/scaling/events
func (h *Handler) ListRecentScalingEvents(c fiber.Ctx) error {
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	events, err := h.repo.ScalingEvents.ListRecent(limit)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list recent scaling events")
	}
	return c.JSON(events)
}

// =========================================================================
// SCALING METRICS HANDLER
// =========================================================================

// GetScalingMetrics returns current scaling metrics.
// GET /api/v1/scaling/metrics
func (h *Handler) GetScalingMetrics(c fiber.Ctx) error {
	// Get agent counts from the database
	agents, err := h.repo.Agents.List(c.Context(), 10000, 0)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch agent metrics")
	}

	totalAgents := len(agents)
	onlineAgents := 0
	busyAgents := 0
	agentsByExecutor := make(map[string]int)
	agentsByLabel := make(map[string]int)

	for _, a := range agents {
		switch a.Status {
		case "online":
			onlineAgents++
		case "busy":
			busyAgents++
		}
		if a.Status == "online" || a.Status == "busy" {
			agentsByExecutor[a.Executor]++
			// Parse labels (stored as JSON array string or comma-separated)
			labels := parseLabelsString(a.Labels)
			for _, l := range labels {
				agentsByLabel[l]++
			}
		}
	}

	// Get active scaling policies count
	policies, _ := h.repo.ScalingPolicies.ListEnabledPolicies()

	// Get queue depth from dispatcher - we'll approximate from policies' stored metrics
	// Since handler doesn't have direct access to the dispatcher, use the latest policy metrics
	queueDepth := 0
	allPolicies, _ := h.repo.ScalingPolicies.ListPolicies()
	for _, p := range allPolicies {
		if p.QueueDepth > queueDepth {
			queueDepth = p.QueueDepth
		}
	}

	return c.JSON(fiber.Map{
		"total_agents":      totalAgents,
		"online_agents":     onlineAgents,
		"busy_agents":       busyAgents,
		"queue_depth":       queueDepth,
		"agents_by_executor": agentsByExecutor,
		"agents_by_label":   agentsByLabel,
		"active_policies":   len(policies),
	})
}

// parseLabelsString parses a label string that could be JSON array or comma-separated.
func parseLabelsString(labels string) []string {
	if labels == "" || labels == "{}" || labels == "[]" {
		return nil
	}
	// Try as comma-separated (simple approach)
	var result []string
	// Remove JSON array brackets if present
	s := labels
	if len(s) >= 2 && s[0] == '[' && s[len(s)-1] == ']' {
		s = s[1 : len(s)-1]
	}
	start := 0
	inQuote := false
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			inQuote = !inQuote
		} else if s[i] == ',' && !inQuote {
			token := trimQuotesAndSpaces(s[start:i])
			if token != "" {
				result = append(result, token)
			}
			start = i + 1
		}
	}
	// Last token
	token := trimQuotesAndSpaces(s[start:])
	if token != "" {
		result = append(result, token)
	}
	return result
}

// trimQuotesAndSpaces removes surrounding quotes and whitespace from a string.
func trimQuotesAndSpaces(s string) string {
	// Trim whitespace
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	s = s[start:end]
	// Trim quotes
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	return s
}
