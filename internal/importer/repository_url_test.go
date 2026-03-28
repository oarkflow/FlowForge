package importer

import "testing"

func TestParseRepositoryURL(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		provider string
		fullName string
	}{
		{
			name:     "github https",
			raw:      "https://github.com/oarkflow/flowforge.git",
			provider: "github",
			fullName: "oarkflow/flowforge",
		},
		{
			name:     "gitlab ssh namespace",
			raw:      "git@gitlab.com:group/platform/flowforge.git",
			provider: "gitlab",
			fullName: "group/platform/flowforge",
		},
		{
			name:     "bitbucket https",
			raw:      "https://bitbucket.org/workspace/flowforge.git",
			provider: "bitbucket",
			fullName: "workspace/flowforge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := parseRepositoryURL(tt.raw)
			if err != nil {
				t.Fatalf("parseRepositoryURL() error = %v", err)
			}
			if ref.Provider != tt.provider {
				t.Fatalf("provider = %q, want %q", ref.Provider, tt.provider)
			}
			if ref.FullName != tt.fullName {
				t.Fatalf("fullName = %q, want %q", ref.FullName, tt.fullName)
			}
		})
	}
}

func TestInjectTokenIntoCloneURL(t *testing.T) {
	got := injectTokenIntoCloneURL("https://github.com/oarkflow/flowforge.git", "secret", "github")
	want := "https://x-access-token:secret@github.com/oarkflow/flowforge.git"
	if got != want {
		t.Fatalf("injectTokenIntoCloneURL() = %q, want %q", got, want)
	}

	got = injectTokenIntoCloneURL("https://gitlab.com/group/project.git", "token", "gitlab")
	want = "https://oauth2:token@gitlab.com/group/project.git"
	if got != want {
		t.Fatalf("injectTokenIntoCloneURL() = %q, want %q", got, want)
	}
}

func TestNormalizeRepositoryMetadata(t *testing.T) {
	meta := normalizeRepositoryMetadata("https://github.com/oarkflow/flowforge.git", "main", "")
	if meta.Provider != "github" {
		t.Fatalf("provider = %q, want github", meta.Provider)
	}
	if meta.FullName != "oarkflow/flowforge" {
		t.Fatalf("fullName = %q, want oarkflow/flowforge", meta.FullName)
	}
	if meta.DefaultBranch != "main" {
		t.Fatalf("defaultBranch = %q, want main", meta.DefaultBranch)
	}
}
