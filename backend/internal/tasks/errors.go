package tasks

import "errors"

var (
	// ErrTaskNotFound is returned when no active task definition matches the given code.
	ErrTaskNotFound = errors.New("task definition not found")
	// ErrTaskNotCompleted is returned when trying to claim a task that isn't completed yet.
	ErrTaskNotCompleted = errors.New("task is not completed yet")
	// ErrTaskAlreadyClaimed is returned when trying to claim a task whose reward was already claimed.
	ErrTaskAlreadyClaimed = errors.New("task reward already claimed")
)
