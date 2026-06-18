package worker

import (
	"context"
	"log/slog"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

const jobQueueSize = 256

type RolloutApplier interface {
	Apply(ctx context.Context, segment domain.Segment) error
}

type RolloutWorker struct {
	applier RolloutApplier
	log     *slog.Logger
	jobs    chan domain.Segment
	done    chan struct{}
}

func NewRolloutWorker(applier RolloutApplier, log *slog.Logger) *RolloutWorker {
	return &RolloutWorker{
		applier: applier,
		log:     log,
		jobs:    make(chan domain.Segment, jobQueueSize),
		done:    make(chan struct{}),
	}
}

func (w *RolloutWorker) Enqueue(segment domain.Segment) {
	select {
	case w.jobs <- segment:
	case <-w.done:
		w.log.Warn("rollout worker stopped; rollout not enqueued", "segment", segment.Slug)
	}
}

func (w *RolloutWorker) Run(ctx context.Context) {
	defer close(w.done)
	for {
		select {
		case <-ctx.Done():
			return
		case segment := <-w.jobs:
			if err := w.applier.Apply(ctx, segment); err != nil {
				w.log.Error("rollout apply failed", "segment", segment.Slug, "err", err)
				continue
			}
			w.log.Info("rollout applied", "segment", segment.Slug)
		}
	}
}

func (w *RolloutWorker) Done() <-chan struct{} {
	return w.done
}
