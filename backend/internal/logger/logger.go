package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/config"
)

func MustLoad(cfg config.LogConfig) *slog.Logger {
	log, err := load(cfg)
	if err != nil {
		panic(fmt.Errorf("create logger: %w", err))
	}

	slog.SetDefault(log)

	return log
}

func load(cfg config.LogConfig) (*slog.Logger, error) {
	opts := &slog.HandlerOptions{
		Level:     parseLevel(cfg.Level),
		AddSource: true,
	}

	var handler slog.Handler

	switch strings.ToLower(cfg.Format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)

	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)

	default:
		return nil, fmt.Errorf("unsupported log format %q", cfg.Format)
	}

	return slog.New(handler), nil
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug

	case "info":
		return slog.LevelInfo

	case "warn", "warning":
		return slog.LevelWarn

	case "error":
		return slog.LevelError

	default:
		return slog.LevelInfo
	}
}
