package importer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// CloneOptions holds optional settings for git clone.
type CloneOptions struct {
	SSHKeyPEM string // PEM-encoded SSH private key
}

// CloneRepo performs a shallow git clone into destDir.
func CloneRepo(ctx context.Context, url, branch, destDir string, opts CloneOptions) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	args := []string{"clone", "--depth=1"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, url, destDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = filepath.Dir(destDir)

	// For SSH key auth, write key to temp file and configure GIT_SSH_COMMAND.
	if opts.SSHKeyPEM != "" {
		keyFile, err := os.CreateTemp("", "flowforge-ssh-key-*")
		if err != nil {
			return fmt.Errorf("create ssh key temp file: %w", err)
		}
		defer os.Remove(keyFile.Name())

		if _, err := keyFile.WriteString(opts.SSHKeyPEM); err != nil {
			keyFile.Close()
			return fmt.Errorf("write ssh key: %w", err)
		}
		keyFile.Close()
		if err := os.Chmod(keyFile.Name(), 0600); err != nil {
			return fmt.Errorf("chmod ssh key: %w", err)
		}

		cmd.Env = append(os.Environ(),
			fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", keyFile.Name()),
		)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}
