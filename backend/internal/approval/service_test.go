package approval

import (
	"testing"

	"github.com/oarkflow/deploy/backend/internal/models"
)

func TestCheckRequired_EmptyRules(t *testing.T) {
	s := &Service{}

	tests := []struct {
		name  string
		rules string
		want  bool
	}{
		{"empty string", "", false},
		{"empty object", "{}", false},
		{"requires approval", `{"require_approval":true,"min_approvals":2}`, true},
		{"no approval required", `{"require_approval":false}`, false},
		{"invalid json", "not json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &models.Environment{ProtectionRules: tt.rules}
			got, rules := s.CheckRequired(env)
			if got != tt.want {
				t.Errorf("CheckRequired() = %v, want %v", got, tt.want)
			}
			if got && rules != nil {
				if rules.RequireApproval != true {
					t.Error("rules.RequireApproval should be true")
				}
			}
		})
	}
}

func TestCheckRequired_WithMinApprovals(t *testing.T) {
	s := &Service{}
	env := &models.Environment{
		ProtectionRules: `{"require_approval":true,"min_approvals":3}`,
	}
	required, rules := s.CheckRequired(env)
	if !required {
		t.Error("should require approval")
	}
	if rules.MinApprovals != 3 {
		t.Errorf("MinApprovals = %d, want 3", rules.MinApprovals)
	}
}

func TestGetApprovers_Valid(t *testing.T) {
	s := &Service{}
	env := &models.Environment{
		RequiredApprovers: `["user-1","user-2","user-3"]`,
	}
	approvers := s.GetApprovers(env)
	if len(approvers) != 3 {
		t.Fatalf("approvers = %d, want 3", len(approvers))
	}
	if approvers[0] != "user-1" {
		t.Errorf("approvers[0] = %q, want %q", approvers[0], "user-1")
	}
}

func TestGetApprovers_Empty(t *testing.T) {
	s := &Service{}
	tests := []struct {
		name  string
		value string
	}{
		{"empty string", ""},
		{"empty array", "[]"},
		{"invalid json", "not json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &models.Environment{RequiredApprovers: tt.value}
			approvers := s.GetApprovers(env)
			if len(approvers) != 0 {
				t.Errorf("approvers = %d, want 0", len(approvers))
			}
		})
	}
}

func TestProtectionRules_ZeroValue(t *testing.T) {
	rules := ProtectionRules{}
	if rules.RequireApproval {
		t.Error("zero value should not require approval")
	}
	if rules.MinApprovals != 0 {
		t.Error("zero value min approvals should be 0")
	}
}
