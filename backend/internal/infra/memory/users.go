// Package memory provides in-memory implementations of the application
// repository interfaces. Used in development and tests; Postgres replaces
// them in Phase 11 behind the same interfaces.
package memory

import (
	"strings"
	"sync"

	"github.com/hokm/platform/internal/app"
)

// UserStore is an in-memory app.UserRepo.
type UserStore struct {
	mu         sync.RWMutex
	byID       map[string]*app.User
	byUsername map[string]*app.User
	byEmail    map[string]*app.User
}

func NewUserStore() *UserStore {
	return &UserStore{
		byID:       make(map[string]*app.User),
		byUsername: make(map[string]*app.User),
		byEmail:    make(map[string]*app.User),
	}
}

func (s *UserStore) Create(u *app.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byUsername[strings.ToLower(u.Username)]; ok {
		return app.ErrUsernameTaken
	}
	if _, ok := s.byEmail[strings.ToLower(u.Email)]; ok {
		return app.ErrEmailTaken
	}
	cp := *u
	s.byID[u.ID] = &cp
	s.byUsername[strings.ToLower(u.Username)] = &cp
	s.byEmail[strings.ToLower(u.Email)] = &cp
	return nil
}

func (s *UserStore) ByUsername(username string) (*app.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.byUsername[strings.ToLower(username)]
	if !ok {
		return nil, app.ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (s *UserStore) ByID(id string) (*app.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.byID[id]
	if !ok {
		return nil, app.ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (s *UserStore) UpdateAvatar(id, seed, style string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.byID[id]
	if !ok {
		return app.ErrUserNotFound
	}
	u.AvatarSeed = seed
	u.AvatarStyle = style
	return nil
}
