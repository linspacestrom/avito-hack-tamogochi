package leaderboard

import "errors"

var (
	ErrValidation         = errors.New("invalid leaderboard request")
	ErrInvalidStoredEvent = errors.New("invalid stored leaderboard event")
)
