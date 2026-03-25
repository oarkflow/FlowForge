package pipeline

import (
	"path/filepath"
	"strings"
)

// ChangeSet represents files changed in a commit/push event.
type ChangeSet struct {
	Files []string // list of changed file paths (relative to repo root)
}

// MatchesPathFilters checks if any changed file matches the pipeline's path filters.
// pathFilters: comma-separated glob patterns like "services/api/**,shared/**"
// ignorePaths: comma-separated glob patterns to exclude like "docs/**,*.md"
// Returns true if any file matches pathFilters AND doesn't match ignorePaths.
// If pathFilters is empty, all files match (no filtering — backward compatible).
func MatchesPathFilters(changeSet *ChangeSet, pathFilters string, ignorePaths string) bool {
	if changeSet == nil || len(changeSet.Files) == 0 {
		return false
	}

	// If no path filters are configured, match everything
	includePatterns := parsePatterns(pathFilters)
	if len(includePatterns) == 0 {
		return true
	}

	excludePatterns := parsePatterns(ignorePaths)

	for _, file := range changeSet.Files {
		file = filepath.Clean(file)

		// Check if file matches any exclude pattern → skip it
		if matchesAny(file, excludePatterns) {
			continue
		}

		// Check if file matches any include pattern → it's a match
		if matchesAny(file, includePatterns) {
			return true
		}
	}

	return false
}

// parsePatterns splits a comma-separated pattern string into individual patterns,
// trimming whitespace and ignoring empty entries.
func parsePatterns(s string) []string {
	if s == "" {
		return nil
	}
	raw := strings.Split(s, ",")
	var patterns []string
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			patterns = append(patterns, p)
		}
	}
	return patterns
}

// matchesAny returns true if the file path matches any of the given glob patterns.
// Supports standard glob patterns plus "**" for recursive directory matching.
func matchesAny(file string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchPathGlob(file, pattern) {
			return true
		}
	}
	return false
}

// matchPathGlob matches a file path against a glob pattern with "**" support.
// "**" matches zero or more path segments (directories).
// Examples:
//   - "services/api/**" matches "services/api/main.go" and "services/api/pkg/handler.go"
//   - "*.md" matches "README.md" but not "docs/README.md"
//   - "**/*.md" matches "README.md" and "docs/README.md"
func matchPathGlob(file string, pattern string) bool {
	// Handle "**" by splitting pattern on "**" and matching segments
	if strings.Contains(pattern, "**") {
		return matchDoubleStarGlob(file, pattern)
	}

	// Use standard filepath.Match for simple globs
	matched, err := filepath.Match(pattern, file)
	if err != nil {
		return false
	}
	return matched
}

// matchDoubleStarGlob handles patterns containing "**".
func matchDoubleStarGlob(file string, pattern string) bool {
	// Split on "**"
	parts := strings.SplitN(pattern, "**", 2)
	prefix := parts[0]
	suffix := ""
	if len(parts) > 1 {
		suffix = parts[1]
	}

	// Remove trailing/leading path separators
	prefix = strings.TrimRight(prefix, "/")
	suffix = strings.TrimLeft(suffix, "/")

	// If prefix is empty, "**" is at the start — match any path
	if prefix == "" {
		if suffix == "" {
			return true // "**" alone matches everything
		}
		// "**/<suffix>" — check every path segment suffix
		segments := pathSegments(file)
		for i := range segments {
			// Try matching suffix against the remainder of the path
			remainder := strings.Join(segments[i:], "/")
			matched, _ := filepath.Match(suffix, remainder)
			if matched {
				return true
			}
			// Also try matching just the filename against the suffix pattern
			if i == len(segments)-1 {
				matched, _ = filepath.Match(suffix, segments[i])
				if matched {
					return true
				}
			}
		}
		return false
	}

	// "<prefix>/**" or "<prefix>/**/<suffix>"
	// Check if file starts under the prefix directory
	if !strings.HasPrefix(file, prefix+"/") && file != prefix {
		return false
	}

	if suffix == "" {
		// "<prefix>/**" matches anything under prefix
		return true
	}

	// "<prefix>/**/<suffix>" — match suffix against the part after prefix
	remainder := strings.TrimPrefix(file, prefix+"/")
	// Try matching the suffix against the end of the path
	matched, _ := filepath.Match(suffix, remainder)
	if matched {
		return true
	}
	// Try matching just the filename
	base := filepath.Base(file)
	matched, _ = filepath.Match(suffix, base)
	return matched
}

// pathSegments splits a file path into its directory/file components.
func pathSegments(path string) []string {
	return strings.Split(filepath.Clean(path), "/")
}
