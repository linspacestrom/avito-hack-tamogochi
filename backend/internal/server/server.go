package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/config"
)

type Server struct {
	server *http.Server
	cfg    config.HTTPConfig
	log    *slog.Logger
}

func New(
	log *slog.Logger,
	cfg config.HTTPConfig,
	handler http.Handler,
) *Server {

	return &Server{
		server: &http.Server{
			Addr:         cfg.Addr(),
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
			Handler:      handler,
		},
		cfg: cfg,
		log: log,
	}
}

func (s *Server) Run() error {
	s.log.Info("starting HTTP server",
		slog.String("addr", s.server.Addr),
	)

	err := s.server.ListenAndServe()

	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}

func (s *Server) Close(ctx context.Context) error {
	s.log.Info("shutting down HTTP server")

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	return nil
}
