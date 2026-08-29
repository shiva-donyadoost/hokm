package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hokm/platform/internal/rating"
)

// ScoreStore is the durable rating/statistics store.
type ScoreStore struct {
	pool *pgxpool.Pool
}

func NewScoreStore(pool *pgxpool.Pool) *ScoreStore { return &ScoreStore{pool: pool} }

// ApplyMatch records the match and updates stats/ratings in one transaction.
func (s *ScoreStore) ApplyMatch(rec rating.MatchRecord) error {
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: apply match begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Match + participants.
	roundsToWin := max(rec.RoundsWonA, rec.RoundsWonB)
	if _, err := tx.Exec(ctx,
		`INSERT INTO games (id, room_id, rounds_to_win, winner_team, finished_at)
		 VALUES ($1, $2, $3, $4, now())`,
		rec.GameID, rec.RoomID, roundsToWin, rec.WinnerTeam); err != nil {
		return fmt.Errorf("postgres: insert game: %w", err)
	}
	for _, p := range rec.Players {
		if _, err := tx.Exec(ctx,
			`INSERT INTO game_players (game_id, seat, user_id, username, is_ai, ai_difficulty, team)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			rec.GameID, p.Seat, p.UserID, p.Username, p.IsAI, p.AIDifficulty, p.Team); err != nil {
			return fmt.Errorf("postgres: insert game player: %w", err)
		}
	}

	// Ensure stats rows exist for humans.
	for _, p := range rec.Players {
		if p.IsAI {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO statistics (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`,
			p.UserID); err != nil {
			return fmt.Errorf("postgres: ensure stats: %w", err)
		}
	}

	// Read current ratings with row locks.
	type teamRatings struct {
		players []rating.MatchPlayer
		values  []float64
		ratings []int
	}
	teamA, teamB := teamRatings{}, teamRatings{}
	for _, p := range rec.Players {
		if p.IsAI {
			continue
		}
		var r int
		if err := tx.QueryRow(ctx,
			`SELECT rating FROM statistics WHERE user_id = $1 FOR UPDATE`, p.UserID,
		).Scan(&r); err != nil {
			return fmt.Errorf("postgres: read rating: %w", err)
		}
		if p.Team == 0 {
			teamA.players = append(teamA.players, p)
			teamA.ratings = append(teamA.ratings, r)
			teamA.values = append(teamA.values, float64(r))
		} else {
			teamB.players = append(teamB.players, p)
			teamB.ratings = append(teamB.ratings, r)
			teamB.values = append(teamB.values, float64(r))
		}
	}
	newA, newB := rating.Update(teamA.values, teamB.values, rec.WinnerTeam == 0)

	apply := func(players []rating.MatchPlayer, old []int, new []float64, wonRounds, lostRounds int, won bool) error {
		for i, p := range players {
			delta := 0
			if won {
				delta = 1
			}
			if _, err := tx.Exec(ctx,
				`UPDATE statistics SET
				   games_played = games_played + 1,
				   wins = wins + $2,
				   losses = losses + $3,
				   rounds_won = rounds_won + $4,
				   rounds_lost = rounds_lost + $5,
				   rating = $6,
				   updated_at = now()
				 WHERE user_id = $1`,
				p.UserID, delta, 1-delta, wonRounds, lostRounds, int(new[i])); err != nil {
				return fmt.Errorf("postgres: update stats: %w", err)
			}
			_ = old
		}
		return nil
	}
	if err := apply(teamA.players, teamA.ratings, newA, rec.RoundsWonA, rec.RoundsWonB, rec.WinnerTeam == 0); err != nil {
		return err
	}
	if err := apply(teamB.players, teamB.ratings, newB, rec.RoundsWonB, rec.RoundsWonA, rec.WinnerTeam == 1); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Leaderboard returns the top n users by rating.
func (s *ScoreStore) Leaderboard(n int) ([]rating.Entry, error) {
	ctx := context.Background()
	rows, err := s.pool.Query(ctx, `
		SELECT u.id, u.username, s.rating, s.games_played, s.wins, s.losses,
		       s.rounds_won, s.rounds_lost, s.updated_at
		FROM statistics s JOIN users u ON u.id = s.user_id
		ORDER BY s.rating DESC, s.wins DESC
		LIMIT $1`, n)
	if err != nil {
		return nil, fmt.Errorf("postgres: leaderboard: %w", err)
	}
	defer rows.Close()
	var out []rating.Entry
	for rows.Next() {
		var e rating.Entry
		if err := rows.Scan(&e.UserID, &e.Username, &e.Rating, &e.GamesPlayed,
			&e.Wins, &e.Losses, &e.RoundsWon, &e.RoundsLost, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("postgres: leaderboard scan: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// StatsOf returns one user's statistics row.
func (s *ScoreStore) StatsOf(userID string) (rating.Entry, error) {
	ctx := context.Background()
	var e rating.Entry
	err := s.pool.QueryRow(ctx, `
		SELECT u.id, u.username, s.rating, s.games_played, s.wins, s.losses,
		       s.rounds_won, s.rounds_lost, s.updated_at
		FROM statistics s JOIN users u ON u.id = s.user_id
		WHERE s.user_id = $1`, userID,
	).Scan(&e.UserID, &e.Username, &e.Rating, &e.GamesPlayed,
		&e.Wins, &e.Losses, &e.RoundsWon, &e.RoundsLost, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return rating.Entry{UserID: userID, Rating: int(rating.StartingRating)}, nil
	}
	if err != nil {
		return e, fmt.Errorf("postgres: stats of: %w", err)
	}
	return e, nil
}
