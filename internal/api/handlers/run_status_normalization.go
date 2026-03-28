package handlers

import "github.com/oarkflow/deploy/backend/internal/models"

func normalizeRunHierarchy(run *models.PipelineRun, stages []models.StageRun, jobs []models.JobRun, steps []models.StepRun) ([]models.StageRun, []models.JobRun, []models.StepRun) {
	if run == nil || !isTerminalRunStatus(run.Status) {
		return stages, jobs, steps
	}

	stageStatuses := make(map[string]string, len(stages))
	for i := range stages {
		if isActiveRunStatus(stages[i].Status) {
			stages[i].Status = normalizedChildStatus(run.Status, stages[i].StartedAt != nil)
		}
		stageStatuses[stages[i].ID] = stages[i].Status
	}

	jobStatuses := make(map[string]string, len(jobs))
	for i := range jobs {
		if isActiveRunStatus(jobs[i].Status) {
			parentStatus := stageStatuses[jobs[i].StageRunID]
			jobs[i].Status = normalizedChildStatus(parentStatus, jobs[i].StartedAt != nil)
		}
		jobStatuses[jobs[i].ID] = jobs[i].Status
	}

	for i := range steps {
		if isActiveRunStatus(steps[i].Status) {
			parentStatus := jobStatuses[steps[i].JobRunID]
			steps[i].Status = normalizedChildStatus(parentStatus, steps[i].StartedAt != nil)
		}
	}

	return stages, jobs, steps
}

func isTerminalRunStatus(status string) bool {
	switch status {
	case "success", "failure", "cancelled", "skipped":
		return true
	default:
		return false
	}
}

func isActiveRunStatus(status string) bool {
	switch status {
	case "queued", "pending", "running", "waiting_approval":
		return true
	default:
		return false
	}
}

func normalizedChildStatus(parentStatus string, started bool) string {
	switch parentStatus {
	case "success":
		if started {
			return "success"
		}
		return "skipped"
	case "failure":
		if started {
			return "failure"
		}
		return "skipped"
	case "cancelled":
		return "cancelled"
	case "skipped":
		return "skipped"
	default:
		if started {
			return parentStatus
		}
		return "skipped"
	}
}
