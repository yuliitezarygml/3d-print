package handler

import (
	"context"
	"net/http"
	"sync"

	"github.com/local/printforge/apps/backend/internal/config"
	"github.com/local/printforge/apps/backend/internal/database"
	httpapi "github.com/local/printforge/apps/backend/internal/http"
)

var (
	initialize sync.Once
	apiHandler http.Handler
	startupErr error
)

// Handler is the Vercel Go Runtime entrypoint. The database pool and router are
// reused while the serverless instance remains warm.
func Handler(w http.ResponseWriter, r *http.Request) {
	initialize.Do(func() {
		cfg, err := config.Load()
		if err != nil {
			startupErr = err
			return
		}
		pool, err := database.Open(context.Background(), cfg.DatabaseURL, cfg.DatabaseMaxConns)
		if err != nil {
			startupErr = err
			return
		}
		apiHandler = httpapi.New(cfg, pool).Routes()
	})
	if startupErr != nil {
		writeStartupError(w)
		return
	}
	apiHandler.ServeHTTP(w, r)
}

func writeStartupError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`{"error":"service configuration is unavailable"}`))
}
