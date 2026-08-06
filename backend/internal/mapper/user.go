package mapper

import (
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/dto"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/entity"
)

func UserToResponse(user entity.User) dto.UserResponse {
	return dto.UserResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Status:      user.Status,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}
