package postgres

import (
	"context"
	"errors"

	db "github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository/postgres/sqlc"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/tasks"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// TasksRepository is a sub-repository of Repository, matching the pattern used by
// RewardsRepository — it goes through r.repo.GetConn(ctx) so it participates correctly in the
// shared trm transaction manager, and uses sqlc-generated queries instead of raw pgx.
type TasksRepository struct {
	repo *Repository
}

func NewTasksRepository(repo *Repository) *TasksRepository {
	return &TasksRepository{repo: repo}
}

func (r *TasksRepository) ListTasksWithProgress(
	ctx context.Context,
	userID string,
) ([]tasks.TaskWithProgress, error) {
	id, err := stringToUUID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := db.New(r.repo.GetConn(ctx)).ListTasksWithProgress(ctx, id)
	if err != nil {
		return nil, err
	}

	result := make([]tasks.TaskWithProgress, 0, len(rows))
	for _, row := range rows {
		result = append(result, taskWithProgressFromListRow(row))
	}
	return result, nil
}

func (r *TasksRepository) GetTaskWithProgressByCode(
	ctx context.Context,
	userID string,
	code string,
) (*tasks.TaskWithProgress, error) {
	id, err := stringToUUID(userID)
	if err != nil {
		return nil, err
	}

	row, err := db.New(r.repo.GetConn(ctx)).GetTaskWithProgressByCode(ctx, db.GetTaskWithProgressByCodeParams{
		UserID: id,
		Code:   code,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, tasks.ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}

	result := taskWithProgressFromGetRow(row)
	return &result, nil
}

func (r *TasksRepository) ClaimProgress(
	ctx context.Context,
	userID string,
	taskDefinitionID string,
) error {
	uID, err := stringToUUID(userID)
	if err != nil {
		return err
	}
	taskID, err := stringToUUID(taskDefinitionID)
	if err != nil {
		return err
	}

	_, err = db.New(r.repo.GetConn(ctx)).ClaimTaskProgress(ctx, db.ClaimTaskProgressParams{
		UserID:           uID,
		TaskDefinitionID: taskID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return tasks.ErrTaskAlreadyClaimed
	}
	return err
}

// taskProgressStatus reads a possibly-NULL progress status/current_value pair (from the
// LEFT JOIN to user_task_progress) as the zero-progress defaults: no row yet means the user
// hasn't started the task, which is StatusInProgress at CurrentValue 0.
func taskProgressStatus(status pgtype.Text) string {
	if !status.Valid {
		return tasks.StatusInProgress
	}
	return status.String
}

func taskWithProgressFromListRow(row db.ListTasksWithProgressRow) tasks.TaskWithProgress {
	return tasks.TaskWithProgress{
		Task: tasks.TaskDefinition{
			ID:                 uuidToString(row.ID),
			Code:               row.Code,
			Type:               row.Type,
			Title:              row.Title,
			Description:        row.Description,
			TargetMetric:       row.TargetMetric,
			TargetValue:        int(row.TargetValue),
			RewardDefinitionID: uuidToString(row.RewardDefinitionID),
		},
		Reward: rewards.RewardDefinition{
			Code:        row.RewardCode,
			Title:       row.RewardTitle,
			Description: row.RewardDescription,
		},
		CurrentValue: int(row.CurrentValue.Int32),
		Status:       taskProgressStatus(row.Status),
		CompletedAt:  timestamptzToTimePtr(row.CompletedAt),
		ClaimedAt:    timestamptzToTimePtr(row.ClaimedAt),
	}
}

func taskWithProgressFromGetRow(row db.GetTaskWithProgressByCodeRow) tasks.TaskWithProgress {
	return tasks.TaskWithProgress{
		Task: tasks.TaskDefinition{
			ID:                 uuidToString(row.ID),
			Code:               row.Code,
			Type:               row.Type,
			Title:              row.Title,
			Description:        row.Description,
			TargetMetric:       row.TargetMetric,
			TargetValue:        int(row.TargetValue),
			RewardDefinitionID: uuidToString(row.RewardDefinitionID),
		},
		Reward: rewards.RewardDefinition{
			Code:        row.RewardCode,
			Title:       row.RewardTitle,
			Description: row.RewardDescription,
		},
		CurrentValue: int(row.CurrentValue.Int32),
		Status:       taskProgressStatus(row.Status),
		CompletedAt:  timestamptzToTimePtr(row.CompletedAt),
		ClaimedAt:    timestamptzToTimePtr(row.ClaimedAt),
	}
}
