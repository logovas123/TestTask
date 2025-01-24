package main

import (
	"context"
	"log/slog"
	"os"

	"SongLibrary/pkg/service"

	"github.com/joho/godotenv"
)

const (
	envDebug = "debug"
	envProd  = "prod"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		slog.Error("error load env file",
			"ERROR", err,
		)
		os.Exit(1)
	}
	slog.Info("parse .env file success")

	levelLog := os.Getenv("LOG_LEVEL")
	if levelLog != "info" && levelLog != "debug" {
		levelLog = "info"
	}

	logger := setupLogger(levelLog)
	logger.Info("setup slog level", "level", levelLog)
	logger.Debug("debug messages are enabled")

	s, err := service.NewService(logger)
	if err != nil {
		logger.Error("error create service:",
			"ERROR", err,
		)
		os.Exit(1)
	}
	logger.Info("service create success")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err = s.Start(ctx, logger); err != nil {
		logger.Error("error start service:",
			"ERROR", err,
		)
		os.Exit(1)
	}

	logger.Info("service gracefull shutdown")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envDebug:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}
