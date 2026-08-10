//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/dailysummary"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/event"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository/postgres"
	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDailySummaryPostgresIntegration(t *testing.T) {
	adminDSN := os.Getenv("TEST_DATABASE_URL")
	runtimeDSN := os.Getenv("TEST_RUNTIME_DATABASE_URL")
	if adminDSN == "" || runtimeDSN == "" {
		t.Skip("test database URLs not set, skipping integration test")
	}

	ctx := context.Background()
	adminPool, err := pgxpool.New(ctx, adminDSN)
	if err != nil {
		t.Fatalf("connect to test database as administrator: %v", err)
	}
	t.Cleanup(adminPool.Close)
	runtimePool, err := pgxpool.New(ctx, runtimeDSN)
	if err != nil {
		t.Fatalf("connect to test database as app_runtime: %v", err)
	}
	t.Cleanup(runtimePool.Close)

	repositories := postgres.New(runtimePool)
	transactionManager := manager.Must(trmpgx.NewDefaultFactory(runtimePool))
	events := event.NewService(repositories.Event, transactionManager)

	t.Run("stable high water and owner isolation", func(t *testing.T) {
		ownerID := createIntegrationUser(t, adminPool)
		otherOwnerID := createIntegrationUser(t, adminPool)
		before, err := repositories.Event.GetHighWater(ctx)
		if err != nil {
			t.Fatalf("GetHighWater() before publish: %v", err)
		}

		publishIntegrationXP(t, ctx, events, ownerID, 10)
		boundedHighWater, err := repositories.Event.GetHighWater(ctx)
		if err != nil {
			t.Fatalf("GetHighWater() bounded: %v", err)
		}
		publishIntegrationXP(t, ctx, events, ownerID, 20)
		publishIntegrationXP(t, ctx, events, otherOwnerID, 30)

		bounded, err := repositories.Event.ListUserEventsByPosition(
			ctx,
			ownerID,
			before,
			boundedHighWater,
			500,
		)
		if err != nil {
			t.Fatalf("ListUserEventsByPosition() bounded: %v", err)
		}
		if len(bounded) != 1 {
			t.Fatalf("bounded owner events = %d, want 1", len(bounded))
		}

		latest, err := repositories.Event.GetHighWater(ctx)
		if err != nil {
			t.Fatalf("GetHighWater() latest: %v", err)
		}
		ownerEvents, err := repositories.Event.ListUserEventsByPosition(
			ctx,
			ownerID,
			boundedHighWater,
			latest,
			500,
		)
		if err != nil {
			t.Fatalf("ListUserEventsByPosition() latest: %v", err)
		}
		if len(ownerEvents) != 1 || ownerEvents[0].OwnerUserID != ownerID {
			t.Fatalf("latest owner events = %+v, want one owner-scoped event", ownerEvents)
		}
	})

	t.Run("event pagination over five hundred rows", func(t *testing.T) {
		ownerID := createIntegrationUser(t, adminPool)
		before, err := repositories.Event.GetHighWater(ctx)
		if err != nil {
			t.Fatalf("GetHighWater() before publish: %v", err)
		}
		publishIntegrationXPBatch(t, ctx, events, ownerID, 501)
		highWater, err := repositories.Event.GetHighWater(ctx)
		if err != nil {
			t.Fatalf("GetHighWater() after publish: %v", err)
		}

		first, err := repositories.Event.ListUserEventsByPosition(ctx, ownerID, before, highWater, 500)
		if err != nil {
			t.Fatalf("first event page: %v", err)
		}
		if len(first) != 500 {
			t.Fatalf("first event page size = %d, want 500", len(first))
		}
		second, err := repositories.Event.ListUserEventsByPosition(
			ctx,
			ownerID,
			first[len(first)-1].GlobalPosition,
			highWater,
			500,
		)
		if err != nil {
			t.Fatalf("second event page: %v", err)
		}
		if len(second) != 1 {
			t.Fatalf("second event page size = %d, want 1", len(second))
		}
	})

	t.Run("concurrent check ins create one summary", func(t *testing.T) {
		ownerID := createIntegrationUser(t, adminPool)
		before, err := repositories.Event.GetHighWater(ctx)
		if err != nil {
			t.Fatalf("GetHighWater() before publish: %v", err)
		}
		created, err := repositories.DailySummary.InsertCheckpoint(ctx, dailysummary.Checkpoint{
			UserID:            ownerID,
			LastCheckInAt:     time.Now().UTC().Add(-25 * time.Hour),
			LastEventPosition: before,
		})
		if err != nil || !created {
			t.Fatalf("InsertCheckpoint() created=%v error=%v", created, err)
		}
		publishIntegrationXP(t, ctx, events, ownerID, 15)

		summaries, errs := checkInConcurrently(
			ctx,
			repositories,
			transactionManager,
			ownerID,
		)
		for _, checkInErr := range errs {
			if checkInErr != nil {
				t.Fatalf("concurrent CheckIn() error = %v", checkInErr)
			}
		}
		shown := 0
		for _, result := range summaries {
			if result.ShouldShow {
				shown++
				if result.Summary == nil || result.Summary.ExperienceEarned != 15 {
					t.Fatalf("shown summary = %+v, want 15 experience", result.Summary)
				}
			}
		}
		if shown != 1 {
			t.Fatalf("shown summaries = %d, want 1", shown)
		}
	})

	t.Run("provider failure rolls back checkpoint", func(t *testing.T) {
		ownerID := createIntegrationUser(t, adminPool)
		position, err := repositories.Event.GetHighWater(ctx)
		if err != nil {
			t.Fatalf("GetHighWater(): %v", err)
		}
		previous := time.Now().UTC().Add(-25 * time.Hour).Truncate(time.Microsecond)
		created, err := repositories.DailySummary.InsertCheckpoint(ctx, dailysummary.Checkpoint{
			UserID: ownerID, LastCheckInAt: previous, LastEventPosition: position,
		})
		if err != nil || !created {
			t.Fatalf("InsertCheckpoint() created=%v error=%v", created, err)
		}

		wantErr := errors.New("pet provider failed")
		service, err := dailysummary.NewService(
			repositories.DailySummary,
			repositories.DailySummary,
			repositories.Event,
			integrationPetProvider{err: wantErr},
			transactionManager,
		)
		if err != nil {
			t.Fatalf("NewService() error = %v", err)
		}
		_, err = service.CheckIn(ctx, ownerID)
		if !errors.Is(err, wantErr) {
			t.Fatalf("CheckIn() error = %v, want provider error", err)
		}

		checkpoint, err := repositories.DailySummary.GetCheckpointForUpdate(ctx, ownerID)
		if err != nil {
			t.Fatalf("GetCheckpointForUpdate() error = %v", err)
		}
		if !checkpoint.LastCheckInAt.Equal(previous) || checkpoint.LastEventPosition != position {
			t.Fatalf("checkpoint after rollback = %+v", checkpoint)
		}
	})

	t.Run("unsupported event is quarantined and checkpoint advances", func(t *testing.T) {
		ownerID := createIntegrationUser(t, adminPool)
		position, err := repositories.Event.GetHighWater(ctx)
		if err != nil {
			t.Fatalf("GetHighWater(): %v", err)
		}
		created, err := repositories.DailySummary.InsertCheckpoint(ctx, dailysummary.Checkpoint{
			UserID:            ownerID,
			LastCheckInAt:     time.Now().UTC().Add(-25 * time.Hour),
			LastEventPosition: position,
		})
		if err != nil || !created {
			t.Fatalf("InsertCheckpoint() created=%v error=%v", created, err)
		}

		stored, err := repositories.Event.Append(ctx, event.AppendRequest{
			EventID:                  uuid.New(),
			AggregateType:            event.AggregatePet,
			AggregateID:              uuid.New(),
			OwnerUserID:              ownerID,
			ExpectedAggregateVersion: 0,
			EventType:                event.CoinsGranted,
			SchemaVersion:            2,
			Payload:                  []byte(`{"amount":10,"source":"integration"}`),
			Metadata:                 []byte(`{}`),
			CommandID:                uuid.New(),
			CommandEventIndex:        0,
			OccurredAt:               time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("append unsupported event: %v", err)
		}

		service, err := dailysummary.NewService(
			repositories.DailySummary,
			repositories.DailySummary,
			repositories.Event,
			integrationPetProvider{},
			transactionManager,
		)
		if err != nil {
			t.Fatalf("NewService() error = %v", err)
		}
		result, err := service.CheckIn(ctx, ownerID)
		if err != nil {
			t.Fatalf("CheckIn() error = %v", err)
		}
		if !result.ShouldShow || result.Summary == nil || result.Summary.SkippedEventCount != 1 {
			t.Fatalf("CheckIn() result = %+v, want one skipped event", result)
		}

		var reason string
		err = adminPool.QueryRow(
			ctx,
			"SELECT reason FROM daily_summary_event_failures WHERE user_id = $1 AND event_id = $2",
			ownerID,
			stored.ID,
		).Scan(&reason)
		if err != nil {
			t.Fatalf("query quarantined event: %v", err)
		}
		if reason != string(dailysummary.EventFailureUnsupportedSchema) {
			t.Fatalf("quarantine reason = %q, want %q", reason, dailysummary.EventFailureUnsupportedSchema)
		}

		var directCount int
		err = runtimePool.QueryRow(ctx, "SELECT count(*) FROM daily_summary_event_failures").Scan(&directCount)
		if err == nil {
			t.Fatal("app_runtime unexpectedly read daily_summary_event_failures directly")
		}

		otherOwnerID := createIntegrationUser(t, adminPool)
		var mismatchedRecorded bool
		err = runtimePool.QueryRow(
			ctx,
			`SELECT app_api.record_daily_summary_event_failure($1, $2, $3, $4, $5, $6)`,
			otherOwnerID,
			stored.ID,
			stored.GlobalPosition,
			string(stored.Type),
			stored.SchemaVersion,
			string(dailysummary.EventFailureUnsupportedSchema),
		).Scan(&mismatchedRecorded)
		if err != nil {
			t.Fatalf("record mismatched quarantine event: %v", err)
		}
		if mismatchedRecorded {
			t.Fatal("quarantine function accepted an event owned by another user")
		}
	})
}

type integrationPetProvider struct {
	err error
}

func (p integrationPetProvider) GetPeriodState(
	context.Context,
	uuid.UUID,
	time.Time,
	time.Time,
) (dailysummary.PetPeriodState, error) {
	return dailysummary.PetPeriodState{PetID: uuid.New()}, p.err
}

func publishIntegrationXP(
	t *testing.T,
	ctx context.Context,
	service *event.Service,
	ownerID uuid.UUID,
	amount int,
) {
	t.Helper()
	command := integrationCommand(ownerID, uuid.New(), uuid.New(), 0, amount)
	if _, err := service.Publish(ctx, command); err != nil {
		t.Fatalf("publish integration experience: %v", err)
	}
}

func publishIntegrationXPBatch(
	t *testing.T,
	ctx context.Context,
	service *event.Service,
	ownerID uuid.UUID,
	count int,
) {
	t.Helper()
	pending := make([]event.PendingEvent, count)
	for index := range pending {
		pending[index] = event.NewExperienceGranted(event.GrantPayload{
			Amount: 1,
			Source: "daily_summary_pagination_test",
		})
	}
	command := event.Command{
		ID:          uuid.New(),
		OwnerUserID: ownerID,
		Streams: []event.Stream{{
			AggregateType:   event.AggregatePet,
			AggregateID:     uuid.New(),
			ExpectedVersion: 0,
			Events:          pending,
		}},
	}
	if _, err := service.Publish(ctx, command); err != nil {
		t.Fatalf("publish integration experience batch: %v", err)
	}
}

func checkInConcurrently(
	ctx context.Context,
	repositories *postgres.Repository,
	transactionManager dailysummary.Transactor,
	ownerID uuid.UUID,
) ([]dailysummary.CheckInResult, []error) {
	const workers = 2
	start := make(chan struct{})
	results := make([]dailysummary.CheckInResult, workers)
	errs := make([]error, workers)
	var wait sync.WaitGroup

	for index := range workers {
		index := index
		wait.Add(1)
		go func() {
			defer wait.Done()
			service, err := dailysummary.NewService(
				repositories.DailySummary,
				repositories.DailySummary,
				repositories.Event,
				integrationPetProvider{},
				transactionManager,
			)
			if err != nil {
				errs[index] = err
				return
			}
			<-start
			results[index], errs[index] = service.CheckIn(ctx, ownerID)
		}()
	}
	close(start)
	wait.Wait()
	return results, errs
}
