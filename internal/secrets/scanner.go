package secrets

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ScanFinding represents a single instance of a potential secret detected in
// a file during repository import.
type ScanFinding struct {
	File       string  `json:"file"`
	Line       int     `json:"line"`
	RuleID     string  `json:"rule_id"`
	RuleName   string  `json:"rule_name"`
	Match      string  `json:"match"` // redacted excerpt
	Confidence float64 `json:"confidence"`
}

// scanRule is a compiled detection rule.
type scanRule struct {
	ID         string
	Name       string
	Pattern    *regexp.Regexp
	Confidence float64
}

// defaultRules contains patterns for common secret types.
var defaultRules = []scanRule{
	{
		ID:         "aws-access-key",
		Name:       "AWS Access Key ID",
		Pattern:    regexp.MustCompile(`(?:^|[^A-Z0-9])AKIA[0-9A-Z]{16}(?:[^A-Z0-9]|$)`),
		Confidence: 0.95,
	},
	{
		ID:         "aws-secret-key",
		Name:       "AWS Secret Access Key",
		Pattern:    regexp.MustCompile(`(?i)(?:aws_?secret_?access_?key|secret_?key)\s*[:=]\s*[A-Za-z0-9/+=]{40}`),
		Confidence: 0.90,
	},
	{
		ID:         "github-pat",
		Name:       "GitHub Personal Access Token",
		Pattern:    regexp.MustCompile(`ghp_[A-Za-z0-9]{36}`),
		Confidence: 0.95,
	},
	{
		ID:         "github-fine-grained-pat",
		Name:       "GitHub Fine-Grained PAT",
		Pattern:    regexp.MustCompile(`github_pat_[A-Za-z0-9_]{82}`),
		Confidence: 0.95,
	},
	{
		ID:         "gitlab-pat",
		Name:       "GitLab Personal Access Token",
		Pattern:    regexp.MustCompile(`glpat-[A-Za-z0-9\-]{20,}`),
		Confidence: 0.95,
	},
	{
		ID:         "slack-token",
		Name:       "Slack Token",
		Pattern:    regexp.MustCompile(`xox[bpors]-[A-Za-z0-9\-]{10,}`),
		Confidence: 0.90,
	},
	{
		ID:         "private-key",
		Name:       "Private Key (PEM)",
		Pattern:    regexp.MustCompile(`-----BEGIN\s+(RSA\s+|EC\s+|DSA\s+|OPENSSH\s+)?PRIVATE\s+KEY-----`),
		Confidence: 0.99,
	},
	{
		ID:         "generic-api-key",
		Name:       "Generic API Key Assignment",
		Pattern:    regexp.MustCompile(`(?i)(?:api_?key|apikey|api_?secret)\s*[:=]\s*['"]?[A-Za-z0-9\-._~]{16,}['"]?`),
		Confidence: 0.60,
	},
	{
		ID:         "generic-password",
		Name:       "Password Assignment",
		Pattern:    regexp.MustCompile(`(?i)(?:password|passwd|pwd)\s*[:=]\s*['"]?[^\s'"]{8,}['"]?`),
		Confidence: 0.55,
	},
	{
		ID:         "generic-secret",
		Name:       "Secret Assignment",
		Pattern:    regexp.MustCompile(`(?i)(?:secret|token)\s*[:=]\s*['"]?[A-Za-z0-9\-._~+/]{16,}['"]?`),
		Confidence: 0.55,
	},
	{
		ID:         "heroku-api-key",
		Name:       "Heroku API Key",
		Pattern:    regexp.MustCompile(`(?i)heroku[_-]?api[_-]?key\s*[:=]\s*[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`),
		Confidence: 0.90,
	},
	{
		ID:         "stripe-key",
		Name:       "Stripe API Key",
		Pattern:    regexp.MustCompile(`(?:sk|pk)_(?:test|live)_[A-Za-z0-9]{20,}`),
		Confidence: 0.90,
	},
	{
		ID:         "gcp-service-account",
		Name:       "GCP Service Account Key",
		Pattern:    regexp.MustCompile(`"type"\s*:\s*"service_account"`),
		Confidence: 0.85,
	},
	{
		ID:         "jwt-token",
		Name:       "JSON Web Token",
		Pattern:    regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`),
		Confidence: 0.70,
	},
	{
		ID:         "sendgrid-key",
		Name:       "SendGrid API Key",
		Pattern:    regexp.MustCompile(`SG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}`),
		Confidence: 0.95,
	},
	{
		ID:         "twilio-key",
		Name:       "Twilio API Key",
		Pattern:    regexp.MustCompile(`SK[0-9a-f]{32}`),
		Confidence: 0.70,
	},
}

// skipExtensions is the set of file extensions to ignore during scanning.
var skipExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".ico": true, ".svg": true, ".woff": true, ".woff2": true,
	".ttf": true, ".eot": true, ".mp3": true, ".mp4": true,
	".zip": true, ".tar": true, ".gz": true, ".bz2": true,
	".rar": true, ".7z": true, ".exe": true, ".dll": true,
	".so": true, ".dylib": true, ".o": true, ".a": true,
	".class": true, ".jar": true, ".war": true, ".pyc": true,
	".wasm": true, ".pdf": true,
}

// skipDirs is the set of directory names to skip.
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	".terraform": true, "__pycache__": true, ".venv": true,
	"dist": true, "build": true, ".next": true,
}

// maxFileSize is the maximum file size (in bytes) we will scan. Files larger
// than this are skipped to avoid excessive memory usage.
const maxFileSize = 1 * 1024 * 1024 // 1 MB

// Scanner searches a directory tree for common secret patterns.
type Scanner struct {
	rules          []scanRule
	minConfidence  float64
	maxFindings    int
}

// NewScanner creates a Scanner with the default rule set.
func NewScanner() *Scanner {
	return &Scanner{
		rules:         defaultRules,
		minConfidence: 0.50,
		maxFindings:   500,
	}
}

// ScanDirectory walks a directory and returns all findings sorted by
// confidence (highest first).
func (s *Scanner) ScanDirectory(root string) ([]ScanFinding, error) {
	var findings []ScanFinding

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible files
		}

		// Skip excluded directories.
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip binary / non-text files.
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if skipExtensions[ext] {
			return nil
		}

		// Skip large files.
		if info.Size() > maxFileSize {
			return nil
		}

		// Cap total findings.
		if len(findings) >= s.maxFindings {
			return filepath.SkipAll
		}

		relPath, _ := filepath.Rel(root, path)
		if relPath == "" {
			relPath = path
		}

		fileFindings, scanErr := s.scanFile(path, relPath)
		if scanErr != nil {
			return nil // skip unreadable files
		}
		findings = append(findings, fileFindings...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("scan directory: %w", err)
	}

	// Sort by confidence desc.
	for i := 1; i < len(findings); i++ {
		for j := i; j > 0 && findings[j].Confidence > findings[j-1].Confidence; j-- {
			findings[j], findings[j-1] = findings[j-1], findings[j]
		}
	}

	return findings, nil
}

func (s *Scanner) scanFile(absPath, relPath string) ([]ScanFinding, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var findings []ScanFinding
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		for _, rule := range s.rules {
			if rule.Confidence < s.minConfidence {
				continue
			}
			if rule.Pattern.MatchString(line) {
				findings = append(findings, ScanFinding{
					File:       relPath,
					Line:       lineNum,
					RuleID:     rule.ID,
					RuleName:   rule.Name,
					Match:      redactMatch(rule.Pattern.FindString(line)),
					Confidence: rule.Confidence,
				})
			}
		}
	}

	return findings, scanner.Err()
}

// redactMatch returns a short, partially redacted excerpt of the match to
// avoid exposing the actual secret value in scan reports.
func redactMatch(match string) string {
	if len(match) <= 8 {
		return "***"
	}
	// Show first 4 and last 4 characters, mask the rest.
	return match[:4] + "..." + match[len(match)-4:]
}
