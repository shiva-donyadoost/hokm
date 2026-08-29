package auth

import (
	"sync"
	"time"
)

// MemoryRefreshStore is an in-memory RefreshStore for development and
// tests. The production implementation (Postgres) arrives in Phase 11.
type MemoryRefreshStore struct {
	mu      sync.Mutex
	entries map[string]memoryRefresh
}

type memoryRefresh struct {
	userID    string
	expiresAt time.Time
}

func NewMemoryRefreshStore() *MemoryRefreshStore {
	return &MemoryRefreshStore{entries: make(map[string]memoryRefresh)}
}

func (s *MemoryRefreshStore) Save(hash, userID string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[hash] = memoryRefresh{userID: userID, expiresAt: expiresAt}
	return nil
}

func (s *MemoryRefreshStore) Consume(hash string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[hash]
	if !ok || time.Now().After(e.expiresAt) {
		return "", ErrInvalidToken
	}
	delete(s.entries, hash)
	return e.userID, nil
}
