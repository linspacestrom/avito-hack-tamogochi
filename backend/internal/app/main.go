package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/config"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository"
	//"github.com/NBx03/avito-hack-tamagotchi/backend/internal/router"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/server"
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

	repo, _, err := repository.New(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("create repository: %w", err)
	}

	log.Info("database connection established",
		slog.String("host", cfg.Database.Host),
		slog.String("database", cfg.Database.Database),
	)

	//httpRouter := router.New(
	//	log,
	//	cfg,
	//)

	api := server.New(
		log,
		cfg.HTTP,
		//httpRouter,
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

	ctx, cancel := context.WithTimeout(
		context.Background(),
		a.cfg.HTTP.ShutdownTimeout,
	)
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
