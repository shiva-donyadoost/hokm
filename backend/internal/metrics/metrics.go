// Package metrics is a tiny dependency-free metrics registry exposing
// Prometheus text format at /api/metrics (Phase 17).
package metrics

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
)

var (
	httpRequests   sync.Map // "method|path|status" → *atomic.Int64
	wsSessions     atomic.Int64
	activeGames    atomic.Int64
	aiDecisions    atomic.Int64
	aiDecisionNano atomic.Int64
	matchesPlayed  atomic.Int64
)

// IncHTTP counts one completed HTTP request.
func IncHTTP(method, path string, status int) {
	key := method + "|" + path + "|" + fmt.Sprint(status)
	v, _ := httpRequests.LoadOrStore(key, &atomic.Int64{})
	v.(*atomic.Int64).Add(1)
}

// WSsessionDelta adjusts the active WebSocket session gauge.
func WSSessions(delta int64) { wsSessions.Add(delta) }

// GamesDelta adjusts the active-match gauge.
func GamesDelta(delta int64) { activeGames.Add(delta) }

// ObserveAIDecision records one AI decision and its duration.
func ObserveAIDecision(dNano int64) {
	aiDecisions.Add(1)
	aiDecisionNano.Add(dNano)
}

// MatchesInc counts one completed match.
func MatchesInc() { matchesPlayed.Add(1) }

// Render produces the Prometheus text exposition.
func Render() string {
	var b []byte
	b = append(b, "# TYPE hokm_http_requests_total counter\n"...)
	type kv struct {
		k string
		v int64
	}
	var rows []kv
	httpRequests.Range(func(key, value any) bool {
		rows = append(rows, kv{key.(string), value.(*atomic.Int64).Load()})
		return true
	})
	sort.Slice(rows, func(i, j int) bool { return rows[i].k < rows[j].k })
	for _, r := range rows {
		b = append(b, fmt.Sprintf("hokm_http_requests_total{req=%q} %d\n", r.k, r.v)...)
	}
	b = append(b, fmt.Sprintf("# TYPE hokm_ws_sessions gauge\nhokm_ws_sessions %d\n", wsSessions.Load())...)
	b = append(b, fmt.Sprintf("# TYPE hokm_games_active gauge\nhokm_games_active %d\n", activeGames.Load())...)
	b = append(b, fmt.Sprintf("# TYPE hokm_ai_decisions_total counter\nhokm_ai_decisions_total %d\n", aiDecisions.Load())...)
	b = append(b, fmt.Sprintf("# TYPE hokm_ai_decision_seconds_total counter\nhokm_ai_decision_seconds_total %f\n", float64(aiDecisionNano.Load())/1e9)...)
	b = append(b, fmt.Sprintf("# TYPE hokm_matches_total counter\nhokm_matches_total %d\n", matchesPlayed.Load())...)
	return string(b)
}
