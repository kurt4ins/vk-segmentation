package worker

import (
	"context"
	"log/slog"
	"time"
)

type ExpiredCleaner interface {
	CleanExpired(ctx context.Context) (int, error)
}

type TTLCleaner struct {
	cleaner  ExpiredCleaner
	interval time.Duration
	log      *slog.Logger
	done     chan struct{}
}

func NewTTLCleaner(cleaner ExpiredCleaner, interval time.Duration, log *slog.Logger) *TTLCleaner {
	return &TTLCleaner{
		cleaner:  cleaner,
		interval: interval,
		log:      log,
		done:     make(chan struct{}),
	}
}

func (w *TTLCleaner) Run(ctx context.Context) {
	defer close(w.done)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			removed, err := w.cleaner.CleanExpired(ctx)
			if err != nil {
				w.log.Error("ttl cleanup failed", "err", err)
				continue
			}
			if removed > 0 {
				w.log.Info("ttl cleanup removed expired memberships", "count", removed)
			}
		}
	}
}

func (w *TTLCleaner) Done() <-chan struct{} {
	return w.done
}
