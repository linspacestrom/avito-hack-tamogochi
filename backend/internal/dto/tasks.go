package dto

type TaskResponse struct {
	Code         string              `json:"code"`
	Type         string              `json:"type"`
	Title        string              `json:"title"`
	Description  string              `json:"description"`
	TargetValue  int                 `json:"targetValue"`
	CurrentValue int                 `json:"currentValue"`
	Status       string              `json:"status"`
	Reward       RewardDefinitionDTO `json:"reward"`
}

type TasksListResponse struct {
	Tasks []TaskResponse `json:"tasks"`
}

type ClaimTaskResponse struct {
	ClaimedReward RewardDefinitionDTO `json:"claimedReward"`
}
