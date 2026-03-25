package pipeline

import (
	"path"
	"strings"
)

// MatchTrigger checks whether the given event matches the pipeline's trigger
// configuration. Returns true if the pipeline should be triggered for this event.
//
// eventType is one of: "push", "pull_request", "create", "release", "schedule", "manual", "api".
// eventData contains contextual information about the event:
//   - "branch": the branch name (for push/PR events)
//   - "tag": the tag name (for tag/release events)
//   - "action": the PR action (opened, synchronize, reopened, closed)
//   - "base_branch": the target branch for PR events
//   - "paths": comma-separated changed file paths (when available)
//   - "ref_type": "tag" or "branch" (for create events)
func MatchTrigger(spec *PipelineSpec, eventType string, eventData map[string]string) bool {
	// Manual and API triggers always match — they're explicitly invoked.
	if eventType == "manual" || eventType == "api" {
		return true
	}

	// If no trigger config is defined at all, match everything for backwards compatibility.
	if spec.On.Push == nil && spec.On.PullRequest == nil && len(spec.On.Schedule) == 0 && spec.On.Manual == nil {
		return true
	}

	switch eventType {
	case "push":
		return matchPushTrigger(spec.On.Push, eventData)

	case "pull_request":
		return matchPullRequestTrigger(spec.On.PullRequest, eventData)

	case "create":
		// Tag creation events match against push tag triggers.
		if eventData["ref_type"] == "tag" {
			return matchTagTrigger(spec.On.Push, eventData)
		}
		// Branch creation — treat like a push event.
		return matchPushTrigger(spec.On.Push, eventData)

	case "release":
		// Releases are treated as tag events.
		return matchTagTrigger(spec.On.Push, eventData)

	case "schedule":
		// Schedule triggers are matched by the scheduler, not by webhook handlers.
		return len(spec.On.Schedule) > 0

	default:
		return false
	}
}

// matchPushTrigger checks if a push event matches the push trigger config.
func matchPushTrigger(trigger *PushTrigger, eventData map[string]string) bool {
	if trigger == nil {
		return false
	}

	branch := eventData["branch"]
	tag := eventData["tag"]

	// If the push is a tag push, delegate to tag matching.
	if tag != "" {
		return matchPatterns(trigger.Tags, tag)
	}

	// Branch matching
	if len(trigger.Branches) > 0 {
		if !matchPatterns(trigger.Branches, branch) {
			return false
		}
	}

	// Path matching (if paths are provided by the webhook)
	changedPaths := eventData["paths"]
	if changedPaths != "" && len(trigger.Paths) > 0 {
		paths := strings.Split(changedPaths, ",")
		if !matchAnyPath(trigger.Paths, paths) {
			return false
		}
	}

	// Ignore paths
	if changedPaths != "" && len(trigger.IgnorePaths) > 0 {
		paths := strings.Split(changedPaths, ",")
		if matchAllPaths(trigger.IgnorePaths, paths) {
			return false // All changed files are in ignore paths
		}
	}

	return true
}

// matchTagTrigger checks if a tag event matches the push trigger's tag patterns.
func matchTagTrigger(trigger *PushTrigger, eventData map[string]string) bool {
	if trigger == nil {
		return false
	}

	tag := eventData["tag"]
	if tag == "" {
		tag = eventData["tag_name"]
	}

	if len(trigger.Tags) == 0 {
		// No tag patterns specified — don't match tag events.
		return false
	}

	return matchPatterns(trigger.Tags, tag)
}

// matchPullRequestTrigger checks if a PR event matches the PR trigger config.
func matchPullRequestTrigger(trigger *PullRequestTrigger, eventData map[string]string) bool {
	if trigger == nil {
		return false
	}

	// Action/type matching (opened, synchronize, reopened, closed)
	action := eventData["action"]
	if len(trigger.Types) > 0 && action != "" {
		matched := false
		for _, t := range trigger.Types {
			if strings.EqualFold(t, action) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Target branch matching
	baseBranch := eventData["base_branch"]
	if len(trigger.Branches) > 0 && baseBranch != "" {
		if !matchPatterns(trigger.Branches, baseBranch) {
			return false
		}
	}

	return true
}

// matchPatterns checks if value matches any of the glob patterns.
// Supports standard glob patterns plus ** for recursive matching.
func matchPatterns(patterns []string, value string) bool {
	for _, pattern := range patterns {
		if matchGlob(pattern, value) {
			return true
		}
	}
	return false
}

// matchAnyPath checks if any of the changed paths match any of the filter patterns.
func matchAnyPath(patterns []string, paths []string) bool {
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		for _, pattern := range patterns {
			if matchGlob(pattern, p) {
				return true
			}
		}
	}
	return false
}

// matchAllPaths checks if ALL changed paths match at least one ignore pattern.
func matchAllPaths(patterns []string, paths []string) bool {
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		matched := false
		for _, pattern := range patterns {
			if matchGlob(pattern, p) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// matchGlob matches a value against a glob pattern.
//
// Supports:
//   - Exact match: "main" matches "main"
//   - Single wildcard: "feature/*" matches "feature/foo" but not "feature/foo/bar"
//   - Double wildcard: "feature/**" matches "feature/foo" and "feature/foo/bar"
//   - Star: "*" matches anything
//   - Standard path.Match patterns
func matchGlob(pattern, value string) bool {
	// Exact match fast path
	if pattern == value {
		return true
	}

	// Universal wildcard
	if pattern == "*" || pattern == "**" {
		return true
	}

	// Handle ** (double-star / recursive match)
	if strings.Contains(pattern, "**") {
		// "feature/**" should match "feature/foo" and "feature/foo/bar/baz"
		// Split on ** and check prefix + any suffix
		parts := strings.SplitN(pattern, "**", 2)
		prefix := parts[0]
		suffix := ""
		if len(parts) > 1 {
			suffix = parts[1]
		}

		// Value must start with prefix
		if prefix != "" && !strings.HasPrefix(value, prefix) {
			return false
		}

		// Value must end with suffix (if any)
		if suffix != "" && !strings.HasSuffix(value, suffix) {
			return false
		}

		return true
	}

	// Try standard path.Match (handles *, ?, [])
	matched, err := path.Match(pattern, value)
	if err == nil && matched {
		return true
	}

	return false
}
