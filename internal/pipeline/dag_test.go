package pipeline

import (
	"testing"
)

func TestBuildStageDAG_NoDeps(t *testing.T) {
	stages := []string{"build", "test", "deploy"}
	dag, err := BuildStageDAG(stages, nil)
	if err != nil {
		t.Fatal(err)
	}
	if dag.HasCycle {
		t.Error("should not have cycle")
	}
	// Without deps, each stage is at its own level (sequential)
	if len(dag.Levels) != 3 {
		t.Errorf("levels = %d, want 3", len(dag.Levels))
	}
}

func TestBuildStageDAG_LinearDeps(t *testing.T) {
	stages := []string{"build", "test", "deploy"}
	deps := map[string][]string{
		"test":   {"build"},
		"deploy": {"test"},
	}
	dag, err := BuildStageDAG(stages, deps)
	if err != nil {
		t.Fatal(err)
	}
	if dag.HasCycle {
		t.Error("should not have cycle")
	}
	if len(dag.Levels) != 3 {
		t.Errorf("levels = %d, want 3", len(dag.Levels))
	}
	if dag.Nodes["build"].Level != 0 {
		t.Errorf("build level = %d, want 0", dag.Nodes["build"].Level)
	}
	if dag.Nodes["test"].Level != 1 {
		t.Errorf("test level = %d, want 1", dag.Nodes["test"].Level)
	}
	if dag.Nodes["deploy"].Level != 2 {
		t.Errorf("deploy level = %d, want 2", dag.Nodes["deploy"].Level)
	}
}

func TestBuildStageDAG_ParallelStages(t *testing.T) {
	stages := []string{"setup", "lint", "test", "build"}
	deps := map[string][]string{
		"lint":  {"setup"},
		"test":  {"setup"},
		"build": {"lint", "test"},
	}
	dag, err := BuildStageDAG(stages, deps)
	if err != nil {
		t.Fatal(err)
	}
	// lint and test should be at same level
	if dag.Nodes["lint"].Level != dag.Nodes["test"].Level {
		t.Errorf("lint (level %d) and test (level %d) should be at same level",
			dag.Nodes["lint"].Level, dag.Nodes["test"].Level)
	}
}

func TestBuildStageDAG_Cycle(t *testing.T) {
	stages := []string{"a", "b", "c"}
	deps := map[string][]string{
		"a": {"c"},
		"b": {"a"},
		"c": {"b"},
	}
	dag, err := BuildStageDAG(stages, deps)
	if err == nil {
		t.Error("should return error for cycle")
	}
	if dag != nil && !dag.HasCycle {
		t.Error("HasCycle should be true")
	}
}

func TestValidateStageDAG_SelfReference(t *testing.T) {
	stages := []string{"build"}
	deps := map[string][]string{"build": {"build"}}
	errs := ValidateStageDAG(stages, deps)
	if len(errs) == 0 {
		t.Error("should report self-reference")
	}
}

func TestValidateStageDAG_UndefinedDep(t *testing.T) {
	stages := []string{"build"}
	deps := map[string][]string{"build": {"nonexistent"}}
	errs := ValidateStageDAG(stages, deps)
	if len(errs) == 0 {
		t.Error("should report undefined dependency")
	}
}

func TestHasStageNeeds(t *testing.T) {
	if HasStageNeeds(nil) {
		t.Error("nil map should return false")
	}
	if HasStageNeeds(map[string][]string{}) {
		t.Error("empty map should return false")
	}
	if HasStageNeeds(map[string][]string{"a": {}}) {
		t.Error("map with empty deps should return false")
	}
	if !HasStageNeeds(map[string][]string{"a": {"b"}}) {
		t.Error("map with deps should return true")
	}
}
