//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/event"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/leaderboard"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository/postgres"
	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const leaderboardProjectionName = "leaderboard_v1"

func TestLeaderboardPostgresIntegration(t *testing.T) {
	adminDSN := os.Getenv("TEST_DATABASE_URL")
	runtimeDSN := os.Getenv("TEST_RUNTIME_DATABASE_URL")
	projectorDSN := os.Getenv("TEST_PROJECTOR_DATABASE_URL")
	if adminDSN == "" || runtimeDSN == "" || projectorDSN == "" {
		t.Skip("test database URLs not set, skipping integration test")
	}

	ctx := context.Background()
	adminPool := openIntegrationPool(t, ctx, adminDSN, "administrator")
	runtimePool := openIntegrationPool(t, ctx, runtimeDSN, "app_runtime")
	projectorPool := openIntegrationPool(t, ctx, projectorDSN, "app_projector")

	runtimeRepositories := postgres.New(runtimePool)
	projectorRepositories := postgres.New(projectorPool)
	runtimeTransactions := manager.Must(trmpgx.NewDefaultFactory(runtimePool))
	projectorTransactions := manager.Must(trmpgx.NewDefaultFactory(projectorPool))
	eventService := event.NewService(runtimeRepositories.Event, runtimeTransactions)
	readService, err := leaderboard.NewService(runtimeRepositories.Leaderboard)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	resetLeaderboardProjection(t, ctx, adminPool)
	t.Cleanup(func() {
		_, _ = adminPool.Exec(context.Background(), "SELECT app_api.reset_leaderboard_projection()")
	})
	initialPosition := globalPosition(t, ctx, adminPool)
	if err = projectorRepositories.Event.SaveProjectionCheckpoint(
		ctx,
		leaderboardProjectionName,
		initialPosition,
	); err != nil {
		t.Fatalf("initialize leaderboard checkpoint: %v", err)
	}

	earlyUserID := createLeaderboardUser(t, adminPool, "Early Player")
	laterUserID := createLeaderboardUser(t, adminPool, "Later Player")
	blockedUserID := createLeaderboardUser(t, adminPool, "Blocked Player")
	earlyPetID := publishLeaderboardPet(t, ctx, eventService, earlyUserID, "Pixel", "cat", 5)
	earlyBestSessionID := publishLeaderboardGame(t, ctx, eventService, earlyUserID, 100)
	laterPetID := publishLeaderboardPet(t, ctx, eventService, laterUserID, "Bolt", "dog", 5)
	publishLeaderboardGame(t, ctx, eventService, laterUserID, 100)
	publishLeaderboardPet(t, ctx, eventService, blockedUserID, "Ghost", "fox", 99)
	publishLeaderboardGame(t, ctx, eventService, blockedUserID, 999)
	publishLeaderboardGame(t, ctx, eventService, earlyUserID, 10)
	if _, err = adminPool.Exec(
		ctx,
		"UPDATE users SET status = 'blocked' WHERE id = $1",
		blockedUserID,
	); err != nil {
		t.Fatalf("block leaderboard user: %v", err)
	}

	t.Run("concurrent projectors serialize one batch", func(t *testing.T) {
		projectors := make([]*leaderboard.Projector, 2)
		for index := range projectors {
			projectors[index], err = leaderboard.NewProjector(
				projectorRepositories.Event,
				projectorRepositories.Leaderboard,
				projectorTransactions,
			)
			if err != nil {
				t.Fatalf("NewProjector() error = %v", err)
			}
		}

		start := make(chan struct{})
		processed := make([]int, len(projectors))
		errs := make([]error, len(projectors))
		var wait sync.WaitGroup
		for index := range projectors {
			index := index
			wait.Add(1)
			go func() {
				defer wait.Done()
				<-start
				processed[index], errs[index] = projectors[index].ProjectNextBatch(ctx)
			}()
		}
		close(start)
		wait.Wait()

		for _, projectErr := range errs {
			if projectErr != nil {
				t.Fatalf("concurrent ProjectNextBatch() error = %v", projectErr)
			}
		}
		if !((processed[0] > 0 && processed[1] == 0) ||
			(processed[1] > 0 && processed[0] == 0)) {
			t.Fatalf("concurrent processed = %v, want one non-zero batch", processed)
		}
	})

	t.Run("rankings use achievement order and exclude blocked users", func(t *testing.T) {
		petEntries, listErr := readService.ListPetLevels(ctx, leaderboard.Page{Limit: 10})
		if listErr != nil {
			t.Fatalf("ListPetLevels() error = %v", listErr)
		}
		if len(petEntries) != 2 || petEntries[0].UserID != earlyUserID ||
			petEntries[0].PetID != earlyPetID || petEntries[0].Rank != 1 ||
			petEntries[1].UserID != laterUserID || petEntries[1].PetID != laterPetID ||
			petEntries[1].Rank != 2 {
			t.Fatalf("pet leaderboard = %+v", petEntries)
		}

		gameEntries, listErr := readService.ListGameScores(ctx, leaderboard.Page{Limit: 10})
		if listErr != nil {
			t.Fatalf("ListGameScores() error = %v", listErr)
		}
		if len(gameEntries) != 2 || gameEntries[0].UserID != earlyUserID ||
			gameEntries[0].BestScore != 100 || gameEntries[0].Rank != 1 ||
			gameEntries[1].UserID != laterUserID || gameEntries[1].Rank != 2 {
			t.Fatalf("game leaderboard = %+v", gameEntries)
		}

		var storedSessionID uuid.UUID
		if queryErr := adminPool.QueryRow(
			ctx,
			"SELECT session_id FROM leaderboard_game_scores WHERE user_id = $1",
			earlyUserID,
		).Scan(&storedSessionID); queryErr != nil {
			t.Fatalf("read stored best session: %v", queryErr)
		}
		if storedSessionID != earlyBestSessionID {
			t.Fatalf("stored session = %s, want %s", storedSessionID, earlyBestSessionID)
		}

		page, listErr := readService.ListPetLevels(ctx, leaderboard.Page{Limit: 1, Offset: 1})
		if listErr != nil || len(page) != 1 || page[0].UserID != laterUserID || page[0].Rank != 2 {
			t.Fatalf("second pet page = %+v error=%v", page, listErr)
		}
	})

	t.Run("user positions are consistent across both views", func(t *testing.T) {
		positions, positionsErr := readService.GetUserPositions(ctx, earlyUserID)
		if positionsErr != nil {
			t.Fatalf("GetUserPositions() error = %v", positionsErr)
		}
		if positions.PetLevelRank == nil || *positions.PetLevelRank != 1 ||
			positions.GameScoreRank == nil || *positions.GameScoreRank != 1 {
			t.Fatalf("early user positions = %+v", positions)
		}
		blockedPositions, positionsErr := readService.GetUserPositions(ctx, blockedUserID)
		if positionsErr != nil || blockedPositions.PetLevelRank != nil ||
			blockedPositions.GameScoreRank != nil {
			t.Fatalf("blocked user positions = %+v error=%v", blockedPositions, positionsErr)
		}
	})

	t.Run("runtime role cannot mutate projection tables", func(t *testing.T) {
		_, permissionErr := runtimePool.Exec(
			ctx,
			`UPDATE leaderboard_pet_levels SET level = 999 WHERE user_id = $1`,
			earlyUserID,
		)
		assertPermissionDenied(t, permissionErr)
		var count int
		permissionErr = runtimePool.QueryRow(
			ctx,
			"SELECT count(*) FROM leaderboard_event_failures",
		).Scan(&count)
		assertPermissionDenied(t, permissionErr)
		_, permissionErr = runtimePool.Exec(ctx, "SELECT app_api.reset_leaderboard_projection()")
		assertPermissionDenied(t, permissionErr)
	})

	t.Run("unsupported event is quarantined and checkpoint advances", func(t *testing.T) {
		before, checkpointErr := projectorRepositories.Event.GetProjectionCheckpoint(
			ctx,
			leaderboardProjectionName,
		)
		if checkpointErr != nil {
			t.Fatalf("GetProjectionCheckpoint() error = %v", checkpointErr)
		}
		sessionID := uuid.New()
		stored, appendErr := runtimeRepositories.Event.Append(ctx, event.AppendRequest{
			EventID:                  uuid.New(),
			AggregateType:            event.AggregateGameSession,
			AggregateID:              sessionID,
			OwnerUserID:              earlyUserID,
			ExpectedAggregateVersion: 0,
			EventType:                event.GameSessionCompleted,
			SchemaVersion:            2,
			Payload:                  []byte(`{"sessionId":"` + sessionID.String() + `","score":200}`),
			Metadata:                 []byte(`{}`),
			CommandID:                uuid.New(),
			CommandEventIndex:        0,
			OccurredAt:               time.Now().UTC(),
		})
		if appendErr != nil {
			t.Fatalf("append unsupported event: %v", appendErr)
		}
		projector, projectorErr := leaderboard.NewProjector(
			projectorRepositories.Event,
			projectorRepositories.Leaderboard,
			projectorTransactions,
		)
		if projectorErr != nil {
			t.Fatalf("NewProjector() error = %v", projectorErr)
		}
		processed, projectorErr := projector.ProjectNextBatch(ctx)
		if projectorErr != nil || processed != 1 {
			t.Fatalf("ProjectNextBatch() processed=%d error=%v", processed, projectorErr)
		}
		after, checkpointErr := projectorRepositories.Event.GetProjectionCheckpoint(
			ctx,
			leaderboardProjectionName,
		)
		if checkpointErr != nil || after <= before {
			t.Fatalf("checkpoint before=%d after=%d error=%v", before, after, checkpointErr)
		}
		var reason string
		if queryErr := adminPool.QueryRow(
			ctx,
			"SELECT reason FROM leaderboard_event_failures WHERE event_id = $1",
			stored.ID,
		).Scan(&reason); queryErr != nil {
			t.Fatalf("read quarantined event: %v", queryErr)
		}
		if reason != string(leaderboard.FailureUnsupportedSchema) {
			t.Fatalf("quarantine reason = %q", reason)
		}
	})

	t.Run("writer failure rolls back read model and checkpoint", func(t *testing.T) {
		userID := createLeaderboardUser(t, adminPool, "Rollback Player")
		petID := publishLeaderboardPet(t, ctx, eventService, userID, "Undo", "owl", 4)
		checkpointBefore, checkpointErr := projectorRepositories.Event.GetProjectionCheckpoint(
			ctx,
			leaderboardProjectionName,
		)
		if checkpointErr != nil {
			t.Fatalf("GetProjectionCheckpoint() error = %v", checkpointErr)
		}
		wantErr := errors.New("forced level write failure")
		projector, projectorErr := leaderboard.NewProjector(
			projectorRepositories.Event,
			failingLevelWriter{
				ProjectionWriter: projectorRepositories.Leaderboard,
				err:              wantErr,
			},
			projectorTransactions,
		)
		if projectorErr != nil {
			t.Fatalf("NewProjector() error = %v", projectorErr)
		}
		if _, projectorErr = projector.ProjectNextBatch(ctx); !errors.Is(projectorErr, wantErr) {
			t.Fatalf("ProjectNextBatch() error = %v, want %v", projectorErr, wantErr)
		}
		checkpointAfter, checkpointErr := projectorRepositories.Event.GetProjectionCheckpoint(
			ctx,
			leaderboardProjectionName,
		)
		if checkpointErr != nil || checkpointAfter != checkpointBefore {
			t.Fatalf("checkpoint before=%d after=%d error=%v", checkpointBefore, checkpointAfter, checkpointErr)
		}
		var count int
		if queryErr := adminPool.QueryRow(
			ctx,
			"SELECT count(*) FROM leaderboard_pet_levels WHERE user_id = $1 AND pet_id = $2",
			userID,
			petID,
		).Scan(&count); queryErr != nil || count != 0 {
			t.Fatalf("rolled back pet count=%d error=%v", count, queryErr)
		}

		recoveryProjector, recoveryErr := leaderboard.NewProjector(
			projectorRepositories.Event,
			projectorRepositories.Leaderboard,
			projectorTransactions,
		)
		if recoveryErr != nil {
			t.Fatalf("NewProjector() recovery error = %v", recoveryErr)
		}
		if processed, recoveryErr := recoveryProjector.ProjectNextBatch(ctx); recoveryErr != nil || processed != 2 {
			t.Fatalf("recovery ProjectNextBatch() processed=%d error=%v", processed, recoveryErr)
		}
	})
}

func resetLeaderboardProjection(
	t *testing.T,
	ctx context.Context,
	adminPool *pgxpool.Pool,
) {
	t.Helper()
	if _, err := adminPool.Exec(ctx, "SELECT app_api.reset_leaderboard_projection()"); err != nil {
		t.Fatalf("reset leaderboard projection: %v", err)
	}
}

func openIntegrationPool(
	t *testing.T,
	ctx context.Context,
	dsn string,
	role string,
) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect as %s: %v", role, err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func createLeaderboardUser(t *testing.T, pool *pgxpool.Pool, displayName string) uuid.UUID {
	t.Helper()
	userID := createIntegrationUser(t, pool)
	if _, err := pool.Exec(
		context.Background(),
		"UPDATE users SET display_name = $2 WHERE id = $1",
		userID,
		displayName,
	); err != nil {
		t.Fatalf("set leaderboard display name: %v", err)
	}
	return userID
}

func publishLeaderboardPet(
	t *testing.T,
	ctx context.Context,
	service *event.Service,
	userID uuid.UUID,
	name string,
	species string,
	level int,
) uuid.UUID {
	t.Helper()
	petID := uuid.New()
	pending := []event.PendingEvent{
		event.NewPetCreated(event.PetCreatedPayload{Name: name, Species: species}),
	}
	if level > 1 {
		pending = append(pending, event.NewLevelReached(event.LevelReachedPayload{Level: level}))
	}
	_, err := service.Publish(ctx, event.Command{
		ID:          uuid.New(),
		OwnerUserID: userID,
		Streams: []event.Stream{{
			AggregateType:   event.AggregatePet,
			AggregateID:     petID,
			ExpectedVersion: 0,
			Events:          pending,
		}},
	})
	if err != nil {
		t.Fatalf("publish leaderboard pet: %v", err)
	}
	return petID
}

func publishLeaderboardGame(
	t *testing.T,
	ctx context.Context,
	service *event.Service,
	userID uuid.UUID,
	score int,
) uuid.UUID {
	t.Helper()
	sessionID := uuid.New()
	_, err := service.Publish(ctx, event.Command{
		ID:          uuid.New(),
		OwnerUserID: userID,
		Streams: []event.Stream{{
			AggregateType:   event.AggregateGameSession,
			AggregateID:     sessionID,
			ExpectedVersion: 0,
			Events: []event.PendingEvent{
				event.NewGameSessionCompleted(event.GameSessionCompletedPayload{
					SessionID: sessionID,
					Score:     score,
				}),
			},
		}},
	})
	if err != nil {
		t.Fatalf("publish leaderboard game: %v", err)
	}
	return sessionID
}

func assertPermissionDenied(t *testing.T, err error) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Fatalf("error = %v, want PostgreSQL permission denied", err)
	}
}

type failingLevelWriter struct {
	leaderboard.ProjectionWriter
	err error
}

func (w failingLevelWriter) AdvancePetLevel(
	context.Context,
	leaderboard.PetProjection,
) (bool, error) {
	return false, w.err
}
