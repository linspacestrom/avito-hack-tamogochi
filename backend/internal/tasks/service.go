package tasks

import (
	"context"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
)

// Repository is what Service needs from storage. Implemented by
// internal/repository/postgres.TasksRepository.
type Repository interface {
	ListTasksWithProgress(ctx context.Context, userID string) ([]TaskWithProgress, error)
	// GetTaskWithProgressByCode returns ErrTaskNotFound if no active task matches code.
	GetTaskWithProgressByCode(ctx context.Context, userID string, code string) (*TaskWithProgress, error)
	// ClaimProgress marks the user's progress on taskDefinitionID as claimed, conditional on
	// it currently being StatusCompleted. Returns ErrTaskAlreadyClaimed if that condition no
	// longer holds — the only way that happens after Service's own check is a concurrent
	// claim winning the race in between.
	ClaimProgress(ctx context.Context, userID string, taskDefinitionID string) error
}

// RewardIssuer grants a task's linked reward to the user — the exact same CreateUserReward
// call rewards.Service uses for the /rewards/issue flow, reused here rather than
// reimplemented. Unlike IssueReward, this skips the RequiredLevel check: completing the task
// is itself the gate, so a second level check on top would be redundant at best and
// contradictory at worst (a low-level task could easily be wired to a higher-level reward).
type RewardIssuer interface {
	CreateUserReward(ctx context.Context, reward rewards.UserReward) (*rewards.UserReward, error)
}

// Transactor runs fn atomically. Implemented by *manager.Manager (go-transaction-manager),
// the same value used elsewhere in this project — defined locally rather than imported from
// the service package to avoid an import cycle (service already imports rewards and tasks).
type Transactor interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Service struct {
	repo         Repository
	rewardIssuer RewardIssuer
	transactor   Transactor
}

func NewService(repo Repository, rewardIssuer RewardIssuer, transactor Transactor) *Service {
	return &Service{repo: repo, rewardIssuer: rewardIssuer, transactor: transactor}
}

// ListTasks returns every active task with the caller's progress on it.
func (s *Service) ListTasks(ctx context.Context, userID string) ([]TaskWithProgress, error) {
	return s.repo.ListTasksWithProgress(ctx, userID)
}

// ClaimTask grants the reward for a completed task and marks it claimed. Fails with
// ErrTaskNotFound if the code doesn't match an active task, ErrTaskNotCompleted if the user
// hasn't reached the task's target yet, or ErrTaskAlreadyClaimed if this claim already
// happened.
//
// NOTE: for a 'daily' task, the same user_task_progress row and the same
// reward_definition_id get reused every time the (not yet built) event dispatcher resets the
// cycle — claiming a repeat completion will hit the same (userId, rewardDefinitionId)
// uniqueness that protects the achievement-style rewards in user_rewards, the same conflict
// already solved for the daily reward cycle via its own separate claim log. Not fixed here
// since nothing resets daily task progress yet (deferred along with the dispatcher) — flagging
// so whoever builds the reset doesn't rediscover this the hard way.
func (s *Service) ClaimTask(ctx context.Context, userID string, taskCode string) (*rewards.RewardDefinition, error) {
	task, err := s.repo.GetTaskWithProgressByCode(ctx, userID, taskCode)
	if err != nil {
		return nil, err
	}

	switch task.Status {
	case StatusClaimed:
		return nil, ErrTaskAlreadyClaimed
	case StatusCompleted:
		// proceed to claim
	default:
		return nil, ErrTaskNotCompleted
	}

	err = s.transactor.Do(ctx, func(txCtx context.Context) error {
		if err := s.repo.ClaimProgress(txCtx, userID, task.Task.ID); err != nil {
			return err
		}
		_, err := s.rewardIssuer.CreateUserReward(txCtx, rewards.UserReward{
			UserID:             userID,
			RewardDefinitionID: task.Task.RewardDefinitionID,
			Status:             rewards.StatusIssued,
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	return &task.Reward, nil
}
