package pipeline

import (
	"encoding/json"
	"strings"
	"testing"
)

// --- Parser Tests ---

func TestParseYAML_BasicPipeline(t *testing.T) {
	yaml := `
version: "1"
name: "Test CI"
on:
  push:
    branches: ["main"]
stages:
  - test
  - build
jobs:
  unit-tests:
    stage: test
    steps:
      - name: Run tests
        run: go test ./...
  compile:
    stage: build
    needs: [unit-tests]
    steps:
      - name: Build
        run: go build ./...
`
	spec, err := ParseYAML(yaml)
	if err != nil {
		t.Fatalf("ParseYAML() error = %v", err)
	}
	if spec.Name != "Test CI" {
		t.Errorf("Name = %q, want %q", spec.Name, "Test CI")
	}
	if spec.Version != "1" {
		t.Errorf("Version = %q, want %q", spec.Version, "1")
	}
	if len(spec.Jobs) != 2 {
		t.Errorf("Jobs count = %d, want 2", len(spec.Jobs))
	}
	if len(spec.Stages) != 2 {
		t.Errorf("Stages count = %d, want 2", len(spec.Stages))
	}
}

func TestParseJSON_BasicPipeline(t *testing.T) {
	jsonStr := `{
		"version": "1",
		"name": "JSON CI",
		"jobs": {
			"test": {
				"stage": "test",
				"steps": [{"name": "Run", "run": "echo hello"}]
			}
		}
	}`
	spec, err := ParseJSON(jsonStr)
	if err != nil {
		t.Fatalf("ParseJSON() error = %v", err)
	}
	if spec.Name != "JSON CI" {
		t.Errorf("Name = %q, want %q", spec.Name, "JSON CI")
	}
}

func TestParse_AutoDetectsJSON(t *testing.T) {
	jsonStr := `{"version":"1","name":"Auto","jobs":{"j":{"steps":[{"run":"echo"}]}}}`
	spec, err := Parse(jsonStr)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Name != "Auto" {
		t.Errorf("Name = %q, want %q", spec.Name, "Auto")
	}
}

func TestParse_AutoDetectsYAML(t *testing.T) {
	yaml := `version: "1"
name: YAML Auto
jobs:
  j:
    steps:
      - run: echo`
	spec, err := Parse(yaml)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Name != "YAML Auto" {
		t.Errorf("Name = %q, want %q", spec.Name, "YAML Auto")
	}
}

func TestParse_EmptyInput(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Error("Parse(\"\") should return error")
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	_, err := ParseYAML("not: valid: yaml: [")
	if err == nil {
		t.Error("ParseYAML() with invalid YAML should return error")
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	_, err := ParseJSON("{invalid json}")
	if err == nil {
		t.Error("ParseJSON() with invalid JSON should return error")
	}
}

func TestApplyDefaults(t *testing.T) {
	yaml := `
version: "1"
name: "Defaults Test"
defaults:
  executor: docker
  image: alpine:latest
  timeout: 30m
  retry:
    count: 3
    delay: 10s
jobs:
  build:
    steps:
      - run: echo hello
  custom:
    executor: local
    image: custom:image
    steps:
      - run: echo custom
`
	spec, err := ParseYAML(yaml)
	if err != nil {
		t.Fatal(err)
	}

	build := spec.Jobs["build"]
	if build.Executor != "docker" {
		t.Errorf("build.Executor = %q, want %q", build.Executor, "docker")
	}
	if build.Image != "alpine:latest" {
		t.Errorf("build.Image = %q, want %q", build.Image, "alpine:latest")
	}
	if build.Timeout != "30m" {
		t.Errorf("build.Timeout = %q, want %q", build.Timeout, "30m")
	}
	if build.Retry == nil || build.Retry.Count != 3 {
		t.Error("build.Retry should inherit from defaults")
	}

	custom := spec.Jobs["custom"]
	if custom.Executor != "local" {
		t.Errorf("custom.Executor = %q, want %q (should not be overridden)", custom.Executor, "local")
	}
	if custom.Image != "custom:image" {
		t.Errorf("custom.Image = %q, want %q (should not be overridden)", custom.Image, "custom:image")
	}
}

func TestParseAndValidate_Valid(t *testing.T) {
	yaml := `
version: "1"
name: "Valid"
on:
  push:
    branches: ["main"]
jobs:
  test:
    steps:
      - run: echo ok
`
	spec, errs, err := ParseAndValidate(yaml)
	if err != nil {
		t.Fatalf("ParseAndValidate() error = %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("unexpected validation errors: %v", errs)
	}
	if spec == nil {
		t.Error("spec should not be nil")
	}
}

func TestParseAndValidate_Invalid(t *testing.T) {
	yaml := `
version: "1"
name: ""
jobs: {}
`
	_, errs, err := ParseAndValidate(yaml)
	if err == nil {
		t.Error("ParseAndValidate() should return error for invalid pipeline")
	}
	if len(errs) == 0 {
		t.Error("should have validation errors")
	}
}

// --- MatrixConfig Custom Marshaling ---

func TestMatrixConfig_UnmarshalYAML(t *testing.T) {
	yaml := `
version: "1"
name: Matrix
jobs:
  test:
    matrix:
      go_version: ["1.21", "1.22"]
      os: ["linux", "darwin"]
    steps:
      - run: echo test
`
	spec, err := ParseYAML(yaml)
	if err != nil {
		t.Fatal(err)
	}
	job := spec.Jobs["test"]
	if job.Matrix == nil {
		t.Fatal("Matrix should not be nil")
	}
	if len(job.Matrix.Entries["go_version"]) != 2 {
		t.Errorf("go_version entries = %d, want 2", len(job.Matrix.Entries["go_version"]))
	}
}

func TestMatrixConfig_JSON_RoundTrip(t *testing.T) {
	m := MatrixConfig{
		Entries: map[string][]string{
			"a": {"1", "2"},
			"b": {"x", "y"},
		},
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var m2 MatrixConfig
	if err := json.Unmarshal(data, &m2); err != nil {
		t.Fatal(err)
	}
	if len(m2.Entries["a"]) != 2 {
		t.Errorf("round-trip failed: entries[a] = %d, want 2", len(m2.Entries["a"]))
	}
}

func TestMatrixConfig_MarshalJSON_Nil(t *testing.T) {
	m := MatrixConfig{}
	data, err := m.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "null" {
		t.Errorf("MarshalJSON() with nil entries = %q, want %q", string(data), "null")
	}
}

// --- CacheList Custom Marshaling ---

func TestCacheList_SingleObjectYAML(t *testing.T) {
	yaml := `
version: "1"
name: Cache
jobs:
  test:
    cache:
      key: test-key
      paths:
        - /tmp/cache
    steps:
      - run: echo test
`
	spec, err := ParseYAML(yaml)
	if err != nil {
		t.Fatal(err)
	}
	job := spec.Jobs["test"]
	if len(job.Cache) != 1 {
		t.Fatalf("Cache entries = %d, want 1", len(job.Cache))
	}
	if job.Cache[0].Key != "test-key" {
		t.Errorf("Cache key = %q, want %q", job.Cache[0].Key, "test-key")
	}
}

func TestCacheList_ArrayYAML(t *testing.T) {
	yaml := `
version: "1"
name: Cache
jobs:
  test:
    cache:
      - key: key1
        paths: [/a]
      - key: key2
        paths: [/b]
    steps:
      - run: echo test
`
	spec, err := ParseYAML(yaml)
	if err != nil {
		t.Fatal(err)
	}
	job := spec.Jobs["test"]
	if len(job.Cache) != 2 {
		t.Fatalf("Cache entries = %d, want 2", len(job.Cache))
	}
}

func TestCacheList_UnmarshalJSON(t *testing.T) {
	jsonSingle := `{"key":"k","paths":["/p"]}`
	var cl CacheList
	if err := json.Unmarshal([]byte(jsonSingle), &cl); err != nil {
		t.Fatal(err)
	}
	if len(cl) != 1 {
		t.Errorf("CacheList len = %d, want 1", len(cl))
	}

	jsonArray := `[{"key":"k1","paths":["/a"]},{"key":"k2","paths":["/b"]}]`
	var cl2 CacheList
	if err := json.Unmarshal([]byte(jsonArray), &cl2); err != nil {
		t.Fatal(err)
	}
	if len(cl2) != 2 {
		t.Errorf("CacheList len = %d, want 2", len(cl2))
	}
}

// --- Validator Tests ---

func TestValidate_MissingName(t *testing.T) {
	spec := &PipelineSpec{Version: "1", Jobs: map[string]JobSpec{"j": {Steps: []StepSpec{{Run: "echo"}}}}}
	errs := Validate(spec)
	found := false
	for _, e := range errs {
		if e.Field == "name" {
			found = true
		}
	}
	if !found {
		t.Error("should report missing name")
	}
}

func TestValidate_MissingVersion(t *testing.T) {
	spec := &PipelineSpec{Name: "Test", Jobs: map[string]JobSpec{"j": {Steps: []StepSpec{{Run: "echo"}}}}}
	errs := Validate(spec)
	found := false
	for _, e := range errs {
		if e.Field == "version" {
			found = true
		}
	}
	if !found {
		t.Error("should report missing version")
	}
}

func TestValidate_NoJobs(t *testing.T) {
	spec := &PipelineSpec{Version: "1", Name: "Test", Jobs: map[string]JobSpec{}}
	errs := Validate(spec)
	found := false
	for _, e := range errs {
		if e.Field == "jobs" {
			found = true
		}
	}
	if !found {
		t.Error("should report no jobs")
	}
}

func TestValidate_StepMissingRunAndUses(t *testing.T) {
	spec := &PipelineSpec{
		Version: "1",
		Name:    "Test",
		Jobs: map[string]JobSpec{
			"j": {Steps: []StepSpec{{Name: "empty"}}},
		},
	}
	errs := Validate(spec)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "either 'uses' or 'run'") {
			found = true
		}
	}
	if !found {
		t.Error("should report step missing uses/run")
	}
}

func TestValidate_StepBothRunAndUses(t *testing.T) {
	spec := &PipelineSpec{
		Version: "1",
		Name:    "Test",
		Jobs: map[string]JobSpec{
			"j": {Steps: []StepSpec{{Run: "echo", Uses: "something"}}},
		},
	}
	errs := Validate(spec)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "both 'uses' and 'run'") {
			found = true
		}
	}
	if !found {
		t.Error("should report step with both uses and run")
	}
}

func TestValidate_UndefinedStageReference(t *testing.T) {
	spec := &PipelineSpec{
		Version: "1",
		Name:    "Test",
		Stages:  []string{"build"},
		Jobs: map[string]JobSpec{
			"j": {Stage: "nonexistent", Steps: []StepSpec{{Run: "echo"}}},
		},
	}
	errs := Validate(spec)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "undefined stage") {
			found = true
		}
	}
	if !found {
		t.Error("should report undefined stage reference")
	}
}

func TestValidate_DuplicateStage(t *testing.T) {
	spec := &PipelineSpec{
		Version: "1",
		Name:    "Test",
		Stages:  []string{"build", "build"},
		Jobs: map[string]JobSpec{
			"j": {Stage: "build", Steps: []StepSpec{{Run: "echo"}}},
		},
	}
	errs := Validate(spec)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "duplicate stage") {
			found = true
		}
	}
	if !found {
		t.Error("should report duplicate stage")
	}
}

func TestValidate_UndefinedJobDependency(t *testing.T) {
	spec := &PipelineSpec{
		Version: "1",
		Name:    "Test",
		Jobs: map[string]JobSpec{
			"j": {Needs: []string{"nonexistent"}, Steps: []StepSpec{{Run: "echo"}}},
		},
	}
	errs := Validate(spec)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "undefined job") {
			found = true
		}
	}
	if !found {
		t.Error("should report undefined job dependency")
	}
}

func TestValidate_DependencyCycle(t *testing.T) {
	spec := &PipelineSpec{
		Version: "1",
		Name:    "Test",
		Jobs: map[string]JobSpec{
			"a": {Needs: []string{"b"}, Steps: []StepSpec{{Run: "echo"}}},
			"b": {Needs: []string{"a"}, Steps: []StepSpec{{Run: "echo"}}},
		},
	}
	errs := Validate(spec)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "cycle") {
			found = true
		}
	}
	if !found {
		t.Error("should report dependency cycle")
	}
}

func TestValidate_EmptyMatrix(t *testing.T) {
	spec := &PipelineSpec{
		Version: "1",
		Name:    "Test",
		Jobs: map[string]JobSpec{
			"j": {Matrix: &MatrixConfig{Entries: map[string][]string{}}, Steps: []StepSpec{{Run: "echo"}}},
		},
	}
	errs := Validate(spec)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "at least one dimension") {
			found = true
		}
	}
	if !found {
		t.Error("should report empty matrix")
	}
}

func TestValidate_NegativeRetry(t *testing.T) {
	spec := &PipelineSpec{
		Version: "1",
		Name:    "Test",
		Jobs: map[string]JobSpec{
			"j": {Retry: &RetryConfig{Count: -1}, Steps: []StepSpec{{Run: "echo"}}},
		},
	}
	errs := Validate(spec)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "non-negative") {
			found = true
		}
	}
	if !found {
		t.Error("should report negative retry count")
	}
}

func TestValidate_ScheduleMissingCron(t *testing.T) {
	spec := &PipelineSpec{
		Version: "1",
		Name:    "Test",
		On: TriggerConfig{
			Schedule: []ScheduleTrigger{{Cron: ""}},
		},
		Jobs: map[string]JobSpec{
			"j": {Steps: []StepSpec{{Run: "echo"}}},
		},
	}
	errs := Validate(spec)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "cron expression is required") {
			found = true
		}
	}
	if !found {
		t.Error("should report missing cron expression")
	}
}

func TestValidate_ManualChoiceNoOptions(t *testing.T) {
	spec := &PipelineSpec{
		Version: "1",
		Name:    "Test",
		On: TriggerConfig{
			Manual: &ManualTrigger{
				Inputs: map[string]ManualInput{
					"env": {Type: "choice", Options: []string{}},
				},
			},
		},
		Jobs: map[string]JobSpec{
			"j": {Steps: []StepSpec{{Run: "echo"}}},
		},
	}
	errs := Validate(spec)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "at least one option") {
			found = true
		}
	}
	if !found {
		t.Error("should report choice input with no options")
	}
}

// --- TopologicalSort Tests ---

func TestTopologicalSort_LinearChain(t *testing.T) {
	spec := &PipelineSpec{
		Jobs: map[string]JobSpec{
			"a": {Steps: []StepSpec{{Run: "echo"}}},
			"b": {Needs: []string{"a"}, Steps: []StepSpec{{Run: "echo"}}},
			"c": {Needs: []string{"b"}, Steps: []StepSpec{{Run: "echo"}}},
		},
	}
	sorted, err := TopologicalSort(spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(sorted) != 3 {
		t.Fatalf("sorted len = %d, want 3", len(sorted))
	}

	pos := map[string]int{}
	for i, n := range sorted {
		pos[n] = i
	}
	if pos["a"] >= pos["b"] || pos["b"] >= pos["c"] {
		t.Errorf("invalid topological order: %v", sorted)
	}
}

func TestTopologicalSort_Cycle(t *testing.T) {
	spec := &PipelineSpec{
		Jobs: map[string]JobSpec{
			"a": {Needs: []string{"b"}, Steps: []StepSpec{{Run: "echo"}}},
			"b": {Needs: []string{"a"}, Steps: []StepSpec{{Run: "echo"}}},
		},
	}
	_, err := TopologicalSort(spec)
	if err == nil {
		t.Error("TopologicalSort() should return error for cycle")
	}
}

func TestTopologicalSort_NoDeps(t *testing.T) {
	spec := &PipelineSpec{
		Jobs: map[string]JobSpec{
			"a": {Steps: []StepSpec{{Run: "echo"}}},
			"b": {Steps: []StepSpec{{Run: "echo"}}},
		},
	}
	sorted, err := TopologicalSort(spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(sorted) != 2 {
		t.Errorf("sorted len = %d, want 2", len(sorted))
	}
}

// --- Stage Needs Validation ---

func TestValidateStageNeeds_ValidGraph(t *testing.T) {
	spec := &PipelineSpec{
		Version: "1",
		Name:    "Test",
		Stages:  []string{"build", "test", "deploy"},
		StageNeeds: map[string][]string{
			"test":   {"build"},
			"deploy": {"test"},
		},
		Jobs: map[string]JobSpec{
			"j1": {Stage: "build", Steps: []StepSpec{{Run: "echo"}}},
			"j2": {Stage: "test", Steps: []StepSpec{{Run: "echo"}}},
			"j3": {Stage: "deploy", Steps: []StepSpec{{Run: "echo"}}},
		},
	}
	errs := Validate(spec)
	for _, e := range errs {
		if strings.Contains(e.Field, "stage_needs") {
			t.Errorf("unexpected stage_needs error: %v", e)
		}
	}
}

func TestValidateStageNeeds_SelfReference(t *testing.T) {
	spec := &PipelineSpec{
		Version: "1",
		Name:    "Test",
		Stages:  []string{"build"},
		StageNeeds: map[string][]string{
			"build": {"build"},
		},
		Jobs: map[string]JobSpec{
			"j": {Stage: "build", Steps: []StepSpec{{Run: "echo"}}},
		},
	}
	errs := Validate(spec)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "self-reference") {
			found = true
		}
	}
	if !found {
		t.Error("should report self-reference in stage needs")
	}
}
