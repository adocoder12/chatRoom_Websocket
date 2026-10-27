package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *Application) SetupRoutes() http.Handler {
	mux := chi.NewRouter()

	mux.Use(middleware.RequestID)
	mux.Use(middleware.ClientIPFromRemoteAddr)
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)

	// JSON errors instead of chi's plain-text defaults
	mux.NotFound(app.notFound)
	mux.MethodNotAllowed(app.methodNotAllowed)

	// Chat page: index.html embedded in the binary
	mux.Get("/", app.HomeHandler)

	mux.Route("/api/v1", func(api chi.Router) {
		// WebSocket: stays open, so NO timeout here
		api.Get("/ws", app.ChatHandler)

		// Normal request/response routes: timeout is fine
		api.Group(func(r chi.Router) {
			r.Use(middleware.Timeout(8 * time.Second))
			r.Get("/health", app.HealthCheckHandler)
		})
	})

	return mux
}
