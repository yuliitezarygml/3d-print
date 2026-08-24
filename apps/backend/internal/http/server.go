package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/local/printforge/apps/backend/internal/config"
)

type Server struct {
	cfg      config.Config
	db       *pgxpool.Pool
	auth     *authService
	catalog  printerCatalog
	telegram *telegramService
	storage  objectStorage
}

type contextKey string

const userContextKey contextKey = "user"

type authUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

func New(cfg config.Config, db *pgxpool.Pool) *Server {
	catalog, err := loadPrinterCatalog()
	if err != nil {
		panic(err)
	}
	store, err := newObjectStorage(cfg)
	if err != nil {
		panic(err)
	}
	server := &Server{cfg: cfg, db: db, auth: newAuthService(cfg, db), catalog: catalog, storage: store}
	server.telegram = newTelegramService(server)
	return server
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(s.securityHeaders)
	r.Use(s.cors)

	r.Get("/health", s.health)
	r.Get("/api/openapi.json", s.openapi)
	r.Get("/api/docs", s.docs)
	r.Post("/api/auth/login", s.login)
	r.Post("/api/auth/refresh", s.refresh)
	r.Get("/api/public/track/{code}", s.publicTrackOrder)
	r.Get("/api/public/track/{code}/receipt.pdf", s.publicOrderReceipt)
	r.Get("/api/public/track/{code}/models/{id}/file", s.publicTrackedModelFile)
	r.Get("/api/public/track/{code}/models/{id}/preview", s.publicTrackedModelPreview)
	r.Get("/api/public/track/{code}/photos/{photoId}", s.publicOrderPhotoFile)
	r.Post("/api/public/requests", s.publicRequest)

	r.Route("/api", func(api chi.Router) {
		api.Use(s.requireAuth)
		api.Get("/me", s.me)
		api.Post("/auth/logout", s.logout)
		api.Get("/dashboard", s.dashboard)
		api.Get("/printers", s.listPrinters)
		api.Get("/printer-catalog", s.listPrinterCatalog)
		api.Post("/printers", s.createPrinter)
		api.Get("/printers/{id}", s.getPrinter)
		api.Patch("/printers/{id}", s.updatePrinter)
		api.Get("/spools", s.listSpools)
		api.Post("/spools", s.createSpool)
		api.Get("/inventory/transactions", s.listInventoryTransactions)
		api.Get("/customers", s.listCustomers)
		api.Post("/customers", s.createCustomer)
		api.Get("/orders", s.listOrders)
		api.Post("/orders", s.createOrder)
		api.Get("/orders/{id}/receipt.pdf", s.orderReceipt)
		api.Patch("/orders/{id}/status", s.updateOrderStatus)
		api.Get("/orders/{id}/events", s.listOrderEvents)
		api.Post("/orders/{id}/events", s.createOrderEvent)
		api.Post("/orders/{id}/photos", s.uploadOrderPhoto)
		api.Get("/orders/{id}/photos/{photoId}", s.orderPhotoFile)
		api.Get("/models", s.listModels)
		api.Post("/models/upload", s.uploadModel)
		api.Get("/models/{id}/file", s.modelFile)
		api.Get("/models/{id}/preview", s.modelPreview)
		api.Post("/models/{id}/preview", s.uploadModelPreview)
		api.Get("/print-jobs", s.listPrintJobs)
		api.Post("/print-jobs", s.createPrintJob)
		api.Patch("/print-jobs/{id}/status", s.updatePrintJobStatus)
		api.Get("/calendar", s.productionCalendar)
		api.Get("/settings", s.getSettings)
		api.Put("/settings", s.updateSettings)
		api.Put("/settings/telegram", s.updateTelegramSettings)
	})
	return r
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		for _, allowed := range s.cfg.AllowedOrigins {
			if strings.TrimSpace(allowed) == origin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
				break
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, apiError{Error: "authentication required"})
			return
		}
		user, err := s.auth.verifyAccessToken(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, apiError{Error: "invalid or expired token"})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey, user)))
	})
}

func currentUser(r *http.Request) authUser {
	user, _ := r.Context().Value(userContextKey).(authUser)
	return user
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unhealthy"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
