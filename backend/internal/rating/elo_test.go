package rating

import (
	"math"
	"testing"
)

func TestExpectedScoreSymmetry(t *testing.T) {
	if ExpectedScore(1000, 1000) != 0.5 {
		t.Fatal("equal ratings should expect 0.5")
	}
	e := ExpectedScore(1200, 1000)
	if e <= 0.5 || e >= 0.9 {
		t.Fatalf("expected score = %v, want (0.5, 0.9)", e)
	}
	if math.Abs(ExpectedScore(1000, 1200)-(1-e)) > 1e-9 {
		t.Fatal("expected scores should be complementary")
	}
}

func TestUpdateWinnerGainsLoserLoses(t *testing.T) {
	newA, newB := Update([]float64{1000, 1000}, []float64{1000, 1000}, true)
	for _, r := range newA {
		if r <= 1000 {
			t.Fatalf("winner rating should rise: %v", newA)
		}
	}
	for _, r := range newB {
		if r >= 1000 {
			t.Fatalf("loser rating should fall: %v", newB)
		}
	}
	// Conservation: zero-sum for equal team sizes.
	sumDelta := (newA[0] + newA[1]) - (newB[0] + newB[1])
	if math.Abs(sumDelta-(2*K)) > 1e-9 {
		t.Fatalf("delta = %v, want 2K", sumDelta)
	}
}

func TestUpdateUnderdogGainsMore(t *testing.T) {
	// Underdog team (800 vs 1200) winning gains more than the favorite.
	_, newB := Update([]float64{1200, 1200}, []float64{800, 800}, false)
	gain := newB[0] - 800
	if gain <= K/2 {
		t.Fatalf("underdog gain = %v, want > K/2", gain)
	}
}
