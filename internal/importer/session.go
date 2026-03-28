package importer

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const sessionTTL = 30 * time.Minute

// session tracks a temporary working directory for an import flow.
type session struct {
	ID        string
	WorkDir   string
	Ephemeral bool // true when WorkDir is a FlowForge temp dir that we created and should clean up
	CreatedAt time.Time
}

// SessionStore manages temporary directories between API calls
// (e.g. between /detect and /project creation).
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*session
	done     chan struct{}
}

// NewSessionStore creates a new store and starts a background sweeper.
func NewSessionStore() *SessionStore {
	s := &SessionStore{
		sessions: make(map[string]*session),
		done:     make(chan struct{}),
	}
	go s.sweepLoop()
	return s
}

// isTempDir returns true when the path looks like a FlowForge-managed temp
// directory that is safe to remove.
func isTempDir(path string) bool {
	return strings.Contains(path, "flowforge-import-") || strings.Contains(path, "flowforge-upload-")
}

// Create stores a working directory and returns a session ID.
func (s *SessionStore) Create(workDir string) string {
	id := uuid.New().String()
	s.mu.Lock()
	s.sessions[id] = &session{
		ID:        id,
		WorkDir:   workDir,
		Ephemeral: isTempDir(workDir),
		CreatedAt: time.Now(),
	}
	s.mu.Unlock()
	return id
}

// Get returns the working directory for a session ID, or empty string if not found/expired.
func (s *SessionStore) Get(id string) string {
	s.mu.RLock()
	sess, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok {
		return ""
	}
	if time.Since(sess.CreatedAt) > sessionTTL {
		s.Remove(id)
		return ""
	}
	return sess.WorkDir
}

// Remove deletes a session and cleans up its working directory ONLY if it is
// a FlowForge-managed temp directory. User-owned local paths are NEVER deleted.
func (s *SessionStore) Remove(id string) {
	s.mu.Lock()
	sess, ok := s.sessions[id]
	if ok {
		delete(s.sessions, id)
	}
	s.mu.Unlock()

	if ok && sess.Ephemeral && sess.WorkDir != "" {
		os.RemoveAll(sess.WorkDir)
	}
}

// Stop halts the background sweeper.
func (s *SessionStore) Stop() {
	close(s.done)
}

// sweepLoop periodically removes expired sessions.
func (s *SessionStore) sweepLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.sweep()
		}
	}
}

func (s *SessionStore) sweep() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, sess := range s.sessions {
		if now.Sub(sess.CreatedAt) > sessionTTL {
			// Only remove temp directories we created — never user-owned paths.
			if sess.Ephemeral && sess.WorkDir != "" {
				os.RemoveAll(sess.WorkDir)
			}
			delete(s.sessions, id)
		}
	}
}
