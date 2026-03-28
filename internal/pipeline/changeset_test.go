package pipeline

import (
	"testing"
)

func TestMatchesPathFilters_NoFilters(t *testing.T) {
	cs := &ChangeSet{Files: []string{"src/main.go", "README.md"}}
	if !MatchesPathFilters(cs, "", "") {
		t.Error("empty filters should match everything")
	}
}

func TestMatchesPathFilters_NilChangeSet(t *testing.T) {
	if MatchesPathFilters(nil, "src/**", "") {
		t.Error("nil changeset should not match")
	}
}

func TestMatchesPathFilters_EmptyFiles(t *testing.T) {
	cs := &ChangeSet{Files: []string{}}
	if MatchesPathFilters(cs, "src/**", "") {
		t.Error("empty files list should not match")
	}
}

func TestMatchesPathFilters_SimpleMatch(t *testing.T) {
	cs := &ChangeSet{Files: []string{"src/main.go"}}
	if !MatchesPathFilters(cs, "src/**", "") {
		t.Error("src/main.go should match src/**")
	}
}

func TestMatchesPathFilters_NoMatch(t *testing.T) {
	cs := &ChangeSet{Files: []string{"docs/README.md"}}
	if MatchesPathFilters(cs, "src/**", "") {
		t.Error("docs/README.md should not match src/**")
	}
}

func TestMatchesPathFilters_MultiplePatterns(t *testing.T) {
	cs := &ChangeSet{Files: []string{"shared/util.go"}}
	if !MatchesPathFilters(cs, "src/**,shared/**", "") {
		t.Error("shared/util.go should match shared/**")
	}
}

func TestMatchesPathFilters_IgnorePaths(t *testing.T) {
	cs := &ChangeSet{Files: []string{"docs/guide.md"}}
	// Matches src/** but the file is in docs — should not match include
	if MatchesPathFilters(cs, "src/**", "docs/**") {
		t.Error("docs file should not match src/** filter")
	}
}

func TestMatchesPathFilters_ExcludeOverridesInclude(t *testing.T) {
	cs := &ChangeSet{Files: []string{"src/README.md"}}
	// Include src/**, but exclude *.md
	if MatchesPathFilters(cs, "src/**", "**/*.md") {
		t.Error("*.md should be excluded even though under src/")
	}
}

func TestMatchesPathFilters_MixedFilesPartialMatch(t *testing.T) {
	cs := &ChangeSet{Files: []string{"docs/notes.txt", "src/handler.go"}}
	if !MatchesPathFilters(cs, "src/**", "") {
		t.Error("should match because src/handler.go is in the changeset")
	}
}

func TestMatchPathGlob_DoubleStarPrefix(t *testing.T) {
	tests := []struct {
		file, pattern string
		want          bool
	}{
		{"src/main.go", "src/**", true},
		{"src/pkg/handler.go", "src/**", true},
		{"README.md", "**/*.md", true},
		{"docs/README.md", "**/*.md", true},
		{"src/main.go", "docs/**", false},
	}
	for _, tt := range tests {
		got := matchPathGlob(tt.file, tt.pattern)
		if got != tt.want {
			t.Errorf("matchPathGlob(%q, %q) = %v, want %v", tt.file, tt.pattern, got, tt.want)
		}
	}
}

func TestMatchDoubleStarGlob_Standalone(t *testing.T) {
	// "**" alone matches everything
	if !matchDoubleStarGlob("any/file/path.go", "**") {
		t.Error("** should match any file path")
	}
}

func TestParsePatterns(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"src/**", 1},
		{"src/**,shared/**", 2},
		{"  src/**  ,  shared/** ,  ", 2},
		{" , , ", 0},
	}
	for _, tt := range tests {
		got := parsePatterns(tt.input)
		if len(got) != tt.want {
			t.Errorf("parsePatterns(%q) = %d patterns, want %d", tt.input, len(got), tt.want)
		}
	}
}

func TestPathSegments(t *testing.T) {
	segs := pathSegments("src/pkg/handler.go")
	if len(segs) != 3 {
		t.Errorf("pathSegments should have 3 parts, got %d: %v", len(segs), segs)
	}
	if segs[0] != "src" || segs[1] != "pkg" || segs[2] != "handler.go" {
		t.Errorf("segments = %v, want [src pkg handler.go]", segs)
	}
}
