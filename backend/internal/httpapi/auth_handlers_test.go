package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func postJSON(t *testing.T, s *Server, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func getAuthed(t *testing.T, s *Server, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func TestRegisterLoginProfileFlow(t *testing.T) {
	s := newTestServer(t)

	// Register.
	rec := postJSON(t, s, "/api/auth/register", map[string]string{
		"username": "ali_reza", "email": "ali@example.com", "password": "s3curePass!",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d: %s", rec.Code, rec.Body.String())
	}
	var regResp struct {
		User   map[string]any `json:"user"`
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &regResp)
	if regResp.Tokens.AccessToken == "" {
		t.Fatal("no access token in register response")
	}

	// Profile with token.
	rec = getAuthed(t, s, "/api/me", regResp.Tokens.AccessToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("me status = %d: %s", rec.Code, rec.Body.String())
	}

	// Login.
	rec = postJSON(t, s, "/api/auth/login", map[string]string{
		"username": "ali_reza", "password": "s3curePass!",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d: %s", rec.Code, rec.Body.String())
	}

	// Wrong password → 401 without user enumeration differences.
	rec = postJSON(t, s, "/api/auth/login", map[string]string{
		"username": "ali_reza", "password": "nope-nope",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad login status = %d", rec.Code)
	}
}

func TestRefreshRotation(t *testing.T) {
	s := newTestServer(t)
	rec := postJSON(t, s, "/api/auth/register", map[string]string{
		"username": "maryam", "email": "m@example.com", "password": "s3curePass!",
	})
	var reg struct {
		Tokens struct {
			RefreshToken string `json:"refresh_token"`
		} `json:"tokens"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &reg)

	// Rotate.
	rec = postJSON(t, s, "/api/auth/refresh", map[string]string{"refresh_token": reg.Tokens.RefreshToken})
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d: %s", rec.Code, rec.Body.String())
	}
	var rr struct {
		Tokens struct {
			RefreshToken string `json:"refresh_token"`
			AccessToken  string `json:"access_token"`
		} `json:"tokens"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &rr)
	if rr.Tokens.RefreshToken == "" || rr.Tokens.AccessToken == "" {
		t.Fatal("refresh response missing tokens")
	}
	// Old token single-use → rejected.
	rec = postJSON(t, s, "/api/auth/refresh", map[string]string{"refresh_token": reg.Tokens.RefreshToken})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("reused refresh status = %d, want 401", rec.Code)
	}
}

func TestRegisterValidation(t *testing.T) {
	s := newTestServer(t)
	cases := []struct {
		name string
		body map[string]string
		want int
	}{
		{"short username", map[string]string{"username": "ab", "email": "a@b.co", "password": "longenough1"}, 422},
		{"bad email", map[string]string{"username": "goodname", "email": "nope", "password": "longenough1"}, 422},
		{"short password", map[string]string{"username": "goodname", "email": "a@b.co", "password": "short"}, 422},
	}
	for _, tc := range cases {
		if rec := postJSON(t, s, "/api/auth/register", tc.body); rec.Code != tc.want {
			t.Errorf("%s: status = %d, want %d (%s)", tc.name, rec.Code, tc.want, rec.Body.String())
		}
	}
	// Duplicate username → 409.
	_ = postJSON(t, s, "/api/auth/register", map[string]string{
		"username": "dupname", "email": "d1@b.co", "password": "longenough1",
	})
	if rec := postJSON(t, s, "/api/auth/register", map[string]string{
		"username": "dupname", "email": "d2@b.co", "password": "longenough1",
	}); rec.Code != http.StatusConflict {
		t.Errorf("duplicate username: status = %d, want 409", rec.Code)
	}
}

func TestRequireAuth(t *testing.T) {
	s := newTestServer(t)
	// No token.
	r := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no-token status = %d, want 401", w.Code)
	}
	// Garbage token.
	rec2 := getAuthed(t, s, "/api/me", "garbage.token.here")
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("garbage-token status = %d, want 401", rec2.Code)
	}
}
