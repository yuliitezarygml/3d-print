package httpapi

import "net/http"

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	var activeOrders, queued, printing, idle, maintenance, spoolCount int
	var filamentGrams, stockValue, revenue, cost, profit, electricity float64
	err := s.db.QueryRow(r.Context(), `SELECT
	(SELECT count(*) FROM orders WHERE status NOT IN ('COMPLETED','CANCELLED')),
	(SELECT count(*) FROM print_jobs WHERE status IN ('QUEUED','READY')),
	(SELECT count(*) FROM printers WHERE status='PRINTING'),
	(SELECT count(*) FROM printers WHERE status='IDLE'),
	(SELECT count(*) FROM printers WHERE status='MAINTENANCE'),
	(SELECT count(*) FROM filament_spools WHERE status='ACTIVE'),
	(SELECT COALESCE(sum(remaining_weight_grams),0) FROM filament_spools WHERE status='ACTIVE'),
	(SELECT COALESCE(sum(remaining_weight_grams*(purchase_price/initial_weight_grams)),0) FROM filament_spools WHERE status='ACTIVE'),
	(SELECT COALESCE(sum(selling_price),0) FROM orders WHERE created_at>=date_trunc('month',now()) AND status<>'CANCELLED'),
	(SELECT COALESCE(sum(total_cost),0) FROM print_jobs WHERE created_at>=date_trunc('month',now()) AND status='SUCCESS'),
	(SELECT COALESCE(sum(electricity_cost),0) FROM print_jobs WHERE created_at>=date_trunc('month',now()) AND status='SUCCESS')`).Scan(&activeOrders, &queued, &printing, &idle, &maintenance, &spoolCount, &filamentGrams, &stockValue, &revenue, &cost, &electricity)
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not load dashboard"})
		return
	}
	profit = revenue - cost
	rows, _ := s.db.Query(r.Context(), `SELECT p.id,p.name,p.manufacturer,p.status,j.id,o.number,COALESCE(m.name,''),j.started_at,j.estimated_minutes FROM printers p LEFT JOIN LATERAL(SELECT * FROM print_jobs WHERE printer_id=p.id AND status='PRINTING' ORDER BY started_at DESC LIMIT 1)j ON true LEFT JOIN orders o ON o.id=j.order_id LEFT JOIN models m ON m.id=j.model_id ORDER BY p.name`)
	printers := make([]map[string]any, 0)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id, name, maker, status string
			var jid, order, model *string
			var started any
			var minutes *int
			if rows.Scan(&id, &name, &maker, &status, &jid, &order, &model, &started, &minutes) == nil {
				printers = append(printers, map[string]any{"id": id, "name": name, "manufacturer": maker, "status": status, "jobId": jid, "orderNumber": order, "modelName": model, "startedAt": started, "estimatedMinutes": minutes})
			}
		}
	}
	writeJSON(w, 200, map[string]any{"activeOrders": activeOrders, "queuedJobs": queued, "printingPrinters": printing, "idlePrinters": idle, "maintenancePrinters": maintenance, "spoolCount": spoolCount, "filamentGrams": filamentGrams, "stockValue": stockValue, "monthlyRevenue": revenue, "monthlyProductionCost": cost, "monthlyProfit": profit, "monthlyElectricityCost": electricity, "printers": printers})
}
