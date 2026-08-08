package postgres

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Small conversion helpers between sqlc/pgtype wire types and the plain string/time.Time
// types used by the rewards package's own domain model (internal/rewards). rewards
// deliberately doesn't depend on google/uuid or pgtype itself — that boundary is kept here.

func uuidToString(id uuid.UUID) string {
	return id.String()
}

func stringToUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

func nullableUUIDToStringPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuid.UUID(u.Bytes).String()
	return &s
}

func stringPtrToNullableUUID(s *string) (pgtype.UUID, error) {
	if s == nil {
		return pgtype.UUID{}, nil
	}
	parsed, err := uuid.Parse(*s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func timestamptzToTimePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	result := t.Time
	return &result
}

func timePtrToTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func timeToTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func int4ToIntPtr(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int32)
	return &n
}
