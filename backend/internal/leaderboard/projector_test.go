package leaderboard

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"testing"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/event"
	"github.com/google/uuid"
)

func TestProjectNextBatchBuildsReadModelsAndKeepsBestScore(t *testing.T) {
	userID := uuid.New()
	petID := uuid.New()
	bestSessionID := uuid.New()
	lowerSessionID := uuid.New()
	state := newProjectionState([]event.Event{
		projectionEvent(t, 1, userID, event.AggregatePet, petID, event.PetCreated,
			event.PetCreatedPayload{Name: " Pixel ", Species: " cat "}),
		projectionEvent(t, 2, userID, event.AggregatePet, petID, event.LevelReached,
			event.LevelReachedPayload{Level: 3}),
		projectionEvent(t, 3, userID, event.AggregateGameSession, bestSessionID,
			event.GameSessionCompleted,
			event.GameSessionCompletedPayload{SessionID: bestSessionID, Score: 50}),
		projectionEvent(t, 4, userID, event.AggregateGameSession, lowerSessionID,
			event.GameSessionCompleted,
			event.GameSessionCompletedPayload{SessionID: lowerSessionID, Score: 10}),
		projectionEvent(t, 5, userID, event.AggregatePet, petID, event.ExperienceGranted,
			event.GrantPayload{Amount: 1, Source: "test"}),
	})
	projector := newTestProjector(t, state)

	processed, err := projector.ProjectNextBatch(context.Background())
	if err != nil {
		t.Fatalf("ProjectNextBatch() error = %v", err)
	}
	if processed != 5 || state.checkpoint != 5 {
		t.Fatalf("processed=%d checkpoint=%d, want 5", processed, state.checkpoint)
	}
	pet := state.pets[userID]
	if pet.PetName != "Pixel" || pet.Species != "cat" || pet.Level != 3 || pet.ReachedPosition != 2 {
		t.Fatalf("pet projection = %+v", pet)
	}
	score := state.scores[userID]
	if score.BestScore != 50 || score.SessionID != bestSessionID || score.AchievedPosition != 3 {
		t.Fatalf("score projection = %+v", score)
	}
	if len(state.failures) != 0 {
		t.Fatalf("failures = %+v, want none", state.failures)
	}

	processed, err = projector.ProjectNextBatch(context.Background())
	if err != nil || processed != 0 {
		t.Fatalf("second ProjectNextBatch() processed=%d error=%v", processed, err)
	}
}

func TestProjectNextBatchQuarantinesInvalidRelevantEvents(t *testing.T) {
	userID := uuid.New()
	petID := uuid.New()
	sessionID := uuid.New()
	invalidAggregate := projectionEvent(
		t,
		1,
		userID,
		event.AggregateGameSession,
		petID,
		event.PetCreated,
		event.PetCreatedPayload{Name: "Pixel", Species: "cat"},
	)
	invalidPayload := projectionEvent(
		t,
		2,
		userID,
		event.AggregatePet,
		petID,
		event.LevelReached,
		event.LevelReachedPayload{Level: 0},
	)
	unsupportedSchema := projectionEvent(
		t,
		3,
		userID,
		event.AggregateGameSession,
		sessionID,
		event.GameSessionCompleted,
		event.GameSessionCompletedPayload{SessionID: sessionID, Score: 1},
	)
	unsupportedSchema.SchemaVersion = 2
	overflowedLevel := projectionEvent(
		t,
		4,
		userID,
		event.AggregatePet,
		petID,
		event.LevelReached,
		event.LevelReachedPayload{Level: int(int64(math.MaxInt32) + 1)},
	)
	state := newProjectionState([]event.Event{
		invalidAggregate,
		invalidPayload,
		unsupportedSchema,
		overflowedLevel,
	})
	projector := newTestProjector(t, state)

	processed, err := projector.ProjectNextBatch(context.Background())
	if err != nil {
		t.Fatalf("ProjectNextBatch() error = %v", err)
	}
	if processed != 4 || state.checkpoint != 4 {
		t.Fatalf("processed=%d checkpoint=%d, want 4", processed, state.checkpoint)
	}
	wantReasons := []EventFailureReason{
		FailureInvalidAggregate,
		FailureInvalidPayload,
		FailureUnsupportedSchema,
		FailureInvalidPayload,
	}
	if len(state.failures) != len(wantReasons) {
		t.Fatalf("failures = %+v", state.failures)
	}
	for index, want := range wantReasons {
		if state.failures[index].Reason != want {
			t.Fatalf("failure[%d] = %s, want %s", index, state.failures[index].Reason, want)
		}
	}
}

func TestProjectNextBatchQuarantinesRepeatedPetCreation(t *testing.T) {
	userID := uuid.New()
	state := newProjectionState([]event.Event{
		projectionEvent(t, 1, userID, event.AggregatePet, uuid.New(), event.PetCreated,
			event.PetCreatedPayload{Name: "One", Species: "cat"}),
		projectionEvent(t, 2, userID, event.AggregatePet, uuid.New(), event.PetCreated,
			event.PetCreatedPayload{Name: "Two", Species: "dog"}),
	})
	projector := newTestProjector(t, state)

	processed, err := projector.ProjectNextBatch(context.Background())
	if err != nil {
		t.Fatalf("ProjectNextBatch() error = %v", err)
	}
	if processed != 2 || len(state.failures) != 1 ||
		state.failures[0].Reason != FailureInvalidSequence {
		t.Fatalf("processed=%d failures=%+v", processed, state.failures)
	}
}

func TestProjectNextBatchDoesNotAdvanceCheckpointWhenQuarantineFails(t *testing.T) {
	userID := uuid.New()
	bad := projectionEvent(
		t,
		1,
		userID,
		event.AggregatePet,
		uuid.New(),
		event.LevelReached,
		event.LevelReachedPayload{Level: 0},
	)
	wantErr := errors.New("quarantine unavailable")
	state := newProjectionState([]event.Event{bad})
	state.failureErr = wantErr
	projector := newTestProjector(t, state)

	processed, err := projector.ProjectNextBatch(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("ProjectNextBatch() error = %v, want %v", err, wantErr)
	}
	if processed != 0 || state.checkpoint != 0 {
		t.Fatalf("processed=%d checkpoint=%d, want no advance", processed, state.checkpoint)
	}
}

func newTestProjector(t *testing.T, state *projectionState) *Projector {
	t.Helper()
	projector, err := NewProjector(state, state, transactorStub{})
	if err != nil {
		t.Fatalf("NewProjector() error = %v", err)
	}
	return projector
}

type projectionState struct {
	events     []event.Event
	checkpoint int64
	pets       map[uuid.UUID]PetProjection
	scores     map[uuid.UUID]GameScoreProjection
	failures   []EventFailure
	failureErr error
}

func newProjectionState(events []event.Event) *projectionState {
	return &projectionState{
		events: events,
		pets:   make(map[uuid.UUID]PetProjection),
		scores: make(map[uuid.UUID]GameScoreProjection),
	}
}

func (*projectionState) LockProjection(context.Context, string) error {
	return nil
}

func (s *projectionState) GetProjectionCheckpoint(context.Context, string) (int64, error) {
	return s.checkpoint, nil
}

func (s *projectionState) ListEventsAfterPosition(
	_ context.Context,
	afterPosition int64,
	limit int32,
) ([]event.Event, error) {
	result := make([]event.Event, 0, limit)
	for _, stored := range s.events {
		if stored.GlobalPosition > afterPosition {
			result = append(result, stored)
			if len(result) == int(limit) {
				break
			}
		}
	}
	return result, nil
}

func (s *projectionState) SaveProjectionCheckpoint(
	_ context.Context,
	_ string,
	position int64,
) error {
	s.checkpoint = position
	return nil
}

func (s *projectionState) InsertPet(
	_ context.Context,
	pet PetProjection,
) (bool, error) {
	if _, exists := s.pets[pet.UserID]; exists {
		return false, nil
	}
	s.pets[pet.UserID] = pet
	return true, nil
}

func (s *projectionState) AdvancePetLevel(
	_ context.Context,
	pet PetProjection,
) (bool, error) {
	current, exists := s.pets[pet.UserID]
	if !exists || current.PetID != pet.PetID || current.Level >= pet.Level {
		return false, nil
	}
	current.Level = pet.Level
	current.ReachedPosition = pet.ReachedPosition
	s.pets[pet.UserID] = current
	return true, nil
}

func (s *projectionState) UpsertGameScore(
	_ context.Context,
	score GameScoreProjection,
) (bool, error) {
	current, exists := s.scores[score.UserID]
	if exists && current.BestScore >= score.BestScore {
		return false, nil
	}
	s.scores[score.UserID] = score
	return true, nil
}

func (s *projectionState) RecordEventFailure(
	_ context.Context,
	failure EventFailure,
) error {
	if s.failureErr != nil {
		return s.failureErr
	}
	s.failures = append(s.failures, failure)
	return nil
}

type transactorStub struct{}

func (transactorStub) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func projectionEvent(
	t *testing.T,
	position int64,
	userID uuid.UUID,
	aggregateType event.AggregateType,
	aggregateID uuid.UUID,
	eventType event.EventType,
	payload any,
) event.Event {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return event.Event{
		GlobalPosition: position,
		ID:             uuid.New(),
		AggregateType:  aggregateType,
		AggregateID:    aggregateID,
		OwnerUserID:    userID,
		Type:           eventType,
		SchemaVersion:  event.SchemaVersionV1,
		Payload:        encoded,
	}
}
