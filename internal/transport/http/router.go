package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kurt4ins/vk-segmentation/api"
	"github.com/kurt4ins/vk-segmentation/internal/transport/http/handler"
	mw "github.com/kurt4ins/vk-segmentation/internal/transport/http/middleware"
)

type RouterDeps struct {
	Logger     *slog.Logger
	ReportsDir string
	Segment    *handler.SegmentHandler
	User       *handler.UserHandler
	History    *handler.HistoryHandler
}

func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(mw.RequestID)
	r.Use(mw.Recoverer(deps.Logger))
	r.Use(mw.Logger(deps.Logger))
	r.Use(mw.Metrics)

	r.Get("/healthz", healthz)
	r.Handle("/metrics", promhttp.Handler())

	r.Get("/swagger", swaggerUI)
	r.Get("/swagger/openapi.yaml", swaggerSpec)

	if deps.ReportsDir != "" {
		fs := http.FileServer(http.Dir(deps.ReportsDir))
		r.Handle(handler.ReportsURLPrefix+"*", http.StripPrefix(handler.ReportsURLPrefix, fs))
	}

	r.Route("/api/v1", func(r chi.Router) {
		if deps.Segment != nil {
			deps.Segment.Register(r)
		}
		if deps.User != nil {
			deps.User.Register(r)
		}
		if deps.History != nil {
			deps.History.Register(r)
		}
	})

	return r
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func swaggerSpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(api.Spec)
}

func swaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerHTML))
}

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>VK Segmentation API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: "/swagger/openapi.yaml",
        dom_id: "#swagger-ui",
      });
    };
  </script>
</body>
</html>`
