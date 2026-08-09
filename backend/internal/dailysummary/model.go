package dailysummary

import (
	"context"
	"time"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/event"
	"github.com/google/uuid"
)

type Checkpoint struct {
	UserID            uuid.UUID
	LastCheckInAt     time.Time
	LastEventPosition int64
}

type EventBoundary struct {
	HighWater  int64
	CapturedAt time.Time
}

type CheckInResult struct {
	ShouldShow bool
	Summary    *Summary
}

type Summary struct {
	PeriodStartedAt   time.Time
	PeriodEndedAt     time.Time
	AbsenceDuration   time.Duration
	Pet               PetChanges
	ExperienceEarned  int64
	CoinsEarned       int64
	BestGameScore     *int64
	Rewards           []EarnedReward
	SkippedEventCount int
}

type PetVitals struct {
	Satiety int
	Mood    int
	Energy  int
}

// PetPeriodState is calculated by the pet module using its lazy degradation rules.
type PetPeriodState struct {
	PetID               uuid.UUID
	Before              PetVitals
	After               PetVitals
	Level               int
	Experience          int64
	NextLevel           int
	NextLevelExperience int64
}

type PetChanges struct {
	PetID    uuid.UUID
	Satiety  StatChange
	Mood     StatChange
	Energy   StatChange
	NextGoal ProgressGoal
}

type StatChange struct {
	Before int
	After  int
	Delta  int
}

type ProgressGoal struct {
	CurrentLevel      int
	CurrentExperience int64
	TargetLevel       int
	TargetExperience  int64
}

type EarnedReward struct {
	UserRewardID       uuid.UUID
	RewardDefinitionID uuid.UUID
}

type EventFailureReason string

const (
	EventFailureInvalidPayload    EventFailureReason = "INVALID_PAYLOAD"
	EventFailureUnsupportedSchema EventFailureReason = "UNSUPPORTED_SCHEMA"
)

type EventFailure struct {
	UserID         uuid.UUID
	EventID        uuid.UUID
	GlobalPosition int64
	EventType      event.EventType
	SchemaVersion  int32
	Reason         EventFailureReason
}

type CheckpointRepository interface {
	InsertCheckpoint(ctx context.Context, checkpoint Checkpoint) (bool, error)
	GetCheckpointForUpdate(ctx context.Context, userID uuid.UUID) (Checkpoint, error)
	UpdateCheckpoint(ctx context.Context, checkpoint Checkpoint) error
}

type EventFailureRepository interface {
	RecordEventFailure(ctx context.Context, failure EventFailure) error
}

type EventReader interface {
	CaptureBoundary(ctx context.Context) (EventBoundary, error)
	ListUserEventsByPosition(
		ctx context.Context,
		ownerUserID uuid.UUID,
		afterPosition int64,
		toPosition int64,
		limit int32,
	) ([]event.Event, error)
}

type PetSummaryProvider interface {
	// GetPeriodState must perform bounded, context-aware local work. Implementations
	// must not call remote services because CheckIn holds a checkpoint row lock.
	GetPeriodState(
		ctx context.Context,
		userID uuid.UUID,
		from time.Time,
		to time.Time,
	) (PetPeriodState, error)
}

type Transactor interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}
