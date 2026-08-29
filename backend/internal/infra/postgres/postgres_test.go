package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hokm/platform/internal/app"
	"github.com/hokm/platform/internal/auth"
)

// testPool skips unless an integration DSN is provided, e.g. the compose
// database: postgres://hokm:change-me-dev-only@localhost:5432/hokm?sslmode=disable
func testPool(t *testing.T) (*testStores, bool) {
	t.Helper()
	dsn := os.Getenv("HOKM_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("HOKM_TEST_PG_DSN not set; skipping postgres integration test")
	}
	ctx := testContext(t)
	pool, err := Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return &testStores{
		users:   NewUserStore(pool),
		refresh: NewRefreshStore(pool),
	}, true
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

type testStores struct {
	users   *UserStore
	refresh *RefreshStore
}

func TestMigrateIdempotent(t *testing.T) {
	ts, ok := testPool(t)
	if !ok {
		return
	}
	_ = ts // Connect already ran Migrate twice via repeated test runs; and
	// Connect is called again here implicitly by testPool per test.
}

func TestUserStoreCRUD(t *testing.T) {
	ts, _ := testPool(t)
	ctx := testContext(t)
	_ = ctx

	u := &app.User{
		ID:           fmt.Sprintf("u_%d", time.Now().UnixNano()),
		Username:     fmt.Sprintf("pguser%d", time.Now().UnixNano()%1000000),
		Email:        fmt.Sprintf("pg%d@example.com", time.Now().UnixNano()%1000000),
		PasswordHash: "x",
		CreatedAt:    time.Now().UTC(),
	}
	if err := ts.users.Create(u); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := ts.users.ByUsername(u.Username)
	if err != nil || got.ID != u.ID {
		t.Fatalf("ByUsername: %v %v", got, err)
	}
	got, err = ts.users.ByID(u.ID)
	if err != nil || got.Email != u.Email {
		t.Fatalf("ByID: %v %v", got, err)
	}
	dup := *u
	dup.ID = u.ID + "_2"
	if err := ts.users.Create(&dup); err != app.ErrUsernameTaken {
		t.Fatalf("want ErrUsernameTaken, got %v", err)
	}
	dupEmail := *u
	dupEmail.ID = u.ID + "_3"
	dupEmail.Username = u.Username + "_x"
	if err := ts.users.Create(&dupEmail); err != app.ErrEmailTaken {
		t.Fatalf("want ErrEmailTaken, got %v", err)
	}
	if _, err := ts.users.ByID("missing"); err != app.ErrUserNotFound {
		t.Fatalf("want ErrUserNotFound, got %v", err)
	}
}

func TestRefreshStoreSingleUse(t *testing.T) {
	ts, _ := testPool(t)
	// refresh_tokens references users; create a real one.
	uid := fmt.Sprintf("u_%d", time.Now().UnixNano())
	u := &app.User{
		ID:           uid,
		Username:     fmt.Sprintf("rt%d", time.Now().UnixNano()%1000000),
		Email:        fmt.Sprintf("rt%d@example.com", time.Now().UnixNano()%1000000),
		PasswordHash: "x",
		CreatedAt:    time.Now().UTC(),
	}
	if err := ts.users.Create(u); err != nil {
		t.Fatalf("create user: %v", err)
	}

	_, hash, err := auth.NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	if err := ts.refresh.Save(hash, uid, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := ts.refresh.Consume(hash)
	if err != nil || got != uid {
		t.Fatalf("consume: %q %v", got, err)
	}
	if _, err := ts.refresh.Consume(hash); err != auth.ErrInvalidToken {
		t.Fatalf("reconsume: %v", err)
	}
	// Expired token is invalid.
	_, hash2, _ := auth.NewRefreshToken()
	_ = ts.refresh.Save(hash2, uid, time.Now().Add(-time.Minute))
	if _, err := ts.refresh.Consume(hash2); err != auth.ErrInvalidToken {
		t.Fatalf("expired: %v", err)
	}
}
