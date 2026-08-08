//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository/postgres"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Run with: go test -tags=integration ./internal/repository/postgres/...
// Requires TEST_DATABASE_URL pointing at a throwaway Postgres database with migrations
// 000001, 000003, 000005, 000006 and 000007 already applied.
func TestRewardsRepository_DuplicateIssuanceIsRejected(t *testing.T) {
	pool := connectTestPool(t)

	repo := postgres.New(pool)
	svc := rewards.NewService(repo.Rewards)
	ctx := context.Background()

	const userID = "11111111-1111-1111-1111-111111111111"
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, "DELETE FROM user_rewards WHERE user_id = $1", userID); err != nil {
			t.Errorf("cleanup failed: %v", err)
		}
	})

	if _, err := svc.IssueReward(ctx, userID, "welcome-coins", nil, 1); err != nil {
		t.Fatalf("first issuance should succeed, got: %v", err)
	}

	_, err := svc.IssueReward(ctx, userID, "welcome-coins", nil, 1)
	if err != rewards.ErrAlreadyIssued {
		t.Fatalf("expected ErrAlreadyIssued on second issuance, got: %v", err)
	}
}

// TestDailyCycleClaim_UpdatesProgressAndLogsAtomically exercises the sqlc-backed
// DailyCycleRepository end to end, including the transaction wrapping progress update and
// claim-log insert together.
func TestDailyCycleClaim_UpdatesProgressAndLogsAtomically(t *testing.T) {
	pool := connectTestPool(t)

	ctx := context.Background()
	transactor := manager.Must(trmpgx.NewDefaultFactory(pool))
	repo := postgres.New(pool)
	svc := rewards.NewDailyCycleService(repo.Rewards, transactor)

	const userID = "22222222-2222-2222-2222-222222222222"
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM user_daily_reward_progress WHERE user_id = $1", userID)
		pool.Exec(ctx, "DELETE FROM daily_reward_claim_log WHERE user_id = $1", userID)
	})

	claimed, err := svc.ClaimToday(ctx, userID)
	if err != nil {
		t.Fatalf("claim should succeed, got: %v", err)
	}
	if claimed.Code != "welcome-coins" {
		t.Fatalf("expected day 1 reward welcome-coins, got %s", claimed.Code)
	}

	var logCount int
	err = pool.QueryRow(ctx,
		"SELECT count(*) FROM daily_reward_claim_log WHERE user_id = $1", userID,
	).Scan(&logCount)
	if err != nil {
		t.Fatalf("query log: %v", err)
	}
	if logCount != 1 {
		t.Fatalf("expected exactly 1 claim log row after one claim, got %d", logCount)
	}

	var currentDay int
	err = pool.QueryRow(ctx,
		"SELECT current_day FROM user_daily_reward_progress WHERE user_id = $1", userID,
	).Scan(&currentDay)
	if err != nil {
		t.Fatalf("query progress: %v", err)
	}
	if currentDay != 2 {
		t.Fatalf("expected progress advanced to day 2, got %d", currentDay)
	}
}

// connectTestPool registers the pool's own Close via t.Cleanup rather than a plain defer —
// t.Cleanup runs LIFO, so callers that register their own cleanup (e.g. deleting test rows)
// after calling this will correctly run that delete *before* the pool closes, not after.
func connectTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}
