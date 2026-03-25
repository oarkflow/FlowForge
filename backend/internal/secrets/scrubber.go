package secrets

import (
	"regexp"
	"strings"
)

// commonPatterns matches well-known sensitive patterns that should always be
// scrubbed from log output regardless of the configured secret list.
var commonPatterns = []*regexp.Regexp{
	// Bearer / token / authorization headers
	regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9\-._~+/]+=*`),
	regexp.MustCompile(`(?i)(token\s*[:=]\s*)[A-Za-z0-9\-._~+/]+=*`),
	regexp.MustCompile(`(?i)(authorization\s*[:=]\s*)[A-Za-z0-9\-._~+/]+=*`),

	// AWS-style keys
	regexp.MustCompile(`(?i)(aws_?(?:access_?key_?id|secret_?access_?key|session_?token)\s*[:=]\s*)\S+`),
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),

	// Generic API keys / passwords in key=value or key: value form
	regexp.MustCompile(`(?i)((?:api_?key|apikey|secret_?key|password|passwd|pwd)\s*[:=]\s*)\S+`),

	// GitHub personal access tokens
	regexp.MustCompile(`ghp_[A-Za-z0-9]{36}`),
	regexp.MustCompile(`github_pat_[A-Za-z0-9_]{82}`),

	// GitLab personal access tokens
	regexp.MustCompile(`glpat-[A-Za-z0-9\-]{20,}`),

	// Slack tokens
	regexp.MustCompile(`xox[bpors]-[A-Za-z0-9\-]+`),

	// Private keys (PEM header)
	regexp.MustCompile(`-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----`),
}

// commonReplacements maps indices from commonPatterns that use a capture group
// prefix to their replacement templates. Patterns at these indices replace only
// the credential portion while preserving the prefix label.
var commonReplacementPrefix = map[int]bool{
	0: true, 1: true, 2: true, 3: true, 5: true,
}

// Scrubber replaces secret values and common sensitive patterns in log lines
// with "***".
type Scrubber struct {
	replacers []*strings.Replacer
	patterns  []*regexp.Regexp
}

// NewScrubber creates a Scrubber that will redact the given literal secret
// values plus common credential patterns from any log line.
func NewScrubber(secretValues []string) *Scrubber {
	s := &Scrubber{}

	// Build a strings.Replacer for exact secret value matches. We filter out
	// very short values (< 4 chars) to avoid false positives.
	var pairs []string
	for _, v := range secretValues {
		if len(v) < 4 {
			continue
		}
		pairs = append(pairs, v, "***")
	}
	if len(pairs) > 0 {
		s.replacers = append(s.replacers, strings.NewReplacer(pairs...))
	}

	// Compile per-secret regex patterns so multi-line or partial matches work.
	for _, v := range secretValues {
		if len(v) < 4 {
			continue
		}
		p, err := regexp.Compile(regexp.QuoteMeta(v))
		if err == nil {
			s.patterns = append(s.patterns, p)
		}
	}

	return s
}

// Scrub replaces all known secret values and common sensitive patterns in the
// given line, returning the sanitised string.
func (s *Scrubber) Scrub(line string) string {
	// First pass: exact secret value replacement via strings.Replacer (fast).
	for _, r := range s.replacers {
		line = r.Replace(line)
	}

	// Second pass: regex-based secret value replacement for any remaining
	// occurrences that the replacer might have missed (e.g. partial overlap).
	for _, p := range s.patterns {
		line = p.ReplaceAllString(line, "***")
	}

	// Third pass: well-known credential patterns.
	for i, p := range commonPatterns {
		if commonReplacementPrefix[i] {
			// Preserve the label prefix, redact only the credential value.
			line = p.ReplaceAllString(line, "${1}***")
		} else {
			line = p.ReplaceAllString(line, "***")
		}
	}

	return line
}
