package handlers

import (
	"testing"
	"time"

	"github.com/oarkflow/deploy/backend/internal/models"
)

func TestNormalizeRunHierarchyMarksActiveChildrenTerminalWhenRunFailed(t *testing.T) {
	now := time.Now()
	run := &models.PipelineRun{Status: "failure"}
	stages := []models.StageRun{
		{ID: "stage-running", Status: "running", StartedAt: &now},
		{ID: "stage-pending", Status: "pending"},
	}
	jobs := []models.JobRun{
		{ID: "job-running", StageRunID: "stage-running", Status: "running", StartedAt: &now},
		{ID: "job-pending", StageRunID: "stage-running", Status: "pending"},
		{ID: "job-stage-pending", StageRunID: "stage-pending", Status: "pending"},
	}
	steps := []models.StepRun{
		{ID: "step-running", JobRunID: "job-running", Status: "running", StartedAt: &now},
		{ID: "step-pending", JobRunID: "job-running", Status: "pending"},
		{ID: "step-stage-pending", JobRunID: "job-stage-pending", Status: "pending"},
	}

	stages, jobs, steps = normalizeRunHierarchy(run, stages, jobs, steps)

	if stages[0].Status != "failure" {
		t.Fatalf("running stage normalized to %q, want failure", stages[0].Status)
	}
	if stages[1].Status != "skipped" {
		t.Fatalf("pending stage normalized to %q, want skipped", stages[1].Status)
	}
	if jobs[0].Status != "failure" {
		t.Fatalf("running job normalized to %q, want failure", jobs[0].Status)
	}
	if jobs[1].Status != "skipped" {
		t.Fatalf("pending job in failed stage normalized to %q, want skipped", jobs[1].Status)
	}
	if jobs[2].Status != "skipped" {
		t.Fatalf("pending job in skipped stage normalized to %q, want skipped", jobs[2].Status)
	}
	if steps[0].Status != "failure" {
		t.Fatalf("running step normalized to %q, want failure", steps[0].Status)
	}
	if steps[1].Status != "skipped" {
		t.Fatalf("pending step in failed job normalized to %q, want skipped", steps[1].Status)
	}
	if steps[2].Status != "skipped" {
		t.Fatalf("pending step in skipped job normalized to %q, want skipped", steps[2].Status)
	}
}

func TestNormalizeRunHierarchyLeavesActiveRunsUntouched(t *testing.T) {
	run := &models.PipelineRun{Status: "running"}
	stages := []models.StageRun{{ID: "stage-1", Status: "running"}}
	jobs := []models.JobRun{{ID: "job-1", StageRunID: "stage-1", Status: "running"}}
	steps := []models.StepRun{{ID: "step-1", JobRunID: "job-1", Status: "running"}}

	stages, jobs, steps = normalizeRunHierarchy(run, stages, jobs, steps)

	if stages[0].Status != "running" || jobs[0].Status != "running" || steps[0].Status != "running" {
		t.Fatal("expected non-terminal run hierarchy to remain unchanged")
	}
}
