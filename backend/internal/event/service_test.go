package event

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type directTransactor struct{}

func (directTransactor) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type memoryStore struct {
	mu        sync.Mutex
	appended  []AppendRequest
	byCommand map[string][]Event
}

func newMemoryStore() *memoryStore {
	return &memoryStore{byCommand: make(map[string][]Event)}
}

func (s *memoryStore) LockCommand(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (s *memoryStore) Append(_ context.Context, request AppendRequest) (Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.appended = append(s.appended, request)
	stored := Event{
		GlobalPosition:    int64(len(s.appended)),
		ID:                request.EventID,
		AggregateType:     request.AggregateType,
		AggregateID:       request.AggregateID,
		OwnerUserID:       request.OwnerUserID,
		AggregateVersion:  request.ExpectedAggregateVersion + 1,
		Type:              request.EventType,
		SchemaVersion:     request.SchemaVersion,
		Payload:           request.Payload,
		Metadata:          request.Metadata,
		ActorUserID:       cloneUUID(request.ActorUserID),
		CommandID:         request.CommandID,
		CommandEventIndex: request.CommandEventIndex,
		OccurredAt:        request.OccurredAt,
		RecordedAt:        request.OccurredAt.Add(time.Millisecond),
	}
	key := commandKey(request.OwnerUserID, request.CommandID)
	s.byCommand[key] = append(s.byCommand[key], stored)
	return stored, nil
}

func (s *memoryStore) GetByID(
	_ context.Context,
	ownerUserID uuid.UUID,
	eventID uuid.UUID,
) (Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, events := range s.byCommand {
		for _, stored := range events {
			if stored.OwnerUserID == ownerUserID && stored.ID == eventID {
				return stored, nil
			}
		}
	}
	return Event{}, ErrEventNotFound
}

func (s *memoryStore) ListByCommandID(
	_ context.Context,
	ownerUserID uuid.UUID,
	commandID uuid.UUID,
) ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	events := s.byCommand[commandKey(ownerUserID, commandID)]
	return append([]Event(nil), events...), nil
}

func (*memoryStore) GetAggregateVersion(
	context.Context,
	uuid.UUID,
	AggregateType,
	uuid.UUID,
) (int64, error) {
	return 0, nil
}

func (*memoryStore) ListAggregateEvents(
	context.Context,
	uuid.UUID,
	AggregateType,
	uuid.UUID,
	int64,
	int32,
) ([]Event, error) {
	return nil, nil
}

func TestServicePublishCreatesOrderedBatch(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	service := NewService(store, directTransactor{})
	fixedTime := time.Date(2026, time.August, 8, 12, 34, 56, 987654321, time.FixedZone("MSK", 3*60*60))
	service.now = func() time.Time { return fixedTime }
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	nextID := 0
	service.newID = func() uuid.UUID {
		id := ids[nextID]
		nextID++
		return id
	}

	ownerID := uuid.New()
	petID := uuid.New()
	gameID := uuid.New()
	command := Command{
		ID:          uuid.New(),
		OwnerUserID: ownerID,
		Streams: []Stream{
			{
				AggregateType:   AggregatePet,
				AggregateID:     petID,
				ExpectedVersion: 4,
				Events: []PendingEvent{
					NewExperienceGranted(GrantPayload{Amount: 10, Source: "mini_game", SourceID: &gameID}),
					NewLevelReached(LevelReachedPayload{Level: 3}),
				},
			},
			{
				AggregateType:   AggregateGameSession,
				AggregateID:     gameID,
				ExpectedVersion: 0,
				Events: []PendingEvent{
					NewGameSessionCompleted(GameSessionCompletedPayload{SessionID: gameID, Score: 120}),
				},
			},
		},
	}

	stored, err := service.Publish(context.Background(), command)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if len(stored) != 3 {
		t.Fatalf("Publish() returned %d events, want 3", len(stored))
	}

	wantVersions := []int64{5, 6, 1}
	for index, request := range store.appended {
		if request.CommandEventIndex != int16(index) {
			t.Errorf("event %d index = %d", index, request.CommandEventIndex)
		}
		if stored[index].AggregateVersion != wantVersions[index] {
			t.Errorf("event %d version = %d, want %d", index, stored[index].AggregateVersion, wantVersions[index])
		}
		wantTime := fixedTime.UTC().Truncate(time.Microsecond)
		if !request.OccurredAt.Equal(wantTime) {
			t.Errorf("event %d occurredAt = %v, want %v", index, request.OccurredAt, wantTime)
		}
	}

	payload, err := DecodePayload[GrantPayload](stored[0])
	if err != nil {
		t.Fatalf("DecodePayload() error = %v", err)
	}
	if payload.Amount != 10 || payload.Source != "mini_game" {
		t.Fatalf("decoded payload = %+v", payload)
	}
}

func TestServicePublishReturnsExistingMatchingCommand(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	service := NewService(store, directTransactor{})
	ownerID := uuid.New()
	petID := uuid.New()
	command := Command{
		ID:          uuid.New(),
		OwnerUserID: ownerID,
		Streams: []Stream{{
			AggregateType:   AggregatePet,
			AggregateID:     petID,
			ExpectedVersion: 0,
			Events: []PendingEvent{
				NewPetCreated(PetCreatedPayload{Name: "Pixel", Species: "cat"}),
			},
		}},
	}

	first, err := service.Publish(context.Background(), command)
	if err != nil {
		t.Fatalf("first Publish() error = %v", err)
	}
	service.now = func() time.Time { return time.Now().Add(24 * time.Hour) }
	second, err := service.Publish(context.Background(), command)
	if err != nil {
		t.Fatalf("second Publish() error = %v", err)
	}
	if len(store.appended) != 1 {
		t.Fatalf("append count = %d, want 1", len(store.appended))
	}
	if first[0].ID != second[0].ID {
		t.Fatalf("replay returned event %s, want %s", second[0].ID, first[0].ID)
	}
}

func TestServicePublishRejectsConflictingReplay(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	service := NewService(store, directTransactor{})
	command := validCommand()
	if _, err := service.Publish(context.Background(), command); err != nil {
		t.Fatalf("first Publish() error = %v", err)
	}
	command.Streams[0].Events[0] = NewExperienceGranted(GrantPayload{Amount: 11, Source: "care"})

	_, err := service.Publish(context.Background(), command)
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("Publish() error = %v, want ErrIdempotencyConflict", err)
	}
}

func TestServicePublishComparesLargeJSONIntegersExactly(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	service := NewService(store, directTransactor{})
	command := validCommand()
	command.Streams[0].Events[0] = NewExperienceGranted(GrantPayload{
		Amount: 9_007_199_254_740_992,
		Source: "care",
	})
	if _, err := service.Publish(context.Background(), command); err != nil {
		t.Fatalf("first Publish() error = %v", err)
	}
	command.Streams[0].Events[0] = NewExperienceGranted(GrantPayload{
		Amount: 9_007_199_254_740_993,
		Source: "care",
	})

	_, err := service.Publish(context.Background(), command)
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("Publish() error = %v, want ErrIdempotencyConflict", err)
	}
}

func TestServicePublishIncludesOccurredAtSourceInFingerprint(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	service := NewService(store, directTransactor{})
	command := validCommand()
	occurredAt := time.Date(2026, time.August, 8, 10, 0, 0, 0, time.UTC)
	command.OccurredAt = &occurredAt
	if _, err := service.Publish(context.Background(), command); err != nil {
		t.Fatalf("first Publish() error = %v", err)
	}
	command.OccurredAt = nil

	_, err := service.Publish(context.Background(), command)
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("Publish() error = %v, want ErrIdempotencyConflict", err)
	}
}

func TestServicePublishValidatesCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Command)
	}{
		{name: "missing command id", mutate: func(command *Command) { command.ID = uuid.Nil }},
		{name: "missing owner", mutate: func(command *Command) { command.OwnerUserID = uuid.Nil }},
		{name: "negative version", mutate: func(command *Command) { command.Streams[0].ExpectedVersion = -1 }},
		{name: "version overflow", mutate: func(command *Command) { command.Streams[0].ExpectedVersion = math.MaxInt64 }},
		{name: "empty stream", mutate: func(command *Command) { command.Streams[0].Events = nil }},
		{name: "invalid payload", mutate: func(command *Command) {
			command.Streams[0].Events[0] = NewExperienceGranted(GrantPayload{Amount: 0, Source: "care"})
		}},
		{name: "duplicate stream", mutate: func(command *Command) {
			command.Streams = append(command.Streams, command.Streams[0])
		}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			command := validCommand()
			test.mutate(&command)
			service := NewService(newMemoryStore(), directTransactor{})
			_, err := service.Publish(context.Background(), command)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("Publish() error = %v, want ErrValidation", err)
			}
		})
	}
}

func validCommand() Command {
	return Command{
		ID:          uuid.New(),
		OwnerUserID: uuid.New(),
		Streams: []Stream{{
			AggregateType:   AggregatePet,
			AggregateID:     uuid.New(),
			ExpectedVersion: 0,
			Events: []PendingEvent{
				NewExperienceGranted(GrantPayload{Amount: 10, Source: "care"}),
			},
		}},
	}
}

func commandKey(ownerUserID, commandID uuid.UUID) string {
	return ownerUserID.String() + ":" + commandID.String()
}
