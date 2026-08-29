package app

import (
	"fmt"
	"sync"
	"testing"
)

type captureSink struct {
	mu   sync.Mutex
	msgs []ChatMessage
}

func (c *captureSink) ChatMessage(m ChatMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, m)
}

func (c *captureSink) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.msgs)
}

func TestChatSendAndHistory(t *testing.T) {
	sink := &captureSink{}
	cs := NewChatService()
	cs.SetSink(sink)

	for i := 1; i <= 3; i++ {
		if _, err := cs.Send("r1", "u1", "alice", fmt.Sprintf("hello %d", i)); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}
	cs.System("r1", "bob joined the room")
	h := cs.History("r1", 10)
	if len(h) != 4 {
		t.Fatalf("history = %d, want 4", len(h))
	}
	if !h[3].IsSystem || h[3].Body != "bob joined the room" {
		t.Fatalf("system msg wrong: %+v", h[3])
	}
	if h[0].Body != "hello 1" {
		t.Fatalf("history order wrong: %+v", h[0])
	}
	if sink.count() != 4 {
		t.Fatalf("sink received %d, want 4", sink.count())
	}
	// Bounded history.
	for i := 0; i < 100; i++ {
		_, _ = cs.Send("r1", "u1", "alice", "spam")
	}
	if got := len(cs.History("r1", 1000)); got > 50 {
		t.Fatalf("history unbounded: %d", got)
	}
}

func TestChatModeration(t *testing.T) {
	cs := NewChatService()
	// Empty / too long rejected.
	if _, err := cs.Send("r", "u", "a", "   "); err == nil {
		t.Fatal("empty message accepted")
	}
	long := make([]byte, 501)
	for i := range long {
		long[i] = 'x'
	}
	if _, err := cs.Send("r", "u", "a", string(long)); err == nil {
		t.Fatal("overlong message accepted")
	}
	// Rate limit: first 5 pass, 6th within 10s fails.
	for i := 0; i < 5; i++ {
		if _, err := cs.Send("r", "fast", "f", "hi"); err != nil {
			t.Fatalf("msg %d rejected: %v", i, err)
		}
	}
	if _, err := cs.Send("r", "fast", "f", "hi again"); err == nil {
		t.Fatal("6th message should be rate limited")
	}
	// Different user unaffected.
	if _, err := cs.Send("r", "slow", "s", "hi"); err != nil {
		t.Fatalf("other user rate limited: %v", err)
	}
}
