package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const trackingAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

func newTrackingCode() (string, error) {
	random := make([]byte, 10)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	for index := range random {
		random[index] = trackingAlphabet[int(random[index])%len(trackingAlphabet)]
	}
	return string(random), nil
}

var orderStatusLabels = map[string]string{
	"DRAFT": "Черновик", "NEW": "Заказ принят", "CONFIRMED": "Подтверждён",
	"WAITING": "Ожидает материалов", "READY_TO_PRINT": "Готов к печати",
	"PRINTING": "Печатается", "POST_PROCESSING": "Постобработка",
	"READY": "Готов к выдаче", "COMPLETED": "Выдан клиенту", "CANCELLED": "Отменён",
}

var orderStatusSequence = []string{"NEW", "CONFIRMED", "READY_TO_PRINT", "PRINTING", "POST_PROCESSING", "READY", "COMPLETED"}

type trackedModel struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	OriginalFilename string  `json:"originalFilename"`
	Format           string  `json:"format"`
	FileSizeBytes    int64   `json:"fileSizeBytes"`
	PreviewPath      *string `json:"-"`
	PreviewURL       *string `json:"previewUrl"`
	DownloadURL      string  `json:"downloadUrl"`
}

type trackedOrder struct {
	Number        string         `json:"number"`
	TrackingCode  string         `json:"trackingCode"`
	Status        string         `json:"status"`
	StatusLabel   string         `json:"statusLabel"`
	SellingPrice  float64        `json:"sellingPrice"`
	PaidAmount    float64        `json:"paidAmount"`
	BalanceDue    float64        `json:"balanceDue"`
	Currency      string         `json:"currency"`
	CustomerName  *string        `json:"customerName"`
	Deadline      *time.Time     `json:"deadline"`
	CreatedAt     time.Time      `json:"createdAt"`
	Notes         string         `json:"notes"`
	Progress      int            `json:"progress"`
	Models        []trackedModel `json:"models"`
	CompanyName   string         `json:"-"`
	PublicBaseURL string         `json:"-"`
}

func (s *Server) loadTrackedOrder(ctx context.Context, code string) (trackedOrder, error) {
	var order trackedOrder
	err := s.db.QueryRow(ctx, `SELECT o.number,o.tracking_code,o.status,o.selling_price,o.paid_amount,o.selling_price-o.paid_amount,s.currency,c.name,o.deadline,o.created_at,COALESCE(o.notes,''),s.company_name,s.public_base_url FROM orders o LEFT JOIN customers c ON c.id=o.customer_id CROSS JOIN settings s WHERE upper(o.tracking_code)=upper($1)`, strings.TrimSpace(code)).Scan(&order.Number, &order.TrackingCode, &order.Status, &order.SellingPrice, &order.PaidAmount, &order.BalanceDue, &order.Currency, &order.CustomerName, &order.Deadline, &order.CreatedAt, &order.Notes, &order.CompanyName, &order.PublicBaseURL)
	if err != nil {
		return trackedOrder{}, errors.New("order not found")
	}
	order.StatusLabel = orderStatusLabels[order.Status]
	order.Progress = orderProgress(order.Status)
	rows, err := s.db.Query(ctx, `SELECT DISTINCT m.id,m.name,m.original_filename,m.format,m.file_size_bytes,m.preview_path FROM models m WHERE m.id IN (SELECT model_id FROM order_models WHERE order_id=(SELECT id FROM orders WHERE tracking_code=$1) UNION SELECT model_id FROM print_jobs WHERE order_id=(SELECT id FROM orders WHERE tracking_code=$1) AND model_id IS NOT NULL) ORDER BY m.name`, order.TrackingCode)
	if err != nil {
		return trackedOrder{}, err
	}
	defer rows.Close()
	order.Models = make([]trackedModel, 0)
	for rows.Next() {
		var model trackedModel
		if err := rows.Scan(&model.ID, &model.Name, &model.OriginalFilename, &model.Format, &model.FileSizeBytes, &model.PreviewPath); err != nil {
			return trackedOrder{}, err
		}
		base := "/api/public/track/" + order.TrackingCode + "/models/" + model.ID
		model.DownloadURL = base + "/file"
		if model.PreviewPath != nil {
			preview := base + "/preview"
			model.PreviewURL = &preview
		}
		order.Models = append(order.Models, model)
	}
	return order, rows.Err()
}

func orderProgress(status string) int {
	if status == "CANCELLED" {
		return 0
	}
	for index, candidate := range orderStatusSequence {
		if status == candidate {
			return int(float64(index) / float64(len(orderStatusSequence)-1) * 100)
		}
	}
	if status == "DRAFT" {
		return 0
	}
	if status == "WAITING" {
		return 25
	}
	return 0
}

func (s *Server) publicTrackOrder(w http.ResponseWriter, r *http.Request) {
	order, err := s.loadTrackedOrder(r.Context(), chi.URLParam(r, "code"))
	if err != nil {
		writeJSON(w, 404, apiError{Error: "заказ с таким кодом не найден"})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, order)
}

func (s *Server) publicTrackedModelFile(w http.ResponseWriter, r *http.Request) {
	if !s.trackedModelAllowed(r, chi.URLParam(r, "code"), chi.URLParam(r, "id")) {
		writeJSON(w, 404, apiError{Error: "model not found"})
		return
	}
	s.serveModelFile(w, r, chi.URLParam(r, "id"))
}

func (s *Server) publicTrackedModelPreview(w http.ResponseWriter, r *http.Request) {
	if !s.trackedModelAllowed(r, chi.URLParam(r, "code"), chi.URLParam(r, "id")) {
		writeJSON(w, 404, apiError{Error: "model preview not found"})
		return
	}
	s.serveModelPreview(w, r, chi.URLParam(r, "id"))
}

func (s *Server) trackedModelAllowed(r *http.Request, code, modelID string) bool {
	var allowed bool
	_ = s.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM orders o WHERE upper(o.tracking_code)=upper($1) AND (EXISTS(SELECT 1 FROM order_models om WHERE om.order_id=o.id AND om.model_id=$2) OR EXISTS(SELECT 1 FROM print_jobs j WHERE j.order_id=o.id AND j.model_id=$2)))`, code, modelID).Scan(&allowed)
	return allowed
}

func (s *Server) updateOrderStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Status string `json:"status"`
	}
	if decodeJSON(r, &input) != nil || orderStatusLabels[input.Status] == "" {
		badRequest(w, "invalid order status")
		return
	}
	id := chi.URLParam(r, "id")
	tag, err := s.db.Exec(r.Context(), `UPDATE orders SET status=$2::order_status,updated_at=now() WHERE id=$1`, id, input.Status)
	if err != nil || tag.RowsAffected() == 0 {
		writeJSON(w, 404, apiError{Error: "order not found"})
		return
	}
	s.audit(r, "STATUS_CHANGE", "Order", id, nil, input)
	writeJSON(w, 200, map[string]any{"id": id, "status": input.Status, "statusLabel": orderStatusLabels[input.Status]})
}

func decodeModelIDs(data []byte) []map[string]any {
	models := make([]map[string]any, 0)
	_ = json.Unmarshal(data, &models)
	return models
}

func trackingMessage(order trackedOrder) string {
	deadline := "не указан"
	if order.Deadline != nil {
		deadline = order.Deadline.Format("02.01.2006 15:04")
	}
	return fmt.Sprintf("<b>Заказ %s</b>\n\nСтатус: <b>%s</b>\nГотовность: %d%%\nСтоимость: %.2f %s\nОплачено: %.2f %s\nОстаток: %.2f %s\nСрок: %s", order.Number, order.StatusLabel, order.Progress, order.SellingPrice, order.Currency, order.PaidAmount, order.Currency, order.BalanceDue, order.Currency, deadline)
}
