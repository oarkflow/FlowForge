package engine

import (
	"strings"
	"testing"

	"github.com/oarkflow/deploy/backend/internal/engine/scheduler"
)

func TestApplyAutogenCompatibilityFixesUpgradesLegacyNodeLintCommand(t *testing.T) {
	config := scheduler.PipelineConfig{
		Stages: []scheduler.StageConfig{
			{
				Name: "test",
				Jobs: []scheduler.JobConfig{
					{
						Name: "lint",
						Steps: []scheduler.StepConfig{
							{Name: "Checkout", Command: "echo checkout"},
							{Name: "Run linter", Command: "cd web && yarn lint"},
						},
					},
				},
			},
		},
	}

	applyAutogenCompatibilityFixes("Node.js CI", &config)

	command := config.Stages[0].Jobs[0].Steps[1].Command
	if !strings.Contains(command, `No lint script found, skipping lint`) {
		t.Fatalf("expected compatibility message in upgraded command, got:\n%s", command)
	}
	if !strings.Contains(command, "cd web") {
		t.Fatalf("expected upgraded command to preserve workdir, got:\n%s", command)
	}
	if !strings.HasSuffix(command, "yarn lint") {
		t.Fatalf("expected upgraded command to keep lint invocation, got:\n%s", command)
	}
}

func TestApplyAutogenCompatibilityFixesLeavesCustomLintCommandsUntouched(t *testing.T) {
	config := scheduler.PipelineConfig{
		Stages: []scheduler.StageConfig{
			{
				Name: "test",
				Jobs: []scheduler.JobConfig{
					{
						Name: "lint",
						Steps: []scheduler.StepConfig{
							{Name: "Run linter", Command: "yarn eslint ."},
						},
					},
				},
			},
		},
	}

	original := config.Stages[0].Jobs[0].Steps[0].Command
	applyAutogenCompatibilityFixes("Node.js CI", &config)

	if got := config.Stages[0].Jobs[0].Steps[0].Command; got != original {
		t.Fatalf("expected custom lint command to remain unchanged, got %q want %q", got, original)
	}
}

func TestApplyAutogenCompatibilityFixesUpgradesLegacyNodeTestCommand(t *testing.T) {
	config := scheduler.PipelineConfig{
		Stages: []scheduler.StageConfig{
			{
				Name: "test",
				Jobs: []scheduler.JobConfig{
					{
						Name: "test",
						Steps: []scheduler.StepConfig{
							{Name: "Checkout", Command: "echo checkout"},
							{Name: "Run tests", Command: "yarn test"},
						},
					},
				},
			},
		},
	}

	applyAutogenCompatibilityFixes("Node.js CI", &config)

	command := config.Stages[0].Jobs[0].Steps[1].Command
	if !strings.Contains(command, `export CI=true`) {
		t.Fatalf("expected upgraded command to force CI mode, got:\n%s", command)
	}
	if !strings.Contains(command, `No test script found, skipping tests`) {
		t.Fatalf("expected upgraded command to tolerate missing test script, got:\n%s", command)
	}
	if !strings.Contains(command, `craco\ test`) {
		t.Fatalf("expected upgraded command to include CRACO/Jest compatibility branch, got:\n%s", command)
	}
	if !strings.Contains(command, "yarn test") {
		t.Fatalf("expected upgraded command to preserve test invocation, got:\n%s", command)
	}
}
