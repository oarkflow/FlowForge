package pipeline

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/models"
)

// MaxTriggerChainDepth limits how deep downstream pipeline triggers can cascade,
// preventing infinite trigger loops (e.g., A→B→C→A).
const MaxTriggerChainDepth = 10

// PipelineTriggerFunc is a function that triggers a pipeline run.
// It is provided by the engine to avoid circular imports.
type PipelineTriggerFunc func(ctx context.Context, pipelineID, triggerType string, triggerData map[string]string) error

// CompositionService handles cross-pipeline triggers (pipeline composition).
// After a pipeline run completes, it checks for downstream pipeline links and
// triggers them based on conditions and link type.
type CompositionService struct {
	repos     *queries.Repositories
	triggerFn PipelineTriggerFunc
}

// NewCompositionService creates a new CompositionService.
func NewCompositionService(repos *queries.Repositories, triggerFn PipelineTriggerFunc) *CompositionService {
	return &CompositionService{
		repos:     repos,
		triggerFn: triggerFn,
	}
}

// TriggerDownstream finds and triggers downstream pipelines after a run completes.
// It respects link conditions (success, failure, always) and passes variables if configured.
// The depth parameter tracks the trigger chain depth to prevent infinite loops.
func (c *CompositionService) TriggerDownstream(ctx context.Context, pipelineID string, runStatus string, variables map[string]string, depth int) error {
	if depth >= MaxTriggerChainDepth {
		log.Warn().
			Str("pipeline_id", pipelineID).
			Int("depth", depth).
			Msg("composition: max trigger chain depth reached, skipping downstream triggers")
		return nil
	}

	links, err := c.repos.PipelineLinks.ListBySource(pipelineID)
	if err != nil {
		return fmt.Errorf("composition: failed to list downstream links: %w", err)
	}

	var triggerErrors []error
	for _, link := range links {
		if !link.Enabled {
			continue
		}

		if !shouldTrigger(link, runStatus) {
			log.Debug().
				Str("link_id", link.ID).
				Str("condition", link.Condition).
				Str("run_status", runStatus).
				Msg("composition: link condition not met, skipping")
			continue
		}

		triggerData := map[string]string{
			"trigger_type":       "pipeline",
			"source_pipeline_id": pipelineID,
			"link_type":          link.LinkType,
			"chain_depth":        fmt.Sprintf("%d", depth+1),
		}

		// Pass variables if configured
		if link.PassVariables && variables != nil {
			for k, v := range variables {
				triggerData[k] = v
			}
		}

		log.Info().
			Str("source_pipeline_id", pipelineID).
			Str("target_pipeline_id", link.TargetPipelineID).
			Str("link_type", link.LinkType).
			Msg("composition: triggering downstream pipeline")

		if err := c.triggerFn(ctx, link.TargetPipelineID, "pipeline", triggerData); err != nil {
			log.Error().Err(err).
				Str("target_pipeline_id", link.TargetPipelineID).
				Msg("composition: failed to trigger downstream pipeline")
			triggerErrors = append(triggerErrors, err)
		}
	}

	if len(triggerErrors) > 0 {
		return fmt.Errorf("composition: %d downstream trigger(s) failed", len(triggerErrors))
	}

	return nil
}

// shouldTrigger checks if a pipeline link's condition matches the run status.
func shouldTrigger(link models.PipelineLink, runStatus string) bool {
	condition := link.Condition
	if condition == "" {
		condition = "success" // default: only trigger on success
	}

	switch condition {
	case "always":
		return true
	case "success":
		return runStatus == "success"
	case "failure":
		return runStatus == "failure"
	default:
		return runStatus == condition
	}
}
