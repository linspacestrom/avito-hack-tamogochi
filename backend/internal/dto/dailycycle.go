package dto

type DailyCycleStatusResponse struct {
	CurrentDay int                 `json:"currentDay"`
	CanClaim   bool                `json:"canClaim"`
	Reward     RewardDefinitionDTO `json:"reward"`
}

type RewardDefinitionDTO struct {
	Code        string `json:"code"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type DailyClaimResponse struct {
	ClaimedReward RewardDefinitionDTO `json:"claimedReward"`
}
