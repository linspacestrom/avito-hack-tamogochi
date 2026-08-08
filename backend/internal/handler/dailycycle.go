package handler

import (
	"context"
	"net/http"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/dto"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/mapper"
	appmiddleware "github.com/NBx03/avito-hack-tamagotchi/backend/internal/middleware"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
)

type DailyCycleService interface {
	GetStatus(ctx context.Context, userID string) (*rewards.DailyCycleStatus, error)
	ClaimToday(ctx context.Context, userID string) (*rewards.RewardDefinition, error)
}

// DailyStatus godoc
// @Summary Get daily reward status
// @Description Returns which day of the 14-day daily reward calendar the user is on and
// @Description whether they can claim it right now.
// @Tags rewards
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.DailyCycleStatusResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /rewards/daily-status [get]
func (h *Handler) DailyStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	status, err := h.dailyCycle.GetStatus(r.Context(), userID.String())
	if err != nil {
		h.handleRewardsError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapper.DailyCycleStatusToResponse(*status))
}

// DailyClaim godoc
// @Summary Claim today's daily reward
// @Description Claims the reward for the user's current day in the 14-day calendar and
// @Description advances the cycle. Fails if already claimed within the last 24 hours.
// @Tags rewards
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.DailyClaimResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /rewards/daily-claim [post]
func (h *Handler) DailyClaim(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	reward, err := h.dailyCycle.ClaimToday(r.Context(), userID.String())
	if err != nil {
		h.handleRewardsError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.DailyClaimResponse{
		ClaimedReward: mapper.RewardDefinitionToDTO(*reward),
	})
}
