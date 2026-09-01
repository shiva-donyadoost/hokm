package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// registerUser creates a user via the API and returns id+token.
func registerUser(t *testing.T, s *Server, username string) (string, string) {
	t.Helper()
	rec := postJSON(t, s, "/api/auth/register", map[string]string{
		"username": username, "email": username + "@example.com", "password": "s3curePass!",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register %s: %d %s", username, rec.Code, rec.Body.String())
	}
	var resp struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode register: %v", err)
	}
	return resp.User.ID, resp.Tokens.AccessToken
}

func authedRequest(t *testing.T, s *Server, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func TestRoomHTTPFlow(t *testing.T) {
	s := newTestServer(t)
	_, tokA := registerUser(t, s, "alice")
	_, tokB := registerUser(t, s, "bob")

	// Create room.
	rec := authedRequest(t, s, http.MethodPost, "/api/rooms", tokA,
		map[string]string{"name": "Friday Night", "visibility": "public"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Room struct {
			ID   string `json:"id"`
			Code string `json:"code"`
		} `json:"room"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	// Public list contains it.
	rec = authedRequest(t, s, http.MethodGet, "/api/rooms", tokB, nil)
	var list struct {
		Rooms []struct {
			ID string `json:"id"`
		} `json:"rooms"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list.Rooms) != 1 || list.Rooms[0].ID != created.Room.ID {
		t.Fatalf("public list wrong: %s", rec.Body.String())
	}

	// Bob joins by code.
	rec = authedRequest(t, s, http.MethodPost, "/api/rooms/join", tokB,
		map[string]string{"code": created.Room.Code})
	if rec.Code != http.StatusOK {
		t.Fatalf("join: %d %s", rec.Code, rec.Body.String())
	}

	// Bob readies up.
	rec = authedRequest(t, s, http.MethodPost, "/api/rooms/"+created.Room.ID+"/ready", tokB,
		map[string]bool{"ready": true})
	if rec.Code != http.StatusOK {
		t.Fatalf("ready: %d %s", rec.Code, rec.Body.String())
	}

	// Alice adds an AI.
	rec = authedRequest(t, s, http.MethodPost, "/api/rooms/"+created.Room.ID+"/ai", tokA,
		map[string]string{"difficulty": "hard"})
	if rec.Code != http.StatusOK {
		t.Fatalf("add ai: %d %s", rec.Code, rec.Body.String())
	}

	// Alice fills remaining empty seats with random AI.
	rec = authedRequest(t, s, http.MethodPost, "/api/rooms/"+created.Room.ID+"/ai/fill", tokA, map[string]any{})
	if rec.Code != http.StatusOK {
		t.Fatalf("fill ai: %d %s", rec.Code, rec.Body.String())
	}

	// Alice cannot be kicked by Bob.
	rec = authedRequest(t, s, http.MethodPost, "/api/rooms/"+created.Room.ID+"/kick", tokB,
		map[string]string{"user_id": "whatever"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-host kick: %d %s", rec.Code, rec.Body.String())
	}

	// Room lookup.
	rec = authedRequest(t, s, http.MethodGet, "/api/rooms/"+created.Room.ID, tokA, nil)
	var got struct {
		Room struct {
			Members []struct {
				Username string `json:"username"`
				IsAI     bool   `json:"is_ai"`
				Ready    bool   `json:"ready"`
			} `json:"members"`
		} `json:"room"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if len(got.Room.Members) != 4 {
		t.Fatalf("members = %d, want 4 after fill", len(got.Room.Members))
	}
	aiN := 0
	for _, m := range got.Room.Members {
		if m.IsAI {
			aiN++
		}
	}
	if aiN != 2 {
		t.Fatalf("AI members = %d, want 2 (one add + one fill)", aiN)
	}

	// Unauthenticated access rejected.
	r := httptest.NewRequest(http.MethodGet, "/api/rooms", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauth list: %d, want 401", w.Code)
	}
}
