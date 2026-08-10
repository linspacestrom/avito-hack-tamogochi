// Package tasks holds the task catalog and per-user task progress, including claiming the
// reward for a completed task.
package tasks

import (
	"time"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
)

// UserTaskProgress.Status values. Matches the CHECK constraint on user_task_progress.status.
const (
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusClaimed    = "claimed"
)

type TaskDefinition struct {
	ID                 string
	Code               string
	Type               string
	Title              string
	Description        string
	TargetMetric       string
	TargetValue        int
	RewardDefinitionID string
}

// TaskWithProgress is a task joined with the calling user's progress on it, plus the reward it
// grants. CurrentValue/Status/CompletedAt/ClaimedAt take their zero values (0, StatusInProgress,
// nil, nil) when the user has no user_task_progress row yet — that's the normal state for a
// task nobody has made progress on, not an error.
type TaskWithProgress struct {
	Task         TaskDefinition
	Reward       rewards.RewardDefinition
	CurrentValue int
	Status       string
	CompletedAt  *time.Time
	ClaimedAt    *time.Time
}
