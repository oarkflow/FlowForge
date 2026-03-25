package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
)

// GetPipelineBadge returns an SVG badge showing the pipeline's last run status.
// This endpoint is PUBLIC (no auth required) so badges can be embedded in READMEs.
func (h *Handler) GetPipelineBadge(c fiber.Ctx) error {
	pipelineID := c.Params("id")

	// Look up the pipeline name
	pipeline, err := h.repo.Pipelines.GetByID(c.Context(), pipelineID)
	if err != nil {
		return svgBadge(c, "pipeline", "not found", "#9f9f9f")
	}

	pipelineName := pipeline.Name

	// Get the latest run for this pipeline
	runs, err := h.repo.Runs.ListByPipeline(c.Context(), pipelineID, 1, 0)
	if err != nil || len(runs) == 0 {
		return svgBadge(c, pipelineName, "unknown", "#9f9f9f")
	}

	lastRun := runs[0]
	var statusText string
	var color string

	switch lastRun.Status {
	case "success":
		statusText = "passing"
		color = "#4c1"
	case "failure":
		statusText = "failing"
		color = "#e05d44"
	case "running":
		statusText = "running"
		color = "#dfb317"
	case "queued", "pending":
		statusText = "pending"
		color = "#dfb317"
	case "cancelled":
		statusText = "cancelled"
		color = "#9f9f9f"
	default:
		statusText = "unknown"
		color = "#9f9f9f"
	}

	return svgBadge(c, pipelineName, statusText, color)
}

// svgBadge generates a shields.io-style SVG badge.
func svgBadge(c fiber.Ctx, label, status, color string) error {
	labelWidth := float64(len(label))*6.5 + 10.0
	statusWidth := float64(len(status))*6.5 + 10.0
	totalWidth := labelWidth + statusWidth

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="20" role="img" aria-label="%s: %s">
  <title>%s: %s</title>
  <linearGradient id="s" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="r">
    <rect width="%.0f" height="20" rx="3" fill="#fff"/>
  </clipPath>
  <g clip-path="url(#r)">
    <rect width="%.0f" height="20" fill="#555"/>
    <rect x="%.0f" width="%.0f" height="20" fill="%s"/>
    <rect width="%.0f" height="20" fill="url(#s)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="110">
    <text aria-hidden="true" x="%.0f" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)">%s</text>
    <text x="%.0f" y="140" transform="scale(.1)">%s</text>
    <text aria-hidden="true" x="%.0f" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)">%s</text>
    <text x="%.0f" y="140" transform="scale(.1)">%s</text>
  </g>
</svg>`,
		totalWidth, label, status,
		label, status,
		totalWidth,
		labelWidth,
		labelWidth, statusWidth, color,
		totalWidth,
		labelWidth/2*10, label,
		labelWidth/2*10, label,
		(labelWidth+statusWidth/2)*10, status,
		(labelWidth+statusWidth/2)*10, status,
	)

	c.Set("Content-Type", "image/svg+xml")
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")
	return c.SendString(svg)
}
