package rewards

import "errors"

var (
	// ErrRewardNotFound is returned when no reward definition matches the given code.
	ErrRewardNotFound = errors.New("reward definition not found")
	// ErrRewardInactive is returned when the reward definition exists but is disabled.
	ErrRewardInactive = errors.New("reward definition is not active")
	// ErrLevelTooLow is returned when the user's level is below the reward's required level.
	ErrLevelTooLow = errors.New("user level is below the reward's required level")
	// ErrAlreadyIssued is returned when this reward was already issued to this user — the
	// (userId, rewardDefinitionId) unique constraint in user_rewards.
	ErrAlreadyIssued = errors.New("reward already issued to this user")
)
