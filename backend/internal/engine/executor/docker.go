package executor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// DockerExecutor runs steps inside Docker containers using the Docker Engine API
// directly via HTTP over a Unix socket, avoiding the need for the full Docker SDK.
//
// Each step gets its own short-lived container but all steps within a job share
// a single Docker named volume mounted at /workspace so that files (checked-out
// source, build artifacts, etc.) persist across steps.
type DockerExecutor struct {
	client     *http.Client
	socketPath string
	mu         sync.Mutex

	// WorkspaceVolume is the name of a Docker volume that will be mounted
	// at /workspace in every container created by this executor. The runner
	// creates the volume before the first step and removes it after the last.
	WorkspaceVolume string

	// pulledImages tracks images already pulled in this executor's lifetime
	// to avoid redundant pulls within the same job.
	pulledImages map[string]bool
}

// NewDockerExecutor creates a new DockerExecutor.
func NewDockerExecutor() *DockerExecutor {
	socketPath := os.Getenv("DOCKER_HOST")
	if socketPath == "" {
		socketPath = "/var/run/docker.sock"
	} else {
		socketPath = strings.TrimPrefix(socketPath, "unix://")
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}

	return &DockerExecutor{
		client: &http.Client{
			Transport: transport,
			Timeout:   0, // No timeout; we use context for cancellation
		},
		socketPath:   socketPath,
		pulledImages: make(map[string]bool),
	}
}

// CreateVolume creates a Docker named volume. Call this once per job before
// executing steps, then set WorkspaceVolume to the returned name.
func (e *DockerExecutor) CreateVolume(ctx context.Context, name string) error {
	body := map[string]string{"Name": name}
	resp, err := e.dockerRequest(ctx, "POST", "/v1.43/volumes/create", body)
	if err != nil {
		return fmt.Errorf("create volume: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create volume failed (status %d): %s", resp.StatusCode, string(b))
	}
	io.Copy(io.Discard, resp.Body)
	return nil
}

// RemoveVolume removes a Docker named volume. Call this after all steps finish.
func (e *DockerExecutor) RemoveVolume(ctx context.Context, name string) {
	resp, err := e.dockerRequest(ctx, "DELETE", fmt.Sprintf("/v1.43/volumes/%s?force=true", name), nil)
	if err == nil {
		resp.Body.Close()
	}
}

// dockerRequest sends an HTTP request to the Docker daemon.
func (e *DockerExecutor) dockerRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, "http://docker"+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return e.client.Do(req)
}

// Execute runs the step inside a Docker container.
func (e *DockerExecutor) Execute(ctx context.Context, step ExecutionStep) (*ExecutionResult, error) {
	return e.ExecuteWithLogs(ctx, step, nil)
}

// ExecuteWithLogs runs a command in a Docker container, streaming output via logWriter.
// If WorkspaceVolume is set, the volume is mounted at /workspace so files persist
// across steps within the same job.
func (e *DockerExecutor) ExecuteWithLogs(ctx context.Context, step ExecutionStep, logWriter LogWriter) (*ExecutionResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if step.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, step.Timeout)
		defer cancel()
	}

	image := step.Env["FLOWFORGE_DOCKER_IMAGE"]
	if image == "" {
		image = "alpine:latest"
	}

	// Pull image (skip if already pulled in this job)
	if !e.pulledImages[image] {
		if err := e.pullImage(ctx, image, step.Env["FLOWFORGE_DOCKER_REGISTRY_AUTH"], logWriter); err != nil {
			return nil, fmt.Errorf("pull image %s: %w", image, err)
		}
		e.pulledImages[image] = true
	}

	// Build container config
	containerConfig := e.buildContainerConfig(step, image)

	// Create container
	containerID, err := e.createContainer(ctx, containerConfig)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	// Always remove container when done
	defer func() {
		removeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		e.removeContainer(removeCtx, containerID)
	}()

	start := time.Now()

	// Start container
	resp, err := e.dockerRequest(ctx, "POST", fmt.Sprintf("/v1.43/containers/%s/start", containerID), nil)
	if err != nil {
		return nil, fmt.Errorf("start container: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("start container: unexpected status %d", resp.StatusCode)
	}

	// Attach to logs (stream stdout/stderr)
	var stdoutBuf, stderrBuf bytes.Buffer
	e.streamContainerLogs(ctx, containerID, &stdoutBuf, &stderrBuf, logWriter)

	// Wait for container to finish
	exitCode, waitErr := e.waitContainer(ctx, containerID)
	duration := time.Since(start)

	result := &ExecutionResult{
		ExitCode: exitCode,
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		Duration: duration,
	}

	if waitErr != nil {
		return result, waitErr
	}

	if ctx.Err() == context.DeadlineExceeded {
		// Try to stop the container
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		e.stopContainer(stopCtx, containerID)
		return result, fmt.Errorf("step %q timed out after %s", step.Name, step.Timeout)
	}
	if ctx.Err() == context.Canceled {
		return result, fmt.Errorf("step %q was cancelled", step.Name)
	}

	return result, nil
}

// dockerContainerConfig represents the Docker container creation request.
type dockerContainerConfig struct {
	Image        string           `json:"Image"`
	Cmd          []string         `json:"Cmd"`
	Env          []string         `json:"Env,omitempty"`
	WorkingDir   string           `json:"WorkingDir,omitempty"`
	HostConfig   dockerHostConfig `json:"HostConfig,omitempty"`
	NetworkMode  string           `json:"NetworkMode,omitempty"`
	AttachStdout bool             `json:"AttachStdout"`
	AttachStderr bool             `json:"AttachStderr"`
	Tty          bool             `json:"Tty"`
}

type dockerHostConfig struct {
	Binds       []string `json:"Binds,omitempty"`
	Privileged  bool     `json:"Privileged,omitempty"`
	NetworkMode string   `json:"NetworkMode,omitempty"`
	NanoCPUs    int64    `json:"NanoCpus,omitempty"`
	Memory      int64    `json:"Memory,omitempty"`
	ExtraHosts  []string `json:"ExtraHosts,omitempty"`
}

func (e *DockerExecutor) buildContainerConfig(step ExecutionStep, image string) dockerContainerConfig {
	// Build environment variables, filtering out FLOWFORGE_DOCKER_* internal vars
	var envList []string
	for k, v := range step.Env {
		if strings.HasPrefix(k, "FLOWFORGE_DOCKER_") {
			continue // Internal config, don't pass to container
		}
		envList = append(envList, k+"="+v)
	}

	config := dockerContainerConfig{
		Image:        image,
		Cmd:          []string{"sh", "-c", step.Command},
		Env:          envList,
		WorkingDir:   "/workspace",
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}

	// Bind mounts
	var binds []string

	// Mount the shared workspace volume for cross-step file persistence.
	// This is the key mechanism: all steps in a job share this volume at
	// /workspace, so checkout files persist for build/test steps.
	if e.WorkspaceVolume != "" {
		binds = append(binds, e.WorkspaceVolume+":/workspace")
	} else if step.WorkDir != "" {
		// Fallback: bind-mount a host directory
		binds = append(binds, step.WorkDir+":/workspace")
	}

	// If the repo clone URL is a local path, bind-mount it into the container
	// at /mnt/source so the checkout action can copy from it.
	if repoURL := step.Env["FLOWFORGE_REPO_CLONE_URL"]; repoURL != "" {
		if strings.HasPrefix(repoURL, "/") || strings.HasPrefix(repoURL, "./") || strings.HasPrefix(repoURL, "../") {
			mountPath := repoURL
			binds = append(binds, mountPath+":/mnt/source:ro")
		}
	}

	// Mount Docker socket if requested (for DinD)
	if step.Env["FLOWFORGE_DOCKER_MOUNT_DOCKER_SOCKET"] == "true" {
		binds = append(binds, "/var/run/docker.sock:/var/run/docker.sock")
	}

	// Cache volumes
	if cacheVolumes := step.Env["FLOWFORGE_DOCKER_CACHE_VOLUMES"]; cacheVolumes != "" {
		for _, v := range strings.Split(cacheVolumes, ",") {
			v = strings.TrimSpace(v)
			if v != "" {
				binds = append(binds, v)
			}
		}
	}

	config.HostConfig.Binds = binds

	// Privileged mode
	if step.Env["FLOWFORGE_DOCKER_PRIVILEGED"] == "true" {
		config.HostConfig.Privileged = true
	}

	// Network mode
	if nm := step.Env["FLOWFORGE_DOCKER_NETWORK_MODE"]; nm != "" {
		config.HostConfig.NetworkMode = nm
	}

	// Resource limits
	if cpus := step.Env["FLOWFORGE_DOCKER_NANO_CPUS"]; cpus != "" {
		var n int64
		fmt.Sscanf(cpus, "%d", &n)
		if n > 0 {
			config.HostConfig.NanoCPUs = n
		}
	}
	if mem := step.Env["FLOWFORGE_DOCKER_MEMORY_BYTES"]; mem != "" {
		var n int64
		fmt.Sscanf(mem, "%d", &n)
		if n > 0 {
			config.HostConfig.Memory = n
		}
	}

	// Extra hosts
	if hosts := step.Env["FLOWFORGE_DOCKER_EXTRA_HOSTS"]; hosts != "" {
		config.HostConfig.ExtraHosts = strings.Split(hosts, ",")
	}

	return config
}

func (e *DockerExecutor) pullImage(ctx context.Context, image, registryAuth string, logWriter LogWriter) error {
	path := fmt.Sprintf("/v1.43/images/create?fromImage=%s", image)

	req, err := http.NewRequestWithContext(ctx, "POST", "http://docker"+path, nil)
	if err != nil {
		return err
	}
	if registryAuth != "" {
		req.Header.Set("X-Registry-Auth", registryAuth)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull image failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Read and optionally stream pull progress
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if logWriter != nil {
			var progress struct {
				Status   string `json:"status"`
				Progress string `json:"progress"`
			}
			if json.Unmarshal(scanner.Bytes(), &progress) == nil && progress.Status != "" {
				msg := progress.Status
				if progress.Progress != "" {
					msg += " " + progress.Progress
				}
				logWriter("system", []byte(msg+"\n"))
			}
		}
	}

	return nil
}

func (e *DockerExecutor) createContainer(ctx context.Context, config dockerContainerConfig) (string, error) {
	resp, err := e.dockerRequest(ctx, "POST", "/v1.43/containers/create", config)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create container failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode create response: %w", err)
	}

	return result.ID, nil
}

func (e *DockerExecutor) streamContainerLogs(ctx context.Context, containerID string, stdoutBuf, stderrBuf *bytes.Buffer, logWriter LogWriter) {
	path := fmt.Sprintf("/v1.43/containers/%s/logs?follow=true&stdout=true&stderr=true&timestamps=false", containerID)
	resp, err := e.dockerRequest(ctx, "GET", path, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Docker multiplexed stream: 8-byte header per frame
	// [0]: stream type (0=stdin, 1=stdout, 2=stderr)
	// [1-3]: reserved
	// [4-7]: big-endian uint32 payload size
	header := make([]byte, 8)
	for {
		_, err := io.ReadFull(resp.Body, header)
		if err != nil {
			break
		}

		streamType := header[0]
		payloadSize := uint32(header[4])<<24 | uint32(header[5])<<16 | uint32(header[6])<<8 | uint32(header[7])

		if payloadSize == 0 {
			continue
		}

		payload := make([]byte, payloadSize)
		_, err = io.ReadFull(resp.Body, payload)
		if err != nil {
			break
		}

		switch streamType {
		case 1: // stdout
			stdoutBuf.Write(payload)
			if logWriter != nil {
				logWriter("stdout", payload)
			}
		case 2: // stderr
			stderrBuf.Write(payload)
			if logWriter != nil {
				logWriter("stderr", payload)
			}
		}
	}
}

func (e *DockerExecutor) waitContainer(ctx context.Context, containerID string) (int, error) {
	path := fmt.Sprintf("/v1.43/containers/%s/wait", containerID)
	resp, err := e.dockerRequest(ctx, "POST", path, nil)
	if err != nil {
		return -1, fmt.Errorf("wait container: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		StatusCode int `json:"StatusCode"`
		Error      struct {
			Message string `json:"Message"`
		} `json:"Error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return -1, fmt.Errorf("decode wait response: %w", err)
	}

	if result.Error.Message != "" {
		return result.StatusCode, fmt.Errorf("container error: %s", result.Error.Message)
	}

	return result.StatusCode, nil
}

func (e *DockerExecutor) stopContainer(ctx context.Context, containerID string) {
	path := fmt.Sprintf("/v1.43/containers/%s/stop?t=10", containerID)
	resp, err := e.dockerRequest(ctx, "POST", path, nil)
	if err == nil {
		resp.Body.Close()
	}
}

func (e *DockerExecutor) removeContainer(ctx context.Context, containerID string) {
	// NOTE: Do NOT use v=true here. That flag removes anonymous volumes
	// associated with the container, which is unnecessary (named workspace
	// volumes are cleaned up explicitly via RemoveVolume) and risky (could
	// interfere with bind-mounted source directories in edge cases).
	path := fmt.Sprintf("/v1.43/containers/%s?force=true", containerID)
	resp, err := e.dockerRequest(ctx, "DELETE", path, nil)
	if err == nil {
		resp.Body.Close()
	}
}

// EncodeRegistryAuth builds a base64-encoded auth string for Docker registry auth.
func EncodeRegistryAuth(username, password string) string {
	auth := map[string]string{
		"username": username,
		"password": password,
	}
	data, _ := json.Marshal(auth)
	return base64.URLEncoding.EncodeToString(data)
}
