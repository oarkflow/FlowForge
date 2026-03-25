package deploy

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// HealthChecker performs HTTP health checks against deployment targets.
type HealthChecker struct {
	client *http.Client
}

// NewHealthChecker creates a new HealthChecker with sensible defaults.
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// CheckHTTP performs a single HTTP health check against the configured URL.
func (h *HealthChecker) CheckHTTP(ctx context.Context, env *models.Environment) (*HealthResult, error) {
	// Build the health check URL
	baseURL := strings.TrimRight(env.HealthCheckURL, "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(env.URL, "/")
	}
	if baseURL == "" {
		return &HealthResult{
			Healthy:   false,
			Error:     "no health check URL configured",
			CheckedAt: time.Now().UTC().Format(time.RFC3339),
		}, nil
	}

	path := env.HealthCheckPath
	if path == "" {
		path = "/health"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	fullURL := baseURL + path

	// Set the timeout from environment config
	if env.HealthCheckTimeout > 0 {
		h.client.Timeout = time.Duration(env.HealthCheckTimeout) * time.Second
	}

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return &HealthResult{
			Healthy:   false,
			Error:     fmt.Sprintf("failed to create request: %v", err),
			CheckedAt: time.Now().UTC().Format(time.RFC3339),
		}, nil
	}

	resp, err := h.client.Do(req)
	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		return &HealthResult{
			Healthy:   false,
			Latency:   latency,
			Error:     fmt.Sprintf("request failed: %v", err),
			CheckedAt: time.Now().UTC().Format(time.RFC3339),
		}, nil
	}
	defer resp.Body.Close()

	expectedStatus := env.HealthCheckExpectedStatus
	if expectedStatus == 0 {
		expectedStatus = 200
	}

	healthy := resp.StatusCode == expectedStatus

	return &HealthResult{
		Healthy:    healthy,
		StatusCode: resp.StatusCode,
		Latency:    latency,
		CheckedAt:  time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// CheckWithRetry performs health checks with retries according to environment configuration.
func (h *HealthChecker) CheckWithRetry(ctx context.Context, env *models.Environment) (*HealthResult, error) {
	retries := env.HealthCheckRetries
	if retries <= 0 {
		retries = 3
	}

	interval := env.HealthCheckInterval
	if interval <= 0 {
		interval = 10
	}

	var lastResult *HealthResult
	for i := 0; i < retries; i++ {
		result, err := h.CheckHTTP(ctx, env)
		if err != nil {
			return nil, err
		}
		lastResult = result

		if result.Healthy {
			return result, nil
		}

		// Wait before retrying (unless this is the last attempt)
		if i < retries-1 {
			select {
			case <-ctx.Done():
				return &HealthResult{
					Healthy:   false,
					Error:     "health check cancelled",
					CheckedAt: time.Now().UTC().Format(time.RFC3339),
				}, ctx.Err()
			case <-time.After(time.Duration(interval) * time.Second):
				// continue to next retry
			}
		}
	}

	return lastResult, nil
}
