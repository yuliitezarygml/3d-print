package httpapi

import "net/http"

func (s *Server) openapi(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"openapi": "3.0.3",
		"info":    map[string]any{"title": "PrintForge API", "version": "0.1.0", "description": "3D-print workshop management API"},
		"servers": []map[string]string{{"url": "/"}},
		"paths": map[string]any{
			"/api/auth/login":                      map[string]any{"post": map[string]any{"summary": "Sign in"}},
			"/api/dashboard":                       map[string]any{"get": map[string]any{"summary": "Workshop dashboard"}},
			"/api/printers":                        map[string]any{"get": map[string]any{"summary": "List printers"}, "post": map[string]any{"summary": "Create printer"}},
			"/api/printer-catalog":                 map[string]any{"get": map[string]any{"summary": "List OrcaSlicer and Bambu Studio printer profiles"}},
			"/api/spools":                          map[string]any{"get": map[string]any{"summary": "List filament spools"}, "post": map[string]any{"summary": "Create spool"}},
			"/api/orders":                          map[string]any{"get": map[string]any{"summary": "List orders"}, "post": map[string]any{"summary": "Create order"}},
			"/api/orders/{id}/receipt.pdf":         map[string]any{"get": map[string]any{"summary": "Download authenticated order receipt PDF"}},
			"/api/models/upload":                   map[string]any{"post": map[string]any{"summary": "Upload STL, OBJ, 3MF or G-code model"}},
			"/api/public/requests":                 map[string]any{"post": map[string]any{"summary": "Create public print request and tracking code"}},
			"/api/public/track/{code}":             map[string]any{"get": map[string]any{"summary": "Public order tracking by secure code"}},
			"/api/public/track/{code}/receipt.pdf": map[string]any{"get": map[string]any{"summary": "Download public order receipt PDF by secure code"}},
			"/api/orders/{id}/events":              map[string]any{"get": map[string]any{"summary": "List order history"}, "post": map[string]any{"summary": "Publish order event"}},
			"/api/orders/{id}/photos":              map[string]any{"post": map[string]any{"summary": "Upload order progress photo"}},
			"/api/calendar":                        map[string]any{"get": map[string]any{"summary": "Production schedule"}},
			"/api/settings/telegram":               map[string]any{"put": map[string]any{"summary": "Configure encrypted Telegram bot token"}},
			"/api/print-jobs":                      map[string]any{"get": map[string]any{"summary": "List print jobs"}, "post": map[string]any{"summary": "Create job with cost calculation"}},
		},
		"components": map[string]any{"securitySchemes": map[string]any{"bearerAuth": map[string]string{"type": "http", "scheme": "bearer", "bearerFormat": "JWT"}}},
	})
}

func (s *Server) docs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html><head><title>PrintForge API</title><link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"></head><body><div id="swagger-ui"></div><script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script><script>SwaggerUIBundle({url:'/api/openapi.json',dom_id:'#swagger-ui',persistAuthorization:true})</script></body></html>`))
}
