package rewards_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
)

type fakeRepository struct {
	definitions map[string]*rewards.RewardDefinition
	created     []rewards.UserReward
	listResult  []rewards.UserReward
}

func (f *fakeRepository) GetRewardDefinitionByCode(
	_ context.Context,
	code string,
) (*rewards.RewardDefinition, error) {
	def, ok := f.definitions[code]
	if !ok {
		return nil, rewards.ErrRewardNotFound
	}
	return def, nil
}

func (f *fakeRepository) CreateUserReward(
	_ context.Context,
	reward rewards.UserReward,
) (*rewards.UserReward, error) {
	f.created = append(f.created, reward)
	return &reward, nil
}

func (f *fakeRepository) ListUserRewards(
	_ context.Context,
	_ string,
) ([]rewards.UserReward, error) {
	return f.listResult, nil
}

func intPtr(v int) *int { return &v }

func TestIssueReward_RewardNotFound(t *testing.T) {
	repo := &fakeRepository{definitions: map[string]*rewards.RewardDefinition{}}
	svc := rewards.NewService(repo)

	_, err := svc.IssueReward(context.Background(), "user-1", "unknown-code", nil, 1)
	if err != rewards.ErrRewardNotFound {
		t.Fatalf("expected ErrRewardNotFound, got %v", err)
	}
}

func TestIssueReward_Inactive(t *testing.T) {
	repo := &fakeRepository{definitions: map[string]*rewards.RewardDefinition{
		"welcome-coins": {
			ID:            "def-1",
			Code:          "welcome-coins",
			IsActive:      false,
			RequiredLevel: 1,
			Value:         json.RawMessage(`{}`),
		},
	}}
	svc := rewards.NewService(repo)

	_, err := svc.IssueReward(context.Background(), "user-1", "welcome-coins", nil, 1)
	if err != rewards.ErrRewardInactive {
		t.Fatalf("expected ErrRewardInactive, got %v", err)
	}
}

func TestIssueReward_LevelTooLow(t *testing.T) {
	repo := &fakeRepository{definitions: map[string]*rewards.RewardDefinition{
		"promo-ad-boost": {
			ID:            "def-2",
			Code:          "promo-ad-boost",
			IsActive:      true,
			RequiredLevel: 3,
			Value:         json.RawMessage(`{}`),
		},
	}}
	svc := rewards.NewService(repo)

	_, err := svc.IssueReward(context.Background(), "user-1", "promo-ad-boost", nil, 1)
	if err != rewards.ErrLevelTooLow {
		t.Fatalf("expected ErrLevelTooLow, got %v", err)
	}
}

func TestIssueReward_Success(t *testing.T) {
	repo := &fakeRepository{definitions: map[string]*rewards.RewardDefinition{
		"welcome-coins": {
			ID:            "def-3",
			Code:          "welcome-coins",
			IsActive:      true,
			RequiredLevel: 1,
			ValidityDays:  intPtr(7),
			Value:         json.RawMessage(`{"amount": 50}`),
		},
	}}
	svc := rewards.NewService(repo)

	reward, err := svc.IssueReward(context.Background(), "user-1", "welcome-coins", nil, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reward.UserID != "user-1" || reward.RewardDefinitionID != "def-3" {
		t.Fatalf("unexpected reward: %+v", reward)
	}
	if reward.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be set for a reward with ValidityDays")
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected exactly one CreateUserReward call, got %d", len(repo.created))
	}
}

func TestListUserRewards_PastExpiryReportsAsExpired(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	repo := &fakeRepository{listResult: []rewards.UserReward{
		{ID: "r1", Status: rewards.StatusIssued, ExpiresAt: &past},
	}}
	svc := rewards.NewService(repo)

	got, err := svc.ListUserRewards(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].Status != rewards.StatusExpired {
		t.Fatalf("expected status %q, got %q", rewards.StatusExpired, got[0].Status)
	}
}

func TestListUserRewards_NotYetExpiredStaysIssued(t *testing.T) {
	future := time.Now().Add(time.Hour)
	repo := &fakeRepository{listResult: []rewards.UserReward{
		{ID: "r1", Status: rewards.StatusIssued, ExpiresAt: &future},
	}}
	svc := rewards.NewService(repo)

	got, err := svc.ListUserRewards(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].Status != rewards.StatusIssued {
		t.Fatalf("expected status %q, got %q", rewards.StatusIssued, got[0].Status)
	}
}

func TestListUserRewards_NoExpiryNeverExpires(t *testing.T) {
	repo := &fakeRepository{listResult: []rewards.UserReward{
		{ID: "r1", Status: rewards.StatusIssued, ExpiresAt: nil},
	}}
	svc := rewards.NewService(repo)

	got, err := svc.ListUserRewards(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].Status != rewards.StatusIssued {
		t.Fatalf("expected status %q, got %q", rewards.StatusIssued, got[0].Status)
	}
}

func TestListUserRewards_RedeemedPastExpiryStaysRedeemed(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	repo := &fakeRepository{listResult: []rewards.UserReward{
		{ID: "r1", Status: rewards.StatusRedeemed, ExpiresAt: &past},
	}}
	svc := rewards.NewService(repo)

	got, err := svc.ListUserRewards(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].Status != rewards.StatusRedeemed {
		t.Fatalf("expiry must not override an already-redeemed reward, got %q", got[0].Status)
	}
}
