package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/app"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/config"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/logger"
)

func main() {
	cfg := config.MustLoad()

	log := logger.MustLoad(cfg.Log)

	application, err := app.New(log, cfg)
	if err != nil {
		log.Error("failed to initialize application",
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	go func() {
		if err := application.Run(); err != nil {
			log.Error("application stopped",
				slog.Any("error", err),
			)
			os.Exit(1)
		}
	}()

	shutdown := make(chan os.Signal, 1)

	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	sig := <-shutdown

	log.Info("received shutdown signal", slog.String("signal", sig.String()))

	application.Stop()
}
