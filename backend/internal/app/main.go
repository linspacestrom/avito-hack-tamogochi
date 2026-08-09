package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/config"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/handler"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository/postgres"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/router"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/server"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/service"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/tasks"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/token"
	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
)

type Repository interface {
	Close()
	Ping(ctx context.Context) error
}

type App struct {
	log    *slog.Logger
	cfg    *config.Config
	server *server.Server
	repo   Repository
}

func New(
	log *slog.Logger,
	cfg *config.Config,
) (*App, error) {

	pool, err := postgres.NewPool(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	repo := postgres.New(pool)

	tokenManager, err := token.New(
		cfg.Auth.Secret,
		cfg.Auth.AccessTokenTTL,
		cfg.Auth.RefreshTokenTTL,
	)
	if err != nil {
		repo.Close()
		return nil, fmt.Errorf("create token manager: %w", err)
	}

	transactionManager := manager.Must(trmpgx.NewDefaultFactory(pool))
	authService := service.NewAuthService(
		repo.User,
		repo.Session,
		transactionManager,
		tokenManager,
	)
	rewardsService := rewards.NewService(repo.Rewards)
	dailyCycleService := rewards.NewDailyCycleService(repo.Rewards, transactionManager)
	tasksService := tasks.NewService(repo.Tasks, repo.Rewards, transactionManager)
	services := service.New(authService, rewardsService, dailyCycleService, tasksService)
	httpHandler := handler.New(log, services.Auth, services.Rewards, services.DailyCycle, services.Tasks)
	httpRouter := router.New(httpHandler, authService, log)

	log.Info("database connection established",
		slog.String("host", cfg.Database.Host),
		slog.String("database", cfg.Database.Database),
	)

	api := server.New(
		log,
		cfg.HTTP,
		httpRouter,
	)

	return &App{
		log:    log,
		cfg:    cfg,
		server: api,
		repo:   repo,
	}, nil
}

func (a *App) Run() error {
	a.log.Info("application started")

	return a.server.Run()
}

func (a *App) Stop() {
	a.log.Info("stopping application")

	ctx, cancel := context.WithTimeout(context.Background(), a.cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := a.server.Close(ctx); err != nil {
		a.log.Error("failed to shutdown http server",
			slog.Any("error", err),
		)
	}

	a.log.Info("closing database connection")

	a.repo.Close()

	a.log.Info("application stopped")
}
