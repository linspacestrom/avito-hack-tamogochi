//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository/postgres"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Run with: go test -tags=integration ./internal/repository/postgres/...
// Requires TEST_DATABASE_URL pointing at a throwaway Postgres database with migrations
// 000001 and 000003 already applied (so the "welcome-coins" seed row exists).
func TestRewardsRepository_DuplicateIssuanceIsRejected(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	// t.Cleanup runs LIFO, so register pool.Close() first — it must run last, after the
	// delete below, otherwise the delete would run against an already-closed pool.
	t.Cleanup(pool.Close)

	repo := postgres.NewRewardsRepository(pool)
	svc := rewards.NewService(repo)

	const userID = "11111111-1111-1111-1111-111111111111"

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, "DELETE FROM user_rewards WHERE user_id = $1", userID); err != nil {
			t.Errorf("cleanup failed: %v", err)
		}
	})

	if _, err := svc.IssueReward(ctx, userID, "welcome-coins", nil, 1); err != nil {
		t.Fatalf("first issuance should succeed, got: %v", err)
	}

	_, err = svc.IssueReward(ctx, userID, "welcome-coins", nil, 1)
	if err != rewards.ErrAlreadyIssued {
		t.Fatalf("expected ErrAlreadyIssued on second issuance, got: %v", err)
	}
}
