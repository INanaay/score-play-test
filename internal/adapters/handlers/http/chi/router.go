package chi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"score-play/internal/adapters/handlers/http/chi/v1/file"
	"score-play/internal/adapters/handlers/http/chi/v1/tag"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// NewRouter builds http.Handler with chi
func NewRouter(logger *slog.Logger, tagHandler *tag.HandlerV1, fileHandler *file.HandlerV1, env string) http.Handler {
	r := chi.NewRouter()

	//handle requestID to facilitate debug (X-Request-ID)
	//It fetches from request if exists, or creates it
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(LoggerMiddleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.RequestSize(5 << 20)) //5mb

	if env != "prod" {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"http://localhost:*", "http://127.0.0.1:*"}, // Ajustez selon vos besoins
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	r.Route("/api/v1", func(r chi.Router) {
		r.Mount("/tag", tagHandler.Routes())
		r.Mount("/file", fileHandler.Routes())
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		resp := HealthResponse{
			Status:    "ok",
			Timestamp: time.Now(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	return r
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}
