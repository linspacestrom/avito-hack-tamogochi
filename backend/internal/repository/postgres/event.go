package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/dailysummary"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/event"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository/postgres/eventsqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

const uuidByteLength = 16

// EventRepository persists events through owner-scoped database functions.
type EventRepository struct {
	repo *Repository
}

var _ event.Store = (*EventRepository)(nil)

// NewEventRepository creates an adapter that uses the transaction from context.
func NewEventRepository(repo *Repository) *EventRepository {
	return &EventRepository{repo: repo}
}

// LockCommand serializes processing of one owner-scoped command ID.
func (r *EventRepository) LockCommand(
	ctx context.Context,
	ownerUserID uuid.UUID,
	commandID uuid.UUID,
) error {
	err := eventsqlc.New(r.repo.GetConn(ctx)).LockEventCommand(
		ctx,
		eventsqlc.LockEventCommandParams{
			OwnerUserID: ownerUserID,
			CommandID:   commandID,
		},
	)
	if err != nil {
		return fmt.Errorf("lock event command: %w", err)
	}

	return nil
}

// Append adds one event using optimistic aggregate version checking.
func (r *EventRepository) Append(
	ctx context.Context,
	request event.AppendRequest,
) (event.Event, error) {
	actorUserID := pgtype.UUID{}
	if request.ActorUserID != nil {
		actorUserID = pgtype.UUID{
			Bytes: [uuidByteLength]byte(*request.ActorUserID),
			Valid: true,
		}
	}

	row, err := eventsqlc.New(r.repo.GetConn(ctx)).AppendEvent(
		ctx,
		eventsqlc.AppendEventParams{
			EventID:                  request.EventID,
			AggregateType:            string(request.AggregateType),
			AggregateID:              request.AggregateID,
			OwnerUserID:              request.OwnerUserID,
			ExpectedAggregateVersion: request.ExpectedAggregateVersion,
			EventType:                string(request.EventType),
			SchemaVersion:            request.SchemaVersion,
			Payload:                  request.Payload,
			Metadata:                 request.Metadata,
			ActorUserID:              actorUserID,
			CommandID:                request.CommandID,
			CommandEventIndex:        request.CommandEventIndex,
			OccurredAt: pgtype.Timestamptz{
				Time:  request.OccurredAt,
				Valid: true,
			},
		},
	)
	if err != nil {
		return event.Event{}, mapAppendError(err)
	}

	return eventFromDB(row)
}

// GetByID loads one event through its owner-scoped database function.
func (r *EventRepository) GetByID(
	ctx context.Context,
	ownerUserID uuid.UUID,
	eventID uuid.UUID,
) (event.Event, error) {
	row, err := eventsqlc.New(r.repo.GetConn(ctx)).GetEventByID(
		ctx,
		eventsqlc.GetEventByIDParams{
			OwnerUserID: ownerUserID,
			EventID:     eventID,
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return event.Event{}, event.ErrEventNotFound
		}
		return event.Event{}, fmt.Errorf("get event by id: %w", err)
	}

	return eventFromDB(row)
}

// ListByCommandID loads a command's events in their original order.
func (r *EventRepository) ListByCommandID(
	ctx context.Context,
	ownerUserID uuid.UUID,
	commandID uuid.UUID,
) ([]event.Event, error) {
	rows, err := eventsqlc.New(r.repo.GetConn(ctx)).ListEventsByCommandID(
		ctx,
		eventsqlc.ListEventsByCommandIDParams{
			OwnerUserID: ownerUserID,
			CommandID:   commandID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list events by command id: %w", err)
	}

	return eventsFromDB(rows)
}

// GetAggregateVersion returns the current version of an owner-scoped stream.
func (r *EventRepository) GetAggregateVersion(
	ctx context.Context,
	ownerUserID uuid.UUID,
	aggregateType event.AggregateType,
	aggregateID uuid.UUID,
) (int64, error) {
	version, err := eventsqlc.New(r.repo.GetConn(ctx)).GetAggregateVersion(
		ctx,
		eventsqlc.GetAggregateVersionParams{
			OwnerUserID:   ownerUserID,
			AggregateType: string(aggregateType),
			AggregateID:   aggregateID,
		},
	)
	if err != nil {
		return 0, fmt.Errorf("get aggregate version: %w", err)
	}

	return version, nil
}

// ListAggregateEvents returns an ordered page of an owner-scoped stream.
func (r *EventRepository) ListAggregateEvents(
	ctx context.Context,
	ownerUserID uuid.UUID,
	aggregateType event.AggregateType,
	aggregateID uuid.UUID,
	afterVersion int64,
	limit int32,
) ([]event.Event, error) {
	rows, err := eventsqlc.New(r.repo.GetConn(ctx)).ListAggregateEvents(
		ctx,
		eventsqlc.ListAggregateEventsParams{
			OwnerUserID:   ownerUserID,
			AggregateType: string(aggregateType),
			AggregateID:   aggregateID,
			AfterVersion:  afterVersion,
			PageSize:      limit,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list aggregate events: %w", err)
	}

	return eventsFromDB(rows)
}

// CaptureBoundary returns a global position and database time from one statement.
func (r *EventRepository) CaptureBoundary(
	ctx context.Context,
) (dailysummary.EventBoundary, error) {
	row, err := eventsqlc.New(r.repo.GetConn(ctx)).GetEventStoreBoundary(ctx)
	if err != nil {
		return dailysummary.EventBoundary{}, fmt.Errorf("capture event store boundary: %w", err)
	}

	return dailysummary.EventBoundary{
		HighWater:  row.HighWater,
		CapturedAt: time.UnixMicro(row.CapturedAtUnixMicro).UTC(),
	}, nil
}

// GetHighWater is kept as a convenience for diagnostics and integration tests.
func (r *EventRepository) GetHighWater(ctx context.Context) (int64, error) {
	boundary, err := r.CaptureBoundary(ctx)
	if err != nil {
		return 0, err
	}
	return boundary.HighWater, nil
}

// ListUserEventsByPosition reads an owner-scoped page bounded by a stable high-water mark.
func (r *EventRepository) ListUserEventsByPosition(
	ctx context.Context,
	ownerUserID uuid.UUID,
	afterPosition int64,
	toPosition int64,
	limit int32,
) ([]event.Event, error) {
	rows, err := eventsqlc.New(r.repo.GetConn(ctx)).ListUserEventsByPosition(
		ctx,
		eventsqlc.ListUserEventsByPositionParams{
			OwnerUserID:   ownerUserID,
			AfterPosition: afterPosition,
			ToPosition:    toPosition,
			PageSize:      limit,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list user events by position: %w", err)
	}

	return dailySummaryEventsFromDB(rows), nil
}

func mapAppendError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return event.ErrVersionConflict
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case "event_store_aggregate_version_unique":
			return event.ErrVersionConflict
		case "event_store_owner_command_event_unique":
			return event.ErrIdempotencyConflict
		}
	}

	return fmt.Errorf("append event: %w", err)
}

func eventsFromDB(rows []eventsqlc.EventStore) ([]event.Event, error) {
	events := make([]event.Event, 0, len(rows))
	for _, row := range rows {
		stored, err := eventFromDB(row)
		if err != nil {
			return nil, err
		}
		events = append(events, stored)
	}
	return events, nil
}

func dailySummaryEventsFromDB(rows []eventsqlc.DailySummaryEventView) []event.Event {
	events := make([]event.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, event.Event{
			GlobalPosition: row.GlobalPosition,
			ID:             row.EventID,
			OwnerUserID:    row.OwnerUserID,
			Type:           event.EventType(row.EventType),
			SchemaVersion:  row.SchemaVersion,
			Payload:        append([]byte(nil), row.Payload...),
		})
	}
	return events
}

func eventFromDB(row eventsqlc.EventStore) (event.Event, error) {
	if !row.OccurredAt.Valid || !row.RecordedAt.Valid {
		return event.Event{}, errors.New("event row has invalid timestamps")
	}

	var actorUserID *uuid.UUID
	if row.ActorUserID.Valid {
		actor := uuid.UUID(row.ActorUserID.Bytes)
		actorUserID = &actor
	}

	return event.Event{
		GlobalPosition:    row.GlobalPosition,
		ID:                row.EventID,
		AggregateType:     event.AggregateType(row.AggregateType),
		AggregateID:       row.AggregateID,
		OwnerUserID:       row.OwnerUserID,
		AggregateVersion:  row.AggregateVersion,
		Type:              event.EventType(row.EventType),
		SchemaVersion:     row.SchemaVersion,
		Payload:           append([]byte(nil), row.Payload...),
		Metadata:          append([]byte(nil), row.Metadata...),
		ActorUserID:       actorUserID,
		CommandID:         row.CommandID,
		CommandEventIndex: row.CommandEventIndex,
		OccurredAt:        row.OccurredAt.Time,
		RecordedAt:        row.RecordedAt.Time,
	}, nil
}
