package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/hokm/platform/internal/app"
	"github.com/hokm/platform/internal/auth"
)

type ctxKey string

const ctxUserID ctxKey = "user_id"

// UserIDFromContext returns the authenticated user id, if any.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ctxUserID).(string)
	return id, ok && id != ""
}

// RequireAuth validates the Bearer token and injects the user id.
func (s *Server) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		token, ok := strings.CutPrefix(h, "Bearer ")
		if !ok || token == "" {
			writeError(w, r, apiError(http.StatusUnauthorized, "unauthorized", "missing bearer token"))
			return
		}
		uid, err := s.tokens.VerifyAccess(token)
		if err != nil {
			if errors.Is(err, auth.ErrExpiredToken) {
				writeError(w, r, apiError(http.StatusUnauthorized, "token_expired", "access token expired"))
				return
			}
			writeError(w, r, apiError(http.StatusUnauthorized, "unauthorized", "invalid token"))
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxUserID, uid)))
	})
}

type registerRequest struct {
	Username   string `json:"username"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	AvatarSeed string `json:"avatar_seed"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, r, apiError(http.StatusBadRequest, "bad_request", "invalid JSON body"))
		return
	}
	u, err := s.users.Register(req.Username, req.Email, req.Password, req.AvatarSeed)
	if err != nil {
		writeError(w, r, s.mapAppError(err))
		return
	}
	_, pair, err := s.users.Login(req.Username, req.Password)
	if err != nil {
		writeError(w, r, s.mapAppError(err))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": u, "tokens": pair})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, r, apiError(http.StatusBadRequest, "bad_request", "invalid JSON body"))
		return
	}
	u, pair, err := s.users.Login(req.Username, req.Password)
	if err != nil {
		writeError(w, r, s.mapAppError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u, "tokens": pair})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, r, apiError(http.StatusBadRequest, "bad_request", "invalid JSON body"))
		return
	}
	pair, err := s.users.Refresh(req.RefreshToken)
	if err != nil {
		writeError(w, r, apiError(http.StatusUnauthorized, "invalid_refresh", "refresh token invalid or expired"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": pair})
}

type updateMeRequest struct {
	AvatarSeed string `json:"avatar_seed"`
}

func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	uid, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, apiError(http.StatusUnauthorized, "unauthorized", "not authenticated"))
		return
	}
	var req updateMeRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, r, apiError(http.StatusBadRequest, "bad_request", "invalid JSON body"))
		return
	}
	u, err := s.users.UpdateAvatar(uid, req.AvatarSeed)
	if err != nil {
		writeError(w, r, s.mapAppError(err))
		return
	}
	if s.rooms != nil {
		s.rooms.UpdateMemberAvatar(uid, u.AvatarSeed)
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	uid, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, apiError(http.StatusUnauthorized, "unauthorized", "not authenticated"))
		return
	}
	u, err := s.users.Profile(uid)
	if err != nil {
		writeError(w, r, s.mapAppError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u})
}

// mapAppError translates application errors to API errors.
func (s *Server) mapAppError(err error) error {
	switch {
	case errors.Is(err, app.ErrUsernameTaken):
		return apiError(http.StatusConflict, "username_taken", "username already taken")
	case errors.Is(err, app.ErrEmailTaken):
		return apiError(http.StatusConflict, "email_taken", "email already registered")
	case errors.Is(err, app.ErrBadCredentials):
		return apiError(http.StatusUnauthorized, "bad_credentials", "invalid username or password")
	case errors.Is(err, app.ErrUserNotFound):
		return apiError(http.StatusNotFound, "user_not_found", "user not found")
	case errors.Is(err, app.ErrValidation):
		return apiError(http.StatusUnprocessableEntity, "validation", err.Error())
	case errors.Is(err, auth.ErrWeakPassword):
		return apiError(http.StatusUnprocessableEntity, "validation", err.Error())
	default:
		return err // falls through to 500 mapping
	}
}
