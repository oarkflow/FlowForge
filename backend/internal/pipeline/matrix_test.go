package pipeline

import (
	"testing"
)

func TestExpandMatrix_SingleDimension(t *testing.T) {
	job := JobSpec{
		Matrix: &MatrixConfig{
			Entries: map[string][]string{"go_version": {"1.21", "1.22"}},
		},
		Steps: []StepSpec{{Run: "echo"}},
	}
	result := ExpandMatrix("test", job)
	if len(result) != 2 {
		t.Fatalf("ExpandMatrix() returned %d jobs, want 2", len(result))
	}
	for _, j := range result {
		if j.Matrix != nil {
			t.Error("expanded job should have nil Matrix")
		}
	}
}

func TestExpandMatrix_TwoDimensions(t *testing.T) {
	job := JobSpec{
		Matrix: &MatrixConfig{
			Entries: map[string][]string{
				"go_version": {"1.21", "1.22"},
				"os":         {"linux", "darwin"},
			},
		},
		Steps: []StepSpec{{Run: "echo"}},
	}
	result := ExpandMatrix("test", job)
	if len(result) != 4 {
		t.Fatalf("ExpandMatrix() returned %d jobs, want 4", len(result))
	}
}

func TestExpandMatrix_NoMatrix(t *testing.T) {
	job := JobSpec{Steps: []StepSpec{{Run: "echo"}}}
	result := ExpandMatrix("test", job)
	if len(result) != 1 {
		t.Fatalf("ExpandMatrix() with no matrix returned %d jobs, want 1", len(result))
	}
	if _, ok := result["test"]; !ok {
		t.Error("should contain original job name")
	}
}

func TestExpandMatrix_NilMatrix(t *testing.T) {
	job := JobSpec{Matrix: nil, Steps: []StepSpec{{Run: "echo"}}}
	result := ExpandMatrix("test", job)
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
}

func TestExpandMatrix_EmptyEntries(t *testing.T) {
	job := JobSpec{
		Matrix: &MatrixConfig{Entries: map[string][]string{}},
		Steps:  []StepSpec{{Run: "echo"}},
	}
	result := ExpandMatrix("test", job)
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
}

func TestExpandMatrix_EnvVarsSet(t *testing.T) {
	job := JobSpec{
		Matrix: &MatrixConfig{
			Entries: map[string][]string{"ver": {"1.0"}},
		},
		Steps: []StepSpec{{Run: "echo"}},
	}
	result := ExpandMatrix("test", job)
	for _, j := range result {
		if v, ok := j.Env["MATRIX_VER"]; !ok || v != "1.0" {
			t.Errorf("MATRIX_VER not set correctly: %v", j.Env)
		}
	}
}

func TestExpandMatrix_PreservesExistingEnv(t *testing.T) {
	job := JobSpec{
		Matrix: &MatrixConfig{
			Entries: map[string][]string{"x": {"1"}},
		},
		Env:   map[string]string{"EXISTING": "value"},
		Steps: []StepSpec{{Run: "echo"}},
	}
	result := ExpandMatrix("test", job)
	for _, j := range result {
		if j.Env["EXISTING"] != "value" {
			t.Error("existing env var not preserved")
		}
	}
}

func TestExpandAllMatrices(t *testing.T) {
	spec := &PipelineSpec{
		Jobs: map[string]JobSpec{
			"regular": {Steps: []StepSpec{{Run: "echo"}}},
			"matrix": {
				Matrix: &MatrixConfig{
					Entries: map[string][]string{"v": {"a", "b"}},
				},
				Steps: []StepSpec{{Run: "echo"}},
			},
		},
	}
	result := ExpandAllMatrices(spec)
	// Should have: 1 regular + 2 expanded = 3
	if len(result) != 3 {
		t.Errorf("ExpandAllMatrices() returned %d jobs, want 3", len(result))
	}
}

func TestToEnvKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"go_version", "GO_VERSION"},
		{"os", "OS"},
		{"my-var", "MY_VAR"},
		{"ALREADY_UPPER", "ALREADY_UPPER"},
		{"mixed.dots", "MIXED_DOTS"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toEnvKey(tt.input)
			if got != tt.want {
				t.Errorf("toEnvKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
