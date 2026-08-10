package dailysummary

import "errors"

var (
	ErrValidation             = errors.New("invalid daily summary request")
	ErrCheckpointNotFound     = errors.New("daily summary checkpoint not found")
	ErrInvalidEvent           = errors.New("invalid daily summary event")
	ErrUnsupportedEventSchema = errors.New("unsupported daily summary event schema")
	ErrPositionRegression     = errors.New("event store position regressed")
	ErrBacklogLimitExceeded   = errors.New("daily summary event backlog limit exceeded")
)
