package mapper

import (
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/dto"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/tasks"
)

func TaskWithProgressToResponse(t tasks.TaskWithProgress) dto.TaskResponse {
	return dto.TaskResponse{
		Code:         t.Task.Code,
		Type:         t.Task.Type,
		Title:        t.Task.Title,
		Description:  t.Task.Description,
		TargetValue:  t.Task.TargetValue,
		CurrentValue: t.CurrentValue,
		Status:       t.Status,
		Reward:       RewardDefinitionToDTO(t.Reward),
	}
}

func TasksToListResponse(list []tasks.TaskWithProgress) dto.TasksListResponse {
	response := dto.TasksListResponse{Tasks: make([]dto.TaskResponse, 0, len(list))}
	for _, t := range list {
		response.Tasks = append(response.Tasks, TaskWithProgressToResponse(t))
	}
	return response
}
