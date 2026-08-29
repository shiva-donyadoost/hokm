package rating

import (
	"testing"
)

func TestMemoryStoreApplyMatch(t *testing.T) {
	s := NewMemoryStore()
	rec := MatchRecord{
		GameID:     "g1",
		RoomID:     "r1",
		RoundsWonA: 7,
		RoundsWonB: 3,
		WinnerTeam: 0,
		Players: []MatchPlayer{
			{UserID: "h1", Username: "human1", Seat: 0, Team: 0},
			{UserID: "ai1", Username: "bot", Seat: 1, Team: 1, IsAI: true, AIDifficulty: "hard"},
			{UserID: "h2", Username: "human2", Seat: 2, Team: 0},
			{UserID: "h3", Username: "human3", Seat: 3, Team: 1},
		},
	}
	if err := s.ApplyMatch(rec); err != nil {
		t.Fatalf("ApplyMatch: %v", err)
	}
	// Winner stats.
	e1, _ := s.StatsOf("h1")
	if e1.Wins != 1 || e1.GamesPlayed != 1 || e1.Rating <= 1000 || e1.RoundsWon != 7 {
		t.Fatalf("winner entry wrong: %+v", e1)
	}
	// Loser stats.
	e3, _ := s.StatsOf("h3")
	if e3.Losses != 1 || e3.Rating >= 1000 || e3.RoundsLost != 7 {
		t.Fatalf("loser entry wrong: %+v", e3)
	}
	// AI players never recorded.
	ai, _ := s.StatsOf("ai1")
	if ai.GamesPlayed != 0 {
		t.Fatalf("AI should not be scored: %+v", ai)
	}
	// Leaderboard ranks winner first.
	lb, _ := s.Leaderboard(10)
	if len(lb) != 3 || lb[0].UserID != "h1" {
		t.Fatalf("leaderboard wrong: %+v", lb)
	}
}

func TestMemoryStoreZeroSum(t *testing.T) {
	s := NewMemoryStore()
	before := 2 * int(StartingRating)
	rec := MatchRecord{
		GameID: "g", WinnerTeam: 1, RoundsWonA: 2, RoundsWonB: 7,
		Players: []MatchPlayer{
			{UserID: "x", Username: "X", Team: 0},
			{UserID: "y", Username: "Y", Team: 1},
		},
	}
	_ = s.ApplyMatch(rec)
	a, _ := s.StatsOf("x")
	b, _ := s.StatsOf("y")
	after := a.Rating + b.Rating
	if before != after {
		t.Fatalf("rating not conserved: %d → %d", before, after)
	}
}
