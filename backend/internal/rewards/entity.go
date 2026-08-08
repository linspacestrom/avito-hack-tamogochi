// Package rewards holds the user reward system, including protection against duplicate
// reward issuance.
package rewards

import (
	"encoding/json"
	"time"
)

// RewardDefinition is a catalog entry — one kind of reward that can be issued to users.
type RewardDefinition struct {
	ID            string
	Code          string
	Title         string
	Description   string
	RequiredLevel int
	ValidityDays  *int
	IsActive      bool
	RewardType    string
	Value         json.RawMessage
}

// UserReward is a single reward issued to a specific user.
type UserReward struct {
	ID                 string
	UserID             string
	RewardDefinitionID string
	SourceEventID      *string
	Status             string
	IssuedAt           time.Time
	ExpiresAt          *time.Time
	RedeemedAt         *time.Time
}
