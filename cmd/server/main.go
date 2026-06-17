package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kurt4ins/vk-segmentation/internal/config"
	"github.com/kurt4ins/vk-segmentation/internal/pkg/logger"
	"github.com/kurt4ins/vk-segmentation/internal/repository/postgres"
	"github.com/kurt4ins/vk-segmentation/internal/service"
	httptransport "github.com/kurt4ins/vk-segmentation/internal/transport/http"
	"github.com/kurt4ins/vk-segmentation/internal/transport/http/handler"
)

func main() {
	if err := run(); err != nil {
		os.Stderr.WriteString("fatal: " + err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logger.New(cfg.LogLevel)

	if cfg.RunMigrations {
		log.Info("running migrations")
		if err := postgres.Migrate(cfg.DBDSN); err != nil {
			return err
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DBDSN)
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Info("connected to postgres")

	transactor := postgres.NewTransactor(pool)
	segmentRepo := postgres.NewSegmentRepo(pool)
	historyRepo := postgres.NewHistoryRepo(pool)

	segmentService := service.NewSegmentService(segmentRepo, historyRepo, transactor)

	segmentHandler := handler.NewSegmentHandler(segmentService)

	router := httptransport.NewRouter(httptransport.RouterDeps{
		Logger:     log,
		ReportsDir: cfg.ReportsDir,
		Segment:    segmentHandler,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	log.Info("server stopped")
	return nil
}
