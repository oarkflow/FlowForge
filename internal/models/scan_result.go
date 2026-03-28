package models

import "time"

type ScanResult struct {
	ID              string    `db:"id" json:"id"`
	RunID           string    `db:"run_id" json:"run_id"`
	ScannerType     string    `db:"scanner_type" json:"scanner_type"` // trivy, grype
	Target          string    `db:"target" json:"target"`
	Vulnerabilities string    `db:"vulnerabilities" json:"vulnerabilities"` // JSON array
	CriticalCount   int       `db:"critical_count" json:"critical_count"`
	HighCount       int       `db:"high_count" json:"high_count"`
	MediumCount     int       `db:"medium_count" json:"medium_count"`
	LowCount        int       `db:"low_count" json:"low_count"`
	Status          string    `db:"status" json:"status"` // pass, fail, error
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

type Vulnerability struct {
	CVEID       string `json:"cve_id"`
	Severity    string `json:"severity"` // critical, high, medium, low
	Package     string `json:"package"`
	Version     string `json:"version"`
	FixVersion  string `json:"fix_version,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}
