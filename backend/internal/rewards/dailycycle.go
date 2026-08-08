package rewards

import (
	"context"
	"errors"
	"time"
)

const (
	DailyCycleLength = 14

	// claimCooldown is how soon after a claim the next one becomes available — claiming
	// again before this has passed is rejected as "too soon".
	claimCooldown = 24 * time.Hour
	// streakGraceWindow is how long a user can go past the cooldown before the streak
	// resets to day 1. Between claimCooldown and streakGraceWindow, the cycle just
	// advances by one day as normal (they claimed "on time", within their day window).
	streakGraceWindow = 48 * time.Hour
)

// ErrTooSoonToClaim is returned when the user already claimed within the last claimCooldown.
var ErrTooSoonToClaim = errors.New("daily reward already claimed, come back later")

// UserDailyRewardProgress is where a user currently stands in the 14-day cycle.
type UserDailyRewardProgress struct {
	UserID         string
	CurrentDay     int
	CycleStartedAt time.Time
	LastClaimedAt  *time.Time
}

type DailyCycleStatus struct {
	CurrentDay int
	CanClaim   bool
	Reward     RewardDefinition
}

// DailyCycleRepository is what DailyCycleService needs from storage. Implemented by
// internal/repository/postgres.RewardsRepository.
type DailyCycleRepository interface {
	// GetProgress returns nil, nil if the user has no progress row yet (hasn't claimed once).
	GetProgress(ctx context.Context, userID string) (*UserDailyRewardProgress, error)
	GetRewardForDay(ctx context.Context, dayNumber int) (*RewardDefinition, error)
	// UpsertProgress creates the row on first claim, updates it on every claim after.
	UpsertProgress(ctx context.Context, progress UserDailyRewardProgress) error
}

type DailyCycleService struct {
	repo DailyCycleRepository
}

func NewDailyCycleService(repo DailyCycleRepository) *DailyCycleService {
	return &DailyCycleService{repo: repo}
}

// GetStatus returns the day the user is currently on and whether they can claim right now.
// It applies the same lazy time-based logic as ClaimToday without writing anything — a user
// who has been away long enough will see themselves back at day 1 before they've even
// claimed, consistent with how pet stat degradation is computed elsewhere in this project.
func (s *DailyCycleService) GetStatus(ctx context.Context, userID string) (*DailyCycleStatus, error) {
	progress, err := s.repo.GetProgress(ctx, userID)
	if err != nil {
		return nil, err
	}

	day, canClaim := effectiveDay(progress)

	reward, err := s.repo.GetRewardForDay(ctx, day)
	if err != nil {
		return nil, err
	}

	return &DailyCycleStatus{
		CurrentDay: day,
		CanClaim:   canClaim,
		Reward:     *reward,
	}, nil
}

// ClaimToday claims the reward for the user's current effective day and advances the cycle.
//
// NOTE: unlike IssueReward, this does not create a UserReward row. reward_definitions here
// are deliberately reused across multiple days and across repeats of the 14-day cycle, which
// is incompatible with the (userId, rewardDefinitionId) uniqueness that protects the
// achievement-style rewards in user_rewards — reusing that table would silently block every
// repeat claim after the first. user_daily_reward_progress is the record of what's been
// claimed and when. If a persistent history of daily claims is needed later (e.g. for the
// "ежедневная сводка" feature to show "reward earned yesterday"), that needs its own
// append-only table — flagging this rather than solving it here, out of scope for this task.
func (s *DailyCycleService) ClaimToday(ctx context.Context, userID string) (*RewardDefinition, error) {
	progress, err := s.repo.GetProgress(ctx, userID)
	if err != nil {
		return nil, err
	}

	day, canClaim := effectiveDay(progress)
	if !canClaim {
		return nil, ErrTooSoonToClaim
	}

	reward, err := s.repo.GetRewardForDay(ctx, day)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	updated := UserDailyRewardProgress{
		UserID:        userID,
		CurrentDay:    nextDay(day),
		LastClaimedAt: &now,
	}
	// CycleStartedAt marks the start of the current unbroken lap: reset it whenever we land
	// back on day 1, whether that's the very first claim, a miss-triggered restart, or a
	// natural 14→1 wraparound. Otherwise carry the existing lap's start forward.
	if updated.CurrentDay == 1 || progress == nil {
		updated.CycleStartedAt = now
	} else {
		updated.CycleStartedAt = progress.CycleStartedAt
	}

	if err := s.repo.UpsertProgress(ctx, updated); err != nil {
		return nil, err
	}

	return reward, nil
}

// effectiveDay computes the day the user is really on right now, and whether a claim is
// available, purely from elapsed time since the last claim — no write happens here.
//
// ⚠️ Skip-day policy: this resets the streak to day 1 once streakGraceWindow has passed
// since the last claim (assumed default: miss a day, start over). ARCHITECTURE.md marks
// reset-vs-freeze as an explicit open question, not yet decided by the team. Reset was
// chosen here because it more directly serves the case's stated goal of habit formation —
// but this needs real confirmation before the daily cycle ships, not just my assumption.
func effectiveDay(progress *UserDailyRewardProgress) (day int, canClaim bool) {
	if progress == nil || progress.LastClaimedAt == nil {
		return 1, true
	}

	elapsed := time.Since(*progress.LastClaimedAt)

	switch {
	case elapsed < claimCooldown:
		return progress.CurrentDay, false
	case elapsed < streakGraceWindow:
		return progress.CurrentDay, true
	default:
		return 1, true
	}
}

func nextDay(day int) int {
	if day >= DailyCycleLength {
		return 1
	}
	return day + 1
}
