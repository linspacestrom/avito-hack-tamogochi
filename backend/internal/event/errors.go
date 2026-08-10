package event

import "errors"

var (
	// ErrValidation indicates that a command violates the Event Store contract
	// and must be corrected before it is retried.
	ErrValidation = errors.New("invalid event command")
	// ErrVersionConflict indicates that an aggregate changed after the caller read it.
	// The caller may reload the aggregate and retry with its current version.
	ErrVersionConflict = errors.New("aggregate version conflict")
	// ErrEventNotFound indicates that an owner-scoped event lookup returned no row.
	ErrEventNotFound = errors.New("event not found")
	// ErrIdempotencyConflict indicates reuse of a command ID with different content.
	// Retrying the unchanged command cannot resolve this conflict.
	ErrIdempotencyConflict = errors.New("idempotency conflict")
)
