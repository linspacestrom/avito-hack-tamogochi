package leaderboard

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/event"
	"github.com/google/uuid"
)

const (
	projectionName      = "leaderboard_v1"
	projectionBatchSize = int32(500)
	initialPetLevel     = int32(1)
	maxPetNameBytes     = 128
	maxSpeciesBytes     = 64
)

type Projector struct {
	events     ProjectionEventStore
	writer     ProjectionWriter
	transactor Transactor
}

// NewProjector creates the transactional Event Store projector.
func NewProjector(
	events ProjectionEventStore,
	writer ProjectionWriter,
	transactor Transactor,
) (*Projector, error) {
	if events == nil || writer == nil || transactor == nil {
		return nil, fmt.Errorf("%w: projector dependencies are required", ErrValidation)
	}
	return &Projector{events: events, writer: writer, transactor: transactor}, nil
}

// ProjectNextBatch applies at most one ordered Event Store page and advances the
// projection checkpoint in the same transaction.
func (p *Projector) ProjectNextBatch(ctx context.Context) (int, error) {
	processed := 0
	err := p.transactor.Do(ctx, func(txCtx context.Context) error {
		if err := p.events.LockProjection(txCtx, projectionName); err != nil {
			return fmt.Errorf("lock leaderboard projection: %w", err)
		}
		checkpoint, err := p.events.GetProjectionCheckpoint(txCtx, projectionName)
		if err != nil {
			return fmt.Errorf("get leaderboard checkpoint: %w", err)
		}
		storedEvents, err := p.events.ListEventsAfterPosition(
			txCtx,
			checkpoint,
			projectionBatchSize,
		)
		if err != nil {
			return fmt.Errorf("list leaderboard events: %w", err)
		}
		if len(storedEvents) > int(projectionBatchSize) {
			return fmt.Errorf("%w: projection page exceeds configured limit", ErrInvalidStoredEvent)
		}

		lastProcessedPosition := checkpoint
		for _, stored := range storedEvents {
			if stored.ID == uuid.Nil || stored.OwnerUserID == uuid.Nil ||
				stored.GlobalPosition <= lastProcessedPosition {
				return fmt.Errorf("%w: event identity or position", ErrInvalidStoredEvent)
			}
			if err := p.applyEvent(txCtx, stored); err != nil {
				failure, ok := projectionFailureFrom(err, stored)
				if !ok {
					return err
				}
				if recordErr := p.writer.RecordEventFailure(txCtx, failure); recordErr != nil {
					return fmt.Errorf("record leaderboard event failure: %w", recordErr)
				}
			}
			lastProcessedPosition = stored.GlobalPosition
		}

		if len(storedEvents) == 0 {
			return nil
		}
		if err := p.events.SaveProjectionCheckpoint(
			txCtx,
			projectionName,
			lastProcessedPosition,
		); err != nil {
			return fmt.Errorf("save leaderboard checkpoint: %w", err)
		}
		processed = len(storedEvents)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return processed, nil
}

func (p *Projector) applyEvent(ctx context.Context, stored event.Event) error {
	switch stored.Type {
	case event.PetCreated, event.LevelReached, event.GameSessionCompleted:
		if stored.SchemaVersion != event.SchemaVersionV1 {
			return newProjectionFailure(FailureUnsupportedSchema, "unsupported event schema")
		}
	default:
		return nil
	}

	switch stored.Type {
	case event.PetCreated:
		if stored.AggregateType != event.AggregatePet || stored.AggregateID == uuid.Nil {
			return newProjectionFailure(FailureInvalidAggregate, "invalid pet aggregate")
		}
		var payload event.PetCreatedPayload
		if err := json.Unmarshal(stored.Payload, &payload); err != nil {
			return newProjectionFailure(FailureInvalidPayload, "decode pet payload")
		}
		payload.Name = strings.TrimSpace(payload.Name)
		payload.Species = strings.TrimSpace(payload.Species)
		if payload.Name == "" || len(payload.Name) > maxPetNameBytes ||
			payload.Species == "" || len(payload.Species) > maxSpeciesBytes {
			return newProjectionFailure(FailureInvalidPayload, "validate pet payload")
		}
		inserted, err := p.writer.InsertPet(ctx, PetProjection{
			UserID:          stored.OwnerUserID,
			PetID:           stored.AggregateID,
			PetName:         payload.Name,
			Species:         payload.Species,
			Level:           initialPetLevel,
			ReachedPosition: stored.GlobalPosition,
		})
		if err != nil {
			return fmt.Errorf("insert leaderboard pet: %w", err)
		}
		if !inserted {
			return newProjectionFailure(FailureInvalidSequence, "pet already exists")
		}
	case event.LevelReached:
		if stored.AggregateType != event.AggregatePet || stored.AggregateID == uuid.Nil {
			return newProjectionFailure(FailureInvalidAggregate, "invalid level aggregate")
		}
		var payload event.LevelReachedPayload
		if err := json.Unmarshal(stored.Payload, &payload); err != nil ||
			payload.Level <= 0 || payload.Level > math.MaxInt32 {
			return newProjectionFailure(FailureInvalidPayload, "decode level payload")
		}
		advanced, err := p.writer.AdvancePetLevel(ctx, PetProjection{
			UserID:          stored.OwnerUserID,
			PetID:           stored.AggregateID,
			Level:           int32(payload.Level),
			ReachedPosition: stored.GlobalPosition,
		})
		if err != nil {
			return fmt.Errorf("advance leaderboard pet: %w", err)
		}
		if !advanced {
			return newProjectionFailure(FailureInvalidSequence, "level did not advance")
		}
	case event.GameSessionCompleted:
		if stored.AggregateType != event.AggregateGameSession || stored.AggregateID == uuid.Nil {
			return newProjectionFailure(FailureInvalidAggregate, "invalid game aggregate")
		}
		var payload event.GameSessionCompletedPayload
		if err := json.Unmarshal(stored.Payload, &payload); err != nil ||
			payload.SessionID == uuid.Nil || payload.Score < 0 {
			return newProjectionFailure(FailureInvalidPayload, "decode game payload")
		}
		if stored.AggregateID != payload.SessionID {
			return newProjectionFailure(FailureInvalidAggregate, "game session mismatch")
		}
		_, err := p.writer.UpsertGameScore(ctx, GameScoreProjection{
			UserID:           stored.OwnerUserID,
			SessionID:        payload.SessionID,
			BestScore:        int64(payload.Score),
			AchievedPosition: stored.GlobalPosition,
		})
		if err != nil {
			return fmt.Errorf("upsert leaderboard game score: %w", err)
		}
	}
	return nil
}

type projectionFailureError struct {
	reason EventFailureReason
	text   string
}

func newProjectionFailure(reason EventFailureReason, text string) error {
	return projectionFailureError{reason: reason, text: text}
}

func (e projectionFailureError) Error() string {
	return e.text
}

func projectionFailureFrom(err error, stored event.Event) (EventFailure, bool) {
	failureErr, ok := err.(projectionFailureError)
	if !ok {
		return EventFailure{}, false
	}
	return EventFailure{
		EventID:        stored.ID,
		UserID:         stored.OwnerUserID,
		GlobalPosition: stored.GlobalPosition,
		EventType:      stored.Type,
		SchemaVersion:  stored.SchemaVersion,
		Reason:         failureErr.reason,
	}, true
}
