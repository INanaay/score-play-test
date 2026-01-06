package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"score-play/internal/adapters/eventbroker/nats"
	"score-play/internal/adapters/repository/postgres"
	"score-play/internal/adapters/storage/minio"
	"score-play/internal/config"
	"score-play/internal/core/service/file"
	"score-play/internal/core/service/minioevent"
	"syscall"
	"time"
)

func main() {

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Load config
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	// Initialize database
	db, err := initDB(cfg.Database)
	if err != nil {
		logger.Error("failed to init database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("failed to close database", "error", err)
		}
	}()
	logger.Info("db connection established")

	minioAdapter, err := minio.NewAdapter(ctx, cfg.Minio, logger)
	if err != nil {
		logger.Error("failed to init minio", "error", err)
		os.Exit(1)
	}
	logger.Info("minio adapter initialized")

	// Initialize repositories
	unitOfWork := postgres.NewUnitOfWork(db)

	// Initialize services
	fileService := file.NewFileService(unitOfWork, minioAdapter, cfg.Upload)
	minioMessageService := minioevent.NewMinioEventService(minioAdapter, unitOfWork, fileService, logger)

	// Initialize NATS consumer
	natsConsumer, err := nats.NewNATSConsumer(cfg.NATS, logger)
	if err != nil {
		logger.Error("failed to create NATS consumer", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := natsConsumer.Close(); err != nil {
			logger.Error("failed to close NATS consumer", "error", err)
		}
	}()
	logger.Info("NATS consumer initialized")

	// Subscribe to NATS
	if err := natsConsumer.Subscribe(ctx, minioMessageService); err != nil {
		logger.Error("failed to subscribe to NATS", "error", err)
		os.Exit(1)
	}
	logger.Info("NATS subscription active")

	// Wait for termination signal
	<-ctx.Done()
	logger.Info("gracefully shutting down video processing service")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Close NATS consumer
	if err := natsConsumer.Close(); err != nil {
		logger.Error("failed to close NATS consumer during shutdown", "error", err)
	}

	// Wait for shutdown context or completion
	<-shutdownCtx.Done()
	if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
		logger.Info("shutdown timeout exceeded")
	}

	logger.Info("video processing service shutdown complete")
}

func initDB(cfg config.DatabaseConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Name,
		cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenCons)
	db.SetMaxIdleConns(cfg.MaxIdleCons)
	db.SetConnMaxLifetime(cfg.ConMaxLifeTime)

	return db, nil
}
