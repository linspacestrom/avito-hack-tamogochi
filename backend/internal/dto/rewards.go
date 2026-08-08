package dto

import "time"

type IssueRewardRequest struct {
	RewardCode string `json:"rewardCode"`
	// UserLevel is supplied by the caller for now — rewards has no way to look up the
	// caller's level itself yet, since the pet module (which owns level) isn't callable
	// from here. This is a known temporary gap: a client can misreport its own level to
	// pass the RequiredLevel check. Replace with a server-side lookup once pet exposes one.
	UserLevel     int     `json:"userLevel"`
	SourceEventID *string `json:"sourceEventId,omitempty"`
}

type UserRewardResponse struct {
	ID                 string     `json:"id"`
	RewardDefinitionID string     `json:"rewardDefinitionId"`
	Status             string     `json:"status"`
	IssuedAt           time.Time  `json:"issuedAt"`
	ExpiresAt          *time.Time `json:"expiresAt,omitempty"`
	RedeemedAt         *time.Time `json:"redeemedAt,omitempty"`
}

type UserRewardsListResponse struct {
	Rewards []UserRewardResponse `json:"rewards"`
}
