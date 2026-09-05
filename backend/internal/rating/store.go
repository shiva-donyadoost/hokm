package rating

import (
	"math"
	"sort"
	"sync"
	"time"
)

// MatchRecord is a completed match snapshot.
type MatchRecord struct {
	GameID     string
	RoomID     string
	RoundsWonA int
	RoundsWonB int
	WinnerTeam int // 0 = A, 1 = B
	// Players indexed by seat.
	Players []MatchPlayer
}

type MatchPlayer struct {
	UserID       string
	Username     string
	Seat         int
	Team         int
	IsAI         bool
	AIDifficulty string
}

// ScoreStore persists statistics and ratings.
type ScoreStore interface {
	// ApplyMatch atomically records a match and updates stats/ratings.
	ApplyMatch(rec MatchRecord) error
	// Leaderboard returns the top n users by wins, then rating, then username.
	Leaderboard(n int) ([]Entry, error)
	// StatsOf returns the stats entry for a user.
	StatsOf(userID string) (Entry, error)
}

// Entry is a leaderboard/statistics row.
type Entry struct {
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	AvatarSeed  string    `json:"avatar_seed,omitempty"`
	Rating      int       `json:"rating"`
	GamesPlayed int       `json:"games_played"`
	Wins        int       `json:"wins"`
	Losses      int       `json:"losses"`
	RoundsWon   int       `json:"rounds_won"`
	RoundsLost  int       `json:"rounds_lost"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MemoryStore is an in-memory ScoreStore for tests and dev fallback.
type MemoryStore struct {
	mu      sync.Mutex
	entries map[string]*Entry
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{entries: make(map[string]*Entry)}
}

func (m *MemoryStore) ApplyMatch(rec MatchRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	get := func(userID, username string) *Entry {
		e, ok := m.entries[userID]
		if !ok {
			e = &Entry{UserID: userID, Username: username, Rating: int(StartingRating)}
			m.entries[userID] = e
		}
		return e
	}
	var ratingsA, ratingsB []float64
	for _, p := range rec.Players {
		if p.IsAI {
			continue
		}
		e := get(p.UserID, p.Username)
		if p.Team == 0 {
			ratingsA = append(ratingsA, float64(e.Rating))
		} else {
			ratingsB = append(ratingsB, float64(e.Rating))
		}
	}
	newA, newB := Update(ratingsA, ratingsB, rec.WinnerTeam == 0)
	idxA, idxB := 0, 0
	aWonRounds, bWonRounds := rec.RoundsWonA, rec.RoundsWonB
	for _, p := range rec.Players {
		if p.IsAI {
			continue
		}
		e := get(p.UserID, p.Username)
		var newRating float64
		if p.Team == 0 {
			newRating = newA[idxA]
			idxA++
			e.RoundsWon += aWonRounds
			e.RoundsLost += bWonRounds
		} else {
			newRating = newB[idxB]
			idxB++
			e.RoundsWon += bWonRounds
			e.RoundsLost += aWonRounds
		}
		e.Rating = int(math.Round(newRating))
		e.GamesPlayed++
		if p.Team == rec.WinnerTeam {
			e.Wins++
		} else {
			e.Losses++
		}
		e.UpdatedAt = time.Now().UTC()
	}
	return nil
}

func (m *MemoryStore) Leaderboard(n int) ([]Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Entry, 0, len(m.entries))
	for _, e := range m.entries {
		out = append(out, *e)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Wins != b.Wins {
			return a.Wins > b.Wins
		}
		if a.Rating != b.Rating {
			return a.Rating > b.Rating
		}
		return a.Username < b.Username
	})
	if n > len(out) {
		n = len(out)
	}
	return out[:n], nil
}

func (m *MemoryStore) StatsOf(userID string) (Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[userID]
	if !ok {
		return Entry{UserID: userID, Rating: int(StartingRating)}, nil
	}
	return *e, nil
}
