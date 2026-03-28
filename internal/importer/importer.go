package importer

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

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
	WorkDir        string                     `json:"work_dir"`
	Detections     []detector.DetectionResult `json:"detections"`
	Profile        detector.ProjectProfile    `json:"profile"`
	Repository     RepositoryMetadata         `json:"repository"`
	GeneratedYAML  string                     `json:"generated_pipeline"`
	DefaultBranch  string                     `json:"default_branch"`
	CloneURL       string                     `json:"clone_url"`
	SecretFindings []secrets.ScanFinding      `json:"secret_findings,omitempty"`
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
	repoMeta := RepositoryMetadata{Provider: "git"}
	var err error

	switch req.SourceType {
	case "git":
		workDir, err = os.MkdirTemp("", "flowforge-import-*")
		if err != nil {
			return nil, fmt.Errorf("create temp dir: %w", err)
		}
		repoMeta = normalizeRepositoryMetadata(req.GitURL, req.Branch, "")
		authCloneURL := req.GitURL
		if req.AccessToken != "" {
			authCloneURL = injectTokenIntoCloneURL(req.GitURL, req.AccessToken, repoMeta.Provider)
		}
		err = CloneRepo(ctx, authCloneURL, req.Branch, workDir, CloneOptions{SSHKeyPEM: req.SSHKeyPEM})
		if err != nil {
			os.RemoveAll(workDir)
			return nil, fmt.Errorf("clone: %w", err)
		}
		cloneURL = req.GitURL
		defaultBranch = req.Branch
		if defaultBranch == "" {
			defaultBranch = "main"
		}
		repoMeta.CloneURL = cloneURL
		repoMeta.DefaultBranch = defaultBranch

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
		repoMeta = RepositoryMetadata{
			Provider:      req.SourceType,
			FullName:      info.FullName,
			CloneURL:      info.CloneURL,
			SSHURL:        info.SSHURL,
			DefaultBranch: info.DefaultBranch,
		}

		workDir, err = os.MkdirTemp("", "flowforge-import-*")
		if err != nil {
			return nil, fmt.Errorf("create temp dir: %w", err)
		}

		// Inject token into HTTPS clone URL for authenticated clone.
		authCloneURL := injectTokenIntoCloneURL(cloneURL, req.AccessToken, req.SourceType)
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
		repoMeta = RepositoryMetadata{
			Provider:      "local",
			FullName:      filepath.Base(req.LocalPath),
			CloneURL:      req.LocalPath,
			DefaultBranch: req.Branch,
		}

	case "upload":
		if req.UploadPath == "" {
			return nil, fmt.Errorf("upload path is required")
		}
		// The upload handler already calls UnwrapSingleSubfolder, but apply
		// it here too in case the path was stored before the unwrap logic
		// was added (e.g. pre-existing sessions).
		workDir = UnwrapSingleSubfolder(req.UploadPath)
		cloneURL = ""
		repoMeta = RepositoryMetadata{
			Provider: "upload",
			FullName: filepath.Base(workDir),
		}

	default:
		return nil, fmt.Errorf("unsupported source type: %s", req.SourceType)
	}

	// Run detection.
	log.Printf("[importer] running detection on workDir=%s", workDir)
	inspection, err := detector.Inspect(workDir)
	if err != nil {
		return nil, fmt.Errorf("detect: %w", err)
	}
	detections := inspection.Detections
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
		generatedYAML = detector.GenerateStarterPipelineForProfile(detections, &inspection.Profile)
	}

	// Run secret scanning on the imported repo.
	scanner := secrets.NewScanner()
	findings, _ := scanner.ScanDirectory(workDir)

	return &ImportResult{
		WorkDir:        workDir,
		Detections:     detections,
		Profile:        inspection.Profile,
		Repository:     repoMeta,
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
