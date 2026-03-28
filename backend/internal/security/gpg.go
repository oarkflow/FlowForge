package security

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CommitVerification holds the result of GPG/SSH signature verification.
type CommitVerification struct {
	Verified  bool   `json:"verified"`
	Signer    string `json:"signer,omitempty"`
	KeyID     string `json:"key_id,omitempty"`
	Algorithm string `json:"algorithm,omitempty"` // gpg, ssh
	Error     string `json:"error,omitempty"`
}

// VerifyCommitSignature verifies the GPG or SSH signature of a git commit.
// It requires the git binary and the commit to exist in the given repo path.
func VerifyCommitSignature(ctx context.Context, repoPath, commitSHA string) (*CommitVerification, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return &CommitVerification{
			Verified: false,
			Error:    "git not available",
		}, nil
	}

	// Try GPG verification first
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "verify-commit", "--raw", commitSHA)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err == nil {
		// Signature verified successfully
		return parseGPGVerification(outputStr), nil
	}

	// Try SSH signature verification
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "log", "--show-signature", "-1", commitSHA)
	output, err = cmd.CombinedOutput()
	outputStr = string(output)

	if err == nil && strings.Contains(outputStr, "Good \"git\" signature") {
		return &CommitVerification{
			Verified:  true,
			Algorithm: "ssh",
			Signer:    extractSSHSigner(outputStr),
		}, nil
	}

	// No valid signature found
	return &CommitVerification{
		Verified: false,
		Error:    "no valid signature found",
	}, nil
}

func parseGPGVerification(output string) *CommitVerification {
	v := &CommitVerification{
		Algorithm: "gpg",
	}

	if strings.Contains(output, "GOODSIG") || strings.Contains(output, "VALIDSIG") {
		v.Verified = true
	}

	// Extract key ID
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "GOODSIG") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				v.KeyID = parts[2]
			}
			if len(parts) >= 4 {
				v.Signer = strings.Join(parts[3:], " ")
			}
		}
		if strings.Contains(line, "VALIDSIG") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				v.KeyID = parts[2]
			}
		}
	}

	if !v.Verified {
		v.Error = "signature verification failed"
	}

	return v
}

func extractSSHSigner(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "signature") {
			parts := strings.SplitAfter(line, "signature for")
			if len(parts) >= 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// EnforceSignedCommits returns an error if the commit is not properly signed.
// Use this as an optional pipeline gate.
func EnforceSignedCommits(ctx context.Context, repoPath, commitSHA string) error {
	v, err := VerifyCommitSignature(ctx, repoPath, commitSHA)
	if err != nil {
		return fmt.Errorf("signature verification error: %w", err)
	}
	if !v.Verified {
		return fmt.Errorf("commit %s is not signed or has invalid signature: %s", commitSHA, v.Error)
	}
	return nil
}
