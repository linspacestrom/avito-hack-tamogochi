package mapper

import (
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/dto"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/entity"
)

func AuthResultToResponse(result *entity.AuthResult) dto.AuthResponse {
	return dto.AuthResponse{
		User:         UserToResponse(result.User),
		AccessToken:  result.Tokens.AccessToken,
		RefreshToken: result.Tokens.RefreshToken,
	}
}

func TokenPairToResponse(tokens *entity.TokenPair) dto.TokenResponse {
	return dto.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}
}
