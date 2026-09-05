package app

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hokm/platform/internal/auth"
)

// User is the account model. Password hashes never leave this package.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	AvatarSeed   string    `json:"avatar_seed,omitempty"`
	AvatarStyle  string    `json:"avatar_style,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	IsGuest      bool      `json:"is_guest"`
}

// UserRepo persists users. Postgres implementation lands in Phase 11.
type UserRepo interface {
	Create(u *User) error
	ByUsername(username string) (*User, error)
	ByID(id string) (*User, error)
	UpdateAvatar(id, seed, style string) error
}

var (
	ErrUsernameTaken  = errors.New("app: username already taken")
	ErrEmailTaken     = errors.New("app: email already taken")
	ErrUserNotFound   = errors.New("app: user not found")
	ErrBadCredentials = errors.New("app: invalid username or password")
	ErrValidation     = errors.New("app: validation failed")
)

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{3,24}$`)
var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// TokenPair is an access/refresh token pair.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // access TTL seconds
}

// UserService implements registration, login, and refresh flows.
type UserService struct {
	repo       UserRepo
	tokens     *auth.TokenManager
	refresh    auth.RefreshStore
	refreshTTL time.Duration
}

func NewUserService(repo UserRepo, tokens *auth.TokenManager, refresh auth.RefreshStore, refreshTTL time.Duration) *UserService {
	return &UserService{repo: repo, tokens: tokens, refresh: refresh, refreshTTL: refreshTTL}
}

// Register validates input, hashes the password, and persists the user.
func (s *UserService) Register(username, email, password, avatarSeed, avatarStyle string) (*User, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	email = strings.ToLower(strings.TrimSpace(email))
	style, seed, err := ValidateAvatarChoice(avatarStyle, avatarSeed)
	if err != nil {
		return nil, err
	}
	if !usernameRe.MatchString(username) {
		return nil, fmt.Errorf("%w: username must be 3-24 chars of letters, digits or underscore", ErrValidation)
	}
	if !emailRe.MatchString(email) {
		return nil, fmt.Errorf("%w: invalid email", ErrValidation)
	}
	if len(password) < 8 || len(password) > 128 {
		return nil, fmt.Errorf("%w: password must be 8-128 characters", ErrValidation)
	}
	if _, err := s.repo.ByUsername(username); err == nil {
		return nil, ErrUsernameTaken
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	u := &User{
		ID:           newID(),
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		AvatarSeed:   seed,
		AvatarStyle:  style,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.repo.Create(u); err != nil {
		return nil, err
	}
	return u, nil
}

// Login verifies credentials and issues a token pair.
func (s *UserService) Login(username, password string) (*User, *TokenPair, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	u, err := s.repo.ByUsername(username)
	if err != nil {
		// Same error for unknown user and wrong password (no enumeration).
		return nil, nil, ErrBadCredentials
	}
	if !auth.VerifyPassword(u.PasswordHash, password) {
		return nil, nil, ErrBadCredentials
	}
	pair, err := s.issuePair(u.ID)
	if err != nil {
		return nil, nil, err
	}
	return u, pair, nil
}

// Refresh rotates a refresh token and issues a new pair.
func (s *UserService) Refresh(token string) (*TokenPair, error) {
	hash := auth.HashRefreshToken(token)
	userID, err := s.refresh.Consume(hash)
	if err != nil {
		return nil, err
	}
	pair, err := s.issuePair(userID)
	if err != nil {
		return nil, err
	}
	return pair, nil
}

// UpdateAvatar persists a whitelisted avatar style+seed for the user.
func (s *UserService) UpdateAvatar(userID, avatarSeed, avatarStyle string) (*User, error) {
	style, seed, err := ValidateAvatarChoice(avatarStyle, avatarSeed)
	if err != nil {
		return nil, err
	}
	if seed == "" {
		return nil, fmt.Errorf("%w: avatar_seed is required", ErrValidation)
	}
	u, err := s.repo.ByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if err := s.repo.UpdateAvatar(userID, seed, style); err != nil {
		return nil, err
	}
	u.AvatarSeed = seed
	u.AvatarStyle = style
	return u, nil
}

// Profile returns the public profile of a user.
func (s *UserService) Profile(userID string) (*User, error) {
	u, err := s.repo.ByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (s *UserService) issuePair(userID string) (*TokenPair, error) {
	access, err := s.tokens.IssueAccess(userID)
	if err != nil {
		return nil, err
	}
	refresh, hash, err := auth.NewRefreshToken()
	if err != nil {
		return nil, err
	}
	if err := s.refresh.Save(hash, userID, time.Now().Add(s.refreshTTL)); err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(s.refreshTTL.Seconds()),
	}, nil
}
