package agent

import (
	"fmt"
	"sync"
	"time"
)

// AgentInfo represents a connected agent in the pool.
type AgentInfo struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Labels       []string          `json:"labels"`
	Executor     string            `json:"executor"`
	OS           string            `json:"os"`
	Arch         string            `json:"arch"`
	CPUCores     int32             `json:"cpu_cores"`
	MemoryMB     int64             `json:"memory_mb"`
	Version      string            `json:"version"`
	Status       string            `json:"status"` // online, busy, draining
	ActiveJobs   int32             `json:"active_jobs"`
	MaxJobs      int32             `json:"max_jobs"`
	CPUUsage     float64           `json:"cpu_usage"`
	MemoryUsage  float64           `json:"memory_usage"`
	IPAddress    string            `json:"ip_address"`
	LastSeenAt   time.Time         `json:"last_seen_at"`
	RegisteredAt time.Time         `json:"registered_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Pool manages a collection of connected agents and handles job dispatching.
type Pool struct {
	mu     sync.RWMutex
	agents map[string]*AgentInfo
}

// NewPool creates a new agent pool.
func NewPool() *Pool {
	return &Pool{
		agents: make(map[string]*AgentInfo),
	}
}

// Register adds a new agent to the pool.
func (p *Pool) Register(agent *AgentInfo) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.agents[agent.ID]; exists {
		// Update existing agent
		p.agents[agent.ID] = agent
		return nil
	}

	agent.Status = "online"
	agent.RegisteredAt = time.Now()
	agent.LastSeenAt = time.Now()
	if agent.MaxJobs <= 0 {
		agent.MaxJobs = int32(agent.CPUCores)
		if agent.MaxJobs <= 0 {
			agent.MaxJobs = 2
		}
	}
	p.agents[agent.ID] = agent
	return nil
}

// Unregister removes an agent from the pool.
func (p *Pool) Unregister(agentID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.agents, agentID)
}

// UpdateHeartbeat updates the agent's last seen timestamp and resource usage.
func (p *Pool) UpdateHeartbeat(agentID string, activeJobs int32, cpuUsage, memoryUsage float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	agent, ok := p.agents[agentID]
	if !ok {
		return fmt.Errorf("agent %s not found in pool", agentID)
	}

	agent.LastSeenAt = time.Now()
	agent.ActiveJobs = activeJobs
	agent.CPUUsage = cpuUsage
	agent.MemoryUsage = memoryUsage

	if agent.Status == "offline" {
		agent.Status = "online"
	}

	return nil
}

// SetStatus updates the agent status.
func (p *Pool) SetStatus(agentID, status string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	agent, ok := p.agents[agentID]
	if !ok {
		return fmt.Errorf("agent %s not found", agentID)
	}

	agent.Status = status
	return nil
}

// Get returns a specific agent's info.
func (p *Pool) Get(agentID string) (*AgentInfo, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	agent, ok := p.agents[agentID]
	if !ok {
		return nil, false
	}
	// Return a copy
	copy := *agent
	return &copy, true
}

// List returns all agents in the pool.
func (p *Pool) List() []*AgentInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*AgentInfo, 0, len(p.agents))
	for _, a := range p.agents {
		copy := *a
		result = append(result, &copy)
	}
	return result
}

// Available returns agents that can accept new jobs.
func (p *Pool) Available() []*AgentInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*AgentInfo, 0)
	for _, a := range p.agents {
		if a.Status == "online" && a.ActiveJobs < a.MaxJobs {
			copy := *a
			result = append(result, &copy)
		}
	}
	return result
}

// SelectAgent picks the best agent for a job based on requirements.
// It matches executor type, labels, and picks the least loaded agent.
func (p *Pool) SelectAgent(executorType string, requiredLabels []string) (*AgentInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var candidates []*AgentInfo
	for _, a := range p.agents {
		if a.Status != "online" {
			continue
		}
		if a.ActiveJobs >= a.MaxJobs {
			continue
		}
		if executorType != "" && a.Executor != executorType {
			continue
		}
		if !hasAllLabels(a.Labels, requiredLabels) {
			continue
		}
		candidates = append(candidates, a)
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available agent matching executor=%s labels=%v", executorType, requiredLabels)
	}

	// Select least loaded agent (lowest active_jobs / max_jobs ratio)
	best := candidates[0]
	bestLoad := float64(best.ActiveJobs) / float64(best.MaxJobs)
	for _, c := range candidates[1:] {
		load := float64(c.ActiveJobs) / float64(c.MaxJobs)
		if load < bestLoad {
			best = c
			bestLoad = load
		}
	}

	copy := *best
	return &copy, nil
}

// IncrementActiveJobs increases the active job count for an agent.
func (p *Pool) IncrementActiveJobs(agentID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	agent, ok := p.agents[agentID]
	if !ok {
		return fmt.Errorf("agent %s not found", agentID)
	}

	agent.ActiveJobs++
	if agent.ActiveJobs >= agent.MaxJobs {
		agent.Status = "busy"
	}
	return nil
}

// DecrementActiveJobs decreases the active job count for an agent.
func (p *Pool) DecrementActiveJobs(agentID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	agent, ok := p.agents[agentID]
	if !ok {
		return fmt.Errorf("agent %s not found", agentID)
	}

	if agent.ActiveJobs > 0 {
		agent.ActiveJobs--
	}
	if agent.Status == "busy" && agent.ActiveJobs < agent.MaxJobs {
		agent.Status = "online"
	}
	return nil
}

// MarkOffline marks agents as offline if they haven't been seen within the timeout.
func (p *Pool) MarkOffline(timeout time.Duration) []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	cutoff := time.Now().Add(-timeout)
	var evicted []string

	for _, a := range p.agents {
		if a.Status != "offline" && a.LastSeenAt.Before(cutoff) {
			a.Status = "offline"
			evicted = append(evicted, a.ID)
		}
	}

	return evicted
}

// Count returns the total number of agents in the pool.
func (p *Pool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.agents)
}

// OnlineCount returns the number of online agents.
func (p *Pool) OnlineCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, a := range p.agents {
		if a.Status == "online" || a.Status == "busy" {
			count++
		}
	}
	return count
}

// CountByLabels counts agents matching the given executor type and labels that are online or busy.
func (p *Pool) CountByLabels(executorType, labels string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Parse comma-separated labels
	var required []string
	if labels != "" {
		for _, l := range splitLabels(labels) {
			l = trimSpace(l)
			if l != "" {
				required = append(required, l)
			}
		}
	}

	count := 0
	for _, a := range p.agents {
		if a.Status != "online" && a.Status != "busy" {
			continue
		}
		if executorType != "" && a.Executor != executorType {
			continue
		}
		if !hasAllLabels(a.Labels, required) {
			continue
		}
		count++
	}
	return count
}

// BusyCount returns the number of busy agents.
func (p *Pool) BusyCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, a := range p.agents {
		if a.Status == "busy" {
			count++
		}
	}
	return count
}

// CountByExecutor returns a map of executor type to agent count (online or busy agents).
func (p *Pool) CountByExecutor() map[string]int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]int)
	for _, a := range p.agents {
		if a.Status == "online" || a.Status == "busy" {
			result[a.Executor]++
		}
	}
	return result
}

// CountByLabel returns a map of label to agent count (online or busy agents).
func (p *Pool) CountByLabel() map[string]int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]int)
	for _, a := range p.agents {
		if a.Status == "online" || a.Status == "busy" {
			for _, l := range a.Labels {
				result[l]++
			}
		}
	}
	return result
}

// splitLabels splits a comma-separated labels string into a slice.
func splitLabels(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

// trimSpace trims leading and trailing whitespace from a string.
func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// hasAllLabels checks if agentLabels contains all required labels.
func hasAllLabels(agentLabels, required []string) bool {
	if len(required) == 0 {
		return true
	}
	labelSet := make(map[string]struct{}, len(agentLabels))
	for _, l := range agentLabels {
		labelSet[l] = struct{}{}
	}
	for _, r := range required {
		if _, ok := labelSet[r]; !ok {
			return false
		}
	}
	return true
}
