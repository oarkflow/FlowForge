package secrets

import (
	"strings"
	"testing"
)

func TestScrubber_ExactSecretReplacement(t *testing.T) {
	s := NewScrubber([]string{"supersecretvalue"})
	line := "connecting with token supersecretvalue to server"
	result := s.Scrub(line)
	if strings.Contains(result, "supersecretvalue") {
		t.Errorf("secret not scrubbed: %s", result)
	}
	if !strings.Contains(result, "***") {
		t.Error("should contain *** replacement")
	}
}

func TestScrubber_ShortSecretsIgnored(t *testing.T) {
	s := NewScrubber([]string{"ab", "x"})
	line := "value ab and x should stay"
	result := s.Scrub(line)
	if !strings.Contains(result, "ab") || !strings.Contains(result, "x") {
		t.Errorf("short secrets should not be scrubbed: %s", result)
	}
}

func TestScrubber_MultipleSecrets(t *testing.T) {
	s := NewScrubber([]string{"secret_one_value", "secret_two_value"})
	line := "first=secret_one_value second=secret_two_value"
	result := s.Scrub(line)
	if strings.Contains(result, "secret_one_value") {
		t.Error("first secret not scrubbed")
	}
	if strings.Contains(result, "secret_two_value") {
		t.Error("second secret not scrubbed")
	}
}

func TestScrubber_EmptySecretList(t *testing.T) {
	s := NewScrubber(nil)
	line := "no secrets here"
	result := s.Scrub(line)
	if result != line {
		t.Errorf("line should be unchanged but common patterns may be applied")
	}
}

func TestScrubber_CommonPatterns_BearerToken(t *testing.T) {
	s := NewScrubber(nil)
	line := "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.payload.signature"
	result := s.Scrub(line)
	if strings.Contains(result, "eyJhbGciOiJIUzI1NiJ9") {
		t.Errorf("bearer token should be scrubbed: %s", result)
	}
}

func TestScrubber_CommonPatterns_AWSKey(t *testing.T) {
	s := NewScrubber(nil)
	line := "found key AKIAIOSFODNN7EXAMPLE"
	result := s.Scrub(line)
	if strings.Contains(result, "AKIAIOSFODNN7EXAMPLE") {
		t.Errorf("AWS key should be scrubbed: %s", result)
	}
}

func TestScrubber_CommonPatterns_GitHubPAT(t *testing.T) {
	s := NewScrubber(nil)
	line := "token=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefgh12"
	result := s.Scrub(line)
	if strings.Contains(result, "ghp_") {
		t.Errorf("GitHub PAT should be scrubbed: %s", result)
	}
}

func TestScrubber_CommonPatterns_GitLabPAT(t *testing.T) {
	s := NewScrubber(nil)
	line := "token=glpat-abcdefghijklmnopqrst"
	result := s.Scrub(line)
	if strings.Contains(result, "glpat-") {
		t.Errorf("GitLab PAT should be scrubbed: %s", result)
	}
}

func TestScrubber_CommonPatterns_SlackToken(t *testing.T) {
	s := NewScrubber(nil)
	line := "using xoxb-123456789012-abcdefghij"
	result := s.Scrub(line)
	if strings.Contains(result, "xoxb-") {
		t.Errorf("Slack token should be scrubbed: %s", result)
	}
}

func TestScrubber_CommonPatterns_PrivateKey(t *testing.T) {
	s := NewScrubber(nil)
	line := "-----BEGIN RSA PRIVATE KEY-----"
	result := s.Scrub(line)
	if strings.Contains(result, "BEGIN RSA PRIVATE KEY") {
		t.Errorf("private key header should be scrubbed: %s", result)
	}
}

func TestScrubber_CommonPatterns_GenericAPIKey(t *testing.T) {
	s := NewScrubber(nil)
	tests := []struct {
		name string
		line string
	}{
		{"api_key=", "api_key=supersecrettoken123"},
		{"password:", "password: mydbpassword"},
		{"secret_key=", "secret_key=foobar12345"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.Scrub(tt.line)
			if !strings.Contains(result, "***") {
				t.Errorf("generic pattern should be scrubbed: %s", result)
			}
		})
	}
}

func TestScrubber_NoFalsePositives(t *testing.T) {
	s := NewScrubber([]string{"mysecretvalue1234"})
	line := "this is a normal log line with no sensitive data"
	result := s.Scrub(line)
	if result != line {
		t.Errorf("normal line should not be modified: %s", result)
	}
}

func TestScrubber_SecretInMiddleOfWord(t *testing.T) {
	s := NewScrubber([]string{"secret1234"})
	line := "prefix_secret1234_suffix"
	result := s.Scrub(line)
	if strings.Contains(result, "secret1234") {
		t.Errorf("secret embedded in text should be scrubbed: %s", result)
	}
}
