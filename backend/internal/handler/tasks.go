package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/dto"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/mapper"
	appmiddleware "github.com/NBx03/avito-hack-tamagotchi/backend/internal/middleware"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/tasks"
	"github.com/go-chi/chi/v5"
)

type TasksService interface {
	ListTasks(ctx context.Context, userID string) ([]tasks.TaskWithProgress, error)
	ClaimTask(ctx context.Context, userID string, taskCode string) (*rewards.RewardDefinition, error)
}

// ListMyTasks godoc
// @Summary List my tasks
// @Description Returns every active task with the authenticated user's progress on it.
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.TasksListResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /tasks/me [get]
func (h *Handler) ListMyTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	list, err := h.tasks.ListTasks(r.Context(), userID.String())
	if err != nil {
		h.handleTasksError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapper.TasksToListResponse(list))
}

// ClaimTask godoc
// @Summary Claim a completed task's reward
// @Description Grants the reward for the given task code and marks it claimed. Fails if the
// @Description task doesn't exist, hasn't been completed yet, or was already claimed.
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Param code path string true "Task code"
// @Success 200 {object} dto.ClaimTaskResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /tasks/{code}/claim [post]
func (h *Handler) ClaimTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	code := chi.URLParam(r, "code")

	reward, err := h.tasks.ClaimTask(r.Context(), userID.String(), code)
	if err != nil {
		h.handleTasksError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ClaimTaskResponse{
		ClaimedReward: mapper.RewardDefinitionToDTO(*reward),
	})
}

func (h *Handler) handleTasksError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, tasks.ErrTaskNotFound):
		writeError(w, http.StatusNotFound, "task definition not found")
	case errors.Is(err, tasks.ErrTaskNotCompleted):
		writeError(w, http.StatusConflict, "task is not completed yet")
	case errors.Is(err, tasks.ErrTaskAlreadyClaimed):
		writeError(w, http.StatusConflict, "task reward already claimed")
	default:
		h.log.Error("tasks request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
