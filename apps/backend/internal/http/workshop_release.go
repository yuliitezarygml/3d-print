package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type orderEvent struct {
	ID        string         `json:"id"`
	EventType string         `json:"eventType"`
	Status    *string        `json:"status"`
	Title     string         `json:"title"`
	Message   string         `json:"message"`
	IsPublic  bool           `json:"isPublic"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"createdAt"`
}

type orderPhoto struct {
	ID               string    `json:"id"`
	Caption          string    `json:"caption"`
	OriginalFilename string    `json:"originalFilename"`
	IsPublic         bool      `json:"isPublic"`
	CreatedAt        time.Time `json:"createdAt"`
	URL              string    `json:"url"`
}

func (s *Server) loadOrderEvents(r *http.Request, orderID string, publicOnly bool) ([]orderEvent, error) {
	rows, err := s.db.Query(r.Context(), `SELECT id,event_type,status::text,title,message,is_public,metadata,created_at FROM order_events WHERE order_id=$1 AND (NOT $2 OR is_public) ORDER BY created_at,id`, orderID, publicOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]orderEvent, 0)
	for rows.Next() {
		var item orderEvent
		var raw []byte
		if err := rows.Scan(&item.ID, &item.EventType, &item.Status, &item.Title, &item.Message, &item.IsPublic, &raw, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Metadata = map[string]any{}
		_ = json.Unmarshal(raw, &item.Metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) listOrderEvents(w http.ResponseWriter, r *http.Request) {
	items, err := s.loadOrderEvents(r, chi.URLParam(r, "id"), false)
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not load order history"})
		return
	}
	writeJSON(w, 200, items)
}

func (s *Server) createOrderEvent(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title    string `json:"title"`
		Message  string `json:"message"`
		IsPublic *bool  `json:"isPublic"`
	}
	if decodeJSON(r, &input) != nil || strings.TrimSpace(input.Title) == "" {
		badRequest(w, "title is required")
		return
	}
	isPublic := true
	if input.IsPublic != nil {
		isPublic = *input.IsPublic
	}
	user := currentUser(r)
	var item orderEvent
	var raw []byte
	err := s.db.QueryRow(r.Context(), `INSERT INTO order_events(order_id,event_type,title,message,is_public,created_by) VALUES($1,'NOTE',$2,$3,$4,$5) RETURNING id,event_type,status::text,title,message,is_public,metadata,created_at`, chi.URLParam(r, "id"), strings.TrimSpace(input.Title), strings.TrimSpace(input.Message), isPublic, user.ID).Scan(&item.ID, &item.EventType, &item.Status, &item.Title, &item.Message, &item.IsPublic, &raw, &item.CreatedAt)
	if err != nil {
		badRequest(w, "could not add order event")
		return
	}
	item.Metadata = map[string]any{}
	writeJSON(w, 201, item)
}

func (s *Server) uploadOrderPhoto(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxImageFileBytes+1024*1024)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		badRequest(w, "photo is too large or invalid")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		badRequest(w, "photo is required")
		return
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".webp" {
		badRequest(w, "photo must be PNG, JPEG or WebP")
		return
	}
	probe := make([]byte, 512)
	n, _ := file.Read(probe)
	if !validImageSignature(ext, probe[:n]) {
		badRequest(w, "invalid photo content")
		return
	}
	_, _ = file.Seek(0, io.SeekStart)
	temp, err := os.CreateTemp("", "printforge-photo-*"+ext)
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not store photo"})
		return
	}
	path := temp.Name()
	defer os.Remove(path)
	written, copyErr := io.Copy(temp, io.LimitReader(file, s.cfg.MaxImageFileBytes+1))
	closeErr := temp.Close()
	if copyErr != nil || closeErr != nil || written > s.cfg.MaxImageFileBytes {
		badRequest(w, "photo exceeds size limit")
		return
	}
	orderID := chi.URLParam(r, "id")
	var exists bool
	if s.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM orders WHERE id=$1)`, orderID).Scan(&exists) != nil || !exists {
		writeJSON(w, 404, apiError{Error: "order not found"})
		return
	}
	key := "orders/" + orderID + "/" + uuid.NewString() + ext
	stored, err := os.Open(path)
	if err != nil || s.storage.Put(r.Context(), key, stored, written, imageMime(ext)) != nil {
		if stored != nil {
			_ = stored.Close()
		}
		writeJSON(w, 500, apiError{Error: "could not store photo"})
		return
	}
	_ = stored.Close()
	caption := strings.TrimSpace(r.FormValue("caption"))
	isPublic := r.FormValue("isPublic") != "false"
	user := currentUser(r)
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		_ = s.storage.Remove(r.Context(), key)
		writeJSON(w, 500, apiError{Error: "could not register photo"})
		return
	}
	defer tx.Rollback(r.Context())
	var eventID, photoID string
	err = tx.QueryRow(r.Context(), `INSERT INTO order_events(order_id,event_type,title,message,is_public,created_by) VALUES($1,'PHOTO','Новая фотография',$2,$3,$4) RETURNING id`, orderID, caption, isPublic, user.ID).Scan(&eventID)
	if err == nil {
		err = tx.QueryRow(r.Context(), `INSERT INTO order_photos(order_id,event_id,storage_path,original_filename,mime_type,file_size_bytes,caption,is_public,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`, orderID, eventID, key, filepath.Base(header.Filename), imageMime(ext), written, caption, isPublic, user.ID).Scan(&photoID)
	}
	if err != nil || tx.Commit(r.Context()) != nil {
		_ = s.storage.Remove(r.Context(), key)
		writeJSON(w, 500, apiError{Error: "could not register photo"})
		return
	}
	writeJSON(w, 201, map[string]any{"id": photoID, "eventId": eventID, "url": "/api/orders/" + orderID + "/photos/" + photoID})
}

func (s *Server) serveOrderPhoto(w http.ResponseWriter, r *http.Request, public bool) {
	orderID := chi.URLParam(r, "id")
	photoID := chi.URLParam(r, "photoId")
	var key, mime string
	query := `SELECT storage_path,mime_type FROM order_photos WHERE id=$1 AND order_id=$2`
	if public {
		query += ` AND is_public`
	}
	if s.db.QueryRow(r.Context(), query, photoID, orderID).Scan(&key, &mime) != nil {
		writeJSON(w, 404, apiError{Error: "photo not found"})
		return
	}
	file, _, _, err := s.storage.Open(r.Context(), key)
	if err != nil {
		writeJSON(w, 404, apiError{Error: "photo not found"})
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = io.Copy(w, file)
}

func (s *Server) orderPhotoFile(w http.ResponseWriter, r *http.Request) {
	s.serveOrderPhoto(w, r, false)
}

func (s *Server) publicOrderPhotoFile(w http.ResponseWriter, r *http.Request) {
	var key, mime string
	err := s.db.QueryRow(r.Context(), `SELECT p.storage_path,p.mime_type FROM order_photos p JOIN orders o ON o.id=p.order_id WHERE p.id=$1 AND p.is_public AND upper(o.tracking_code)=upper($2)`, chi.URLParam(r, "photoId"), chi.URLParam(r, "code")).Scan(&key, &mime)
	if err != nil {
		writeJSON(w, 404, apiError{Error: "photo not found"})
		return
	}
	file, _, _, err := s.storage.Open(r.Context(), key)
	if err != nil {
		writeJSON(w, 404, apiError{Error: "photo not found"})
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = io.Copy(w, file)
}

func (s *Server) publicRequest(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxModelFileBytes+2*1024*1024)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		badRequest(w, "request is too large or invalid")
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	if name == "" || (email == "" && phone == "") {
		badRequest(w, "name and email or phone are required")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		badRequest(w, "3D model is required")
		return
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".stl" && ext != ".obj" && ext != ".3mf" && ext != ".gcode" && ext != ".gco" {
		badRequest(w, "unsupported model format")
		return
	}
	probe := make([]byte, 512)
	n, _ := file.Read(probe)
	if !validModelSignature(ext, probe[:n]) {
		badRequest(w, "invalid model content")
		return
	}
	_, _ = file.Seek(0, io.SeekStart)
	temp, err := os.CreateTemp("", "printforge-request-*"+ext)
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not process model"})
		return
	}
	path := temp.Name()
	defer os.Remove(path)
	written, copyErr := io.Copy(temp, io.LimitReader(file, s.cfg.MaxModelFileBytes+1))
	closeErr := temp.Close()
	if copyErr != nil || closeErr != nil || written > s.cfg.MaxModelFileBytes {
		badRequest(w, "model exceeds size limit")
		return
	}
	key := modelObjectKey(uuid.NewString() + ext)
	stored, err := os.Open(path)
	if err != nil || s.storage.Put(r.Context(), key, stored, written, modelMime(ext)) != nil {
		if stored != nil {
			_ = stored.Close()
		}
		writeJSON(w, 500, apiError{Error: "could not store model"})
		return
	}
	_ = stored.Close()
	trackingCode, err := newTrackingCode()
	if err != nil {
		_ = s.storage.Remove(r.Context(), key)
		writeJSON(w, 500, apiError{Error: "could not create request"})
		return
	}
	quantity, _ := strconv.Atoi(r.FormValue("quantity"))
	if quantity < 1 {
		quantity = 1
	}
	notes := strings.TrimSpace(r.FormValue("notes"))
	material := strings.TrimSpace(r.FormValue("material"))
	color := strings.TrimSpace(r.FormValue("color"))
	minutes, grams, metadata := analyseSlicerEstimate(path, ext)
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		_ = s.storage.Remove(r.Context(), key)
		writeJSON(w, 500, apiError{Error: "could not create request"})
		return
	}
	defer tx.Rollback(r.Context())
	var customerID, modelID, orderID, number string
	err = tx.QueryRow(r.Context(), `INSERT INTO customers(name,email,phone) VALUES($1,NULLIF($2,''),NULLIF($3,'')) RETURNING id`, name, email, phone).Scan(&customerID)
	if err == nil {
		err = tx.QueryRow(r.Context(), `INSERT INTO models(name,original_filename,storage_path,mime_type,file_size_bytes,format,customer_id,estimated_print_minutes,estimated_filament_grams,slicer_metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`, strings.TrimSuffix(filepath.Base(header.Filename), ext), filepath.Base(header.Filename), key, modelMime(ext), written, strings.ToUpper(strings.TrimPrefix(ext, ".")), customerID, minutes, grams, metadata).Scan(&modelID)
	}
	if err == nil {
		err = tx.QueryRow(r.Context(), `INSERT INTO orders(number,tracking_code,customer_id,status,notes,source,requested_material,requested_color,requested_quantity) VALUES('ORD-'||to_char(now(),'YYYY')||'-'||lpad(nextval('order_number_seq')::text,5,'0'),$1,$2,'DRAFT',$3,'PUBLIC',NULLIF($4,''),NULLIF($5,''),$6) RETURNING id,number`, trackingCode, customerID, notes, material, color, quantity).Scan(&orderID, &number)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO order_models(order_id,model_id) VALUES($1,$2)`, orderID, modelID)
	}
	if err != nil || tx.Commit(r.Context()) != nil {
		_ = s.storage.Remove(r.Context(), key)
		writeJSON(w, 500, apiError{Error: "could not create request"})
		return
	}
	writeJSON(w, 201, map[string]any{"orderId": orderID, "number": number, "trackingCode": trackingCode, "status": "DRAFT", "trackingUrl": "/track/" + trackingCode})
}

func (s *Server) productionCalendar(w http.ResponseWriter, r *http.Request) {
	from := time.Now().AddDate(0, 0, -7)
	to := time.Now().AddDate(0, 1, 0)
	if value, err := time.Parse(time.RFC3339, r.URL.Query().Get("from")); err == nil {
		from = value
	}
	if value, err := time.Parse(time.RFC3339, r.URL.Query().Get("to")); err == nil {
		to = value
	}
	rows, err := s.db.Query(r.Context(), `SELECT j.id,j.status,p.name,COALESCE(o.number,''),COALESCE(m.name,''),COALESCE(j.scheduled_start,j.created_at),COALESCE(j.scheduled_end,COALESCE(j.scheduled_start,j.created_at)+(j.estimated_minutes||' minutes')::interval) FROM print_jobs j JOIN printers p ON p.id=j.printer_id LEFT JOIN orders o ON o.id=j.order_id LEFT JOIN models m ON m.id=j.model_id WHERE COALESCE(j.scheduled_start,j.created_at)<$2 AND COALESCE(j.scheduled_end,COALESCE(j.scheduled_start,j.created_at)+(j.estimated_minutes||' minutes')::interval)>$1 ORDER BY 6`, from, to)
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not load calendar"})
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, status, printer, order, model string
		var start, end time.Time
		if rows.Scan(&id, &status, &printer, &order, &model, &start, &end) == nil {
			items = append(items, map[string]any{"id": id, "status": status, "printerName": printer, "orderNumber": order, "modelName": model, "start": start, "end": end})
		}
	}
	writeJSON(w, 200, items)
}
