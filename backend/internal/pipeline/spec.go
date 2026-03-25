package pipeline

import "encoding/json"

// PipelineSpec is the top-level pipeline configuration matching flowforge.yml.
type PipelineSpec struct {
	Version    string                `yaml:"version" json:"version"`
	Name       string                `yaml:"name" json:"name"`
	On         TriggerConfig         `yaml:"on" json:"on"`
	Defaults   *DefaultsConfig       `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	Env        map[string]string     `yaml:"env,omitempty" json:"env,omitempty"`
	Stages     []string              `yaml:"stages,omitempty" json:"stages,omitempty"`
	StageNeeds map[string][]string   `yaml:"stage_needs,omitempty" json:"stage_needs,omitempty"` // stage-level DAG dependencies: stage_name → [dependency_stage_names]
	Jobs       map[string]JobSpec    `yaml:"jobs" json:"jobs"`
	Notify     *NotifyConfig         `yaml:"notify,omitempty" json:"notify,omitempty"`
}

// TriggerConfig holds all pipeline trigger definitions.
type TriggerConfig struct {
	Push        *PushTrigger        `yaml:"push,omitempty" json:"push,omitempty"`
	PullRequest *PullRequestTrigger `yaml:"pull_request,omitempty" json:"pull_request,omitempty"`
	Schedule    []ScheduleTrigger   `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	Manual      *ManualTrigger      `yaml:"manual,omitempty" json:"manual,omitempty"`
}

// PushTrigger configures push-based pipeline triggers.
type PushTrigger struct {
	Branches    []string `yaml:"branches,omitempty" json:"branches,omitempty"`
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Paths       []string `yaml:"paths,omitempty" json:"paths,omitempty"`
	IgnorePaths []string `yaml:"ignore_paths,omitempty" json:"ignore_paths,omitempty"`
}

// PullRequestTrigger configures PR-based pipeline triggers.
type PullRequestTrigger struct {
	Types    []string `yaml:"types,omitempty" json:"types,omitempty"`
	Branches []string `yaml:"branches,omitempty" json:"branches,omitempty"`
}

// ScheduleTrigger configures cron-based pipeline triggers.
type ScheduleTrigger struct {
	Cron     string `yaml:"cron" json:"cron"`
	Timezone string `yaml:"timezone,omitempty" json:"timezone,omitempty"`
	Branch   string `yaml:"branch,omitempty" json:"branch,omitempty"`
}

// ManualTrigger configures manual pipeline triggers with input parameters.
type ManualTrigger struct {
	Inputs map[string]ManualInput `yaml:"inputs,omitempty" json:"inputs,omitempty"`
}

// ManualInput describes a single input parameter for manual triggers.
type ManualInput struct {
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool     `yaml:"required,omitempty" json:"required,omitempty"`
	Default     string   `yaml:"default,omitempty" json:"default,omitempty"`
	Type        string   `yaml:"type,omitempty" json:"type,omitempty"`
	Options     []string `yaml:"options,omitempty" json:"options,omitempty"`
}

// DefaultsConfig provides default settings applied to all jobs unless overridden.
type DefaultsConfig struct {
	Timeout  string       `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retry    *RetryConfig `yaml:"retry,omitempty" json:"retry,omitempty"`
	Executor string       `yaml:"executor,omitempty" json:"executor,omitempty"`
	Image    string       `yaml:"image,omitempty" json:"image,omitempty"`
}

// RetryConfig configures step/job retry behavior.
type RetryConfig struct {
	Count int      `yaml:"count" json:"count"`
	Delay string   `yaml:"delay,omitempty" json:"delay,omitempty"`
	On    []string `yaml:"on,omitempty" json:"on,omitempty"`
}

// JobSpec defines a single job within the pipeline.
type JobSpec struct {
	Stage            string            `yaml:"stage" json:"stage"`
	Executor         string            `yaml:"executor,omitempty" json:"executor,omitempty"`
	Image            string            `yaml:"image,omitempty" json:"image,omitempty"`
	Env              map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Cache            CacheList         `yaml:"cache,omitempty" json:"cache,omitempty"`
	Steps            []StepSpec        `yaml:"steps" json:"steps"`
	Needs            []string          `yaml:"needs,omitempty" json:"needs,omitempty"`
	Matrix           *MatrixConfig     `yaml:"matrix,omitempty" json:"matrix,omitempty"`
	When             string            `yaml:"when,omitempty" json:"when,omitempty"`
	Environment      string            `yaml:"environment,omitempty" json:"environment,omitempty"`
	ApprovalRequired bool              `yaml:"approval_required,omitempty" json:"approval_required,omitempty"`
	Privileged       bool              `yaml:"privileged,omitempty" json:"privileged,omitempty"`
	Timeout          string            `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retry            *RetryConfig      `yaml:"retry,omitempty" json:"retry,omitempty"`
	ContinueOnError  bool              `yaml:"continue_on_error,omitempty" json:"continue_on_error,omitempty"`
}

// StepSpec defines a single step within a job.
type StepSpec struct {
	Name    string            `yaml:"name,omitempty" json:"name,omitempty"`
	Uses    string            `yaml:"uses,omitempty" json:"uses,omitempty"`
	Run     string            `yaml:"run,omitempty" json:"run,omitempty"`
	With    map[string]string `yaml:"with,omitempty" json:"with,omitempty"`
	Outputs []string          `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	If      string            `yaml:"if,omitempty" json:"if,omitempty"`
	Env     map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
}

// MatrixConfig defines the matrix build configuration. Keys map to lists of
// values; jobs are expanded to cover the full Cartesian product of all keys.
type MatrixConfig struct {
	Entries map[string][]string `yaml:"-" json:"-"`
}

// UnmarshalYAML implements custom YAML unmarshalling for MatrixConfig, which
// is a dynamic map of string keys to string-slice values.
func (m *MatrixConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	raw := make(map[string][]string)
	if err := unmarshal(&raw); err != nil {
		return err
	}
	m.Entries = raw
	return nil
}

// MarshalYAML implements custom YAML marshalling for MatrixConfig.
func (m MatrixConfig) MarshalYAML() (interface{}, error) {
	return m.Entries, nil
}

// UnmarshalJSON implements custom JSON unmarshalling for MatrixConfig.
func (m *MatrixConfig) UnmarshalJSON(data []byte) error {
	raw := make(map[string][]string)
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.Entries = raw
	return nil
}

// MarshalJSON implements custom JSON marshalling for MatrixConfig.
func (m MatrixConfig) MarshalJSON() ([]byte, error) {
	if m.Entries == nil {
		return []byte("null"), nil
	}
	return json.Marshal(m.Entries)
}

// CacheConfig defines a cache entry for a job.
type CacheConfig struct {
	Key   string   `yaml:"key" json:"key"`
	Paths []string `yaml:"paths" json:"paths"`
}

// CacheList is a list of CacheConfig that accepts either a single object or
// a list in YAML/JSON. This allows both:
//
//	cache:
//	  key: foo
//	  paths: [bar]
//
// and:
//
//	cache:
//	  - key: foo
//	    paths: [bar]
type CacheList []CacheConfig

// UnmarshalYAML accepts a single CacheConfig map or a list of CacheConfig.
func (cl *CacheList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try as list first
	var list []CacheConfig
	if err := unmarshal(&list); err == nil {
		*cl = list
		return nil
	}
	// Fall back to single object
	var single CacheConfig
	if err := unmarshal(&single); err != nil {
		return err
	}
	*cl = CacheList{single}
	return nil
}

// UnmarshalJSON accepts a single CacheConfig object or a list of CacheConfig.
func (cl *CacheList) UnmarshalJSON(data []byte) error {
	var list []CacheConfig
	if err := json.Unmarshal(data, &list); err == nil {
		*cl = list
		return nil
	}
	var single CacheConfig
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	*cl = CacheList{single}
	return nil
}

// NotifyConfig defines notification rules for the pipeline.
type NotifyConfig struct {
	OnFailure    []NotifyChannel `yaml:"on_failure,omitempty" json:"on_failure,omitempty"`
	OnSuccess    []NotifyChannel `yaml:"on_success,omitempty" json:"on_success,omitempty"`
	OnDeployment []NotifyChannel `yaml:"on_deployment,omitempty" json:"on_deployment,omitempty"`
}

// NotifyChannel references a notification channel and optional conditions.
type NotifyChannel struct {
	Channel string `yaml:"channel" json:"channel"`
	When    string `yaml:"when,omitempty" json:"when,omitempty"`
}
