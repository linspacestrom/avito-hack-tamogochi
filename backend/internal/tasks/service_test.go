package tasks_test

import (
	"context"
	"testing"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/tasks"
)

type fakeRepository struct {
	byCode  map[string]*tasks.TaskWithProgress
	claimed []string // task definition IDs passed to ClaimProgress
	// claimShouldFail simulates a concurrent claim winning the race between Service's
	// read and the transactional write.
	claimShouldFail bool
}

func (f *fakeRepository) ListTasksWithProgress(
	_ context.Context,
	_ string,
) ([]tasks.TaskWithProgress, error) {
	result := make([]tasks.TaskWithProgress, 0, len(f.byCode))
	for _, t := range f.byCode {
		result = append(result, *t)
	}
	return result, nil
}

func (f *fakeRepository) GetTaskWithProgressByCode(
	_ context.Context,
	_ string,
	code string,
) (*tasks.TaskWithProgress, error) {
	t, ok := f.byCode[code]
	if !ok {
		return nil, tasks.ErrTaskNotFound
	}
	return t, nil
}

func (f *fakeRepository) ClaimProgress(_ context.Context, _ string, taskDefinitionID string) error {
	if f.claimShouldFail {
		return tasks.ErrTaskAlreadyClaimed
	}
	f.claimed = append(f.claimed, taskDefinitionID)
	return nil
}

type fakeRewardIssuer struct {
	issued []rewards.UserReward
}

func (f *fakeRewardIssuer) CreateUserReward(
	_ context.Context,
	reward rewards.UserReward,
) (*rewards.UserReward, error) {
	f.issued = append(f.issued, reward)
	return &reward, nil
}

// fakeTransactor just runs fn directly — an in-memory fake repository has no real
// transactions to coordinate, so there's nothing for it to do beyond satisfying the interface.
type fakeTransactor struct{}

func (fakeTransactor) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestListTasks_ReturnsAllTasksFromRepository(t *testing.T) {
	repo := &fakeRepository{byCode: map[string]*tasks.TaskWithProgress{
		"daily-login": {Task: tasks.TaskDefinition{Code: "daily-login"}, Status: tasks.StatusInProgress},
	}}
	svc := tasks.NewService(repo, &fakeRewardIssuer{}, fakeTransactor{})

	list, err := svc.ListTasks(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 task, got %d", len(list))
	}
}

func TestClaimTask_UnknownCodeReturnsNotFound(t *testing.T) {
	repo := &fakeRepository{byCode: map[string]*tasks.TaskWithProgress{}}
	svc := tasks.NewService(repo, &fakeRewardIssuer{}, fakeTransactor{})

	_, err := svc.ClaimTask(context.Background(), "user-1", "unknown-code")
	if err != tasks.ErrTaskNotFound {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestClaimTask_NotCompletedYetFails(t *testing.T) {
	repo := &fakeRepository{byCode: map[string]*tasks.TaskWithProgress{
		"daily-login": {Task: tasks.TaskDefinition{Code: "daily-login"}, Status: tasks.StatusInProgress},
	}}
	svc := tasks.NewService(repo, &fakeRewardIssuer{}, fakeTransactor{})

	_, err := svc.ClaimTask(context.Background(), "user-1", "daily-login")
	if err != tasks.ErrTaskNotCompleted {
		t.Fatalf("expected ErrTaskNotCompleted, got %v", err)
	}
}

func TestClaimTask_AlreadyClaimedFails(t *testing.T) {
	repo := &fakeRepository{byCode: map[string]*tasks.TaskWithProgress{
		"daily-login": {Task: tasks.TaskDefinition{Code: "daily-login"}, Status: tasks.StatusClaimed},
	}}
	svc := tasks.NewService(repo, &fakeRewardIssuer{}, fakeTransactor{})

	_, err := svc.ClaimTask(context.Background(), "user-1", "daily-login")
	if err != tasks.ErrTaskAlreadyClaimed {
		t.Fatalf("expected ErrTaskAlreadyClaimed, got %v", err)
	}
}

func TestClaimTask_CompletedTaskIssuesRewardAndMarksClaimed(t *testing.T) {
	repo := &fakeRepository{byCode: map[string]*tasks.TaskWithProgress{
		"daily-login": {
			Task: tasks.TaskDefinition{
				ID:                 "task-def-1",
				Code:               "daily-login",
				RewardDefinitionID: "reward-def-1",
			},
			Reward: rewards.RewardDefinition{ID: "reward-def-1", Code: "welcome-coins"},
			Status: tasks.StatusCompleted,
		},
	}}
	issuer := &fakeRewardIssuer{}
	svc := tasks.NewService(repo, issuer, fakeTransactor{})

	reward, err := svc.ClaimTask(context.Background(), "user-1", "daily-login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reward.Code != "welcome-coins" {
		t.Fatalf("expected the task's linked reward, got %+v", reward)
	}
	if len(repo.claimed) != 1 || repo.claimed[0] != "task-def-1" {
		t.Fatalf("expected ClaimProgress called once with task-def-1, got %v", repo.claimed)
	}
	if len(issuer.issued) != 1 || issuer.issued[0].RewardDefinitionID != "reward-def-1" {
		t.Fatalf("expected CreateUserReward called once with reward-def-1, got %+v", issuer.issued)
	}
	if issuer.issued[0].UserID != "user-1" {
		t.Fatalf("expected reward issued to user-1, got %q", issuer.issued[0].UserID)
	}
}

func TestClaimTask_ConcurrentClaimLosesRaceCleanly(t *testing.T) {
	repo := &fakeRepository{
		byCode: map[string]*tasks.TaskWithProgress{
			"daily-login": {
				Task:   tasks.TaskDefinition{ID: "task-def-1", RewardDefinitionID: "reward-def-1"},
				Status: tasks.StatusCompleted,
			},
		},
		claimShouldFail: true,
	}
	issuer := &fakeRewardIssuer{}
	svc := tasks.NewService(repo, issuer, fakeTransactor{})

	_, err := svc.ClaimTask(context.Background(), "user-1", "daily-login")
	if err != tasks.ErrTaskAlreadyClaimed {
		t.Fatalf("expected ErrTaskAlreadyClaimed, got %v", err)
	}
	if len(issuer.issued) != 0 {
		t.Fatalf("expected no reward issued when ClaimProgress fails, got %+v", issuer.issued)
	}
}
