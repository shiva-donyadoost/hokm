package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/hokm/platform/internal/app"
	"github.com/hokm/platform/internal/auth"
	"github.com/hokm/platform/internal/infra/memory"
	"github.com/hokm/platform/internal/rating"
	"github.com/hokm/platform/internal/room"
)

func TestStatsAndLeaderboardHTTP(t *testing.T) {
	tokens := auth.NewTokenManager("test-secret-value-at-least-long", time.Minute)
	users := app.NewUserService(memory.NewUserStore(), tokens, auth.NewMemoryRefreshStore(), time.Hour)
	rooms := room.NewManager()
	scores := rating.NewMemoryStore()
	s := NewServer(users, tokens, rooms, nil, nil, scores)

	uid, tok := registerUser(t, s, "ranked")
	if err := scores.ApplyMatch(rating.MatchRecord{
		GameID: "g1", RoomID: "r1", RoundsWonA: 1, RoundsWonB: 0, WinnerTeam: 0,
		Players: []rating.MatchPlayer{
			{UserID: uid, Username: "ranked", Seat: 0, Team: 0},
			{UserID: "opp", Username: "opp", Seat: 1, Team: 1},
		},
	}); err != nil {
		t.Fatalf("ApplyMatch: %v", err)
	}

	rec := getAuthed(t, s, "/api/stats", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("stats: %d %s", rec.Code, rec.Body.String())
	}
	var stats struct {
		Stats rating.Entry `json:"stats"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if stats.Stats.GamesPlayed != 1 || stats.Stats.Wins != 1 {
		t.Fatalf("stats = %+v, want 1 game 1 win", stats.Stats)
	}

	rec = getAuthed(t, s, "/api/leaderboard", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("leaderboard: %d %s", rec.Code, rec.Body.String())
	}
	var lb struct {
		Entries []rating.Entry `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &lb); err != nil {
		t.Fatalf("decode leaderboard: %v", err)
	}
	if len(lb.Entries) < 1 || lb.Entries[0].UserID != uid {
		t.Fatalf("leaderboard = %+v, want ranked first", lb.Entries)
	}
}
