package importer

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/oarkflow/deploy/backend/internal/detector"
	"github.com/oarkflow/deploy/backend/internal/integrations"
	"github.com/oarkflow/deploy/backend/internal/integrations/bitbucket"
	"github.com/oarkflow/deploy/backend/internal/integrations/github"
	"github.com/oarkflow/deploy/backend/internal/integrations/gitlab"
	"github.com/oarkflow/deploy/backend/internal/secrets"
)

// ImportRequest describes what to import and how.
type ImportRequest struct {
	SourceType  string // "git", "github", "gitlab", "bitbucket", "local", "upload"
	GitURL      string
	SSHKeyPEM   string
	Provider    string
	RepoOwner   string
	RepoName    string
	AccessToken string // ephemeral provider token
	LocalPath   string
	UploadPath  string // temp path from /import/upload
	Branch      string
}

// ImportResult holds the outcome of an import detection.
type ImportResult struct {
	WorkDir        string                    `json:"work_dir"`
	Detections     []detector.DetectionResult `json:"detections"`
	GeneratedYAML  string                    `json:"generated_pipeline"`
	DefaultBranch  string                    `json:"default_branch"`
	CloneURL       string                    `json:"clone_url"`
	SecretFindings []secrets.ScanFinding     `json:"secret_findings,omitempty"`
}

// Service orchestrates project import: resolve source -> detect -> generate pipeline.
type Service struct {
	encKey   []byte
	sessions *SessionStore
}

// New creates a new import service.
func New(encKey []byte) *Service {
	return &Service{
		encKey:   encKey,
		sessions: NewSessionStore(),
	}
}

// Sessions returns the session store for managing temp dirs.
func (s *Service) Sessions() *SessionStore {
	return s.sessions
}

// Import resolves the source to a local directory, runs detection, and generates a pipeline.
func (s *Service) Import(ctx context.Context, req ImportRequest) (*ImportResult, error) {
	var workDir string
	var cloneURL string
	var defaultBranch string
	var err error

	switch req.SourceType {
	case "git":
		workDir, err = os.MkdirTemp("", "flowforge-import-*")
		if err != nil {
			return nil, fmt.Errorf("create temp dir: %w", err)
		}
		err = CloneRepo(ctx, req.GitURL, req.Branch, workDir, CloneOptions{SSHKeyPEM: req.SSHKeyPEM})
		if err != nil {
			os.RemoveAll(workDir)
			return nil, fmt.Errorf("clone: %w", err)
		}
		cloneURL = req.GitURL
		defaultBranch = req.Branch
		if defaultBranch == "" {
			defaultBranch = "main"
		}

	case "github", "gitlab", "bitbucket":
		provider := s.newProvider(req.SourceType, req.AccessToken)
		if provider == nil {
			return nil, fmt.Errorf("unsupported provider: %s", req.SourceType)
		}
		info, err := provider.GetRepo(ctx, req.RepoOwner, req.RepoName)
		if err != nil {
			return nil, fmt.Errorf("get repo: %w", err)
		}
		cloneURL = info.CloneURL
		defaultBranch = info.DefaultBranch

		workDir, err = os.MkdirTemp("", "flowforge-import-*")
		if err != nil {
			return nil, fmt.Errorf("create temp dir: %w", err)
		}

		// Inject token into HTTPS clone URL for authenticated clone.
		authCloneURL := injectTokenInURL(cloneURL, req.AccessToken)
		branch := req.Branch
		if branch == "" {
			branch = defaultBranch
		}
		if err := CloneRepo(ctx, authCloneURL, branch, workDir, CloneOptions{}); err != nil {
			os.RemoveAll(workDir)
			return nil, fmt.Errorf("clone: %w", err)
		}

	case "local":
		if req.LocalPath == "" {
			return nil, fmt.Errorf("local path is required")
		}
		info, err := os.Stat(req.LocalPath)
		if err != nil || !info.IsDir() {
			return nil, fmt.Errorf("local path does not exist or is not a directory")
		}
		workDir = req.LocalPath
		cloneURL = req.LocalPath

	case "upload":
		if req.UploadPath == "" {
			return nil, fmt.Errorf("upload path is required")
		}
		// The upload handler already calls UnwrapSingleSubfolder, but apply
		// it here too in case the path was stored before the unwrap logic
		// was added (e.g. pre-existing sessions).
		workDir = UnwrapSingleSubfolder(req.UploadPath)
		cloneURL = ""

	default:
		return nil, fmt.Errorf("unsupported source type: %s", req.SourceType)
	}

	// Run detection.
	log.Printf("[importer] running detection on workDir=%s", workDir)
	detections, err := detector.Detect(workDir)
	if err != nil {
		return nil, fmt.Errorf("detect: %w", err)
	}
	log.Printf("[importer] detection found %d results", len(detections))

	// If the project already contains a flowforge.yml / .flowforge.yml, use
	// that as the pipeline config instead of auto-generating one.
	generatedYAML := ""
	for _, name := range []string{"flowforge.yml", ".flowforge.yml", "flowforge.yaml", ".flowforge.yaml"} {
		candidate := filepath.Join(workDir, name)
		if data, err := os.ReadFile(candidate); err == nil && len(data) > 0 {
			generatedYAML = string(data)
			log.Printf("[importer] found existing pipeline config: %s", name)
			break
		}
	}
	if generatedYAML == "" {
		generatedYAML = detector.GenerateStarterPipeline(detections)
	}

	// Run secret scanning on the imported repo.
	scanner := secrets.NewScanner()
	findings, _ := scanner.ScanDirectory(workDir)

	return &ImportResult{
		WorkDir:        workDir,
		Detections:     detections,
		GeneratedYAML:  generatedYAML,
		DefaultBranch:  defaultBranch,
		CloneURL:       cloneURL,
		SecretFindings: findings,
	}, nil
}

// ListProviderRepos lists repos for a given provider using the provided token.
func (s *Service) ListProviderRepos(ctx context.Context, provider, token string, opts integrations.ListReposOptions) ([]integrations.RepoInfo, int, error) {
	p := s.newProvider(provider, token)
	if p == nil {
		return nil, 0, fmt.Errorf("unsupported provider: %s", provider)
	}
	return p.ListRepos(ctx, opts)
}

// Cleanup removes a temporary work directory.
func (s *Service) Cleanup(workDir string) {
	if workDir != "" && strings.Contains(workDir, "flowforge-import-") {
		os.RemoveAll(workDir)
	}
}

func (s *Service) newProvider(providerType, token string) integrations.SCMProvider {
	switch providerType {
	case "github":
		return github.NewClient(token)
	case "gitlab":
		return gitlab.NewClient(token)
	case "bitbucket":
		return bitbucket.NewClient(token)
	default:
		return nil
	}
}

// injectTokenInURL adds an OAuth token to an HTTPS clone URL for authenticated cloning.
func injectTokenInURL(cloneURL, token string) string {
	if token == "" {
		return cloneURL
	}
	// https://github.com/owner/repo.git -> https://oauth2:TOKEN@github.com/owner/repo.git
	if strings.HasPrefix(cloneURL, "https://") {
		return "https://oauth2:" + token + "@" + strings.TrimPrefix(cloneURL, "https://")
	}
	return cloneURL
}
