package pipeline

import (
	"fmt"
	"strings"
)

// ValidationError represents a single validation error with context about
// where the error occurred.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface for ValidationError.
func (v ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", v.Field, v.Message)
}

// Validate performs comprehensive validation of a PipelineSpec and returns all
// errors found.
func Validate(spec *PipelineSpec) []ValidationError {
	var errs []ValidationError

	errs = append(errs, validateTopLevel(spec)...)
	errs = append(errs, validateStages(spec)...)
	errs = append(errs, validateJobs(spec)...)
	errs = append(errs, validateDependencies(spec)...)
	errs = append(errs, validateTriggers(spec)...)

	return errs
}

// validateTopLevel checks required top-level fields.
func validateTopLevel(spec *PipelineSpec) []ValidationError {
	var errs []ValidationError

	if spec.Name == "" {
		errs = append(errs, ValidationError{Field: "name", Message: "pipeline name is required"})
	}
	if spec.Version == "" {
		errs = append(errs, ValidationError{Field: "version", Message: "pipeline version is required"})
	}
	if len(spec.Jobs) == 0 {
		errs = append(errs, ValidationError{Field: "jobs", Message: "at least one job is required"})
	}

	return errs
}

// validateStages ensures all stages referenced by jobs are defined in the
// top-level stages list (if stages are specified).
func validateStages(spec *PipelineSpec) []ValidationError {
	var errs []ValidationError

	// If no explicit stages are defined, skip stage validation.
	if len(spec.Stages) == 0 {
		return nil
	}

	stageSet := make(map[string]bool, len(spec.Stages))
	for _, s := range spec.Stages {
		if stageSet[s] {
			errs = append(errs, ValidationError{
				Field:   "stages",
				Message: fmt.Sprintf("duplicate stage %q", s),
			})
		}
		stageSet[s] = true
	}

	for name, job := range spec.Jobs {
		if job.Stage == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("jobs.%s.stage", name),
				Message: "stage is required when stages are defined",
			})
			continue
		}
		if !stageSet[job.Stage] {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("jobs.%s.stage", name),
				Message: fmt.Sprintf("references undefined stage %q", job.Stage),
			})
		}
	}

	return errs
}

// validateJobs validates each individual job definition.
func validateJobs(spec *PipelineSpec) []ValidationError {
	var errs []ValidationError

	for name, job := range spec.Jobs {
		prefix := fmt.Sprintf("jobs.%s", name)

		if len(job.Steps) == 0 {
			errs = append(errs, ValidationError{
				Field:   prefix + ".steps",
				Message: "at least one step is required",
			})
		}

		// Validate each step
		for i, step := range job.Steps {
			stepPrefix := fmt.Sprintf("%s.steps[%d]", prefix, i)
			if step.Uses == "" && step.Run == "" {
				errs = append(errs, ValidationError{
					Field:   stepPrefix,
					Message: "step must have either 'uses' or 'run'",
				})
			}
			if step.Uses != "" && step.Run != "" {
				errs = append(errs, ValidationError{
					Field:   stepPrefix,
					Message: "step cannot have both 'uses' and 'run'",
				})
			}
		}

		// Validate needs references
		for _, dep := range job.Needs {
			if _, ok := spec.Jobs[dep]; !ok {
				errs = append(errs, ValidationError{
					Field:   prefix + ".needs",
					Message: fmt.Sprintf("references undefined job %q", dep),
				})
			}
		}

		// Validate matrix
		if job.Matrix != nil {
			if len(job.Matrix.Entries) == 0 {
				errs = append(errs, ValidationError{
					Field:   prefix + ".matrix",
					Message: "matrix must have at least one dimension",
				})
			}
			for key, values := range job.Matrix.Entries {
				if len(values) == 0 {
					errs = append(errs, ValidationError{
						Field:   fmt.Sprintf("%s.matrix.%s", prefix, key),
						Message: "matrix dimension must have at least one value",
					})
				}
			}
		}

		// Validate retry
		if job.Retry != nil {
			if job.Retry.Count < 0 {
				errs = append(errs, ValidationError{
					Field:   prefix + ".retry.count",
					Message: "retry count must be non-negative",
				})
			}
		}
	}

	return errs
}

// validateDependencies checks that job dependencies form a DAG using
// topological sort (Kahn's algorithm). Returns errors if cycles are detected.
func validateDependencies(spec *PipelineSpec) []ValidationError {
	var errs []ValidationError

	// Build adjacency list and in-degree count
	inDegree := make(map[string]int, len(spec.Jobs))
	dependents := make(map[string][]string, len(spec.Jobs))

	for name := range spec.Jobs {
		inDegree[name] = 0
	}

	for name, job := range spec.Jobs {
		for _, dep := range job.Needs {
			if _, exists := spec.Jobs[dep]; !exists {
				// Already reported in validateJobs
				continue
			}
			dependents[dep] = append(dependents[dep], name)
			inDegree[name]++
		}
	}

	// Kahn's algorithm — find all nodes with no dependencies
	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}

	visited := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		visited++
		for _, dep := range dependents[node] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if visited != len(spec.Jobs) {
		// Find which jobs are part of the cycle
		var cycleJobs []string
		for name, deg := range inDegree {
			if deg > 0 {
				cycleJobs = append(cycleJobs, name)
			}
		}
		errs = append(errs, ValidationError{
			Field:   "jobs",
			Message: fmt.Sprintf("dependency cycle detected involving jobs: %s", strings.Join(cycleJobs, ", ")),
		})
	}

	return errs
}

// validateTriggers validates trigger configurations.
func validateTriggers(spec *PipelineSpec) []ValidationError {
	var errs []ValidationError

	on := &spec.On

	if on.Push == nil && on.PullRequest == nil && len(on.Schedule) == 0 && on.Manual == nil {
		// No triggers is allowed (API-only), but we note it as a warning
		// For now, no error.
	}

	// Validate schedule triggers
	for i, sched := range on.Schedule {
		if sched.Cron == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("on.schedule[%d].cron", i),
				Message: "cron expression is required",
			})
		}
	}

	// Validate manual inputs
	if on.Manual != nil {
		for name, input := range on.Manual.Inputs {
			if input.Type == "choice" && len(input.Options) == 0 {
				errs = append(errs, ValidationError{
					Field:   fmt.Sprintf("on.manual.inputs.%s.options", name),
					Message: "choice input must have at least one option",
				})
			}
		}
	}

	return errs
}

// TopologicalSort returns the job names in a valid execution order. It assumes
// the DAG has already been validated (no cycles).
func TopologicalSort(spec *PipelineSpec) ([]string, error) {
	inDegree := make(map[string]int, len(spec.Jobs))
	dependents := make(map[string][]string, len(spec.Jobs))

	for name := range spec.Jobs {
		inDegree[name] = 0
	}

	for name, job := range spec.Jobs {
		for _, dep := range job.Needs {
			dependents[dep] = append(dependents[dep], name)
			inDegree[name]++
		}
	}

	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		sorted = append(sorted, node)
		for _, dep := range dependents[node] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(sorted) != len(spec.Jobs) {
		return nil, fmt.Errorf("dependency cycle detected")
	}

	return sorted, nil
}
