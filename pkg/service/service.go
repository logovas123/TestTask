package service

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"SongLibrary/pkg/handlers"
	"SongLibrary/pkg/storage/repository/postgres"
)

type Service struct {
	SongHandler *handlers.SongHandler
	Mux         http.Handler
}

func NewService(logger *slog.Logger) (*Service, error) {
	pool, err := postgres.NewConnPostgres(logger)
	if err != nil {
		logger.Error("error create coonect to db:",
			"error", err,
		)
		return nil, err
	}

	songRepo := postgres.NewSongPostgresRepository(pool)
	logger.Info("new db repository create success")

	songHandler := &handlers.SongHandler{
		SongRepo: songRepo,
		Logger:   logger,
	}
	logger.Info("song handler create success")

	mux := handlers.NewMuxServer(songHandler, logger)
	logger.Info("create new router success")

	return &Service{
		SongHandler: songHandler,
		Mux:         mux,
	}, nil
}

func (s *Service) Start(ctx context.Context, logger *slog.Logger) error {
	host := os.Getenv("SERVER_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	addr := net.JoinHostPort(host, port)
	logger.Debug("get result addr", "addr", addr)

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:         addr,
		Handler:      s.Mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("server start")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("error listen server:", "error", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("error shutdown server:", "error", err)
		return err
	}
	logger.Info("server stopped success")

	s.SongHandler.SongRepo.Close()
	logger.Info("db pool success closed")

	return nil
}
