package dailysummary

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/event"
	"github.com/google/uuid"
)

func TestCheckInInitializesCheckpointWithoutSummary(t *testing.T) {
	userID := uuid.New()
	now := mustTime(t, "2026-08-09T09:00:00+03:00")
	repo := &checkpointStub{}
	events := &eventReaderStub{highWater: 17}
	pets := &petProviderStub{}
	service := newTestService(t, repo, events, pets, now)

	result, err := service.CheckIn(context.Background(), userID)
	if err != nil {
		t.Fatalf("CheckIn() error = %v", err)
	}
	if result.ShouldShow || result.Summary != nil {
		t.Fatalf("first CheckIn() result = %+v, want no summary", result)
	}
	if repo.checkpoint.LastEventPosition != events.highWater {
		t.Fatalf("checkpoint position = %d, want %d", repo.checkpoint.LastEventPosition, events.highWater)
	}
	if !repo.checkpoint.LastCheckInAt.Equal(now.UTC()) {
		t.Fatalf("checkpoint time = %s, want %s", repo.checkpoint.LastCheckInAt, now.UTC())
	}
	if pets.called {
		t.Fatal("pet provider was called for the first check-in")
	}
}

func TestCheckInSameMoscowDayAdvancesCheckpointWithoutSummary(t *testing.T) {
	userID := uuid.New()
	previous := mustTime(t, "2026-08-09T08:00:00+03:00")
	now := mustTime(t, "2026-08-09T22:00:00+03:00")
	repo := &checkpointStub{exists: true, checkpoint: Checkpoint{
		UserID:            userID,
		LastCheckInAt:     previous,
		LastEventPosition: 4,
	}}
	events := &eventReaderStub{highWater: 11}
	pets := &petProviderStub{}
	service := newTestService(t, repo, events, pets, now)

	result, err := service.CheckIn(context.Background(), userID)
	if err != nil {
		t.Fatalf("CheckIn() error = %v", err)
	}
	if result.ShouldShow || result.Summary != nil {
		t.Fatalf("same-day CheckIn() result = %+v, want no summary", result)
	}
	if repo.checkpoint.LastEventPosition != 11 || !repo.checkpoint.LastCheckInAt.Equal(now.UTC()) {
		t.Fatalf("checkpoint = %+v, want current time and position", repo.checkpoint)
	}
	if len(events.requests) != 0 {
		t.Fatalf("event pages read = %d, want 0", len(events.requests))
	}
}

func TestCheckInBuildsSummaryFromBoundedEventPages(t *testing.T) {
	userID := uuid.New()
	previous := mustTime(t, "2026-08-08T20:00:00+03:00")
	now := mustTime(t, "2026-08-09T09:00:00+03:00")
	rewardPayload := event.RewardPayload{
		UserRewardID:       uuid.New(),
		RewardDefinitionID: uuid.New(),
	}
	repo := &checkpointStub{exists: true, checkpoint: Checkpoint{
		UserID:            userID,
		LastCheckInAt:     previous,
		LastEventPosition: 3,
	}}
	events := &eventReaderStub{
		highWater: 9,
		events: []event.Event{
			storedEvent(t, userID, 4, event.ExperienceGranted, event.GrantPayload{Amount: 12, Source: "game"}),
			storedEvent(t, userID, 5, event.CoinsGranted, event.GrantPayload{Amount: 7, Source: "game"}),
			storedEvent(t, userID, 6, event.GameSessionCompleted, event.GameSessionCompletedPayload{SessionID: uuid.New(), Score: 42}),
			storedEvent(t, userID, 7, event.GameSessionCompleted, event.GameSessionCompletedPayload{SessionID: uuid.New(), Score: 31}),
			storedEvent(t, userID, 8, event.RewardGranted, rewardPayload),
			storedEvent(t, userID, 9, event.LevelReached, event.LevelReachedPayload{Level: 2}),
		},
		pageLimit: 2,
	}
	pets := &petProviderStub{state: PetPeriodState{
		PetID:               uuid.New(),
		Before:              PetVitals{Satiety: 90, Mood: 80, Energy: 20},
		After:               PetVitals{Satiety: 60, Mood: 70, Energy: 100},
		Level:               2,
		Experience:          140,
		NextLevel:           3,
		NextLevelExperience: 200,
	}}
	service := newTestService(t, repo, events, pets, now)

	result, err := service.CheckIn(context.Background(), userID)
	if err != nil {
		t.Fatalf("CheckIn() error = %v", err)
	}
	if !result.ShouldShow || result.Summary == nil {
		t.Fatalf("CheckIn() result = %+v, want summary", result)
	}
	summary := result.Summary
	if summary.ExperienceEarned != 12 || summary.CoinsEarned != 7 {
		t.Fatalf("earned = xp %d, coins %d", summary.ExperienceEarned, summary.CoinsEarned)
	}
	if summary.BestGameScore == nil || *summary.BestGameScore != 42 {
		t.Fatalf("best score = %v, want 42", summary.BestGameScore)
	}
	if len(summary.Rewards) != 1 || summary.Rewards[0].UserRewardID != rewardPayload.UserRewardID {
		t.Fatalf("rewards = %+v, want user reward %s", summary.Rewards, rewardPayload.UserRewardID)
	}
	if summary.Pet.Satiety.Delta != -30 || summary.Pet.Mood.Delta != -10 || summary.Pet.Energy.Delta != 80 {
		t.Fatalf("pet changes = %+v", summary.Pet)
	}
	if summary.Pet.NextGoal.TargetLevel != 3 || summary.Pet.NextGoal.TargetExperience != 200 {
		t.Fatalf("next goal = %+v", summary.Pet.NextGoal)
	}
	if summary.AbsenceDuration != 13*time.Hour {
		t.Fatalf("absence = %s, want 13h", summary.AbsenceDuration)
	}
	if len(events.requests) != 3 {
		t.Fatalf("event page requests = %d, want 3", len(events.requests))
	}
	if repo.checkpoint.LastEventPosition != 9 || !repo.checkpoint.LastCheckInAt.Equal(now.UTC()) {
		t.Fatalf("checkpoint = %+v, want high water 9", repo.checkpoint)
	}
}

func TestCheckInUsesMoscowCalendarBoundary(t *testing.T) {
	userID := uuid.New()
	previous := mustTime(t, "2026-08-08T23:59:00+03:00")
	now := mustTime(t, "2026-08-09T00:01:00+03:00")
	repo := &checkpointStub{exists: true, checkpoint: Checkpoint{
		UserID: userID, LastCheckInAt: previous, LastEventPosition: 0,
	}}
	service := newTestService(
		t,
		repo,
		&eventReaderStub{},
		&petProviderStub{state: PetPeriodState{PetID: uuid.New()}},
		now,
	)

	result, err := service.CheckIn(context.Background(), userID)
	if err != nil {
		t.Fatalf("CheckIn() error = %v", err)
	}
	if !result.ShouldShow {
		t.Fatal("CheckIn() did not show summary across Moscow midnight")
	}
}

func TestCheckInRollsBackCheckpointWhenPetProviderFails(t *testing.T) {
	userID := uuid.New()
	previous := mustTime(t, "2026-08-08T20:00:00+03:00")
	now := mustTime(t, "2026-08-09T09:00:00+03:00")
	wantErr := errors.New("pet unavailable")
	repo := &checkpointStub{exists: true, checkpoint: Checkpoint{
		UserID: userID, LastCheckInAt: previous, LastEventPosition: 2,
	}}
	pets := &petProviderStub{err: wantErr}
	service := newTestService(t, repo, &eventReaderStub{highWater: 3}, pets, now)

	_, err := service.CheckIn(context.Background(), userID)
	if !errors.Is(err, wantErr) {
		t.Fatalf("CheckIn() error = %v, want %v", err, wantErr)
	}
	if repo.updates != 0 {
		t.Fatalf("checkpoint updates = %d, want 0", repo.updates)
	}
}

func TestCheckInQuarantinesUnsupportedRelevantEventSchema(t *testing.T) {
	userID := uuid.New()
	previous := mustTime(t, "2026-08-08T20:00:00+03:00")
	now := mustTime(t, "2026-08-09T09:00:00+03:00")
	bad := storedEvent(t, userID, 1, event.CoinsGranted, event.GrantPayload{Amount: 1, Source: "test"})
	bad.SchemaVersion = 2
	repo := &checkpointStub{exists: true, checkpoint: Checkpoint{
		UserID: userID, LastCheckInAt: previous, LastEventPosition: 0,
	}}
	service := newTestService(
		t,
		repo,
		&eventReaderStub{highWater: 1, events: []event.Event{bad}},
		&petProviderStub{},
		now,
	)

	result, err := service.CheckIn(context.Background(), userID)
	if err != nil {
		t.Fatalf("CheckIn() error = %v", err)
	}
	if !result.ShouldShow || result.Summary == nil || result.Summary.SkippedEventCount != 1 {
		t.Fatalf("CheckIn() result = %+v, want one skipped event", result)
	}
	if len(repo.failures) != 1 || repo.failures[0].Reason != EventFailureUnsupportedSchema {
		t.Fatalf("event failures = %+v, want unsupported schema", repo.failures)
	}
	if repo.updates != 1 || repo.checkpoint.LastEventPosition != 1 {
		t.Fatalf("checkpoint = %+v updates=%d, want advanced checkpoint", repo.checkpoint, repo.updates)
	}
}

func TestCheckInRollsBackWhenQuarantineWriteFails(t *testing.T) {
	userID := uuid.New()
	previous := mustTime(t, "2026-08-08T20:00:00+03:00")
	now := mustTime(t, "2026-08-09T09:00:00+03:00")
	bad := storedEvent(t, userID, 1, event.CoinsGranted, event.GrantPayload{Amount: 1, Source: "test"})
	bad.SchemaVersion = 2
	wantErr := errors.New("quarantine unavailable")
	repo := &checkpointStub{
		exists: true,
		checkpoint: Checkpoint{
			UserID: userID, LastCheckInAt: previous, LastEventPosition: 0,
		},
		failureErr: wantErr,
	}
	service := newTestService(
		t,
		repo,
		&eventReaderStub{highWater: 1, events: []event.Event{bad}},
		&petProviderStub{},
		now,
	)

	_, err := service.CheckIn(context.Background(), userID)
	if !errors.Is(err, wantErr) {
		t.Fatalf("CheckIn() error = %v, want %v", err, wantErr)
	}
	if repo.updates != 0 {
		t.Fatalf("checkpoint updates = %d, want 0", repo.updates)
	}
}

func TestCheckInRejectsOversizedBacklog(t *testing.T) {
	userID := uuid.New()
	previous := mustTime(t, "2026-08-08T20:00:00+03:00")
	now := mustTime(t, "2026-08-09T09:00:00+03:00")
	events := make([]event.Event, maxEventsPerCheckIn+1)
	for index := range events {
		events[index] = storedEvent(
			t,
			userID,
			int64(index+1),
			event.ExperienceGranted,
			event.GrantPayload{Amount: 1, Source: "test"},
		)
	}
	repo := &checkpointStub{exists: true, checkpoint: Checkpoint{
		UserID: userID, LastCheckInAt: previous, LastEventPosition: 0,
	}}
	service := newTestService(
		t,
		repo,
		&eventReaderStub{highWater: int64(len(events)), events: events},
		&petProviderStub{},
		now,
	)

	_, err := service.CheckIn(context.Background(), userID)
	if !errors.Is(err, ErrBacklogLimitExceeded) {
		t.Fatalf("CheckIn() error = %v, want ErrBacklogLimitExceeded", err)
	}
	if repo.updates != 0 {
		t.Fatalf("checkpoint updates = %d, want 0", repo.updates)
	}
}

func TestCheckInRejectsPositionRegression(t *testing.T) {
	userID := uuid.New()
	now := mustTime(t, "2026-08-09T09:00:00+03:00")
	repo := &checkpointStub{exists: true, checkpoint: Checkpoint{
		UserID: userID, LastCheckInAt: now.Add(-time.Hour), LastEventPosition: 9,
	}}
	service := newTestService(t, repo, &eventReaderStub{highWater: 8}, &petProviderStub{}, now)

	_, err := service.CheckIn(context.Background(), userID)
	if !errors.Is(err, ErrPositionRegression) {
		t.Fatalf("CheckIn() error = %v, want ErrPositionRegression", err)
	}
}

func TestCheckInRecapturesBoundaryAfterConcurrentWait(t *testing.T) {
	userID := uuid.New()
	repo := newLockingCheckpointStub(Checkpoint{
		UserID:            userID,
		LastCheckInAt:     mustTime(t, "2026-08-08T20:00:00+03:00"),
		LastEventPosition: 0,
	})
	reader := newScriptedEventReader(map[string][]EventBoundary{
		"older": {
			{HighWater: 1, CapturedAt: mustTime(t, "2026-08-09T08:00:00+03:00")},
			{HighWater: 3, CapturedAt: mustTime(t, "2026-08-09T08:03:00+03:00")},
		},
		"newer": {
			{HighWater: 2, CapturedAt: mustTime(t, "2026-08-09T08:01:00+03:00")},
			{HighWater: 2, CapturedAt: mustTime(t, "2026-08-09T08:02:00+03:00")},
		},
	})
	pets := &petProviderStub{state: PetPeriodState{PetID: uuid.New()}}
	service, err := NewService(repo, repo, reader, pets, lockingTransactorStub{repo: repo})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	olderCtx := context.WithValue(context.Background(), checkInLabelKey{}, "older")
	newerCtx := context.WithValue(context.Background(), checkInLabelKey{}, "newer")
	olderDone := make(chan struct{})
	var olderResult CheckInResult
	var olderErr error
	go func() {
		defer close(olderDone)
		olderResult, olderErr = service.CheckIn(olderCtx, userID)
	}()

	<-repo.olderInsertReached
	newerResult, newerErr := service.CheckIn(newerCtx, userID)
	if newerErr != nil {
		t.Fatalf("newer CheckIn() error = %v", newerErr)
	}
	close(repo.allowOlderInsert)
	<-olderDone
	if olderErr != nil {
		t.Fatalf("older CheckIn() error = %v", olderErr)
	}
	if !newerResult.ShouldShow || olderResult.ShouldShow {
		t.Fatalf(
			"concurrent results: newer=%+v older=%+v, want only newer summary",
			newerResult,
			olderResult,
		)
	}
	if repo.checkpoint.LastEventPosition != 3 ||
		!repo.checkpoint.LastCheckInAt.Equal(mustTime(t, "2026-08-09T08:03:00+03:00").UTC()) {
		t.Fatalf("final checkpoint = %+v, want recaptured older-call boundary", repo.checkpoint)
	}
}

func newTestService(
	t *testing.T,
	repo testDailySummaryRepository,
	events EventReader,
	pets PetSummaryProvider,
	now time.Time,
) *Service {
	t.Helper()
	if stub, ok := events.(*eventReaderStub); ok {
		stub.capturedAt = now
	}
	service, err := NewService(repo, repo, events, pets, transactorStub{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

type testDailySummaryRepository interface {
	CheckpointRepository
	EventFailureRepository
}

type checkpointStub struct {
	exists     bool
	checkpoint Checkpoint
	updates    int
	failures   []EventFailure
	failureErr error
}

func (s *checkpointStub) RecordEventFailure(_ context.Context, failure EventFailure) error {
	if s.failureErr != nil {
		return s.failureErr
	}
	s.failures = append(s.failures, failure)
	return nil
}

func (s *checkpointStub) InsertCheckpoint(_ context.Context, checkpoint Checkpoint) (bool, error) {
	if s.exists {
		return false, nil
	}
	s.exists = true
	s.checkpoint = checkpoint
	return true, nil
}

func (s *checkpointStub) GetCheckpointForUpdate(_ context.Context, _ uuid.UUID) (Checkpoint, error) {
	if !s.exists {
		return Checkpoint{}, ErrCheckpointNotFound
	}
	return s.checkpoint, nil
}

func (s *checkpointStub) UpdateCheckpoint(_ context.Context, checkpoint Checkpoint) error {
	s.checkpoint = checkpoint
	s.updates++
	return nil
}

type eventPageRequest struct {
	after int64
	to    int64
	limit int32
}

type eventReaderStub struct {
	highWater  int64
	capturedAt time.Time
	events     []event.Event
	pageLimit  int
	requests   []eventPageRequest
}

func (s *eventReaderStub) CaptureBoundary(context.Context) (EventBoundary, error) {
	return EventBoundary{HighWater: s.highWater, CapturedAt: s.capturedAt}, nil
}

func (s *eventReaderStub) ListUserEventsByPosition(
	_ context.Context,
	ownerUserID uuid.UUID,
	afterPosition int64,
	toPosition int64,
	limit int32,
) ([]event.Event, error) {
	s.requests = append(s.requests, eventPageRequest{after: afterPosition, to: toPosition, limit: limit})
	pageLimit := int(limit)
	if s.pageLimit > 0 && s.pageLimit < pageLimit {
		pageLimit = s.pageLimit
	}
	result := make([]event.Event, 0, pageLimit)
	for _, stored := range s.events {
		if stored.OwnerUserID == ownerUserID &&
			stored.GlobalPosition > afterPosition &&
			stored.GlobalPosition <= toPosition {
			result = append(result, stored)
			if len(result) == pageLimit {
				break
			}
		}
	}
	return result, nil
}

type petProviderStub struct {
	state  PetPeriodState
	err    error
	called bool
}

func (s *petProviderStub) GetPeriodState(
	context.Context,
	uuid.UUID,
	time.Time,
	time.Time,
) (PetPeriodState, error) {
	s.called = true
	return s.state, s.err
}

type transactorStub struct{}

func (transactorStub) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type checkInLabelKey struct{}

type lockingCheckpointStub struct {
	transactionLock    sync.Mutex
	stateLock          sync.Mutex
	checkpoint         Checkpoint
	olderInsertReached chan struct{}
	allowOlderInsert   chan struct{}
	olderSignalOnce    sync.Once
}

func newLockingCheckpointStub(checkpoint Checkpoint) *lockingCheckpointStub {
	return &lockingCheckpointStub{
		checkpoint:         checkpoint,
		olderInsertReached: make(chan struct{}),
		allowOlderInsert:   make(chan struct{}),
	}
}

func (s *lockingCheckpointStub) InsertCheckpoint(
	ctx context.Context,
	_ Checkpoint,
) (bool, error) {
	if ctx.Value(checkInLabelKey{}) == "older" {
		s.olderSignalOnce.Do(func() { close(s.olderInsertReached) })
		<-s.allowOlderInsert
	}
	return false, nil
}

func (s *lockingCheckpointStub) GetCheckpointForUpdate(
	context.Context,
	uuid.UUID,
) (Checkpoint, error) {
	s.transactionLock.Lock()
	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	return s.checkpoint, nil
}

func (s *lockingCheckpointStub) UpdateCheckpoint(
	_ context.Context,
	checkpoint Checkpoint,
) error {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	s.checkpoint = checkpoint
	return nil
}

func (*lockingCheckpointStub) RecordEventFailure(context.Context, EventFailure) error {
	return nil
}

type lockingTransactorStub struct {
	repo *lockingCheckpointStub
}

func (s lockingTransactorStub) Do(ctx context.Context, fn func(context.Context) error) error {
	err := fn(ctx)
	s.repo.transactionLock.Unlock()
	return err
}

type scriptedEventReader struct {
	lock       sync.Mutex
	boundaries map[string][]EventBoundary
	calls      map[string]int
}

func newScriptedEventReader(boundaries map[string][]EventBoundary) *scriptedEventReader {
	return &scriptedEventReader{boundaries: boundaries, calls: make(map[string]int)}
}

func (s *scriptedEventReader) CaptureBoundary(ctx context.Context) (EventBoundary, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	label, _ := ctx.Value(checkInLabelKey{}).(string)
	index := s.calls[label]
	values := s.boundaries[label]
	if index >= len(values) {
		return EventBoundary{}, errors.New("unexpected boundary capture")
	}
	s.calls[label]++
	return values[index], nil
}

func (*scriptedEventReader) ListUserEventsByPosition(
	context.Context,
	uuid.UUID,
	int64,
	int64,
	int32,
) ([]event.Event, error) {
	return nil, nil
}

func storedEvent(
	t *testing.T,
	ownerID uuid.UUID,
	position int64,
	eventType event.EventType,
	payload any,
) event.Event {
	t.Helper()
	return storedEventWithID(t, ownerID, uuid.New(), position, eventType, payload)
}

func storedEventWithID(
	t *testing.T,
	ownerID uuid.UUID,
	eventID uuid.UUID,
	position int64,
	eventType event.EventType,
	payload any,
) event.Event {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal event payload: %v", err)
	}
	return event.Event{
		GlobalPosition: position,
		ID:             eventID,
		OwnerUserID:    ownerID,
		Type:           eventType,
		SchemaVersion:  event.SchemaVersionV1,
		Payload:        encoded,
		RecordedAt:     time.Unix(position, 0).UTC(),
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}
