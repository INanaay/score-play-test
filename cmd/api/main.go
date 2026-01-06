package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"score-play/internal/adapters/handlers/http/chi"
	file2 "score-play/internal/adapters/handlers/http/chi/v1/file"
	"score-play/internal/adapters/handlers/http/chi/v1/tag"
	"score-play/internal/adapters/repository/postgres"
	"score-play/internal/adapters/storage/minio"
	"score-play/internal/config"
	"score-play/internal/core/port"
	"score-play/internal/core/service/cleanup"
	"score-play/internal/core/service/file"
	tagservice "score-play/internal/core/service/tag"
	"sync"
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

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := initDB(cfg.Database)
	if err != nil {
		logger.Error("failed to init database", "error", err)
		os.Exit(1)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			logger.Error("failed to close database", "error", err)
			os.Exit(1)
		}
	}(db)
	logger.Info("db connection established")

	//storage
	minioAdapter, err := minio.NewAdapter(ctx, cfg.Minio, logger)
	if err != nil {
		logger.Error("failed to init minio", "error", err)
		os.Exit(1)
	}

	//repositories
	tagRepo := postgres.NewSqlTagRepository(db)
	unitOfWork := postgres.NewUnitOfWork(db)

	tagService := tagservice.NewTagService(tagRepo)
	fileService := file.NewFileService(unitOfWork, minioAdapter, cfg.Upload)
	cleanupService := cleanup.NewCleanupService(unitOfWork, minioAdapter, logger)

	//http
	tagHandler := tag.NewTagHandlerV1(tagService, logger)
	fileHandler := file2.NewFileHandlerV1(fileService, logger)

	router := chi.NewRouter(logger, tagHandler, fileHandler, cfg.Env.Env)
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("starting server", "host", cfg.Server.Host, "port", cfg.Server.Port)
		servErr := server.ListenAndServe()
		if servErr != nil && !errors.Is(servErr, http.ErrServerClosed) {
			logger.Error("failed to start server", "error", servErr)
			stop()
		}
	}()

	// init cleanup task
	wg.Add(1)
	go func() {
		defer wg.Done()
		initCleanupTask(ctx, cleanupService, cfg.Upload.CleanupEvery, logger)
	}()

	//wait for context cancel
	<-ctx.Done()
	logger.Info("gracefully shutting down app")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to shutdown server", "error", err)
	} else {
		logger.Info("server gracefully shutdown complete")
	}

	wg.Wait()
	logger.Info("app shutdown complete")

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
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenCons)
	db.SetMaxIdleConns(cfg.MaxIdleCons)
	db.SetConnMaxLifetime(cfg.ConMaxLifeTime)

	return db, nil
}

func initCleanupTask(ctx context.Context, service port.CleanupService, every time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()

	logger.Info("cleanup task initialized", "interval", every)

	for {
		select {
		case <-ticker.C:
			logger.Info("cleanup task starting")
			err := service.CleanupExpiredSessions(ctx, time.Now().Add(every))
			if err != nil {
				logger.Error("failed to cleanup expired files", "error", err)
			} else {
				logger.Info("cleanup task completed successfully")
			}
		case <-ctx.Done():
			logger.Info("cleanup task stopped")
			return
		}
	}

}
