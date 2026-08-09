package dailysummary

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/event"
	"github.com/google/uuid"
)

const (
	summaryTimezone     = "Europe/Moscow"
	eventPageSize       = int32(500)
	maxEventsPerCheckIn = 10_000
	checkInTimeout      = 4 * time.Second
)

type Service struct {
	checkpoints CheckpointRepository
	failures    EventFailureRepository
	events      EventReader
	pets        PetSummaryProvider
	transactor  Transactor
	location    *time.Location
}

// NewService creates a Daily Summary service with transactional persistence,
// owner-scoped event access, and lazy pet-state calculation.
func NewService(
	checkpoints CheckpointRepository,
	failures EventFailureRepository,
	events EventReader,
	pets PetSummaryProvider,
	transactor Transactor,
) (*Service, error) {
	if checkpoints == nil || failures == nil || events == nil || pets == nil || transactor == nil {
		return nil, fmt.Errorf("%w: dependencies are required", ErrValidation)
	}

	location, err := time.LoadLocation(summaryTimezone)
	if err != nil {
		return nil, fmt.Errorf("load daily summary timezone: %w", err)
	}

	return &Service{
		checkpoints: checkpoints,
		failures:    failures,
		events:      events,
		pets:        pets,
		transactor:  transactor,
		location:    location,
	}, nil
}

// CheckIn advances the user's activity checkpoint and returns a summary on the first
// check-in of a new calendar day in Europe/Moscow. Callers must supply userID from
// the authenticated server context, never from a client-controlled owner field.
func (s *Service) CheckIn(ctx context.Context, userID uuid.UUID) (CheckInResult, error) {
	if userID == uuid.Nil {
		return CheckInResult{}, fmt.Errorf("%w: user ID is required", ErrValidation)
	}

	result := CheckInResult{}
	checkInCtx, cancel := context.WithTimeout(ctx, checkInTimeout)
	defer cancel()

	err := s.transactor.Do(checkInCtx, func(txCtx context.Context) error {
		initialBoundary, err := s.events.CaptureBoundary(txCtx)
		if err != nil {
			return fmt.Errorf("capture initial event store boundary: %w", err)
		}
		created, err := s.checkpoints.InsertCheckpoint(txCtx, Checkpoint{
			UserID:            userID,
			LastCheckInAt:     normalizeBoundaryTime(initialBoundary.CapturedAt),
			LastEventPosition: initialBoundary.HighWater,
		})
		if err != nil {
			return fmt.Errorf("insert daily summary checkpoint: %w", err)
		}

		checkpoint, err := s.checkpoints.GetCheckpointForUpdate(txCtx, userID)
		if err != nil {
			return fmt.Errorf("lock daily summary checkpoint: %w", err)
		}

		boundary, err := s.events.CaptureBoundary(txCtx)
		if err != nil {
			return fmt.Errorf("capture event store boundary: %w", err)
		}
		now := normalizeBoundaryTime(boundary.CapturedAt)
		summaryEndPosition := boundary.HighWater
		if now.IsZero() {
			return fmt.Errorf("%w: event store boundary time is required", ErrValidation)
		}
		if summaryEndPosition < checkpoint.LastEventPosition {
			return fmt.Errorf(
				"%w: checkpoint=%d high_water=%d",
				ErrPositionRegression,
				checkpoint.LastEventPosition,
				summaryEndPosition,
			)
		}
		if checkpoint.LastCheckInAt.After(now) {
			return fmt.Errorf("%w: last check-in is in the future", ErrValidation)
		}

		if created || sameCalendarDay(checkpoint.LastCheckInAt, now, s.location) {
			return s.checkpoints.UpdateCheckpoint(txCtx, Checkpoint{
				UserID:            userID,
				LastCheckInAt:     now,
				LastEventPosition: summaryEndPosition,
			})
		}
		activity, err := s.collectActivity(
			txCtx,
			userID,
			checkpoint.LastEventPosition,
			summaryEndPosition,
		)
		if err != nil {
			return err
		}

		petState, err := s.pets.GetPeriodState(
			txCtx,
			userID,
			checkpoint.LastCheckInAt,
			now,
		)
		if err != nil {
			return fmt.Errorf("get pet period state: %w", err)
		}

		summary := Summary{
			PeriodStartedAt:   checkpoint.LastCheckInAt,
			PeriodEndedAt:     now,
			AbsenceDuration:   now.Sub(checkpoint.LastCheckInAt),
			Pet:               petChanges(petState),
			ExperienceEarned:  activity.experience,
			CoinsEarned:       activity.coins,
			BestGameScore:     activity.bestGameScore,
			Rewards:           activity.rewards,
			SkippedEventCount: activity.skippedEvents,
		}

		if err := s.checkpoints.UpdateCheckpoint(txCtx, Checkpoint{
			UserID:            userID,
			LastCheckInAt:     now,
			LastEventPosition: summaryEndPosition,
		}); err != nil {
			return fmt.Errorf("update daily summary checkpoint: %w", err)
		}

		result = CheckInResult{ShouldShow: true, Summary: &summary}
		return nil
	})
	if err != nil {
		return CheckInResult{}, err
	}

	return result, nil
}

type activitySummary struct {
	experience    int64
	coins         int64
	bestGameScore *int64
	rewards       []EarnedReward
	skippedEvents int
}

// collectActivity aggregates relevant events between the persisted checkpoint and
// the captured snapshot boundary. Invalid payloads are quarantined and excluded.
func (s *Service) collectActivity(
	ctx context.Context,
	userID uuid.UUID,
	checkpointPosition int64,
	summaryEndPosition int64,
) (activitySummary, error) {
	activity := activitySummary{rewards: make([]EarnedReward, 0)}
	lastProcessedPosition := checkpointPosition
	processedEvents := 0

	for lastProcessedPosition < summaryEndPosition {
		page, err := s.events.ListUserEventsByPosition(
			ctx,
			userID,
			lastProcessedPosition,
			summaryEndPosition,
			eventPageSize,
		)
		if err != nil {
			return activitySummary{}, fmt.Errorf("list daily summary events: %w", err)
		}
		if len(page) == 0 {
			break
		}
		if len(page) > int(eventPageSize) {
			return activitySummary{}, fmt.Errorf("%w: event page exceeds configured limit", ErrInvalidEvent)
		}

		for _, stored := range page {
			if processedEvents >= maxEventsPerCheckIn {
				return activitySummary{}, ErrBacklogLimitExceeded
			}
			processedEvents++
			if stored.OwnerUserID != userID ||
				stored.ID == uuid.Nil ||
				stored.GlobalPosition <= lastProcessedPosition ||
				stored.GlobalPosition > summaryEndPosition {
				return activitySummary{}, fmt.Errorf(
					"%w: event position or owner is outside the requested range",
					ErrInvalidEvent,
				)
			}
			if err := applyEvent(&activity, stored); err != nil {
				reason, quarantinable := eventFailureReason(err)
				if !quarantinable {
					return activitySummary{}, err
				}
				if recordErr := s.failures.RecordEventFailure(ctx, EventFailure{
					UserID:         userID,
					EventID:        stored.ID,
					GlobalPosition: stored.GlobalPosition,
					EventType:      stored.Type,
					SchemaVersion:  stored.SchemaVersion,
					Reason:         reason,
				}); recordErr != nil {
					return activitySummary{}, fmt.Errorf("record daily summary event failure: %w", recordErr)
				}
				activity.skippedEvents++
			}
			lastProcessedPosition = stored.GlobalPosition
		}
	}

	return activity, nil
}

// applyEvent validates one supported payload version and applies its contribution
// to the in-memory summary accumulator.
func applyEvent(activity *activitySummary, stored event.Event) error {
	switch stored.Type {
	case event.ExperienceGranted, event.CoinsGranted, event.GameSessionCompleted, event.RewardGranted:
		if stored.SchemaVersion != event.SchemaVersionV1 {
			return fmt.Errorf(
				"%w: type=%s version=%d",
				ErrUnsupportedEventSchema,
				stored.Type,
				stored.SchemaVersion,
			)
		}
	default:
		return nil
	}

	switch stored.Type {
	case event.ExperienceGranted, event.CoinsGranted:
		payload, err := event.DecodePayload[event.GrantPayload](stored)
		if err != nil || payload.Amount <= 0 {
			return fmt.Errorf("%w: decode %s payload", ErrInvalidEvent, stored.Type)
		}
		amount := int64(payload.Amount)
		if stored.Type == event.ExperienceGranted {
			value, err := addEarnedAmount(activity.experience, amount)
			if err != nil {
				return err
			}
			activity.experience = value
		} else {
			value, err := addEarnedAmount(activity.coins, amount)
			if err != nil {
				return err
			}
			activity.coins = value
		}
	case event.GameSessionCompleted:
		payload, err := event.DecodePayload[event.GameSessionCompletedPayload](stored)
		if err != nil || payload.SessionID == uuid.Nil || payload.Score < 0 {
			return fmt.Errorf("%w: decode %s payload", ErrInvalidEvent, stored.Type)
		}
		score := int64(payload.Score)
		if activity.bestGameScore == nil || score > *activity.bestGameScore {
			activity.bestGameScore = &score
		}
	case event.RewardGranted:
		payload, err := event.DecodePayload[event.RewardPayload](stored)
		if err != nil || payload.UserRewardID == uuid.Nil || payload.RewardDefinitionID == uuid.Nil {
			return fmt.Errorf("%w: decode %s payload", ErrInvalidEvent, stored.Type)
		}
		activity.rewards = append(activity.rewards, EarnedReward{
			UserRewardID:       payload.UserRewardID,
			RewardDefinitionID: payload.RewardDefinitionID,
		})
	}

	return nil
}

func eventFailureReason(err error) (EventFailureReason, bool) {
	switch {
	case errors.Is(err, ErrUnsupportedEventSchema):
		return EventFailureUnsupportedSchema, true
	case errors.Is(err, ErrInvalidEvent):
		return EventFailureInvalidPayload, true
	default:
		return "", false
	}
}

func addEarnedAmount(accumulatedAmount, earnedAmount int64) (int64, error) {
	if earnedAmount > 0 && accumulatedAmount > math.MaxInt64-earnedAmount {
		return 0, fmt.Errorf("%w: earned amount overflow", ErrInvalidEvent)
	}
	return accumulatedAmount + earnedAmount, nil
}

func petChanges(state PetPeriodState) PetChanges {
	return PetChanges{
		PetID:   state.PetID,
		Satiety: buildStatChange(state.Before.Satiety, state.After.Satiety),
		Mood:    buildStatChange(state.Before.Mood, state.After.Mood),
		Energy:  buildStatChange(state.Before.Energy, state.After.Energy),
		NextGoal: ProgressGoal{
			CurrentLevel:      state.Level,
			CurrentExperience: state.Experience,
			TargetLevel:       state.NextLevel,
			TargetExperience:  state.NextLevelExperience,
		},
	}
}

func buildStatChange(previousValue, currentValue int) StatChange {
	return StatChange{
		Before: previousValue,
		After:  currentValue,
		Delta:  currentValue - previousValue,
	}
}

func sameCalendarDay(left, right time.Time, location *time.Location) bool {
	leftDate := left.In(location)
	rightDate := right.In(location)
	return leftDate.Year() == rightDate.Year() && leftDate.YearDay() == rightDate.YearDay()
}

func normalizeBoundaryTime(value time.Time) time.Time {
	return value.UTC().Truncate(time.Microsecond)
}
