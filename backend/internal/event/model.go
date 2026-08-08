package event

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SchemaVersionV1 identifies the initial payload contract for domain events.
const SchemaVersionV1 int32 = 1

// Event is an immutable domain event returned from the Event Store.
type Event struct {
	GlobalPosition    int64
	ID                uuid.UUID
	AggregateType     AggregateType
	AggregateID       uuid.UUID
	OwnerUserID       uuid.UUID
	AggregateVersion  int64
	Type              EventType
	SchemaVersion     int32
	Payload           json.RawMessage
	Metadata          json.RawMessage
	ActorUserID       *uuid.UUID
	CommandID         uuid.UUID
	CommandEventIndex int16
	OccurredAt        time.Time
	RecordedAt        time.Time
}

// Command groups events that must be appended atomically and idempotently.
type Command struct {
	ID          uuid.UUID
	OwnerUserID uuid.UUID
	ActorUserID *uuid.UUID
	OccurredAt  *time.Time
	Streams     []Stream
}

// Stream describes ordered events for one aggregate and its expected version.
type Stream struct {
	AggregateType   AggregateType
	AggregateID     uuid.UUID
	ExpectedVersion int64
	Events          []PendingEvent
}

// AppendRequest contains a validated event ready for persistence.
type AppendRequest struct {
	EventID                  uuid.UUID
	AggregateType            AggregateType
	AggregateID              uuid.UUID
	OwnerUserID              uuid.UUID
	ExpectedAggregateVersion int64
	EventType                EventType
	SchemaVersion            int32
	Payload                  json.RawMessage
	Metadata                 json.RawMessage
	ActorUserID              *uuid.UUID
	CommandID                uuid.UUID
	CommandEventIndex        int16
	OccurredAt               time.Time
}

// Store defines owner-scoped Event Store persistence operations.
type Store interface {
	LockCommand(ctx context.Context, ownerUserID, commandID uuid.UUID) error
	Append(ctx context.Context, request AppendRequest) (Event, error)
	GetByID(ctx context.Context, ownerUserID, eventID uuid.UUID) (Event, error)
	ListByCommandID(ctx context.Context, ownerUserID, commandID uuid.UUID) ([]Event, error)
	GetAggregateVersion(
		ctx context.Context,
		ownerUserID uuid.UUID,
		aggregateType AggregateType,
		aggregateID uuid.UUID,
	) (int64, error)
	ListAggregateEvents(
		ctx context.Context,
		ownerUserID uuid.UUID,
		aggregateType AggregateType,
		aggregateID uuid.UUID,
		afterVersion int64,
		limit int32,
	) ([]Event, error)
}

// Transactor joins Event operations to an existing transaction or starts a new one.
type Transactor interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

// DecodePayload decodes a stored payload into its version-specific Go type.
func DecodePayload[T any](stored Event) (T, error) {
	var payload T
	err := json.Unmarshal(stored.Payload, &payload)
	return payload, err
}
