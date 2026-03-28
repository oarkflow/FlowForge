package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ParseYAML parses a YAML pipeline configuration string into a PipelineSpec.
func ParseYAML(data string) (*PipelineSpec, error) {
	var spec PipelineSpec
	if err := yaml.Unmarshal([]byte(data), &spec); err != nil {
		return nil, fmt.Errorf("yaml parse error: %w", err)
	}
	if err := applyDefaults(&spec); err != nil {
		return nil, fmt.Errorf("defaults error: %w", err)
	}
	return &spec, nil
}

// ParseJSON parses a JSON pipeline configuration string into a PipelineSpec.
func ParseJSON(data string) (*PipelineSpec, error) {
	var spec PipelineSpec
	if err := json.Unmarshal([]byte(data), &spec); err != nil {
		return nil, fmt.Errorf("json parse error: %w", err)
	}
	if err := applyDefaults(&spec); err != nil {
		return nil, fmt.Errorf("defaults error: %w", err)
	}
	return &spec, nil
}

// Parse auto-detects the format (YAML or JSON) and parses the pipeline config.
// It tries JSON first (if the trimmed input starts with '{'), otherwise YAML.
func Parse(data string) (*PipelineSpec, error) {
	trimmed := strings.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty pipeline configuration")
	}
	if trimmed[0] == '{' {
		return ParseJSON(data)
	}
	return ParseYAML(data)
}

// ParseAndValidate parses the pipeline config and runs full validation.
func ParseAndValidate(data string) (*PipelineSpec, []ValidationError, error) {
	spec, err := Parse(data)
	if err != nil {
		return nil, nil, err
	}
	errs := Validate(spec)
	if len(errs) > 0 {
		return spec, errs, fmt.Errorf("validation failed with %d error(s)", len(errs))
	}
	return spec, nil, nil
}

// applyDefaults populates jobs with values from the defaults section where
// the job does not have its own value set.
func applyDefaults(spec *PipelineSpec) error {
	if spec.Defaults == nil {
		return nil
	}
	d := spec.Defaults
	for name, job := range spec.Jobs {
		if job.Executor == "" && d.Executor != "" {
			job.Executor = d.Executor
		}
		if job.Image == "" && d.Image != "" {
			job.Image = d.Image
		}
		if job.Timeout == "" && d.Timeout != "" {
			job.Timeout = d.Timeout
		}
		if job.Retry == nil && d.Retry != nil {
			copied := *d.Retry
			job.Retry = &copied
		}
		spec.Jobs[name] = job
	}
	return nil
}
