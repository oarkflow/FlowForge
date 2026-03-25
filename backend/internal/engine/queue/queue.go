package queue

import (
	"container/heap"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// Job represents a unit of work to be dispatched by the scheduler.
type Job struct {
	ID            string          `json:"id"`
	PipelineRunID string          `json:"pipeline_run_id"`
	Priority      int             `json:"priority"` // Higher value = higher priority
	CreatedAt     time.Time       `json:"created_at"`
	Config        json.RawMessage `json:"config"`
	Cancelled     bool            `json:"-"`
}

// PriorityQueue is a thread-safe, in-memory priority queue for jobs.
// Jobs with higher Priority values are dequeued first. Among jobs with equal
// priority, the one created earliest is dequeued first (FIFO).
type PriorityQueue struct {
	mu    sync.Mutex
	cond  *sync.Cond
	items jobHeap
	index map[string]int // job ID -> position in heap for O(1) cancel
	closed bool
}

// NewPriorityQueue creates a new empty PriorityQueue.
func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		items: make(jobHeap, 0),
		index: make(map[string]int),
	}
	pq.cond = sync.NewCond(&pq.mu)
	heap.Init(&pq.items)
	return pq
}

// Enqueue adds a job to the priority queue.
func (pq *PriorityQueue) Enqueue(job *Job) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.closed {
		return errors.New("queue is closed")
	}

	if _, exists := pq.index[job.ID]; exists {
		return errors.New("job already in queue: " + job.ID)
	}

	entry := &jobEntry{
		job:   job,
		index: len(pq.items),
	}
	heap.Push(&pq.items, entry)
	pq.index[job.ID] = entry.index

	// Signal one waiting dequeue goroutine
	pq.cond.Signal()
	return nil
}

// Dequeue removes and returns the highest-priority job from the queue.
// It blocks until a job is available or the queue is closed.
func (pq *PriorityQueue) Dequeue() (*Job, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	for {
		// Skip cancelled jobs at the top
		for pq.items.Len() > 0 {
			top := pq.items[0]
			if top.job.Cancelled {
				entry := heap.Pop(&pq.items).(*jobEntry)
				delete(pq.index, entry.job.ID)
				continue
			}
			break
		}

		if pq.items.Len() > 0 {
			entry := heap.Pop(&pq.items).(*jobEntry)
			delete(pq.index, entry.job.ID)
			return entry.job, nil
		}

		if pq.closed {
			return nil, errors.New("queue is closed and empty")
		}

		// Wait until a job is enqueued or the queue is closed
		pq.cond.Wait()
	}
}

// TryDequeue attempts to dequeue a job without blocking.
// Returns nil, nil if the queue is empty.
func (pq *PriorityQueue) TryDequeue() (*Job, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	// Skip cancelled jobs at the top
	for pq.items.Len() > 0 {
		top := pq.items[0]
		if top.job.Cancelled {
			entry := heap.Pop(&pq.items).(*jobEntry)
			delete(pq.index, entry.job.ID)
			continue
		}
		break
	}

	if pq.items.Len() == 0 {
		return nil, nil
	}

	entry := heap.Pop(&pq.items).(*jobEntry)
	delete(pq.index, entry.job.ID)
	return entry.job, nil
}

// Cancel marks a job as cancelled. Returns true if the job was found and cancelled.
func (pq *PriorityQueue) Cancel(jobID string) bool {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	idx, exists := pq.index[jobID]
	if !exists {
		return false
	}

	if idx >= 0 && idx < len(pq.items) {
		pq.items[idx].job.Cancelled = true
	}
	return true
}

// Len returns the number of non-cancelled jobs in the queue.
func (pq *PriorityQueue) Len() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	count := 0
	for _, entry := range pq.items {
		if !entry.job.Cancelled {
			count++
		}
	}
	return count
}

// Close closes the queue, causing any blocked Dequeue calls to return an error.
func (pq *PriorityQueue) Close() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.closed = true
	pq.cond.Broadcast()
}

// --- Internal heap implementation ---

type jobEntry struct {
	job   *Job
	index int // position in the heap slice, maintained by heap.Interface
}

type jobHeap []*jobEntry

func (h jobHeap) Len() int { return len(h) }

func (h jobHeap) Less(i, j int) bool {
	// Higher priority first
	if h[i].job.Priority != h[j].job.Priority {
		return h[i].job.Priority > h[j].job.Priority
	}
	// Same priority: earlier created first (FIFO)
	return h[i].job.CreatedAt.Before(h[j].job.CreatedAt)
}

func (h jobHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *jobHeap) Push(x interface{}) {
	entry := x.(*jobEntry)
	entry.index = len(*h)
	*h = append(*h, entry)
}

func (h *jobHeap) Pop() interface{} {
	old := *h
	n := len(old)
	entry := old[n-1]
	old[n-1] = nil // avoid memory leak
	entry.index = -1
	*h = old[:n-1]
	return entry
}
