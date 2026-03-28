package pipeline

import (
	"testing"
)

func TestMatchTrigger_ManualAlwaysMatches(t *testing.T) {
	spec := &PipelineSpec{}
	if !MatchTrigger(spec, "manual", nil) {
		t.Error("manual trigger should always match")
	}
}

func TestMatchTrigger_APIAlwaysMatches(t *testing.T) {
	spec := &PipelineSpec{}
	if !MatchTrigger(spec, "api", nil) {
		t.Error("API trigger should always match")
	}
}

func TestMatchTrigger_NoConfig_MatchesAll(t *testing.T) {
	spec := &PipelineSpec{}
	if !MatchTrigger(spec, "push", map[string]string{"branch": "main"}) {
		t.Error("no trigger config should match all events")
	}
}

func TestMatchTrigger_PushBranch(t *testing.T) {
	spec := &PipelineSpec{
		On: TriggerConfig{
			Push: &PushTrigger{Branches: []string{"main", "develop"}},
		},
	}
	tests := []struct {
		branch string
		want   bool
	}{
		{"main", true},
		{"develop", true},
		{"feature/test", false},
	}
	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			got := MatchTrigger(spec, "push", map[string]string{"branch": tt.branch})
			if got != tt.want {
				t.Errorf("MatchTrigger(branch=%q) = %v, want %v", tt.branch, got, tt.want)
			}
		})
	}
}

func TestMatchTrigger_PushWildcard(t *testing.T) {
	spec := &PipelineSpec{
		On: TriggerConfig{
			Push: &PushTrigger{Branches: []string{"feature/**"}},
		},
	}
	tests := []struct {
		branch string
		want   bool
	}{
		{"feature/test", true},
		{"feature/deep/nested", true},
		{"main", false},
	}
	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			got := MatchTrigger(spec, "push", map[string]string{"branch": tt.branch})
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchTrigger_PushPaths(t *testing.T) {
	spec := &PipelineSpec{
		On: TriggerConfig{
			Push: &PushTrigger{
				Branches: []string{"main"},
				Paths:    []string{"src/**"},
			},
		},
	}
	tests := []struct {
		name  string
		paths string
		want  bool
	}{
		{"match", "src/main.go,src/utils.go", true},
		{"no match", "docs/README.md", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchTrigger(spec, "push", map[string]string{
				"branch": "main",
				"paths":  tt.paths,
			})
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchTrigger_PushIgnorePaths(t *testing.T) {
	spec := &PipelineSpec{
		On: TriggerConfig{
			Push: &PushTrigger{
				Branches:    []string{"main"},
				IgnorePaths: []string{"docs/**", "*.md"},
			},
		},
	}
	// All changed files match ignore paths → should not trigger
	got := MatchTrigger(spec, "push", map[string]string{
		"branch": "main",
		"paths":  "docs/guide.md",
	})
	if got {
		t.Error("should not trigger when all paths are ignored")
	}
}

func TestMatchTrigger_PushTag(t *testing.T) {
	spec := &PipelineSpec{
		On: TriggerConfig{
			Push: &PushTrigger{Tags: []string{"v*"}},
		},
	}
	tests := []struct {
		tag  string
		want bool
	}{
		{"v1.0.0", true},
		{"release", false},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got := MatchTrigger(spec, "push", map[string]string{"tag": tt.tag})
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchTrigger_PullRequest(t *testing.T) {
	spec := &PipelineSpec{
		On: TriggerConfig{
			PullRequest: &PullRequestTrigger{
				Types:    []string{"opened", "synchronize"},
				Branches: []string{"main"},
			},
		},
	}
	tests := []struct {
		name       string
		action     string
		baseBranch string
		want       bool
	}{
		{"opened to main", "opened", "main", true},
		{"sync to main", "synchronize", "main", true},
		{"closed to main", "closed", "main", false},
		{"opened to develop", "opened", "develop", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchTrigger(spec, "pull_request", map[string]string{
				"action":      tt.action,
				"base_branch": tt.baseBranch,
			})
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchTrigger_Schedule(t *testing.T) {
	spec := &PipelineSpec{
		On: TriggerConfig{
			Schedule: []ScheduleTrigger{{Cron: "0 2 * * *"}},
		},
	}
	if !MatchTrigger(spec, "schedule", nil) {
		t.Error("schedule trigger should match when schedule config exists")
	}
}

func TestMatchTrigger_NoPushConfig(t *testing.T) {
	spec := &PipelineSpec{
		On: TriggerConfig{
			PullRequest: &PullRequestTrigger{},
		},
	}
	if MatchTrigger(spec, "push", map[string]string{"branch": "main"}) {
		t.Error("push should not match when only PR trigger is configured")
	}
}

func TestMatchTrigger_CreateTag(t *testing.T) {
	spec := &PipelineSpec{
		On: TriggerConfig{
			Push: &PushTrigger{Tags: []string{"v*"}},
		},
	}
	got := MatchTrigger(spec, "create", map[string]string{
		"ref_type": "tag",
		"tag":      "v2.0",
	})
	if !got {
		t.Error("create tag event should match tag triggers")
	}
}

func TestMatchTrigger_UnknownEvent(t *testing.T) {
	spec := &PipelineSpec{
		On: TriggerConfig{Push: &PushTrigger{Branches: []string{"main"}}},
	}
	if MatchTrigger(spec, "unknown", nil) {
		t.Error("unknown event type should not match")
	}
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern, value string
		want           bool
	}{
		{"main", "main", true},
		{"main", "develop", false},
		{"*", "anything", true},
		{"**", "anything/deep", true},
		{"feature/*", "feature/foo", true},
		{"feature/**", "feature/foo/bar", true},
		{"feature/**", "main", false},
	}
	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.value, func(t *testing.T) {
			got := matchGlob(tt.pattern, tt.value)
			if got != tt.want {
				t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.want)
			}
		})
	}
}
