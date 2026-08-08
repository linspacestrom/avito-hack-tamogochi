package mapper

import (
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/dto"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
)

func RewardDefinitionToDTO(def rewards.RewardDefinition) dto.RewardDefinitionDTO {
	return dto.RewardDefinitionDTO{
		Code:        def.Code,
		Title:       def.Title,
		Description: def.Description,
	}
}

func DailyCycleStatusToResponse(status rewards.DailyCycleStatus) dto.DailyCycleStatusResponse {
	return dto.DailyCycleStatusResponse{
		CurrentDay: status.CurrentDay,
		CanClaim:   status.CanClaim,
		Reward:     RewardDefinitionToDTO(status.Reward),
	}
}
