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
	"github.com/kurt4ins/vk-segmentation/internal/worker"
)

func main() {
	if err := run(); err != nil {
		_, _ = os.Stderr.WriteString("fatal: " + err.Error() + "\n")
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
	userRepo := postgres.NewUserRepo(pool)
	membershipRepo := postgres.NewMembershipRepo(pool)

	rolloutService := service.NewRolloutService(userRepo, membershipRepo, historyRepo, segmentRepo, transactor, cfg.RolloutBatchSize)
	rolloutWorker := worker.NewRolloutWorker(rolloutService, log)

	segmentService := service.NewSegmentService(segmentRepo, historyRepo, transactor, rolloutWorker)
	userService := service.NewUserService(userRepo, segmentRepo, membershipRepo, historyRepo, transactor)
	membershipService := service.NewMembershipService(userRepo, segmentRepo, membershipRepo, historyRepo, transactor)
	historyService := service.NewHistoryService(historyRepo, cfg.ReportsDir)

	ttlCleaner := worker.NewTTLCleaner(membershipService, cfg.TTLCleanInterval, log)

	workerCtx, cancelWorker := context.WithCancel(ctx)
	defer cancelWorker()
	go rolloutWorker.Run(workerCtx)
	go ttlCleaner.Run(workerCtx)

	segmentHandler := handler.NewSegmentHandler(segmentService)
	userHandler := handler.NewUserHandler(userService, membershipService)
	historyHandler := handler.NewHistoryHandler(historyService)

	router := httptransport.NewRouter(httptransport.RouterDeps{
		Logger:     log,
		ReportsDir: cfg.ReportsDir,
		Segment:    segmentHandler,
		User:       userHandler,
		History:    historyHandler,
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

	cancelWorker()
	for _, wk := range []struct {
		name string
		done <-chan struct{}
	}{
		{"rollout worker", rolloutWorker.Done()},
		{"ttl cleaner", ttlCleaner.Done()},
	} {
		select {
		case <-wk.done:
		case <-time.After(15 * time.Second):
			log.Warn("worker did not stop in time", "worker", wk.name)
		}
	}

	log.Info("server stopped")
	return nil
}
