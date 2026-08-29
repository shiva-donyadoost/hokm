// Package rating implements the match-recording and rating pipeline.
// The Elo implementation is deliberately simple and swappable
// (Glicko/TrueSkill can replace it via the same interface).
package rating

import (
	"math"
)

// K is the Elo K-factor for all players. Higher K → faster movement.
const K = 32.0

const StartingRating = 1000.0

// ExpectedScore returns the expected win probability of A against B.
func ExpectedScore(ratingA, ratingB float64) float64 {
	return 1.0 / (1.0 + math.Pow(10, (ratingB-ratingA)/400.0))
}

// Update returns the new ratings for the A team and B team members given
// the actual outcome (1 = A wins, 0 = B wins, 0.5 = draw). Team games use
// the average opponent rating as the opposing strength.
func Update(ratingsA, ratingsB []float64, aWon bool) (newA, newB []float64) {
	actualA, actualB := 1.0, 0.0
	if !aWon {
		actualA, actualB = 0.0, 1.0
	}
	avgB := avg(ratingsB)
	avgA := avg(ratingsA)
	newA = make([]float64, len(ratingsA))
	newB = make([]float64, len(ratingsB))
	for i, r := range ratingsA {
		newA[i] = r + K*(actualA-ExpectedScore(r, avgB))
	}
	for i, r := range ratingsB {
		newB[i] = r + K*(actualB-ExpectedScore(r, avgA))
	}
	return newA, newB
}

func avg(xs []float64) float64 {
	if len(xs) == 0 {
		return StartingRating
	}
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}
