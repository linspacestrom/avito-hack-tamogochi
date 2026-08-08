package mapper

import (
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/dto"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
)

func UserRewardToResponse(reward rewards.UserReward) dto.UserRewardResponse {
	return dto.UserRewardResponse{
		ID:                 reward.ID,
		RewardDefinitionID: reward.RewardDefinitionID,
		Status:             reward.Status,
		IssuedAt:           reward.IssuedAt,
		ExpiresAt:          reward.ExpiresAt,
		RedeemedAt:         reward.RedeemedAt,
	}
}

func UserRewardsToListResponse(rewardList []rewards.UserReward) dto.UserRewardsListResponse {
	response := dto.UserRewardsListResponse{
		Rewards: make([]dto.UserRewardResponse, 0, len(rewardList)),
	}
	for _, reward := range rewardList {
		response.Rewards = append(response.Rewards, UserRewardToResponse(reward))
	}
	return response
}
