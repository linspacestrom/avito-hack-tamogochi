package event

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	maxCommandEvents    = 1000
	minPageSize         = 1
	maxPageSize         = 500
	maxPayloadSizeBytes = 64 * 1024
	versionIncrement    = 1
)

var (
	serverTimeMetadata = json.RawMessage(`{"occurredAtSource":"server"}`)
	callerTimeMetadata = json.RawMessage(`{"occurredAtSource":"caller"}`)
)

// Service validates and atomically publishes owner-scoped event commands.
type Service struct {
	store      Store
	transactor Transactor
	now        func() time.Time
	newID      func() uuid.UUID
}

type preparedEvent struct {
	eventID          uuid.UUID
	aggregateType    AggregateType
	aggregateID      uuid.UUID
	aggregateVersion int64
	eventType        EventType
	payload          json.RawMessage
	commandIndex     int16
}

type preparedCommand struct {
	commandID          uuid.UUID
	ownerUserID        uuid.UUID
	actorUserID        *uuid.UUID
	occurredAt         time.Time
	occurredAtExplicit bool
	metadata           json.RawMessage
	events             []preparedEvent
}

// eventIdentity contains only persisted fields that participate in idempotency.
// Generated IDs, global positions, and recording times are intentionally excluded.
type eventIdentity struct {
	ownerUserID      uuid.UUID
	commandID        uuid.UUID
	commandIndex     int16
	aggregateType    AggregateType
	aggregateID      uuid.UUID
	aggregateVersion int64
	eventType        EventType
	schemaVersion    int32
}

// NewService creates an Event Service backed by the supplied store and transactor.
func NewService(store Store, transactor Transactor) *Service {
	return &Service{
		store:      store,
		transactor: transactor,
		now:        time.Now,
		newID:      uuid.New,
	}
}

// Publish appends every event in a command or returns the original events for an exact replay.
func (s *Service) Publish(ctx context.Context, command Command) ([]Event, error) {
	prepared, err := s.prepare(command)
	if err != nil {
		return nil, err
	}

	var result []Event
	err = s.transactor.Do(ctx, func(txCtx context.Context) error {
		// The lock and the following lookup are separate statements so READ COMMITTED
		// observes a concurrent command that committed while this transaction waited.
		if lockErr := s.store.LockCommand(
			txCtx,
			prepared.ownerUserID,
			prepared.commandID,
		); lockErr != nil {
			return fmt.Errorf("lock event command: %w", lockErr)
		}

		existing, listErr := s.store.ListByCommandID(
			txCtx,
			prepared.ownerUserID,
			prepared.commandID,
		)
		if listErr != nil {
			return fmt.Errorf("list command events: %w", listErr)
		}
		if len(existing) > 0 {
			if !commandMatches(prepared, existing) {
				return ErrIdempotencyConflict
			}

			result = existing
			return nil
		}

		result = make([]Event, 0, len(prepared.events))
		for _, pending := range prepared.events {
			stored, appendErr := s.store.Append(txCtx, AppendRequest{
				EventID:                  pending.eventID,
				AggregateType:            pending.aggregateType,
				AggregateID:              pending.aggregateID,
				OwnerUserID:              prepared.ownerUserID,
				ExpectedAggregateVersion: pending.aggregateVersion - versionIncrement,
				EventType:                pending.eventType,
				SchemaVersion:            SchemaVersionV1,
				Payload:                  pending.payload,
				Metadata:                 prepared.metadata,
				ActorUserID:              prepared.actorUserID,
				CommandID:                prepared.commandID,
				CommandEventIndex:        pending.commandIndex,
				OccurredAt:               prepared.occurredAt,
			})
			if appendErr != nil {
				return fmt.Errorf("append event %d: %w", pending.commandIndex, appendErr)
			}

			result = append(result, stored)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("publish event command: %w", err)
	}

	return result, nil
}

// GetByID returns one event visible to its owner.
func (s *Service) GetByID(
	ctx context.Context,
	ownerUserID uuid.UUID,
	eventID uuid.UUID,
) (Event, error) {
	if ownerUserID == uuid.Nil || eventID == uuid.Nil {
		return Event{}, fmt.Errorf("%w: owner and event IDs are required", ErrValidation)
	}

	stored, err := s.store.GetByID(ctx, ownerUserID, eventID)
	if err != nil {
		return Event{}, fmt.Errorf("get event: %w", err)
	}

	return stored, nil
}

// ListByCommandID returns a command's events in command index order.
func (s *Service) ListByCommandID(
	ctx context.Context,
	ownerUserID uuid.UUID,
	commandID uuid.UUID,
) ([]Event, error) {
	if ownerUserID == uuid.Nil || commandID == uuid.Nil {
		return nil, fmt.Errorf("%w: owner and command IDs are required", ErrValidation)
	}

	events, err := s.store.ListByCommandID(ctx, ownerUserID, commandID)
	if err != nil {
		return nil, fmt.Errorf("list command events: %w", err)
	}

	return events, nil
}

// GetAggregateVersion returns the current owner-scoped stream version.
func (s *Service) GetAggregateVersion(
	ctx context.Context,
	ownerUserID uuid.UUID,
	aggregateType AggregateType,
	aggregateID uuid.UUID,
) (int64, error) {
	if err := validateAggregate(ownerUserID, aggregateType, aggregateID); err != nil {
		return 0, err
	}

	version, err := s.store.GetAggregateVersion(
		ctx,
		ownerUserID,
		aggregateType,
		aggregateID,
	)
	if err != nil {
		return 0, fmt.Errorf("get aggregate version: %w", err)
	}

	return version, nil
}

// ListAggregateEvents returns an ordered page after the supplied aggregate version.
func (s *Service) ListAggregateEvents(
	ctx context.Context,
	ownerUserID uuid.UUID,
	aggregateType AggregateType,
	aggregateID uuid.UUID,
	afterVersion int64,
	limit int32,
) ([]Event, error) {
	if err := validateAggregate(ownerUserID, aggregateType, aggregateID); err != nil {
		return nil, err
	}
	if afterVersion < 0 || limit < minPageSize || limit > maxPageSize {
		return nil, fmt.Errorf("%w: invalid pagination", ErrValidation)
	}

	events, err := s.store.ListAggregateEvents(
		ctx,
		ownerUserID,
		aggregateType,
		aggregateID,
		afterVersion,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list aggregate events: %w", err)
	}

	return events, nil
}

func (s *Service) prepare(command Command) (preparedCommand, error) {
	if command.ID == uuid.Nil || command.OwnerUserID == uuid.Nil {
		return preparedCommand{}, fmt.Errorf(
			"%w: command and owner IDs are required",
			ErrValidation,
		)
	}
	if command.ActorUserID != nil && *command.ActorUserID == uuid.Nil {
		return preparedCommand{}, fmt.Errorf("%w: actor ID is empty", ErrValidation)
	}
	if len(command.Streams) == 0 {
		return preparedCommand{}, fmt.Errorf("%w: command has no streams", ErrValidation)
	}

	occurredAt := s.now().UTC().Truncate(time.Microsecond)
	explicitTime := command.OccurredAt != nil
	if explicitTime {
		if command.OccurredAt.IsZero() {
			return preparedCommand{}, fmt.Errorf("%w: occurred time is empty", ErrValidation)
		}
		occurredAt = command.OccurredAt.UTC().Truncate(time.Microsecond)
	}

	seenStreams := make(map[string]struct{}, len(command.Streams))
	prepared := preparedCommand{
		commandID:          command.ID,
		ownerUserID:        command.OwnerUserID,
		actorUserID:        cloneUUID(command.ActorUserID),
		occurredAt:         occurredAt,
		occurredAtExplicit: explicitTime,
		metadata:           serverTimeMetadata,
	}
	// Recording the time source makes explicit and server-assigned timestamps
	// distinguishable when a command ID is replayed.
	if explicitTime {
		prepared.metadata = callerTimeMetadata
	}

	for _, stream := range command.Streams {
		if err := validateAggregate(
			command.OwnerUserID,
			stream.AggregateType,
			stream.AggregateID,
		); err != nil {
			return preparedCommand{}, err
		}
		if stream.ExpectedVersion < 0 {
			return preparedCommand{}, fmt.Errorf("%w: negative expected version", ErrValidation)
		}
		if len(stream.Events) == 0 {
			return preparedCommand{}, fmt.Errorf("%w: stream has no events", ErrValidation)
		}
		if stream.ExpectedVersion > math.MaxInt64-int64(len(stream.Events))*versionIncrement {
			return preparedCommand{}, fmt.Errorf("%w: aggregate version overflows", ErrValidation)
		}

		streamKey := string(stream.AggregateType) + ":" + stream.AggregateID.String()
		if _, exists := seenStreams[streamKey]; exists {
			return preparedCommand{}, fmt.Errorf("%w: aggregate stream is duplicated", ErrValidation)
		}
		seenStreams[streamKey] = struct{}{}

		for streamIndex, pending := range stream.Events {
			if len(prepared.events) >= maxCommandEvents {
				return preparedCommand{}, fmt.Errorf("%w: command has too many events", ErrValidation)
			}

			payload, err := encodeAndValidatePayload(pending)
			if err != nil {
				return preparedCommand{}, err
			}

			prepared.events = append(prepared.events, preparedEvent{
				eventID:       s.newID(),
				aggregateType: stream.AggregateType,
				aggregateID:   stream.AggregateID,
				aggregateVersion: stream.ExpectedVersion +
					int64(streamIndex)*versionIncrement + versionIncrement,
				eventType:    pending.eventType,
				payload:      payload,
				commandIndex: int16(len(prepared.events)),
			})
		}
	}

	return prepared, nil
}

// commandMatches verifies the semantic fingerprint of an idempotent replay.
func commandMatches(expected preparedCommand, stored []Event) bool {
	if len(expected.events) != len(stored) {
		return false
	}

	for index, actual := range stored {
		if !eventMatches(expected, expected.events[index], actual) {
			return false
		}
	}

	return true
}

func eventMatches(
	expected preparedCommand,
	pending preparedEvent,
	actual Event,
) bool {
	expectedIdentity := eventIdentity{
		ownerUserID:      expected.ownerUserID,
		commandID:        expected.commandID,
		commandIndex:     pending.commandIndex,
		aggregateType:    pending.aggregateType,
		aggregateID:      pending.aggregateID,
		aggregateVersion: pending.aggregateVersion,
		eventType:        pending.eventType,
		schemaVersion:    SchemaVersionV1,
	}
	actualIdentity := eventIdentity{
		ownerUserID:      actual.OwnerUserID,
		commandID:        actual.CommandID,
		commandIndex:     actual.CommandEventIndex,
		aggregateType:    actual.AggregateType,
		aggregateID:      actual.AggregateID,
		aggregateVersion: actual.AggregateVersion,
		eventType:        actual.Type,
		schemaVersion:    actual.SchemaVersion,
	}

	if expectedIdentity != actualIdentity {
		return false
	}
	if !sameUUID(actual.ActorUserID, expected.actorUserID) {
		return false
	}
	if !sameJSON(actual.Payload, pending.payload) ||
		!sameJSON(actual.Metadata, expected.metadata) {
		return false
	}

	return !expected.occurredAtExplicit || actual.OccurredAt.Equal(expected.occurredAt)
}

func encodeAndValidatePayload(pending PendingEvent) (json.RawMessage, error) {
	var err error
	switch payload := pending.payload.(type) {
	case PetCreatedPayload:
		if pending.eventType != PetCreated || blank(payload.Name) || blank(payload.Species) {
			err = ErrValidation
		}
	case CareActionCompletedPayload:
		if pending.eventType != CareActionCompleted || blank(payload.Action) {
			err = ErrValidation
		}
	case GrantPayload:
		if (pending.eventType != ExperienceGranted && pending.eventType != CoinsGranted) ||
			payload.Amount <= 0 || blank(payload.Source) || invalidOptionalUUID(payload.SourceID) {
			err = ErrValidation
		}
	case LevelReachedPayload:
		if pending.eventType != LevelReached || payload.Level <= 0 {
			err = ErrValidation
		}
	case GameSessionCompletedPayload:
		if pending.eventType != GameSessionCompleted || payload.SessionID == uuid.Nil || payload.Score < 0 {
			err = ErrValidation
		}
	case RewardPayload:
		if (pending.eventType != RewardGranted &&
			pending.eventType != RewardRedeemed &&
			pending.eventType != RewardExpired) ||
			payload.UserRewardID == uuid.Nil ||
			payload.RewardDefinitionID == uuid.Nil ||
			invalidOptionalUUID(payload.SourceEventID) {
			err = ErrValidation
		}
	default:
		err = ErrValidation
	}
	if err != nil {
		return nil, fmt.Errorf("%w: invalid %s payload", err, pending.eventType)
	}

	payload, err := json.Marshal(pending.payload)
	if err != nil {
		return nil, fmt.Errorf("%w: encode %s payload: %v", ErrValidation, pending.eventType, err)
	}
	if len(payload) > maxPayloadSizeBytes {
		return nil, fmt.Errorf("%w: payload is too large", ErrValidation)
	}

	return payload, nil
}

func validateAggregate(
	ownerUserID uuid.UUID,
	aggregateType AggregateType,
	aggregateID uuid.UUID,
) error {
	if ownerUserID == uuid.Nil || aggregateID == uuid.Nil {
		return fmt.Errorf("%w: owner and aggregate IDs are required", ErrValidation)
	}
	if aggregateType != AggregatePet &&
		aggregateType != AggregateReward &&
		aggregateType != AggregateGameSession {
		return fmt.Errorf("%w: unsupported aggregate type", ErrValidation)
	}

	return nil
}

// sameJSON compares JSON values without converting large integers to float64.
func sameJSON(left, right json.RawMessage) bool {
	leftValue, leftOK := decodeJSON(left)
	rightValue, rightOK := decodeJSON(right)
	if !leftOK || !rightOK {
		return false
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func decodeJSON(value json.RawMessage) (any, bool) {
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.UseNumber()

	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, false
	}

	return decoded, true
}

func sameUUID(left, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func cloneUUID(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func invalidOptionalUUID(value *uuid.UUID) bool {
	return value != nil && *value == uuid.Nil
}

func blank(value string) bool {
	return strings.TrimSpace(value) == ""
}
