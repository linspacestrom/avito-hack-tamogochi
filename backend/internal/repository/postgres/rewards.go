package postgres

import (
	"context"
	"encoding/json"
	"errors"

	db "github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository/postgres/sqlc"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const uniqueViolationCode = "23505"

// RewardsRepository is a sub-repository of Repository, matching the pattern used by
// UserRepository/SessionRepository/PetRepository — it goes through r.repo.GetConn(ctx) so it
// participates correctly in the shared trm transaction manager, and uses sqlc-generated
// queries instead of raw pgx (see the note in sqlc/models.go about how those were produced).
type RewardsRepository struct {
	repo *Repository
}

func NewRewardsRepository(repo *Repository) *RewardsRepository {
	return &RewardsRepository{repo: repo}
}

func (r *RewardsRepository) GetRewardDefinitionByCode(
	ctx context.Context,
	code string,
) (*rewards.RewardDefinition, error) {
	row, err := db.New(r.repo.GetConn(ctx)).GetRewardDefinitionByCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, rewards.ErrRewardNotFound
	}
	if err != nil {
		return nil, err
	}

	result := rewardDefinitionFromDB(row)
	return &result, nil
}

func (r *RewardsRepository) CreateUserReward(
	ctx context.Context,
	reward rewards.UserReward,
) (*rewards.UserReward, error) {
	userID, err := stringToUUID(reward.UserID)
	if err != nil {
		return nil, err
	}
	rewardDefinitionID, err := stringToUUID(reward.RewardDefinitionID)
	if err != nil {
		return nil, err
	}
	sourceEventID, err := stringPtrToNullableUUID(reward.SourceEventID)
	if err != nil {
		return nil, err
	}

	row, err := db.New(r.repo.GetConn(ctx)).CreateUserReward(ctx, db.CreateUserRewardParams{
		UserID:             userID,
		RewardDefinitionID: rewardDefinitionID,
		SourceEventID:      sourceEventID,
		Status:             reward.Status,
		ExpiresAt:          timePtrToTimestamptz(reward.ExpiresAt),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, rewards.ErrAlreadyIssued
		}
		return nil, err
	}

	result := userRewardFromDB(row)
	return &result, nil
}

func (r *RewardsRepository) ListUserRewards(
	ctx context.Context,
	userID string,
) ([]rewards.UserReward, error) {
	id, err := stringToUUID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := db.New(r.repo.GetConn(ctx)).ListUserRewards(ctx, id)
	if err != nil {
		return nil, err
	}

	result := make([]rewards.UserReward, 0, len(rows))
	for _, row := range rows {
		result = append(result, userRewardFromDB(row))
	}
	return result, nil
}

func rewardDefinitionFromDB(def db.RewardDefinition) rewards.RewardDefinition {
	return rewards.RewardDefinition{
		ID:            uuidToString(def.ID),
		Code:          def.Code,
		Title:         def.Title,
		Description:   def.Description,
		RequiredLevel: int(def.RequiredLevel),
		ValidityDays:  int4ToIntPtr(def.ValidityDays),
		IsActive:      def.IsActive,
		RewardType:    def.RewardType,
		Value:         json.RawMessage(def.Value),
	}
}

func userRewardFromDB(reward db.UserReward) rewards.UserReward {
	return rewards.UserReward{
		ID:                 uuidToString(reward.ID),
		UserID:             uuidToString(reward.UserID),
		RewardDefinitionID: uuidToString(reward.RewardDefinitionID),
		SourceEventID:      nullableUUIDToStringPtr(reward.SourceEventID),
		Status:             reward.Status,
		IssuedAt:           reward.IssuedAt.Time,
		ExpiresAt:          timestamptzToTimePtr(reward.ExpiresAt),
		RedeemedAt:         timestamptzToTimePtr(reward.RedeemedAt),
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == uniqueViolationCode
	}
	return false
}
