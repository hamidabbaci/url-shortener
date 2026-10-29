package store

import (
	"errors"
	"time"
)

// خطاهایی که ممکنه پیش بیاد
var (
	ErrNotFound   = errors.New("link not found")
	ErrCodeExists = errors.New("code already exists")
)

// Link مدل اصلی ماست
type Link struct {
	Code        string
	OriginalURL string
	CreatedAt   time.Time
}

// MemoryStore لینک‌ها رو تو حافظه نگه میداره
type MemoryStore struct {
	links map[string]Link
}

// NewMemoryStore یه استور خالی می‌سازه
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		links: make(map[string]Link),
	}
}

// Save یه لینک جدید ذخیره میکنه
func (s *MemoryStore) Save(link Link) error {
	if _, exists := s.links[link.Code]; exists {
		return ErrCodeExists
	}
	s.links[link.Code] = link
	return nil
}

// Get لینک رو با کد پیدا میکنه
func (s *MemoryStore) Get(code string) (Link, error) {
	link, ok := s.links[code]
	if !ok {
		return Link{}, ErrNotFound
	}
	return link, nil
}
