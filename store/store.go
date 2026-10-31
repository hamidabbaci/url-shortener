package store

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrNotFound   = errors.New("link not found")
	ErrCodeExists = errors.New("code already exists")
)

type Link struct {
	Code        string
	OriginalURL string
	CreatedAt   time.Time
}

type MemoryStore struct {
	mu    sync.RWMutex
	links map[string]Link
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		links: make(map[string]Link),
	}
}

func (s *MemoryStore) Save(link Link) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.links[link.Code]; exists {
		return ErrCodeExists
	}
	s.links[link.Code] = link
	return nil
}

func (s *MemoryStore) Get(code string) (Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	link, ok := s.links[code]
	if !ok {
		return Link{}, ErrNotFound
	}
	return link, nil
}
