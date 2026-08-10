//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository/postgres"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/tasks"
	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
)

// Run with: go test -tags=integration ./internal/repository/postgres/...
// Requires TEST_DATABASE_URL pointing at a throwaway Postgres database with migrations
// 000001 through 000008 already applied (needs the seeded task_definitions/reward_definitions
// catalog from 000003 and 000008).
func TestTasksRepository_ListShowsUnstartedTasksWithZeroProgress(t *testing.T) {
	pool := connectTestPool(t)

	repo := postgres.New(pool)
	svc := tasks.NewService(repo.Tasks, repo.Rewards, manager.Must(trmpgx.NewDefaultFactory(pool)))
	ctx := context.Background()

	const userID = "33333333-3333-3333-3333-333333333333"

	list, err := svc.ListTasks(ctx, userID)
	if err != nil {
		t.Fatalf("list should succeed, got: %v", err)
	}
	if len(list) < 12 {
		t.Fatalf("expected at least the 12 seeded tasks, got %d", len(list))
	}

	for _, task := range list {
		if task.Status != tasks.StatusInProgress || task.CurrentValue != 0 {
			t.Fatalf("expected zero-progress default for untouched task %s, got %+v", task.Task.Code, task)
		}
	}
}

func TestTasksRepository_ClaimIssuesRewardAndMarksClaimed(t *testing.T) {
	pool := connectTestPool(t)

	ctx := context.Background()
	transactor := manager.Must(trmpgx.NewDefaultFactory(pool))
	repo := postgres.New(pool)
	svc := tasks.NewService(repo.Tasks, repo.Rewards, transactor)

	const userID = "44444444-4444-4444-4444-444444444444"
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM user_rewards WHERE user_id = $1", userID)
		pool.Exec(ctx, "DELETE FROM user_task_progress WHERE user_id = $1", userID)
	})

	var taskDefID string
	err := pool.QueryRow(ctx,
		"SELECT id FROM task_definitions WHERE code = 'avito-open-app'",
	).Scan(&taskDefID)
	if err != nil {
		t.Fatalf("lookup seeded task: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO user_task_progress (user_id, task_definition_id, current_value, status, completed_at)
		VALUES ($1, $2, 1, 'completed', now())
	`, userID, taskDefID)
	if err != nil {
		t.Fatalf("seed completed progress: %v", err)
	}

	reward, err := svc.ClaimTask(ctx, userID, "avito-open-app")
	if err != nil {
		t.Fatalf("claim should succeed, got: %v", err)
	}
	if reward.Code != "welcome-coins" {
		t.Fatalf("expected the task's linked reward welcome-coins, got %s", reward.Code)
	}

	var status string
	err = pool.QueryRow(ctx,
		"SELECT status FROM user_task_progress WHERE user_id = $1 AND task_definition_id = $2",
		userID, taskDefID,
	).Scan(&status)
	if err != nil {
		t.Fatalf("query progress: %v", err)
	}
	if status != "claimed" {
		t.Fatalf("expected status claimed, got %s", status)
	}

	var rewardCount int
	err = pool.QueryRow(ctx,
		"SELECT count(*) FROM user_rewards WHERE user_id = $1", userID,
	).Scan(&rewardCount)
	if err != nil {
		t.Fatalf("query user_rewards: %v", err)
	}
	if rewardCount != 1 {
		t.Fatalf("expected exactly 1 user_rewards row after claim, got %d", rewardCount)
	}

	// Claiming again must fail — the task is already claimed.
	_, err = svc.ClaimTask(ctx, userID, "avito-open-app")
	if err != tasks.ErrTaskAlreadyClaimed {
		t.Fatalf("expected ErrTaskAlreadyClaimed on second claim, got: %v", err)
	}
}

func TestTasksRepository_ClaimBeforeCompletedFails(t *testing.T) {
	pool := connectTestPool(t)

	ctx := context.Background()
	transactor := manager.Must(trmpgx.NewDefaultFactory(pool))
	repo := postgres.New(pool)
	svc := tasks.NewService(repo.Tasks, repo.Rewards, transactor)

	const userID = "55555555-5555-5555-5555-555555555555"

	_, err := svc.ClaimTask(ctx, userID, "avito-view-listings")
	if err != tasks.ErrTaskNotCompleted {
		t.Fatalf("expected ErrTaskNotCompleted for a task with no progress row, got: %v", err)
	}
}
