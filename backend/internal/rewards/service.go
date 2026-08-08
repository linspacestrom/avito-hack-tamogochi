package rewards

import (
	"context"
	"time"
)

// UserReward.Status values. Matches the CHECK constraint on user_rewards.status.
const (
	StatusIssued   = "issued"
	StatusRedeemed = "redeemed"
	StatusExpired  = "expired"
	StatusRevoked  = "revoked"
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
		Status:             StatusIssued,
		ExpiresAt:          expiresAt,
	})
	if err != nil {
		return nil, err
	}

	return reward, nil
}

// ListUserRewards returns every reward issued to userID, most recent first. Rewards past
// their expiry are reported as StatusExpired even though the stored row still says
// StatusIssued — expiration is computed lazily on read here, the same "don't write until
// something forces a write" approach used for pet stat degradation and the daily reward
// cycle elsewhere in this project. Nothing is written back to the database by this call.
func (s *Service) ListUserRewards(ctx context.Context, userID string) ([]UserReward, error) {
	userRewards, err := s.repo.ListUserRewards(ctx, userID)
	if err != nil {
		return nil, err
	}

	for i := range userRewards {
		userRewards[i].Status = effectiveStatus(userRewards[i])
	}

	return userRewards, nil
}

// effectiveStatus reports StatusExpired for a reward that is still StatusIssued but past its
// ExpiresAt. Rewards already redeemed or revoked are left alone — expiry only applies to
// rewards that were never used.
func effectiveStatus(reward UserReward) string {
	if reward.Status == StatusIssued && reward.ExpiresAt != nil && reward.ExpiresAt.Before(time.Now()) {
		return StatusExpired
	}
	return reward.Status
}
