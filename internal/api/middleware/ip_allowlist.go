package middleware

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// IPAllowlistEntry represents a row in the ip_allowlist table.
type IPAllowlistEntry struct {
	ID        string  `db:"id" json:"id"`
	ProjectID *string `db:"project_id" json:"project_id,omitempty"`
	Scope     string  `db:"scope" json:"scope"` // "global" or "project"
	CIDR      string  `db:"cidr" json:"cidr"`
	Label     string  `db:"label" json:"label"`
}

// IPAllowlist is a middleware that restricts access to routes based on the
// client's IP address. It supports both global and per-project allowlists
// using CIDR notation (e.g. "10.0.0.0/8", "192.168.1.100/32").
//
// When no entries exist the middleware is permissive (allow all).
type IPAllowlist struct {
	db *sqlx.DB

	mu      sync.RWMutex
	global  []*net.IPNet
	project map[string][]*net.IPNet // projectID -> CIDRs
}

// NewIPAllowlist creates a new IPAllowlist middleware backed by the given
// database. Call Reload() once after creation to populate the in-memory cache.
func NewIPAllowlist(db *sqlx.DB) *IPAllowlist {
	return &IPAllowlist{
		db:      db,
		project: make(map[string][]*net.IPNet),
	}
}

// Reload reads allowlist entries from the database and rebuilds the in-memory
// CIDR cache. Call this on startup and whenever entries change.
func (a *IPAllowlist) Reload(ctx context.Context) error {
	var entries []IPAllowlistEntry
	err := a.db.SelectContext(ctx, &entries, `SELECT id, project_id, scope, cidr, label FROM ip_allowlist`)
	if err != nil {
		return fmt.Errorf("ip allowlist reload: %w", err)
	}

	global := make([]*net.IPNet, 0)
	project := make(map[string][]*net.IPNet)

	for _, e := range entries {
		_, cidr, err := net.ParseCIDR(normalizeCIDR(e.CIDR))
		if err != nil {
			log.Warn().Str("cidr", e.CIDR).Err(err).Msg("ip allowlist: skipping invalid CIDR")
			continue
		}
		if e.Scope == "global" {
			global = append(global, cidr)
		} else if e.ProjectID != nil {
			project[*e.ProjectID] = append(project[*e.ProjectID], cidr)
		}
	}

	a.mu.Lock()
	a.global = global
	a.project = project
	a.mu.Unlock()

	log.Info().Int("global", len(global)).Int("projects", len(project)).Msg("ip allowlist reloaded")
	return nil
}

// CheckGlobal returns a Fiber handler that enforces the global allowlist.
// If no global entries exist, all requests pass through.
func (a *IPAllowlist) CheckGlobal() fiber.Handler {
	return func(c fiber.Ctx) error {
		a.mu.RLock()
		cidrs := a.global
		a.mu.RUnlock()

		if len(cidrs) == 0 {
			return c.Next() // no restrictions
		}

		ip := net.ParseIP(extractIP(c.IP()))
		if ip == nil {
			return fiber.NewError(fiber.StatusForbidden, "unable to determine client IP")
		}

		for _, cidr := range cidrs {
			if cidr.Contains(ip) {
				return c.Next()
			}
		}

		log.Warn().Str("ip", c.IP()).Msg("ip allowlist: global access denied")
		return fiber.NewError(fiber.StatusForbidden, "IP address not in allowlist")
	}
}

// CheckProject returns a Fiber handler that enforces per-project allowlists.
// The project ID is extracted from the route parameter specified by paramName.
// If no project-specific entries exist, it falls through to the global list.
// If neither has entries, all requests pass.
func (a *IPAllowlist) CheckProject(paramName string) fiber.Handler {
	return func(c fiber.Ctx) error {
		projectID := c.Params(paramName)
		if projectID == "" {
			return c.Next()
		}

		a.mu.RLock()
		projectCIDRs := a.project[projectID]
		globalCIDRs := a.global
		a.mu.RUnlock()

		// No restrictions at all -> pass
		if len(projectCIDRs) == 0 && len(globalCIDRs) == 0 {
			return c.Next()
		}

		ip := net.ParseIP(extractIP(c.IP()))
		if ip == nil {
			return fiber.NewError(fiber.StatusForbidden, "unable to determine client IP")
		}

		// Check project-specific first.
		for _, cidr := range projectCIDRs {
			if cidr.Contains(ip) {
				return c.Next()
			}
		}

		// Fallback to global.
		for _, cidr := range globalCIDRs {
			if cidr.Contains(ip) {
				return c.Next()
			}
		}

		log.Warn().Str("ip", c.IP()).Str("project", projectID).Msg("ip allowlist: project access denied")
		return fiber.NewError(fiber.StatusForbidden, "IP address not in allowlist")
	}
}

// normalizeCIDR ensures a plain IP gets a /32 (IPv4) or /128 (IPv6) suffix.
func normalizeCIDR(s string) string {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, "/") {
		if strings.Contains(s, ":") {
			return s + "/128"
		}
		return s + "/32"
	}
	return s
}

// extractIP strips any port suffix from an IP string.
func extractIP(raw string) string {
	if host, _, err := net.SplitHostPort(raw); err == nil {
		return host
	}
	return raw
}
