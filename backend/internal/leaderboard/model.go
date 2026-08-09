package leaderboard

import (
	"context"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/event"
	"github.com/google/uuid"
)

const (
	DefaultPageSize int32 = 20
	MaxPageSize     int32 = 100
)

type Page struct {
	Limit  int32
	Offset int32
}

type PetLevelEntry struct {
	Rank        int64
	UserID      uuid.UUID
	DisplayName string
	PetID       uuid.UUID
	PetName     string
	Species     string
	Level       int32
}

type GameScoreEntry struct {
	Rank        int64
	UserID      uuid.UUID
	DisplayName string
	BestScore   int64
}

type UserPositions struct {
	PetLevelRank  *int64
	GameScoreRank *int64
}

type PetProjection struct {
	UserID          uuid.UUID
	PetID           uuid.UUID
	PetName         string
	Species         string
	Level           int32
	ReachedPosition int64
}

type GameScoreProjection struct {
	UserID           uuid.UUID
	SessionID        uuid.UUID
	BestScore        int64
	AchievedPosition int64
}

type EventFailureReason string

const (
	FailureInvalidPayload    EventFailureReason = "INVALID_PAYLOAD"
	FailureInvalidAggregate  EventFailureReason = "INVALID_AGGREGATE"
	FailureInvalidSequence   EventFailureReason = "INVALID_SEQUENCE"
	FailureUnsupportedSchema EventFailureReason = "UNSUPPORTED_SCHEMA"
)

type EventFailure struct {
	EventID        uuid.UUID
	UserID         uuid.UUID
	GlobalPosition int64
	EventType      event.EventType
	SchemaVersion  int32
	Reason         EventFailureReason
}

type ProjectionEventStore interface {
	LockProjection(ctx context.Context, projectionName string) error
	GetProjectionCheckpoint(ctx context.Context, projectionName string) (int64, error)
	ListEventsAfterPosition(ctx context.Context, afterPosition int64, limit int32) ([]event.Event, error)
	SaveProjectionCheckpoint(ctx context.Context, projectionName string, position int64) error
}

type ProjectionWriter interface {
	InsertPet(ctx context.Context, pet PetProjection) (bool, error)
	AdvancePetLevel(ctx context.Context, pet PetProjection) (bool, error)
	UpsertGameScore(ctx context.Context, score GameScoreProjection) (bool, error)
	RecordEventFailure(ctx context.Context, failure EventFailure) error
}

type Reader interface {
	ListPetLevels(ctx context.Context, page Page) ([]PetLevelEntry, error)
	ListGameScores(ctx context.Context, page Page) ([]GameScoreEntry, error)
	GetUserPositions(ctx context.Context, userID uuid.UUID) (UserPositions, error)
}

type Transactor interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}
