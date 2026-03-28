package agent

import (
	"testing"
	"time"
)

func newTestAgent(id string, executor string, labels []string, cpuCores int32) *AgentInfo {
	return &AgentInfo{
		ID:       id,
		Name:     "agent-" + id,
		Labels:   labels,
		Executor: executor,
		OS:       "linux",
		Arch:     "amd64",
		CPUCores: cpuCores,
		MemoryMB: 4096,
		Version:  "1.0",
		MaxJobs:  cpuCores,
	}
}

func TestPool_Register(t *testing.T) {
	p := NewPool()
	agent := newTestAgent("a1", "docker", []string{"linux"}, 4)

	if err := p.Register(agent); err != nil {
		t.Fatal(err)
	}
	if p.Count() != 1 {
		t.Errorf("Count() = %d, want 1", p.Count())
	}

	got, ok := p.Get("a1")
	if !ok {
		t.Fatal("agent not found")
	}
	if got.Status != "online" {
		t.Errorf("Status = %q, want %q", got.Status, "online")
	}
	if got.MaxJobs != 4 {
		t.Errorf("MaxJobs = %d, want 4", got.MaxJobs)
	}
}

func TestPool_Register_DefaultMaxJobs(t *testing.T) {
	p := NewPool()
	agent := &AgentInfo{ID: "a1", CPUCores: 0}
	p.Register(agent)

	got, _ := p.Get("a1")
	if got.MaxJobs != 2 {
		t.Errorf("MaxJobs = %d, want 2 (default)", got.MaxJobs)
	}
}

func TestPool_Register_Update(t *testing.T) {
	p := NewPool()
	p.Register(newTestAgent("a1", "docker", nil, 4))
	p.Register(&AgentInfo{ID: "a1", Name: "updated", Executor: "local"})

	got, _ := p.Get("a1")
	if got.Name != "updated" {
		t.Errorf("Name = %q, want %q (should be updated)", got.Name, "updated")
	}
	if p.Count() != 1 {
		t.Errorf("Count() = %d, want 1 (update should not add)", p.Count())
	}
}

func TestPool_Unregister(t *testing.T) {
	p := NewPool()
	p.Register(newTestAgent("a1", "docker", nil, 4))
	p.Unregister("a1")
	if p.Count() != 0 {
		t.Errorf("Count() = %d, want 0", p.Count())
	}
	_, ok := p.Get("a1")
	if ok {
		t.Error("agent should not be found after unregister")
	}
}

func TestPool_UpdateHeartbeat(t *testing.T) {
	p := NewPool()
	p.Register(newTestAgent("a1", "docker", nil, 4))

	err := p.UpdateHeartbeat("a1", 2, 50.0, 60.0)
	if err != nil {
		t.Fatal(err)
	}

	got, _ := p.Get("a1")
	if got.ActiveJobs != 2 {
		t.Errorf("ActiveJobs = %d, want 2", got.ActiveJobs)
	}
	if got.CPUUsage != 50.0 {
		t.Errorf("CPUUsage = %f, want 50.0", got.CPUUsage)
	}
}

func TestPool_UpdateHeartbeat_NotFound(t *testing.T) {
	p := NewPool()
	err := p.UpdateHeartbeat("nonexistent", 0, 0, 0)
	if err == nil {
		t.Error("should return error for unknown agent")
	}
}

func TestPool_UpdateHeartbeat_RestoredOnline(t *testing.T) {
	p := NewPool()
	agent := newTestAgent("a1", "docker", nil, 4)
	p.Register(agent)

	// Mark offline manually
	p.SetStatus("a1", "offline")
	got, _ := p.Get("a1")
	if got.Status != "offline" {
		t.Fatalf("Status should be offline, got %q", got.Status)
	}

	// Heartbeat should restore to online
	p.UpdateHeartbeat("a1", 0, 0, 0)
	got, _ = p.Get("a1")
	if got.Status != "online" {
		t.Errorf("Status = %q, want %q after heartbeat", got.Status, "online")
	}
}

func TestPool_SelectAgent_LeastLoaded(t *testing.T) {
	p := NewPool()
	a1 := newTestAgent("a1", "docker", nil, 4)
	a2 := newTestAgent("a2", "docker", nil, 4)
	p.Register(a1)
	p.Register(a2)

	// Make a1 busy
	p.IncrementActiveJobs("a1")
	p.IncrementActiveJobs("a1")

	selected, err := p.SelectAgent("docker", nil)
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != "a2" {
		t.Errorf("should select least loaded agent a2, got %q", selected.ID)
	}
}

func TestPool_SelectAgent_ByExecutor(t *testing.T) {
	p := NewPool()
	p.Register(newTestAgent("docker-agent", "docker", nil, 4))
	p.Register(newTestAgent("k8s-agent", "kubernetes", nil, 4))

	selected, err := p.SelectAgent("kubernetes", nil)
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != "k8s-agent" {
		t.Errorf("should select k8s executor, got %q", selected.ID)
	}
}

func TestPool_SelectAgent_ByLabels(t *testing.T) {
	p := NewPool()
	p.Register(newTestAgent("a1", "docker", []string{"gpu", "linux"}, 4))
	p.Register(newTestAgent("a2", "docker", []string{"linux"}, 4))

	selected, err := p.SelectAgent("docker", []string{"gpu"})
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != "a1" {
		t.Errorf("should select agent with gpu label, got %q", selected.ID)
	}
}

func TestPool_SelectAgent_NoMatch(t *testing.T) {
	p := NewPool()
	p.Register(newTestAgent("a1", "docker", nil, 4))

	_, err := p.SelectAgent("kubernetes", nil)
	if err == nil {
		t.Error("should return error when no matching agent")
	}
}

func TestPool_SelectAgent_SkipsBusy(t *testing.T) {
	p := NewPool()
	agent := newTestAgent("a1", "docker", nil, 1)
	p.Register(agent)

	p.IncrementActiveJobs("a1") // now at capacity

	_, err := p.SelectAgent("docker", nil)
	if err == nil {
		t.Error("should not select an agent at max capacity")
	}
}

func TestPool_IncrementActiveJobs_SetsBusy(t *testing.T) {
	p := NewPool()
	agent := newTestAgent("a1", "docker", nil, 2)
	p.Register(agent)

	p.IncrementActiveJobs("a1")
	got, _ := p.Get("a1")
	if got.Status != "online" {
		t.Errorf("Status = %q, want online (not at capacity yet)", got.Status)
	}

	p.IncrementActiveJobs("a1")
	got, _ = p.Get("a1")
	if got.Status != "busy" {
		t.Errorf("Status = %q, want busy (at capacity)", got.Status)
	}
}

func TestPool_DecrementActiveJobs_RestoresOnline(t *testing.T) {
	p := NewPool()
	agent := newTestAgent("a1", "docker", nil, 1)
	p.Register(agent)

	p.IncrementActiveJobs("a1") // busy
	p.DecrementActiveJobs("a1") // back to online

	got, _ := p.Get("a1")
	if got.Status != "online" {
		t.Errorf("Status = %q, want online after decrement", got.Status)
	}
}

func TestPool_DecrementActiveJobs_NoNegative(t *testing.T) {
	p := NewPool()
	p.Register(newTestAgent("a1", "docker", nil, 4))

	// Decrement when already 0
	p.DecrementActiveJobs("a1")
	got, _ := p.Get("a1")
	if got.ActiveJobs != 0 {
		t.Errorf("ActiveJobs = %d, should not go negative", got.ActiveJobs)
	}
}

func TestPool_MarkOffline(t *testing.T) {
	p := NewPool()
	agent := newTestAgent("a1", "docker", nil, 4)
	agent.LastSeenAt = time.Now().Add(-10 * time.Minute)
	p.Register(agent)
	// Override LastSeenAt after registration
	p.mu.Lock()
	p.agents["a1"].LastSeenAt = time.Now().Add(-10 * time.Minute)
	p.mu.Unlock()

	evicted := p.MarkOffline(5 * time.Minute)
	if len(evicted) != 1 || evicted[0] != "a1" {
		t.Errorf("evicted = %v, want [a1]", evicted)
	}

	got, _ := p.Get("a1")
	if got.Status != "offline" {
		t.Errorf("Status = %q, want offline", got.Status)
	}
}

func TestPool_Available(t *testing.T) {
	p := NewPool()
	p.Register(newTestAgent("a1", "docker", nil, 4))
	p.Register(newTestAgent("a2", "docker", nil, 1))

	// Fill a2 to capacity
	p.IncrementActiveJobs("a2")

	available := p.Available()
	if len(available) != 1 {
		t.Errorf("Available() = %d, want 1", len(available))
	}
	if available[0].ID != "a1" {
		t.Errorf("available agent ID = %q, want a1", available[0].ID)
	}
}

func TestPool_List(t *testing.T) {
	p := NewPool()
	p.Register(newTestAgent("a1", "docker", nil, 4))
	p.Register(newTestAgent("a2", "local", nil, 2))

	list := p.List()
	if len(list) != 2 {
		t.Errorf("List() = %d, want 2", len(list))
	}
}

func TestPool_OnlineCount(t *testing.T) {
	p := NewPool()
	p.Register(newTestAgent("a1", "docker", nil, 4))
	p.Register(newTestAgent("a2", "docker", nil, 2))

	if p.OnlineCount() != 2 {
		t.Errorf("OnlineCount() = %d, want 2", p.OnlineCount())
	}

	p.SetStatus("a2", "offline")
	if p.OnlineCount() != 1 {
		t.Errorf("OnlineCount() = %d, want 1 after offline", p.OnlineCount())
	}
}

func TestPool_BusyCount(t *testing.T) {
	p := NewPool()
	a := newTestAgent("a1", "docker", nil, 1)
	p.Register(a)

	if p.BusyCount() != 0 {
		t.Error("BusyCount should be 0 initially")
	}

	p.IncrementActiveJobs("a1")
	if p.BusyCount() != 1 {
		t.Errorf("BusyCount() = %d, want 1", p.BusyCount())
	}
}

func TestPool_CountByExecutor(t *testing.T) {
	p := NewPool()
	p.Register(newTestAgent("a1", "docker", nil, 4))
	p.Register(newTestAgent("a2", "docker", nil, 4))
	p.Register(newTestAgent("a3", "kubernetes", nil, 4))

	counts := p.CountByExecutor()
	if counts["docker"] != 2 {
		t.Errorf("docker count = %d, want 2", counts["docker"])
	}
	if counts["kubernetes"] != 1 {
		t.Errorf("kubernetes count = %d, want 1", counts["kubernetes"])
	}
}

func TestPool_CountByLabel(t *testing.T) {
	p := NewPool()
	p.Register(newTestAgent("a1", "docker", []string{"gpu", "linux"}, 4))
	p.Register(newTestAgent("a2", "docker", []string{"linux"}, 4))

	counts := p.CountByLabel()
	if counts["linux"] != 2 {
		t.Errorf("linux count = %d, want 2", counts["linux"])
	}
	if counts["gpu"] != 1 {
		t.Errorf("gpu count = %d, want 1", counts["gpu"])
	}
}

func TestHasAllLabels(t *testing.T) {
	tests := []struct {
		agent    []string
		required []string
		want     bool
	}{
		{[]string{"a", "b", "c"}, []string{"a", "b"}, true},
		{[]string{"a", "b"}, []string{"a", "b", "c"}, false},
		{[]string{"a"}, nil, true},
		{nil, nil, true},
		{nil, []string{"a"}, false},
	}
	for _, tt := range tests {
		got := hasAllLabels(tt.agent, tt.required)
		if got != tt.want {
			t.Errorf("hasAllLabels(%v, %v) = %v, want %v", tt.agent, tt.required, got, tt.want)
		}
	}
}

func TestSplitLabels(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"a,b,c", 3},
		{"single", 1},
		{"", 1}, // splits "" into [""]
	}
	for _, tt := range tests {
		got := splitLabels(tt.input)
		if len(got) != tt.want {
			t.Errorf("splitLabels(%q) = %d parts, want %d", tt.input, len(got), tt.want)
		}
	}
}

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"  hello  ", "hello"},
		{"hello", "hello"},
		{"\thello\t", "hello"},
		{"  ", ""},
	}
	for _, tt := range tests {
		got := trimSpace(tt.input)
		if got != tt.want {
			t.Errorf("trimSpace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
