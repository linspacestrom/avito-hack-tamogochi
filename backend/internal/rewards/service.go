package rewards

import (
	"context"
	"time"
)

// Repository is what Service needs from storage. Implemented by
// internal/repository/postgres.RewardsRepository.
type Repository interface {
	GetRewardDefinitionByCode(ctx context.Context, code string) (*RewardDefinition, error)
	CreateUserReward(ctx context.Context, reward UserReward) (*UserReward, error)
	ListUserRewards(ctx context.Context, userID string) ([]UserReward, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// IssueReward grants the reward identified by rewardCode to userID, if the user meets the
// level requirement and hasn't already received it. userLevel is passed in by the caller —
// rewards doesn't query the pet module's data directly, that stays the pet module's own.
func (s *Service) IssueReward(
	ctx context.Context,
	userID string,
	rewardCode string,
	sourceEventID *string,
	userLevel int,
) (*UserReward, error) {
	def, err := s.repo.GetRewardDefinitionByCode(ctx, rewardCode)
	if err != nil {
		return nil, err
	}

	if !def.IsActive {
		return nil, ErrRewardInactive
	}

	if userLevel < def.RequiredLevel {
		return nil, ErrLevelTooLow
	}

	var expiresAt *time.Time
	if def.ValidityDays != nil {
		t := time.Now().AddDate(0, 0, *def.ValidityDays)
		expiresAt = &t
	}

	reward, err := s.repo.CreateUserReward(ctx, UserReward{
		UserID:             userID,
		RewardDefinitionID: def.ID,
		SourceEventID:      sourceEventID,
		Status:             "issued",
		ExpiresAt:          expiresAt,
	})
	if err != nil {
		return nil, err
	}

	return reward, nil
}

// ListUserRewards returns every reward issued to userID, most recent first.
func (s *Service) ListUserRewards(ctx context.Context, userID string) ([]UserReward, error) {
	return s.repo.ListUserRewards(ctx, userID)
}
