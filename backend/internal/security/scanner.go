package security

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/models"
)

// Scanner is the interface for vulnerability scanners.
type Scanner interface {
	Scan(ctx context.Context, target string) (*models.ScanResult, error)
	Name() string
}

// ScanService manages vulnerability scanning.
type ScanService struct {
	repo     *queries.ScanResultRepo
	scanners []Scanner
}

// NewScanService creates a new scan service with the specified scanners.
func NewScanService(repo *queries.ScanResultRepo) *ScanService {
	svc := &ScanService{repo: repo}

	// Auto-detect available scanners
	if _, err := exec.LookPath("trivy"); err == nil {
		svc.scanners = append(svc.scanners, &TrivyScanner{})
	}
	if _, err := exec.LookPath("grype"); err == nil {
		svc.scanners = append(svc.scanners, &GrypeScanner{})
	}

	return svc
}

// RunScan runs all available scanners against the given target and stores results.
func (s *ScanService) RunScan(ctx context.Context, runID, target string) ([]models.ScanResult, error) {
	var results []models.ScanResult

	for _, scanner := range s.scanners {
		result, err := scanner.Scan(ctx, target)
		if err != nil {
			log.Error().Err(err).Str("scanner", scanner.Name()).Msg("security: scan failed")
			// Store error result
			errResult := &models.ScanResult{
				RunID:       runID,
				ScannerType: scanner.Name(),
				Target:      target,
				Status:      "error",
				Vulnerabilities: "[]",
			}
			_ = s.repo.Create(ctx, errResult)
			results = append(results, *errResult)
			continue
		}

		result.RunID = runID
		if err := s.repo.Create(ctx, result); err != nil {
			log.Error().Err(err).Msg("security: failed to store scan result")
		}
		results = append(results, *result)
	}

	return results, nil
}

// CheckGate evaluates whether the scan results pass the quality gate.
// Returns false if any critical or high vulnerabilities are found.
func (s *ScanService) CheckGate(results []models.ScanResult, maxCritical, maxHigh int) bool {
	for _, r := range results {
		if r.CriticalCount > maxCritical || r.HighCount > maxHigh {
			return false
		}
	}
	return true
}

// GetByRunID returns all scan results for a pipeline run.
func (s *ScanService) GetByRunID(ctx context.Context, runID string) ([]models.ScanResult, error) {
	return s.repo.ListByRunID(ctx, runID, 100, 0)
}

// --------------------------------------------------------------------------
// Trivy Scanner
// --------------------------------------------------------------------------

// TrivyScanner shells out to trivy for filesystem/container scanning.
type TrivyScanner struct{}

func (t *TrivyScanner) Name() string { return "trivy" }

func (t *TrivyScanner) Scan(ctx context.Context, target string) (*models.ScanResult, error) {
	// Run trivy with JSON output
	cmd := exec.CommandContext(ctx, "trivy", "fs", "--format", "json", "--severity", "CRITICAL,HIGH,MEDIUM,LOW", target)
	output, err := cmd.Output()
	if err != nil {
		// Trivy exits with code 1 if vulnerabilities found
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("trivy: %s", string(exitErr.Stderr))
		}
		// Still try to parse output if available
		if len(output) == 0 {
			return nil, fmt.Errorf("trivy: %w", err)
		}
	}

	return parseTrivyOutput(output, target)
}

func parseTrivyOutput(output []byte, target string) (*models.ScanResult, error) {
	// Trivy JSON output structure (simplified)
	var trivyResult struct {
		Results []struct {
			Vulnerabilities []struct {
				VulnerabilityID string `json:"VulnerabilityID"`
				Severity        string `json:"Severity"`
				PkgName         string `json:"PkgName"`
				InstalledVersion string `json:"InstalledVersion"`
				FixedVersion    string `json:"FixedVersion"`
				Title           string `json:"Title"`
				Description     string `json:"Description"`
			} `json:"Vulnerabilities"`
		} `json:"Results"`
	}

	if err := json.Unmarshal(output, &trivyResult); err != nil {
		return nil, fmt.Errorf("trivy: parse output: %w", err)
	}

	var vulns []models.Vulnerability
	critical, high, medium, low := 0, 0, 0, 0

	for _, r := range trivyResult.Results {
		for _, v := range r.Vulnerabilities {
			vulns = append(vulns, models.Vulnerability{
				CVEID:       v.VulnerabilityID,
				Severity:    strings.ToLower(v.Severity),
				Package:     v.PkgName,
				Version:     v.InstalledVersion,
				FixVersion:  v.FixedVersion,
				Title:       v.Title,
				Description: v.Description,
			})
			switch strings.ToLower(v.Severity) {
			case "critical":
				critical++
			case "high":
				high++
			case "medium":
				medium++
			case "low":
				low++
			}
		}
	}

	vulnJSON, _ := json.Marshal(vulns)
	status := "pass"
	if critical > 0 || high > 0 {
		status = "fail"
	}

	return &models.ScanResult{
		ScannerType:     "trivy",
		Target:          target,
		Vulnerabilities: string(vulnJSON),
		CriticalCount:   critical,
		HighCount:       high,
		MediumCount:     medium,
		LowCount:        low,
		Status:          status,
	}, nil
}

// --------------------------------------------------------------------------
// Grype Scanner
// --------------------------------------------------------------------------

// GrypeScanner shells out to grype for SBOM-based vulnerability scanning.
type GrypeScanner struct{}

func (g *GrypeScanner) Name() string { return "grype" }

func (g *GrypeScanner) Scan(ctx context.Context, target string) (*models.ScanResult, error) {
	cmd := exec.CommandContext(ctx, "grype", target, "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("grype: %s", string(exitErr.Stderr))
		}
		if len(output) == 0 {
			return nil, fmt.Errorf("grype: %w", err)
		}
	}

	return parseGrypeOutput(output, target)
}

func parseGrypeOutput(output []byte, target string) (*models.ScanResult, error) {
	var grypeResult struct {
		Matches []struct {
			Vulnerability struct {
				ID       string `json:"id"`
				Severity string `json:"severity"`
				Fix      struct {
					Versions []string `json:"versions"`
				} `json:"fix"`
				Description string `json:"description"`
			} `json:"vulnerability"`
			Artifact struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"artifact"`
		} `json:"matches"`
	}

	if err := json.Unmarshal(output, &grypeResult); err != nil {
		return nil, fmt.Errorf("grype: parse output: %w", err)
	}

	var vulns []models.Vulnerability
	critical, high, medium, low := 0, 0, 0, 0

	for _, m := range grypeResult.Matches {
		fixVersion := ""
		if len(m.Vulnerability.Fix.Versions) > 0 {
			fixVersion = m.Vulnerability.Fix.Versions[0]
		}
		vulns = append(vulns, models.Vulnerability{
			CVEID:       m.Vulnerability.ID,
			Severity:    strings.ToLower(m.Vulnerability.Severity),
			Package:     m.Artifact.Name,
			Version:     m.Artifact.Version,
			FixVersion:  fixVersion,
			Description: m.Vulnerability.Description,
		})
		switch strings.ToLower(m.Vulnerability.Severity) {
		case "critical":
			critical++
		case "high":
			high++
		case "medium":
			medium++
		case "low":
			low++
		}
	}

	vulnJSON, _ := json.Marshal(vulns)
	status := "pass"
	if critical > 0 || high > 0 {
		status = "fail"
	}

	return &models.ScanResult{
		ScannerType:     "grype",
		Target:          target,
		Vulnerabilities: string(vulnJSON),
		CriticalCount:   critical,
		HighCount:       high,
		MediumCount:     medium,
		LowCount:        low,
		Status:          status,
	}, nil
}
