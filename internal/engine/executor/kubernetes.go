package executor

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// KubernetesExecutor runs steps as Kubernetes Jobs using the Kubernetes API
// directly via HTTP, avoiding the need for client-go.
type KubernetesExecutor struct {
	apiServerURL string
	token        string
	client       *http.Client
	namespace    string
}

// NewKubernetesExecutor creates a new KubernetesExecutor.
// It attempts in-cluster config first, then falls back to environment variables.
func NewKubernetesExecutor() *KubernetesExecutor {
	e := &KubernetesExecutor{
		namespace: "default",
	}

	// Try in-cluster configuration
	if token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		e.token = string(token)
		e.apiServerURL = "https://kubernetes.default.svc"
		e.namespace = "default"
		if ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
			e.namespace = string(ns)
		}
		// In-cluster: use the service account CA
		e.client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
			},
			Timeout: 0,
		}
	} else {
		// Fallback to environment
		e.apiServerURL = os.Getenv("KUBERNETES_API_URL")
		if e.apiServerURL == "" {
			e.apiServerURL = "https://localhost:6443"
		}
		e.token = os.Getenv("KUBERNETES_TOKEN")
		if ns := os.Getenv("KUBERNETES_NAMESPACE"); ns != "" {
			e.namespace = ns
		}
		e.client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // Dev mode; prod should use proper CA
				},
			},
			Timeout: 0,
		}
	}

	return e
}

// k8sRequest sends an authenticated request to the Kubernetes API.
func (e *KubernetesExecutor) k8sRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := e.apiServerURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if e.token != "" {
		req.Header.Set("Authorization", "Bearer "+e.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	return e.client.Do(req)
}

// Execute runs the step as a Kubernetes Job.
func (e *KubernetesExecutor) Execute(ctx context.Context, step ExecutionStep) (*ExecutionResult, error) {
	return e.ExecuteWithLogs(ctx, step, nil)
}

// ExecuteWithLogs runs a command as a Kubernetes Job, streaming output via logWriter.
func (e *KubernetesExecutor) ExecuteWithLogs(ctx context.Context, step ExecutionStep, logWriter LogWriter) (*ExecutionResult, error) {
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, step.Timeout)
		defer cancel()
	}

	namespace := e.namespace
	if ns := step.Env["FLOWFORGE_K8S_NAMESPACE"]; ns != "" {
		namespace = ns
	}

	image := step.Env["FLOWFORGE_K8S_IMAGE"]
	if image == "" {
		image = "alpine:latest"
	}

	jobName := sanitizeK8sName(fmt.Sprintf("ff-%s-%d", step.Name, time.Now().UnixMilli()))

	// Create the Job
	job := e.buildJobSpec(jobName, namespace, image, step)

	if logWriter != nil {
		logWriter("system", []byte(fmt.Sprintf("[flowforge] Creating Kubernetes Job %s in namespace %s\n", jobName, namespace)))
	}

	resp, err := e.k8sRequest(ctx, "POST",
		fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs", namespace), job)
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create job failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Ensure cleanup
	defer func() {
		deleteCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		e.deleteJob(deleteCtx, namespace, jobName)
	}()

	start := time.Now()

	// Wait for pod to be created and get its name
	podName, err := e.waitForPod(ctx, namespace, jobName, logWriter)
	if err != nil {
		return nil, fmt.Errorf("wait for pod: %w", err)
	}

	// Stream pod logs
	var stdoutBuf bytes.Buffer
	e.streamPodLogs(ctx, namespace, podName, &stdoutBuf, logWriter)

	// Wait for Job completion
	exitCode, waitErr := e.waitForJobCompletion(ctx, namespace, jobName, podName)
	duration := time.Since(start)

	result := &ExecutionResult{
		ExitCode: exitCode,
		Stdout:   stdoutBuf.String(),
		Duration: duration,
	}

	if ctx.Err() == context.DeadlineExceeded {
		return result, fmt.Errorf("step %q timed out after %s", step.Name, step.Timeout)
	}
	if ctx.Err() == context.Canceled {
		return result, fmt.Errorf("step %q was cancelled", step.Name)
	}

	return result, waitErr
}

type k8sJob struct {
	APIVersion string      `json:"apiVersion"`
	Kind       string      `json:"kind"`
	Metadata   k8sMeta     `json:"metadata"`
	Spec       k8sJobSpec  `json:"spec"`
}

type k8sMeta struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

type k8sJobSpec struct {
	BackoffLimit          *int32          `json:"backoffLimit,omitempty"`
	ActiveDeadlineSeconds *int64          `json:"activeDeadlineSeconds,omitempty"`
	Template              k8sPodTemplate  `json:"template"`
}

type k8sPodTemplate struct {
	Metadata k8sMeta    `json:"metadata,omitempty"`
	Spec     k8sPodSpec `json:"spec"`
}

type k8sPodSpec struct {
	RestartPolicy  string           `json:"restartPolicy"`
	Containers     []k8sContainer   `json:"containers"`
	ServiceAccount string           `json:"serviceAccountName,omitempty"`
	NodeSelector   map[string]string `json:"nodeSelector,omitempty"`
	Volumes        []k8sVolume      `json:"volumes,omitempty"`
}

type k8sContainer struct {
	Name         string           `json:"name"`
	Image        string           `json:"image"`
	Command      []string         `json:"command"`
	Args         []string         `json:"args"`
	Env          []k8sEnvVar      `json:"env,omitempty"`
	Resources    *k8sResources    `json:"resources,omitempty"`
	VolumeMounts []k8sVolumeMount `json:"volumeMounts,omitempty"`
	WorkingDir   string           `json:"workingDir,omitempty"`
}

type k8sEnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type k8sResources struct {
	Requests map[string]string `json:"requests,omitempty"`
	Limits   map[string]string `json:"limits,omitempty"`
}

type k8sVolume struct {
	Name                  string `json:"name"`
	PersistentVolumeClaim *struct {
		ClaimName string `json:"claimName"`
	} `json:"persistentVolumeClaim,omitempty"`
	EmptyDir *struct{} `json:"emptyDir,omitempty"`
}

type k8sVolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
}

func (e *KubernetesExecutor) buildJobSpec(jobName, namespace, image string, step ExecutionStep) k8sJob {
	labels := map[string]string{
		"app.kubernetes.io/managed-by": "flowforge",
		"flowforge/job":                jobName,
	}

	// Build env vars, filtering out FLOWFORGE_K8S_* internal vars
	var envVars []k8sEnvVar
	for k, v := range step.Env {
		if strings.HasPrefix(k, "FLOWFORGE_K8S_") {
			continue
		}
		envVars = append(envVars, k8sEnvVar{Name: k, Value: v})
	}

	container := k8sContainer{
		Name:       "step",
		Image:      image,
		Command:    []string{"sh"},
		Args:       []string{"-c", step.Command},
		Env:        envVars,
		WorkingDir: "/workspace",
	}

	// Resource limits
	resources := &k8sResources{}
	hasResources := false
	if cpuReq := step.Env["FLOWFORGE_K8S_CPU_REQUEST"]; cpuReq != "" {
		if resources.Requests == nil {
			resources.Requests = make(map[string]string)
		}
		resources.Requests["cpu"] = cpuReq
		hasResources = true
	}
	if cpuLim := step.Env["FLOWFORGE_K8S_CPU_LIMIT"]; cpuLim != "" {
		if resources.Limits == nil {
			resources.Limits = make(map[string]string)
		}
		resources.Limits["cpu"] = cpuLim
		hasResources = true
	}
	if memReq := step.Env["FLOWFORGE_K8S_MEMORY_REQUEST"]; memReq != "" {
		if resources.Requests == nil {
			resources.Requests = make(map[string]string)
		}
		resources.Requests["memory"] = memReq
		hasResources = true
	}
	if memLim := step.Env["FLOWFORGE_K8S_MEMORY_LIMIT"]; memLim != "" {
		if resources.Limits == nil {
			resources.Limits = make(map[string]string)
		}
		resources.Limits["memory"] = memLim
		hasResources = true
	}
	if hasResources {
		container.Resources = resources
	}

	// Volumes
	var volumes []k8sVolume
	var volumeMounts []k8sVolumeMount

	// Workspace volume
	if pvc := step.Env["FLOWFORGE_K8S_WORKSPACE_PVC"]; pvc != "" {
		volumes = append(volumes, k8sVolume{
			Name: "workspace",
			PersistentVolumeClaim: &struct {
				ClaimName string `json:"claimName"`
			}{ClaimName: pvc},
		})
		volumeMounts = append(volumeMounts, k8sVolumeMount{
			Name:      "workspace",
			MountPath: "/workspace",
		})
	} else {
		volumes = append(volumes, k8sVolume{
			Name:     "workspace",
			EmptyDir: &struct{}{},
		})
		volumeMounts = append(volumeMounts, k8sVolumeMount{
			Name:      "workspace",
			MountPath: "/workspace",
		})
	}

	// Cache volume
	if cachePVC := step.Env["FLOWFORGE_K8S_CACHE_PVC"]; cachePVC != "" {
		volumes = append(volumes, k8sVolume{
			Name: "cache",
			PersistentVolumeClaim: &struct {
				ClaimName string `json:"claimName"`
			}{ClaimName: cachePVC},
		})
		volumeMounts = append(volumeMounts, k8sVolumeMount{
			Name:      "cache",
			MountPath: "/cache",
		})
	}

	container.VolumeMounts = volumeMounts

	podSpec := k8sPodSpec{
		RestartPolicy: "Never",
		Containers:    []k8sContainer{container},
		Volumes:       volumes,
	}

	// Service account
	if sa := step.Env["FLOWFORGE_K8S_SERVICE_ACCOUNT"]; sa != "" {
		podSpec.ServiceAccount = sa
	}

	// Node selector
	if ns := step.Env["FLOWFORGE_K8S_NODE_SELECTOR"]; ns != "" {
		nodeSelector := make(map[string]string)
		for _, pair := range strings.Split(ns, ",") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				nodeSelector[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
		if len(nodeSelector) > 0 {
			podSpec.NodeSelector = nodeSelector
		}
	}

	jobSpec := k8sJobSpec{
		Template: k8sPodTemplate{
			Metadata: k8sMeta{Labels: labels},
			Spec:     podSpec,
		},
	}

	backoffLimit := int32(0)
	jobSpec.BackoffLimit = &backoffLimit

	// Active deadline
	if deadline := step.Env["FLOWFORGE_K8S_ACTIVE_DEADLINE_SECONDS"]; deadline != "" {
		var d int64
		fmt.Sscanf(deadline, "%d", &d)
		if d > 0 {
			jobSpec.ActiveDeadlineSeconds = &d
		}
	}

	return k8sJob{
		APIVersion: "batch/v1",
		Kind:       "Job",
		Metadata: k8sMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: jobSpec,
	}
}

func (e *KubernetesExecutor) waitForPod(ctx context.Context, namespace, jobName string, logWriter LogWriter) (string, error) {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods?labelSelector=job-name=%s", namespace, jobName)

	for i := 0; i < 120; i++ { // Wait up to 2 minutes
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		resp, err := e.k8sRequest(ctx, "GET", path, nil)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		var podList struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
				Status struct {
					Phase             string `json:"phase"`
					ContainerStatuses []struct {
						State struct {
							Waiting *struct {
								Reason  string `json:"reason"`
								Message string `json:"message"`
							} `json:"waiting"`
							Running    *struct{} `json:"running"`
							Terminated *struct {
								ExitCode int32 `json:"exitCode"`
							} `json:"terminated"`
						} `json:"state"`
					} `json:"containerStatuses"`
				} `json:"status"`
			} `json:"items"`
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err := json.Unmarshal(body, &podList); err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		for _, pod := range podList.Items {
			podName := pod.Metadata.Name

			// Check for ImagePullBackOff
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.State.Waiting != nil {
					reason := cs.State.Waiting.Reason
					if reason == "ImagePullBackOff" || reason == "ErrImagePull" {
						return "", fmt.Errorf("image pull error: %s", cs.State.Waiting.Message)
					}
				}
			}

			// Pod is running or completed
			if pod.Status.Phase == "Running" || pod.Status.Phase == "Succeeded" || pod.Status.Phase == "Failed" {
				if logWriter != nil {
					logWriter("system", []byte(fmt.Sprintf("[flowforge] Pod %s is %s\n", podName, pod.Status.Phase)))
				}
				return podName, nil
			}
		}

		time.Sleep(1 * time.Second)
	}

	return "", fmt.Errorf("timed out waiting for pod to start")
}

func (e *KubernetesExecutor) streamPodLogs(ctx context.Context, namespace, podName string, buf *bytes.Buffer, logWriter LogWriter) {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log?follow=true&container=step", namespace, podName)

	resp, err := e.k8sRequest(ctx, "GET", path, nil)
	if err != nil {
		// Try non-follow mode as fallback
		path = fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log?container=step", namespace, podName)
		resp, err = e.k8sRequest(ctx, "GET", path, nil)
		if err != nil {
			return
		}
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		lineWithNL := make([]byte, len(line)+1)
		copy(lineWithNL, line)
		lineWithNL[len(line)] = '\n'

		buf.Write(lineWithNL)
		if logWriter != nil {
			logWriter("stdout", lineWithNL)
		}
	}
}

func (e *KubernetesExecutor) waitForJobCompletion(ctx context.Context, namespace, jobName, podName string) (int, error) {
	jobPath := fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs/%s", namespace, jobName)
	podPath := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s", namespace, podName)

	for i := 0; i < 7200; i++ { // Wait up to 2 hours
		select {
		case <-ctx.Done():
			return -1, ctx.Err()
		default:
		}

		// Check job status
		resp, err := e.k8sRequest(ctx, "GET", jobPath, nil)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		var job struct {
			Status struct {
				Succeeded  int32 `json:"succeeded"`
				Failed     int32 `json:"failed"`
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
			} `json:"status"`
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err := json.Unmarshal(body, &job); err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		if job.Status.Succeeded > 0 {
			return 0, nil
		}

		if job.Status.Failed > 0 {
			// Get exit code from pod
			exitCode := e.getPodExitCode(ctx, podPath)
			return exitCode, nil
		}

		for _, cond := range job.Status.Conditions {
			if cond.Type == "Complete" && cond.Status == "True" {
				return 0, nil
			}
			if cond.Type == "Failed" && cond.Status == "True" {
				exitCode := e.getPodExitCode(ctx, podPath)
				return exitCode, nil
			}
		}

		time.Sleep(2 * time.Second)
	}

	return -1, fmt.Errorf("timed out waiting for job completion")
}

func (e *KubernetesExecutor) getPodExitCode(ctx context.Context, podPath string) int {
	resp, err := e.k8sRequest(ctx, "GET", podPath, nil)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()

	var pod struct {
		Status struct {
			ContainerStatuses []struct {
				State struct {
					Terminated *struct {
						ExitCode int `json:"exitCode"`
					} `json:"terminated"`
				} `json:"state"`
			} `json:"containerStatuses"`
		} `json:"status"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &pod); err != nil {
		return 1
	}

	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Terminated != nil {
			return cs.State.Terminated.ExitCode
		}
	}

	return 1
}

func (e *KubernetesExecutor) deleteJob(ctx context.Context, namespace, jobName string) {
	propagation := "Background"
	body := map[string]interface{}{
		"propagationPolicy": propagation,
	}
	resp, err := e.k8sRequest(ctx, "DELETE",
		fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs/%s", namespace, jobName), body)
	if err == nil {
		resp.Body.Close()
	}
}

var k8sNameRegexp = regexp.MustCompile(`[^a-z0-9-]`)

// sanitizeK8sName converts a string to a valid Kubernetes resource name.
func sanitizeK8sName(name string) string {
	name = strings.ToLower(name)
	name = k8sNameRegexp.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if len(name) > 63 {
		name = name[:63]
	}
	name = strings.TrimRight(name, "-")
	if name == "" {
		name = "flowforge-job"
	}
	return name
}
