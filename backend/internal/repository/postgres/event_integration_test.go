//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/event"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository/postgres"
	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const postgresInsufficientPrivilegeCode = "42501"

// Run with administrative and app_runtime connections to a throwaway, fully migrated database:
// go test -tags=integration ./internal/repository/postgres/...
func TestEventServicePostgresIntegration(t *testing.T) {
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

	t.Run("runtime cannot read event store directly", func(t *testing.T) {
		var count int
		err := runtimePool.QueryRow(ctx, "SELECT count(*) FROM event_store").Scan(&count)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != postgresInsufficientPrivilegeCode {
			t.Fatalf("direct event_store read error = %v, want insufficient_privilege", err)
		}
	})

	t.Run("concurrent identical command", func(t *testing.T) {
		ownerID := createIntegrationUser(t, adminPool)
		command := integrationCommand(ownerID, uuid.New(), uuid.New(), 0, 10)

		results, errs := publishConcurrently(ctx, events, command, command)
		assertOneError(t, errs, nil, nil)
		if results[0][0].ID != results[1][0].ID {
			t.Fatalf("concurrent replay returned different events: %s and %s", results[0][0].ID, results[1][0].ID)
		}
	})

	t.Run("conflicting command reuse", func(t *testing.T) {
		ownerID := createIntegrationUser(t, adminPool)
		commandID := uuid.New()
		petID := uuid.New()
		first := integrationCommand(ownerID, commandID, petID, 0, 10)
		second := integrationCommand(ownerID, commandID, petID, 0, 11)

		_, errs := publishConcurrently(ctx, events, first, second)
		assertOneError(t, errs, nil, event.ErrIdempotencyConflict)
	})

	t.Run("same command id for different owners", func(t *testing.T) {
		commandID := uuid.New()
		first := integrationCommand(createIntegrationUser(t, adminPool), commandID, uuid.New(), 0, 10)
		second := integrationCommand(createIntegrationUser(t, adminPool), commandID, uuid.New(), 0, 10)

		_, errs := publishConcurrently(ctx, events, first, second)
		assertOneError(t, errs, nil, nil)
	})

	t.Run("concurrent stream version conflict", func(t *testing.T) {
		ownerID := createIntegrationUser(t, adminPool)
		petID := uuid.New()
		first := integrationCommand(ownerID, uuid.New(), petID, 0, 10)
		second := integrationCommand(ownerID, uuid.New(), petID, 0, 20)

		_, errs := publishConcurrently(ctx, events, first, second)
		assertOneError(t, errs, nil, event.ErrVersionConflict)
	})

	t.Run("later append failure rolls back package", func(t *testing.T) {
		ownerID := createIntegrationUser(t, adminPool)
		existingPetID := uuid.New()
		if _, err := events.Publish(
			ctx,
			integrationCommand(ownerID, uuid.New(), existingPetID, 0, 10),
		); err != nil {
			t.Fatalf("seed aggregate: %v", err)
		}

		newPetID := uuid.New()
		commandID := uuid.New()
		command := commandWithTwoStreams(ownerID, commandID, newPetID, existingPetID)
		_, err := events.Publish(ctx, command)
		if !errors.Is(err, event.ErrVersionConflict) {
			t.Fatalf("Publish() error = %v, want ErrVersionConflict", err)
		}

		version, err := events.GetAggregateVersion(ctx, ownerID, event.AggregatePet, newPetID)
		if err != nil {
			t.Fatalf("GetAggregateVersion() error = %v", err)
		}
		if version != 0 {
			t.Fatalf("rolled back stream version = %d, want 0", version)
		}
		stored, err := events.ListByCommandID(ctx, ownerID, commandID)
		if err != nil {
			t.Fatalf("ListByCommandID() error = %v", err)
		}
		if len(stored) != 0 {
			t.Fatalf("rolled back command contains %d events", len(stored))
		}
	})

	t.Run("external rollback removes command and positions", func(t *testing.T) {
		ownerID := createIntegrationUser(t, adminPool)
		petID := uuid.New()
		command := integrationCommand(ownerID, uuid.New(), petID, 0, 10)
		before := globalPosition(t, ctx, adminPool)
		rollback := errors.New("force outer rollback")

		err := transactionManager.Do(ctx, func(txCtx context.Context) error {
			first, publishErr := events.Publish(txCtx, command)
			if publishErr != nil {
				return publishErr
			}
			second, publishErr := events.Publish(txCtx, command)
			if publishErr != nil {
				return publishErr
			}
			if first[0].ID != second[0].ID {
				return errors.New("repeat inside transaction returned a different event")
			}
			return rollback
		})
		if !errors.Is(err, rollback) {
			t.Fatalf("outer transaction error = %v, want rollback marker", err)
		}

		stored, err := events.ListByCommandID(ctx, ownerID, command.ID)
		if err != nil {
			t.Fatalf("ListByCommandID() error = %v", err)
		}
		if len(stored) != 0 {
			t.Fatalf("rolled back outer transaction contains %d events", len(stored))
		}
		if after := globalPosition(t, ctx, adminPool); after != before {
			t.Fatalf("global position after rollback = %d, want %d", after, before)
		}
	})
}

func integrationCommand(
	ownerID uuid.UUID,
	commandID uuid.UUID,
	petID uuid.UUID,
	expectedVersion int64,
	amount int,
) event.Command {
	return event.Command{
		ID:          commandID,
		OwnerUserID: ownerID,
		Streams: []event.Stream{{
			AggregateType:   event.AggregatePet,
			AggregateID:     petID,
			ExpectedVersion: expectedVersion,
			Events: []event.PendingEvent{
				event.NewExperienceGranted(event.GrantPayload{Amount: amount, Source: "integration_test"}),
			},
		}},
	}
}

func commandWithTwoStreams(
	ownerID uuid.UUID,
	commandID uuid.UUID,
	newPetID uuid.UUID,
	existingPetID uuid.UUID,
) event.Command {
	return event.Command{
		ID:          commandID,
		OwnerUserID: ownerID,
		Streams: []event.Stream{
			{
				AggregateType:   event.AggregatePet,
				AggregateID:     newPetID,
				ExpectedVersion: 0,
				Events: []event.PendingEvent{
					event.NewExperienceGranted(event.GrantPayload{Amount: 1, Source: "integration_test"}),
				},
			},
			{
				AggregateType:   event.AggregatePet,
				AggregateID:     existingPetID,
				ExpectedVersion: 0,
				Events: []event.PendingEvent{
					event.NewExperienceGranted(event.GrantPayload{Amount: 1, Source: "integration_test"}),
				},
			},
		},
	}
}

func publishConcurrently(
	ctx context.Context,
	service *event.Service,
	commands ...event.Command,
) ([][]event.Event, []error) {
	start := make(chan struct{})
	results := make([][]event.Event, len(commands))
	errs := make([]error, len(commands))
	var wait sync.WaitGroup

	for index := range commands {
		index := index
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			results[index], errs[index] = service.Publish(ctx, commands[index])
		}()
	}
	close(start)
	wait.Wait()
	return results, errs
}

func assertOneError(t *testing.T, actual []error, first, second error) {
	t.Helper()
	want := []error{first, second}
	matched := make([]bool, len(want))
	for _, actualErr := range actual {
		found := false
		for index, wantErr := range want {
			if matched[index] {
				continue
			}
			if (wantErr == nil && actualErr == nil) ||
				(wantErr != nil && errors.Is(actualErr, wantErr)) {
				matched[index] = true
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("unexpected concurrent result error: %v", actualErr)
		}
	}
}

func createIntegrationUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(
		context.Background(),
		`INSERT INTO users (id, email, display_name, password_hash)
		 VALUES ($1, $2, 'Event Test', 'unused')`,
		id,
		fmt.Sprintf("event-%s@example.test", id),
	)
	if err != nil {
		t.Fatalf("create integration user: %v", err)
	}
	return id
}

func globalPosition(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int64 {
	t.Helper()
	var position int64
	if err := pool.QueryRow(
		ctx,
		"SELECT last_position FROM event_store_position WHERE singleton = TRUE",
	).Scan(&position); err != nil {
		t.Fatalf("read global position: %v", err)
	}
	return position
}
