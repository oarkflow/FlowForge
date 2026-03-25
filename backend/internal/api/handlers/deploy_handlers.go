package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/oarkflow/deploy/backend/internal/deploy"
)

// =========================================================================
// DEPLOYMENT STRATEGY HANDLERS
// =========================================================================

type updateStrategyInput struct {
	Strategy                  string `json:"strategy"`
	StrategyConfig            string `json:"strategy_config"`
	HealthCheckURL            string `json:"health_check_url"`
	HealthCheckInterval       int    `json:"health_check_interval"`
	HealthCheckTimeout        int    `json:"health_check_timeout"`
	HealthCheckRetries        int    `json:"health_check_retries"`
	HealthCheckPath           string `json:"health_check_path"`
	HealthCheckExpectedStatus int    `json:"health_check_expected_status"`
}

// GetStrategyConfig returns the strategy configuration for an environment.
func (h *Handler) GetStrategyConfig(c fiber.Ctx) error {
	envID := c.Params("eid")

	env, err := h.repo.Environments.GetByID(c.Context(), envID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "environment not found")
	}

	return c.JSON(fiber.Map{
		"strategy":                    env.Strategy,
		"strategy_config":             env.StrategyConfig,
		"health_check_url":            env.HealthCheckURL,
		"health_check_interval":       env.HealthCheckInterval,
		"health_check_timeout":        env.HealthCheckTimeout,
		"health_check_retries":        env.HealthCheckRetries,
		"health_check_path":           env.HealthCheckPath,
		"health_check_expected_status": env.HealthCheckExpectedStatus,
	})
}

// UpdateStrategyConfig updates the strategy configuration for an environment.
func (h *Handler) UpdateStrategyConfig(c fiber.Ctx) error {
	envID := c.Params("eid")

	var input updateStrategyInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	// Validate strategy type
	validStrategies := map[string]bool{
		"recreate":   true,
		"rolling":    true,
		"blue_green": true,
		"canary":     true,
	}
	if input.Strategy != "" && !validStrategies[input.Strategy] {
		return fiber.NewError(fiber.StatusBadRequest, "invalid strategy — must be one of: recreate, rolling, blue_green, canary")
	}

	env, err := h.repo.Environments.GetByID(c.Context(), envID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "environment not found")
	}

	// Apply defaults for empty values
	strategy := input.Strategy
	if strategy == "" {
		strategy = env.Strategy
	}
	if strategy == "" {
		strategy = "recreate"
	}
	strategyConfig := input.StrategyConfig
	if strategyConfig == "" {
		strategyConfig = env.StrategyConfig
	}
	if strategyConfig == "" {
		strategyConfig = "{}"
	}
	healthCheckURL := input.HealthCheckURL
	if healthCheckURL == "" {
		healthCheckURL = env.HealthCheckURL
	}
	healthCheckInterval := input.HealthCheckInterval
	if healthCheckInterval == 0 {
		healthCheckInterval = env.HealthCheckInterval
	}
	if healthCheckInterval == 0 {
		healthCheckInterval = 30
	}
	healthCheckTimeout := input.HealthCheckTimeout
	if healthCheckTimeout == 0 {
		healthCheckTimeout = env.HealthCheckTimeout
	}
	if healthCheckTimeout == 0 {
		healthCheckTimeout = 10
	}
	healthCheckRetries := input.HealthCheckRetries
	if healthCheckRetries == 0 {
		healthCheckRetries = env.HealthCheckRetries
	}
	if healthCheckRetries == 0 {
		healthCheckRetries = 3
	}
	healthCheckPath := input.HealthCheckPath
	if healthCheckPath == "" {
		healthCheckPath = env.HealthCheckPath
	}
	if healthCheckPath == "" {
		healthCheckPath = "/health"
	}
	healthCheckExpectedStatus := input.HealthCheckExpectedStatus
	if healthCheckExpectedStatus == 0 {
		healthCheckExpectedStatus = env.HealthCheckExpectedStatus
	}
	if healthCheckExpectedStatus == 0 {
		healthCheckExpectedStatus = 200
	}

	if err := h.repo.Environments.UpdateStrategy(c.Context(), envID,
		strategy, strategyConfig,
		healthCheckURL, healthCheckInterval, healthCheckTimeout,
		healthCheckRetries, healthCheckPath, healthCheckExpectedStatus); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update strategy configuration")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update_strategy", "environment", envID,
		fiber.Map{"strategy": strategy})

	// Return the updated environment
	env, _ = h.repo.Environments.GetByID(c.Context(), envID)
	return c.JSON(env)
}

type advanceCanaryInput struct {
	Weight int `json:"weight"`
}

// AdvanceCanary advances a canary deployment to the specified weight.
func (h *Handler) AdvanceCanary(c fiber.Ctx) error {
	envID := c.Params("eid")
	depID := c.Params("did")

	var input advanceCanaryInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if input.Weight < 0 || input.Weight > 100 {
		return fiber.NewError(fiber.StatusBadRequest, "weight must be between 0 and 100")
	}

	// Verify environment exists
	if _, err := h.repo.Environments.GetByID(c.Context(), envID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "environment not found")
	}

	// Verify deployment exists and belongs to this environment
	dep, err := h.repo.Deployments.GetByID(c.Context(), depID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "deployment not found")
	}
	if dep.EnvironmentID != envID {
		return fiber.NewError(fiber.StatusBadRequest, "deployment does not belong to this environment")
	}

	manager := deploy.NewManager(h.repo)
	if err := manager.AdvanceCanary(c.Context(), depID, input.Weight); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "advance_canary", "deployment", depID,
		fiber.Map{"weight": input.Weight, "environment_id": envID})

	// Return updated deployment
	dep, _ = h.repo.Deployments.GetByID(c.Context(), depID)
	return c.JSON(dep)
}

// CheckDeploymentHealth performs a health check for a deployment.
func (h *Handler) CheckDeploymentHealth(c fiber.Ctx) error {
	envID := c.Params("eid")
	depID := c.Params("did")

	env, err := h.repo.Environments.GetByID(c.Context(), envID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "environment not found")
	}

	// Verify deployment exists and belongs to this environment
	dep, err := h.repo.Deployments.GetByID(c.Context(), depID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "deployment not found")
	}
	if dep.EnvironmentID != envID {
		return fiber.NewError(fiber.StatusBadRequest, "deployment does not belong to this environment")
	}

	manager := deploy.NewManager(h.repo)
	result, err := manager.CheckHealth(c.Context(), env, depID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "health check failed: "+err.Error())
	}

	return c.JSON(result)
}

// GetDeploymentPlan returns the deployment plan/strategy state for a deployment.
func (h *Handler) GetDeploymentPlan(c fiber.Ctx) error {
	envID := c.Params("eid")
	depID := c.Params("did")

	// Verify environment exists
	if _, err := h.repo.Environments.GetByID(c.Context(), envID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "environment not found")
	}

	dep, err := h.repo.Deployments.GetByID(c.Context(), depID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "deployment not found")
	}
	if dep.EnvironmentID != envID {
		return fiber.NewError(fiber.StatusBadRequest, "deployment does not belong to this environment")
	}

	return c.JSON(fiber.Map{
		"deployment_id":        dep.ID,
		"strategy":             dep.Strategy,
		"canary_weight":        dep.CanaryWeight,
		"health_check_results": dep.HealthCheckResults,
		"strategy_state":       dep.StrategyState,
		"status":               dep.Status,
	})
}

// helper to safely parse int from string with a default
func parseIntOrDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
