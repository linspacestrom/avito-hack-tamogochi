package rewards_test

import (
	"context"
	"testing"
	"time"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
)

type fakeDailyCycleRepository struct {
	progress map[string]*rewards.UserDailyRewardProgress
	catalog  map[int]*rewards.RewardDefinition
}

func newFakeDailyCycleRepository() *fakeDailyCycleRepository {
	catalog := make(map[int]*rewards.RewardDefinition, rewards.DailyCycleLength)
	for day := 1; day <= rewards.DailyCycleLength; day++ {
		catalog[day] = &rewards.RewardDefinition{ID: "def", Code: "day-reward"}
	}
	return &fakeDailyCycleRepository{
		progress: map[string]*rewards.UserDailyRewardProgress{},
		catalog:  catalog,
	}
}

func (f *fakeDailyCycleRepository) GetProgress(
	_ context.Context,
	userID string,
) (*rewards.UserDailyRewardProgress, error) {
	return f.progress[userID], nil
}

func (f *fakeDailyCycleRepository) GetRewardForDay(
	_ context.Context,
	dayNumber int,
) (*rewards.RewardDefinition, error) {
	def, ok := f.catalog[dayNumber]
	if !ok {
		return nil, rewards.ErrRewardNotFound
	}
	return def, nil
}

func (f *fakeDailyCycleRepository) UpsertProgress(
	_ context.Context,
	progress rewards.UserDailyRewardProgress,
) error {
	p := progress
	f.progress[progress.UserID] = &p
	return nil
}

func (f *fakeDailyCycleRepository) LogClaim(
	_ context.Context,
	_ string,
	_ int,
	_ string,
	_ time.Time,
) error {
	return nil
}

// fakeTransactor just runs fn directly — an in-memory fake repository has no real
// transactions to coordinate, so there's nothing for it to do beyond satisfying the interface.
type fakeTransactor struct{}

func (fakeTransactor) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestDailyCycle_FirstClaimEverStartsAtDayOne(t *testing.T) {
	repo := newFakeDailyCycleRepository()
	svc := rewards.NewDailyCycleService(repo, fakeTransactor{})

	status, err := svc.GetStatus(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.CurrentDay != 1 || !status.CanClaim {
		t.Fatalf("expected day 1, canClaim=true, got %+v", status)
	}
}

func TestDailyCycle_ClaimAdvancesToNextDay(t *testing.T) {
	repo := newFakeDailyCycleRepository()
	svc := rewards.NewDailyCycleService(repo, fakeTransactor{})
	ctx := context.Background()

	if _, err := svc.ClaimToday(ctx, "user-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	status, err := svc.GetStatus(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.CurrentDay != 2 {
		t.Fatalf("expected day 2 after first claim, got %d", status.CurrentDay)
	}
	if status.CanClaim {
		t.Fatal("expected canClaim=false immediately after claiming (cooldown)")
	}
}

func TestDailyCycle_ClaimingTwiceTooSoonFails(t *testing.T) {
	repo := newFakeDailyCycleRepository()
	svc := rewards.NewDailyCycleService(repo, fakeTransactor{})
	ctx := context.Background()

	if _, err := svc.ClaimToday(ctx, "user-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := svc.ClaimToday(ctx, "user-1")
	if err != rewards.ErrTooSoonToClaim {
		t.Fatalf("expected ErrTooSoonToClaim, got %v", err)
	}
}

func TestDailyCycle_ClaimingOnScheduleAdvancesNormally(t *testing.T) {
	repo := newFakeDailyCycleRepository()
	claimedAt := time.Now().Add(-30 * time.Hour) // within the 24-48h "on time" window
	repo.progress["user-1"] = &rewards.UserDailyRewardProgress{
		UserID:         "user-1",
		CurrentDay:     3,
		CycleStartedAt: time.Now().Add(-72 * time.Hour),
		LastClaimedAt:  &claimedAt,
	}
	svc := rewards.NewDailyCycleService(repo, fakeTransactor{})

	status, err := svc.GetStatus(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.CurrentDay != 3 || !status.CanClaim {
		t.Fatalf("expected day 3, canClaim=true, got %+v", status)
	}
}

func TestDailyCycle_MissingADayResetsToDayOne(t *testing.T) {
	repo := newFakeDailyCycleRepository()
	claimedAt := time.Now().Add(-72 * time.Hour) // past the 48h grace window
	repo.progress["user-1"] = &rewards.UserDailyRewardProgress{
		UserID:         "user-1",
		CurrentDay:     5,
		CycleStartedAt: time.Now().Add(-96 * time.Hour),
		LastClaimedAt:  &claimedAt,
	}
	svc := rewards.NewDailyCycleService(repo, fakeTransactor{})
	ctx := context.Background()

	status, err := svc.GetStatus(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.CurrentDay != 1 || !status.CanClaim {
		t.Fatalf("expected reset to day 1, canClaim=true, got %+v", status)
	}

	if _, err := svc.ClaimToday(ctx, "user-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.progress["user-1"].CurrentDay != 2 {
		t.Fatalf("expected day 2 after claiming the reset day 1, got %d", repo.progress["user-1"].CurrentDay)
	}
}

func TestDailyCycle_Day14WrapsAroundToDayOne(t *testing.T) {
	repo := newFakeDailyCycleRepository()
	claimedAt := time.Now().Add(-30 * time.Hour)
	repo.progress["user-1"] = &rewards.UserDailyRewardProgress{
		UserID:         "user-1",
		CurrentDay:     14,
		CycleStartedAt: time.Now().Add(-13 * 24 * time.Hour),
		LastClaimedAt:  &claimedAt,
	}
	svc := rewards.NewDailyCycleService(repo, fakeTransactor{})

	if _, err := svc.ClaimToday(context.Background(), "user-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := repo.progress["user-1"]
	if got.CurrentDay != 1 {
		t.Fatalf("expected wraparound to day 1 after claiming day 14, got %d", got.CurrentDay)
	}
	if !got.CycleStartedAt.After(time.Now().Add(-time.Minute)) {
		t.Fatal("expected CycleStartedAt to reset to ~now on a fresh lap")
	}
}
