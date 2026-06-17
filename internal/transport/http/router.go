package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kurt4ins/vk-segmentation/internal/transport/http/handler"
	mw "github.com/kurt4ins/vk-segmentation/internal/transport/http/middleware"
)

type RouterDeps struct {
	Logger     *slog.Logger
	ReportsDir string
	Segment    *handler.SegmentHandler
	User       *handler.UserHandler
}

func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(mw.RequestID)
	r.Use(mw.Recoverer(deps.Logger))
	r.Use(mw.Logger(deps.Logger))
	r.Use(mw.Metrics)

	r.Get("/healthz", healthz)
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/api/v1", func(r chi.Router) {
		if deps.Segment != nil {
			deps.Segment.Register(r)
		}
		if deps.User != nil {
			deps.User.Register(r)
		}
	})

	return r
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
