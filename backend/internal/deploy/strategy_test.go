package deploy

import (
	"testing"
)

func TestNewStrategy_Recreate(t *testing.T) {
	s := NewStrategy(StrategyRecreate)
	if s == nil {
		t.Fatal("strategy should not be nil")
	}
	if s.Type() != StrategyRecreate {
		t.Errorf("Type() = %q, want %q", s.Type(), StrategyRecreate)
	}
}

func TestNewStrategy_Rolling(t *testing.T) {
	s := NewStrategy(StrategyRolling)
	if s == nil {
		t.Fatal("strategy should not be nil")
	}
	if s.Type() != StrategyRolling {
		t.Errorf("Type() = %q, want %q", s.Type(), StrategyRolling)
	}
}

func TestNewStrategy_BlueGreen(t *testing.T) {
	s := NewStrategy(StrategyBlueGreen)
	if s == nil {
		t.Fatal("strategy should not be nil")
	}
	if s.Type() != StrategyBlueGreen {
		t.Errorf("Type() = %q, want %q", s.Type(), StrategyBlueGreen)
	}
}

func TestNewStrategy_Canary(t *testing.T) {
	s := NewStrategy(StrategyCanary)
	if s == nil {
		t.Fatal("strategy should not be nil")
	}
	if s.Type() != StrategyCanary {
		t.Errorf("Type() = %q, want %q", s.Type(), StrategyCanary)
	}
}

func TestNewStrategy_Unknown_DefaultsToRecreate(t *testing.T) {
	s := NewStrategy("unknown_strategy")
	if s == nil {
		t.Fatal("strategy should not be nil")
	}
	if s.Type() != StrategyRecreate {
		t.Errorf("Type() = %q, want %q (default)", s.Type(), StrategyRecreate)
	}
}

func TestNewStrategy_EmptyString_DefaultsToRecreate(t *testing.T) {
	s := NewStrategy("")
	if s.Type() != StrategyRecreate {
		t.Errorf("Type() = %q, want %q (default)", s.Type(), StrategyRecreate)
	}
}

func TestStrategyTypeConstants(t *testing.T) {
	tests := []struct {
		strategy StrategyType
		want     string
	}{
		{StrategyRecreate, "recreate"},
		{StrategyRolling, "rolling"},
		{StrategyBlueGreen, "blue_green"},
		{StrategyCanary, "canary"},
	}
	for _, tt := range tests {
		if string(tt.strategy) != tt.want {
			t.Errorf("StrategyType %q != %q", tt.strategy, tt.want)
		}
	}
}

func TestDeploymentPlan_Fields(t *testing.T) {
	plan := DeploymentPlan{
		DeploymentID:      "dep-1",
		EnvironmentID:     "env-1",
		StrategyType:      StrategyRolling,
		TotalSteps:        5,
		EstimatedDuration: 300,
		Steps: []DeployStep{
			{Name: "step1", Description: "First step", Order: 1},
			{Name: "step2", Description: "Second step", Order: 2},
		},
	}
	if plan.DeploymentID != "dep-1" {
		t.Errorf("DeploymentID = %q", plan.DeploymentID)
	}
	if len(plan.Steps) != 2 {
		t.Errorf("Steps count = %d, want 2", len(plan.Steps))
	}
}

func TestHealthResult_Fields(t *testing.T) {
	result := HealthResult{
		Healthy:    true,
		StatusCode: 200,
		Latency:    50,
	}
	if !result.Healthy {
		t.Error("should be healthy")
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d", result.StatusCode)
	}
}
