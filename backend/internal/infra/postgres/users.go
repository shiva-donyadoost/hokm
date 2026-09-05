package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hokm/platform/internal/app"
	"github.com/hokm/platform/internal/auth"
)

// isUniqueViolation reports PostgreSQL unique-violation (23505) errors.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// UserStore is the durable app.UserRepo.
type UserStore struct {
	pool *pgxpool.Pool
}

func NewUserStore(pool *pgxpool.Pool) *UserStore { return &UserStore{pool: pool} }

func scanUser(row pgx.Row) (*app.User, error) {
	var u app.User
	var avatar *string
	var style *string
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.IsGuest, &u.CreatedAt, &avatar, &style)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, app.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: scan user: %w", err)
	}
	if avatar != nil {
		u.AvatarSeed = *avatar
	}
	if style != nil {
		u.AvatarStyle = *style
	}
	return &u, nil
}

const userCols = `id, username, email, password_hash, is_guest, created_at, avatar_seed, avatar_style`

func (s *UserStore) Create(u *app.User) error {
	_, err := s.pool.Exec(context.Background(),
		`INSERT INTO users (id, username, email, password_hash, is_guest, created_at, avatar_seed, avatar_style)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		u.ID, u.Username, u.Email, u.PasswordHash, u.IsGuest, u.CreatedAt, nullIfEmpty(u.AvatarSeed), nullIfEmpty(u.AvatarStyle),
	)
	if err != nil {
		// Unique violation → translate to domain errors.
		if isUniqueViolation(err) {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.ConstraintName == "users_email_key" {
				return app.ErrEmailTaken
			}
			return app.ErrUsernameTaken
		}
		return fmt.Errorf("postgres: insert user: %w", err)
	}
	return nil
}

func (s *UserStore) ByUsername(username string) (*app.User, error) {
	return scanUser(s.pool.QueryRow(context.Background(),
		`SELECT `+userCols+` FROM users WHERE lower(username) = lower($1)`, username))
}

func (s *UserStore) ByID(id string) (*app.User, error) {
	return scanUser(s.pool.QueryRow(context.Background(),
		`SELECT `+userCols+` FROM users WHERE id = $1`, id))
}

// RefreshStore is the durable single-use refresh-token store.
type RefreshStore struct {
	pool *pgxpool.Pool
}

func NewRefreshStore(pool *pgxpool.Pool) *RefreshStore { return &RefreshStore{pool: pool} }

func (s *RefreshStore) Save(hash, userID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(context.Background(),
		`INSERT INTO refresh_tokens (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		hash, userID, expiresAt)
	if err != nil {
		return fmt.Errorf("postgres: save refresh: %w", err)
	}
	return nil
}

// Consume atomically deletes the token and returns its owner; single use.
func (s *RefreshStore) Consume(hash string) (string, error) {
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("postgres: consume refresh begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID string
	var expiresAt time.Time
	err = tx.QueryRow(ctx,
		`SELECT user_id, expires_at FROM refresh_tokens WHERE token_hash = $1 FOR UPDATE`, hash,
	).Scan(&userID, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", auth.ErrInvalidToken
	}
	if err != nil {
		return "", fmt.Errorf("postgres: consume refresh select: %w", err)
	}
	if time.Now().After(expiresAt) {
		_, _ = tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, hash)
		_ = tx.Commit(ctx)
		return "", auth.ErrInvalidToken
	}
	if _, err := tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, hash); err != nil {
		return "", fmt.Errorf("postgres: consume refresh delete: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("postgres: consume refresh commit: %w", err)
	}
	return userID, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *UserStore) UpdateAvatar(id, seed, style string) error {
	tag, err := s.pool.Exec(context.Background(),
		`UPDATE users SET avatar_seed = $2, avatar_style = $3 WHERE id = $1`, id, seed, style)
	if err != nil {
		return fmt.Errorf("postgres: update avatar: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrUserNotFound
	}
	return nil
}
