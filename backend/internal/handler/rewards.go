package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/dto"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/mapper"
	appmiddleware "github.com/NBx03/avito-hack-tamagotchi/backend/internal/middleware"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
)

type RewardsService interface {
	IssueReward(
		ctx context.Context,
		userID string,
		rewardCode string,
		sourceEventID *string,
		userLevel int,
	) (*rewards.UserReward, error)
	ListUserRewards(ctx context.Context, userID string) ([]rewards.UserReward, error)
}

// IssueReward godoc
// @Summary Issue a reward
// @Description Issues the reward identified by rewardCode to the authenticated user, if the
// @Description level requirement is met and it hasn't already been issued to them.
// @Tags rewards
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.IssueRewardRequest true "Reward to issue"
// @Success 201 {object} dto.UserRewardResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /rewards/issue [post]
func (h *Handler) IssueReward(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var request dto.IssueRewardRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	reward, err := h.rewards.IssueReward(
		r.Context(),
		userID.String(),
		request.RewardCode,
		request.SourceEventID,
		request.UserLevel,
	)
	if err != nil {
		h.handleRewardsError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, mapper.UserRewardToResponse(*reward))
}

// ListMyRewards godoc
// @Summary List my rewards
// @Description Returns every reward issued to the authenticated user, most recent first.
// @Tags rewards
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.UserRewardsListResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /rewards/me [get]
func (h *Handler) ListMyRewards(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	userRewards, err := h.rewards.ListUserRewards(r.Context(), userID.String())
	if err != nil {
		h.handleRewardsError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapper.UserRewardsToListResponse(userRewards))
}

func (h *Handler) handleRewardsError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, rewards.ErrRewardNotFound):
		writeError(w, http.StatusNotFound, "reward definition not found")
	case errors.Is(err, rewards.ErrRewardInactive):
		writeError(w, http.StatusBadRequest, "reward definition is not active")
	case errors.Is(err, rewards.ErrLevelTooLow):
		writeError(w, http.StatusBadRequest, "user level is below the reward's required level")
	case errors.Is(err, rewards.ErrAlreadyIssued):
		writeError(w, http.StatusConflict, "reward already issued to this user")
	default:
		h.log.Error("rewards request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
