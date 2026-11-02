package store

import (
	"errors"
	"time"
)

type Link struct {
	Code        string
	OriginalURL string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

var (
	ErrNotFound   = errors.New("link not found")
	ErrCodeExists = errors.New("code already exists")
	ErrExpired    = errors.New("link expired")
)
