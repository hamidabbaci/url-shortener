package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) Save(link Link) error {
	_, err := s.pool.Exec(context.Background(),
		`INSERT INTO links (code, original_url, created_at, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		link.Code, link.OriginalURL, link.CreatedAt, link.ExpiresAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrCodeExists
		}
		return err
	}
	return nil
}

func (s *PostgresStore) Get(code string) (Link, error) {
	var link Link
	err := s.pool.QueryRow(context.Background(),
		`SELECT code, original_url, created_at, expires_at
		 FROM links WHERE code = $1`, code,
	).Scan(&link.Code, &link.OriginalURL, &link.CreatedAt, &link.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Link{}, ErrNotFound
		}
		return Link{}, err
	}
	if time.Now().UTC().After(link.ExpiresAt) {
		return Link{}, ErrExpired
	}
	return link, nil
}
func (s *PostgresStore) Close() {
	s.pool.Close()
}
