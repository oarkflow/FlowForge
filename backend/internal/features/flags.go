package features

import (
	"context"
	"encoding/json"
	"math/rand"

	"github.com/gofiber/fiber/v3"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/models"
)

// FlagService checks if a feature is enabled for a given user/org.
type FlagService struct {
	repo *queries.FeatureFlagRepo
}

// NewFlagService creates a new feature flag service.
func NewFlagService(repo *queries.FeatureFlagRepo) *FlagService {
	return &FlagService{repo: repo}
}

// IsEnabled checks if a feature flag is enabled for the given user and org.
func (s *FlagService) IsEnabled(ctx context.Context, flagName, userID, orgID string) bool {
	flag, err := s.repo.GetByName(ctx, flagName)
	if err != nil {
		return false // Flag not found means disabled
	}
	if flag.Enabled == 0 {
		return false
	}

	// Check target users
	if flag.TargetUsers != "[]" && flag.TargetUsers != "" {
		var targetUsers []string
		if json.Unmarshal([]byte(flag.TargetUsers), &targetUsers) == nil && len(targetUsers) > 0 {
			for _, u := range targetUsers {
				if u == userID {
					return true
				}
			}
			// If target users are specified and user is not in the list, check orgs
		}
	}

	// Check target orgs
	if flag.TargetOrgs != "[]" && flag.TargetOrgs != "" {
		var targetOrgs []string
		if json.Unmarshal([]byte(flag.TargetOrgs), &targetOrgs) == nil && len(targetOrgs) > 0 {
			for _, o := range targetOrgs {
				if o == orgID {
					return true
				}
			}
			// If target orgs are specified and org is not in the list, check rollout
		}
	}

	// Rollout percentage
	if flag.RolloutPercentage < 100 {
		return rand.Intn(100) < flag.RolloutPercentage
	}

	return true
}

// List returns all feature flags.
func (s *FlagService) List(ctx context.Context, limit, offset int) ([]models.FeatureFlag, error) {
	return s.repo.List(ctx, limit, offset)
}

// Update updates a feature flag.
func (s *FlagService) Update(ctx context.Context, flag *models.FeatureFlag) error {
	return s.repo.Update(ctx, flag)
}

// Create creates a new feature flag.
func (s *FlagService) Create(ctx context.Context, flag *models.FeatureFlag) error {
	return s.repo.Create(ctx, flag)
}

// Middleware injects feature flags into the request context.
func Middleware(svc *FlagService) fiber.Handler {
	return func(c fiber.Ctx) error {
		flags, _ := svc.List(c.Context(), 1000, 0)
		enabledFlags := make(map[string]bool)
		userID, _ := c.Locals("user_id").(string)
		for _, f := range flags {
			enabledFlags[f.Name] = svc.IsEnabled(c.Context(), f.Name, userID, "")
		}
		c.Locals("feature_flags", enabledFlags)
		return c.Next()
	}
}
