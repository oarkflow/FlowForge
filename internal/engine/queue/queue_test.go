package queue

import (
	"sync"
	"testing"
	"time"
)

func TestPriorityQueue_EnqueueDequeue(t *testing.T) {
	pq := NewPriorityQueue()
	defer pq.Close()

	job := &Job{ID: "j1", Priority: 5, CreatedAt: time.Now()}
	if err := pq.Enqueue(job); err != nil {
		t.Fatal(err)
	}
	if pq.Len() != 1 {
		t.Errorf("Len() = %d, want 1", pq.Len())
	}

	got, err := pq.TryDequeue()
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "j1" {
		t.Errorf("dequeued job ID = %q, want %q", got.ID, "j1")
	}
	if pq.Len() != 0 {
		t.Errorf("Len() = %d, want 0", pq.Len())
	}
}

func TestPriorityQueue_PriorityOrdering(t *testing.T) {
	pq := NewPriorityQueue()
	defer pq.Close()

	now := time.Now()
	pq.Enqueue(&Job{ID: "low", Priority: 1, CreatedAt: now})
	pq.Enqueue(&Job{ID: "high", Priority: 10, CreatedAt: now})
	pq.Enqueue(&Job{ID: "mid", Priority: 5, CreatedAt: now})

	// Should come out: high (10), mid (5), low (1)
	j1, _ := pq.TryDequeue()
	if j1.ID != "high" {
		t.Errorf("first dequeue = %q, want %q", j1.ID, "high")
	}
	j2, _ := pq.TryDequeue()
	if j2.ID != "mid" {
		t.Errorf("second dequeue = %q, want %q", j2.ID, "mid")
	}
	j3, _ := pq.TryDequeue()
	if j3.ID != "low" {
		t.Errorf("third dequeue = %q, want %q", j3.ID, "low")
	}
}

func TestPriorityQueue_FIFOForSamePriority(t *testing.T) {
	pq := NewPriorityQueue()
	defer pq.Close()

	now := time.Now()
	pq.Enqueue(&Job{ID: "first", Priority: 5, CreatedAt: now})
	pq.Enqueue(&Job{ID: "second", Priority: 5, CreatedAt: now.Add(time.Second)})
	pq.Enqueue(&Job{ID: "third", Priority: 5, CreatedAt: now.Add(2 * time.Second)})

	j1, _ := pq.TryDequeue()
	j2, _ := pq.TryDequeue()
	j3, _ := pq.TryDequeue()

	if j1.ID != "first" || j2.ID != "second" || j3.ID != "third" {
		t.Errorf("FIFO order violated: got %s, %s, %s", j1.ID, j2.ID, j3.ID)
	}
}

func TestPriorityQueue_TryDequeue_EmptyQueue(t *testing.T) {
	pq := NewPriorityQueue()
	defer pq.Close()

	job, err := pq.TryDequeue()
	if err != nil {
		t.Fatal(err)
	}
	if job != nil {
		t.Error("TryDequeue on empty queue should return nil")
	}
}

func TestPriorityQueue_Cancel(t *testing.T) {
	pq := NewPriorityQueue()
	defer pq.Close()

	pq.Enqueue(&Job{ID: "cancel-me", Priority: 10, CreatedAt: time.Now()})
	pq.Enqueue(&Job{ID: "keep-me", Priority: 5, CreatedAt: time.Now()})

	ok := pq.Cancel("cancel-me")
	if !ok {
		t.Error("Cancel should return true for existing job")
	}

	// Cancelled job should be skipped
	got, _ := pq.TryDequeue()
	if got.ID != "keep-me" {
		t.Errorf("dequeued = %q, want %q (cancelled job should be skipped)", got.ID, "keep-me")
	}
}

func TestPriorityQueue_Cancel_NotFound(t *testing.T) {
	pq := NewPriorityQueue()
	defer pq.Close()

	ok := pq.Cancel("nonexistent")
	if ok {
		t.Error("Cancel should return false for nonexistent job")
	}
}

func TestPriorityQueue_DuplicateID(t *testing.T) {
	pq := NewPriorityQueue()
	defer pq.Close()

	pq.Enqueue(&Job{ID: "dup", Priority: 1, CreatedAt: time.Now()})
	err := pq.Enqueue(&Job{ID: "dup", Priority: 2, CreatedAt: time.Now()})
	if err == nil {
		t.Error("should reject duplicate job ID")
	}
}

func TestPriorityQueue_Close_RejectsEnqueue(t *testing.T) {
	pq := NewPriorityQueue()
	pq.Close()

	err := pq.Enqueue(&Job{ID: "after-close", Priority: 1, CreatedAt: time.Now()})
	if err == nil {
		t.Error("should reject enqueue after close")
	}
}

func TestPriorityQueue_Dequeue_BlocksUntilJob(t *testing.T) {
	pq := NewPriorityQueue()
	defer pq.Close()

	var got *Job
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		got, err = pq.Dequeue()
		if err != nil {
			t.Errorf("Dequeue error: %v", err)
		}
	}()

	// Give goroutine time to block
	time.Sleep(50 * time.Millisecond)

	pq.Enqueue(&Job{ID: "delayed", Priority: 1, CreatedAt: time.Now()})
	wg.Wait()

	if got == nil || got.ID != "delayed" {
		t.Errorf("Dequeue should unblock with the job, got %v", got)
	}
}

func TestPriorityQueue_Dequeue_UnblocksOnClose(t *testing.T) {
	pq := NewPriorityQueue()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := pq.Dequeue()
		if err == nil {
			t.Error("Dequeue should return error when queue is closed")
		}
	}()

	time.Sleep(50 * time.Millisecond)
	pq.Close()
	wg.Wait()
}

func TestPriorityQueue_Len_ExcludesCancelled(t *testing.T) {
	pq := NewPriorityQueue()
	defer pq.Close()

	pq.Enqueue(&Job{ID: "a", Priority: 1, CreatedAt: time.Now()})
	pq.Enqueue(&Job{ID: "b", Priority: 1, CreatedAt: time.Now()})
	pq.Enqueue(&Job{ID: "c", Priority: 1, CreatedAt: time.Now()})

	if pq.Len() != 3 {
		t.Errorf("Len() = %d, want 3", pq.Len())
	}

	pq.Cancel("b")
	if pq.Len() != 2 {
		t.Errorf("Len() after cancel = %d, want 2", pq.Len())
	}
}

func TestPriorityQueue_ConcurrentEnqueueDequeue(t *testing.T) {
	pq := NewPriorityQueue()
	defer pq.Close()

	const n = 100
	var wg sync.WaitGroup

	// Enqueue concurrently
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			pq.Enqueue(&Job{
				ID:        string(rune('A'+id%26)) + time.Now().String(),
				Priority:  id % 10,
				CreatedAt: time.Now(),
			})
		}(i)
	}
	wg.Wait()

	// Dequeue all
	count := 0
	for {
		j, _ := pq.TryDequeue()
		if j == nil {
			break
		}
		count++
	}
	if count != n {
		t.Errorf("dequeued %d jobs, want %d", count, n)
	}
}
