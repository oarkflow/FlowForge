package tenant

import (
	"github.com/gofiber/fiber/v3"
)

// TenantMiddleware injects the user's organization ID into the request context
// for tenant-scoped queries. When multi-tenant mode is enabled, all database
// queries should filter by org_id.
func TenantMiddleware(enabled bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !enabled {
			return c.Next()
		}

		// The org_id can come from:
		// 1. JWT claims (if the user belongs to an org)
		// 2. Query parameter (admin override)
		// 3. X-Tenant-ID header
		orgID := c.Get("X-Tenant-ID")
		if orgID == "" {
			orgID = c.Query("org_id")
		}
		if orgID == "" {
			orgID, _ = c.Locals("org_id").(string)
		}

		c.Locals("tenant_id", orgID)
		return c.Next()
	}
}

// GetTenantID retrieves the current tenant ID from the context.
func GetTenantID(c fiber.Ctx) string {
	id, _ := c.Locals("tenant_id").(string)
	return id
}

// ResourceLimits defines per-tenant resource limits.
type ResourceLimits struct {
	MaxProjects       int `json:"max_projects"`
	MaxPipelines      int `json:"max_pipelines"`
	MaxConcurrentRuns int `json:"max_concurrent_runs"`
	MaxAgents         int `json:"max_agents"`
	MaxStorageMB      int `json:"max_storage_mb"`
}

// DefaultLimits returns the default resource limits for a tenant.
func DefaultLimits() ResourceLimits {
	return ResourceLimits{
		MaxProjects:       50,
		MaxPipelines:      200,
		MaxConcurrentRuns: 10,
		MaxAgents:         20,
		MaxStorageMB:      10240, // 10 GB
	}
}

// UsageTracker tracks resource usage per tenant for billing purposes.
type UsageTracker struct {
	// In a production system, this would use a proper time-series database
	// or billing service. For now, we track basic counters.
}

// NewUsageTracker creates a new usage tracker.
func NewUsageTracker() *UsageTracker {
	return &UsageTracker{}
}

// TrackBuildMinutes records build time usage for the given tenant.
func (t *UsageTracker) TrackBuildMinutes(orgID string, minutes float64) {
	// In production: write to time-series DB or billing API
}

// TrackStorageUsage records storage usage for the given tenant.
func (t *UsageTracker) TrackStorageUsage(orgID string, bytesUsed int64) {
	// In production: write to billing API
}
