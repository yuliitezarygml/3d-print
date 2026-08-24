package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type printJobInput struct {
	OrderID                *string    `json:"orderId"`
	ModelID                *string    `json:"modelId"`
	PrinterID              string     `json:"printerId"`
	SpoolID                string     `json:"spoolId"`
	Quantity               int        `json:"quantity"`
	EstimatedMinutes       int        `json:"estimatedMinutes"`
	EstimatedFilamentGrams float64    `json:"estimatedFilamentGrams"`
	LabourHours            float64    `json:"labourHours"`
	PostProcessingCost     float64    `json:"postProcessingCost"`
	PackagingCost          float64    `json:"packagingCost"`
	OtherCost              float64    `json:"otherCost"`
	MarkupPercent          *float64   `json:"markupPercent"`
	Notes                  string     `json:"notes"`
	ScheduledStart         *time.Time `json:"scheduledStart"`
}

type costCalculation struct {
	PowerWatts         float64 `json:"powerWatts"`
	ElectricityRate    float64 `json:"electricityPricePerKwh"`
	EnergyKwh          float64 `json:"energyKwh"`
	MaterialCost       float64 `json:"materialCost"`
	ElectricityCost    float64 `json:"electricityCost"`
	MachineCost        float64 `json:"machineCost"`
	LabourCost         float64 `json:"labourCost"`
	PostProcessingCost float64 `json:"postProcessingCost"`
	PackagingCost      float64 `json:"packagingCost"`
	OtherCost          float64 `json:"otherCost"`
	TotalCost          float64 `json:"totalCost"`
	MarkupPercent      float64 `json:"markupPercent"`
	SuggestedPrice     float64 `json:"suggestedPrice"`
}

func calculateCosts(minutes int, grams, powerWatts, spoolPrice, spoolInitial, electricityRate, machineRate, purchasePrice, depreciationHours, labourHours, labourRate, post, packaging, other, markup float64) costCalculation {
	hours := float64(minutes) / 60
	energy := powerWatts / 1000 * hours
	material := 0.0
	if spoolInitial > 0 {
		material = grams * (spoolPrice / spoolInitial)
	}
	depreciation := 0.0
	if depreciationHours > 0 {
		depreciation = purchasePrice / depreciationHours * hours
	}
	machine := machineRate*hours + depreciation
	labour := labourHours * labourRate
	total := material + energy*electricityRate + machine + labour + post + packaging + other
	return costCalculation{PowerWatts: powerWatts, ElectricityRate: electricityRate, EnergyKwh: energy, MaterialCost: material, ElectricityCost: energy * electricityRate, MachineCost: machine, LabourCost: labour, PostProcessingCost: post, PackagingCost: packaging, OtherCost: other, TotalCost: total, MarkupPercent: markup, SuggestedPrice: total * (1 + markup/100)}
}

func (s *Server) createPrintJob(w http.ResponseWriter, r *http.Request) {
	var in printJobInput
	if decodeJSON(r, &in) != nil || in.PrinterID == "" || in.SpoolID == "" || in.EstimatedMinutes < 0 || in.EstimatedFilamentGrams < 0 || in.LabourHours < 0 || in.Quantity < 0 {
		badRequest(w, "printer, spool, time and filament are required")
		return
	}
	if in.Quantity == 0 {
		in.Quantity = 1
	}
	if in.OrderID != nil && in.ModelID != nil {
		var attached bool
		if err := s.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM order_models WHERE order_id=$1 AND model_id=$2)`, in.OrderID, in.ModelID).Scan(&attached); err != nil || !attached {
			badRequest(w, "selected model is not attached to this order")
			return
		}
	}
	var power, purchase, depreciation, spoolPrice, spoolInitial, electricity, machine, labourRate, defaultMarkup float64
	err := s.db.QueryRow(r.Context(), `SELECT p.power_watts,p.purchase_price,p.depreciation_hours,s.purchase_price,s.initial_weight_grams,c.electricity_price_per_kwh,c.machine_rate_per_hour,c.labour_rate_per_hour,c.default_markup_percent FROM printers p CROSS JOIN filament_spools s CROSS JOIN settings c WHERE p.id=$1 AND s.id=$2 AND c.id=true`, in.PrinterID, in.SpoolID).Scan(&power, &purchase, &depreciation, &spoolPrice, &spoolInitial, &electricity, &machine, &labourRate, &defaultMarkup)
	if err != nil {
		badRequest(w, "printer or spool not found")
		return
	}
	markup := defaultMarkup
	if in.MarkupPercent != nil {
		markup = *in.MarkupPercent
	}
	cost := calculateCosts(in.EstimatedMinutes, in.EstimatedFilamentGrams, power, spoolPrice, spoolInitial, electricity, machine, purchase, depreciation, in.LabourHours, labourRate, in.PostProcessingCost, in.PackagingCost, in.OtherCost, markup)
	var scheduledEnd *time.Time
	if in.ScheduledStart != nil {
		value := in.ScheduledStart.Add(time.Duration(in.EstimatedMinutes) * time.Minute)
		scheduledEnd = &value
	}
	var id string
	err = s.db.QueryRow(r.Context(), `INSERT INTO print_jobs(order_id,model_id,printer_id,spool_id,quantity,estimated_minutes,estimated_filament_grams,power_watts,electricity_price_per_kwh,estimated_energy_kwh,material_cost,electricity_cost,machine_cost,labour_cost,post_processing_cost,packaging_cost,other_cost,total_cost,markup_percent,suggested_price,notes,scheduled_start,scheduled_end)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)RETURNING id`, in.OrderID, in.ModelID, in.PrinterID, in.SpoolID, in.Quantity, in.EstimatedMinutes, in.EstimatedFilamentGrams, cost.PowerWatts, cost.ElectricityRate, cost.EnergyKwh, cost.MaterialCost, cost.ElectricityCost, cost.MachineCost, cost.LabourCost, cost.PostProcessingCost, cost.PackagingCost, cost.OtherCost, cost.TotalCost, cost.MarkupPercent, cost.SuggestedPrice, in.Notes, in.ScheduledStart, scheduledEnd).Scan(&id)
	if err != nil {
		badRequest(w, "could not create print job")
		return
	}
	s.audit(r, "CREATE", "PrintJob", id, nil, in)
	writeJSON(w, 201, map[string]any{"id": id, "status": "QUEUED", "costs": cost})
}

func (s *Server) listPrintJobs(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT j.id,j.status,j.quantity,j.estimated_minutes,j.actual_minutes,j.estimated_filament_grams,j.actual_filament_grams,j.power_watts,j.electricity_price_per_kwh,j.estimated_energy_kwh,j.actual_energy_kwh,j.material_cost,j.electricity_cost,j.machine_cost,j.labour_cost,j.post_processing_cost,j.packaging_cost,j.other_cost,j.total_cost,j.markup_percent,j.suggested_price,j.created_at,p.id,p.name,s.id,s.code,s.material,s.color_name,o.id,o.number,m.id,m.name FROM print_jobs j JOIN printers p ON p.id=j.printer_id JOIN filament_spools s ON s.id=j.spool_id LEFT JOIN orders o ON o.id=j.order_id LEFT JOIN models m ON m.id=j.model_id ORDER BY j.created_at DESC`)
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not load print jobs"})
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, status, pid, pname, sid, scode, mat, color string
		var qty, est int
		var actualMin *int
		var estFil, power, rate, estEnergy, material, electricity, machine, labour, post, pack, other, total, markup, price float64
		var actualFil, actualEnergy *float64
		var created time.Time
		var oid, onum, mid, mname *string
		if rows.Scan(&id, &status, &qty, &est, &actualMin, &estFil, &actualFil, &power, &rate, &estEnergy, &actualEnergy, &material, &electricity, &machine, &labour, &post, &pack, &other, &total, &markup, &price, &created, &pid, &pname, &sid, &scode, &mat, &color, &oid, &onum, &mid, &mname) == nil {
			items = append(items, map[string]any{"id": id, "status": status, "quantity": qty, "estimatedMinutes": est, "actualMinutes": actualMin, "estimatedFilamentGrams": estFil, "actualFilamentGrams": actualFil, "powerWatts": power, "electricityPricePerKwh": rate, "estimatedEnergyKwh": estEnergy, "actualEnergyKwh": actualEnergy, "materialCost": material, "electricityCost": electricity, "machineCost": machine, "labourCost": labour, "postProcessingCost": post, "packagingCost": pack, "otherCost": other, "totalCost": total, "markupPercent": markup, "suggestedPrice": price, "createdAt": created, "printerId": pid, "printerName": pname, "spoolId": sid, "spoolCode": scode, "material": mat, "colorName": color, "orderId": oid, "orderNumber": onum, "modelId": mid, "modelName": mname})
		}
	}
	writeJSON(w, 200, items)
}

func (s *Server) updatePrintJobStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status              string   `json:"status"`
		ActualMinutes       *int     `json:"actualMinutes"`
		ActualFilamentGrams *float64 `json:"actualFilamentGrams"`
		ActualEnergyKwh     *float64 `json:"actualEnergyKwh"`
	}
	if decodeJSON(r, &in) != nil {
		badRequest(w, "invalid status data")
		return
	}
	id := chi.URLParam(r, "id")
	if in.Status == "SUCCESS" {
		if in.ActualMinutes == nil || in.ActualFilamentGrams == nil || *in.ActualMinutes < 0 || *in.ActualFilamentGrams < 0 {
			badRequest(w, "actualMinutes and actualFilamentGrams are required to complete a job")
			return
		}
		result, err := s.completeJob(r.Context(), currentUser(r), id, *in.ActualMinutes, *in.ActualFilamentGrams, in.ActualEnergyKwh)
		if err != nil {
			badRequest(w, err.Error())
			return
		}
		var orderID *string
		_ = s.db.QueryRow(r.Context(), `SELECT order_id FROM print_jobs WHERE id=$1`, id).Scan(&orderID)
		if orderID != nil {
			go s.telegram.notifyOrderStatus(*orderID)
		}
		writeJSON(w, 200, result)
		return
	}
	valid := map[string]bool{"QUEUED": true, "READY": true, "PRINTING": true, "PAUSED": true, "FAILED": true, "CANCELLED": true}
	if !valid[in.Status] {
		badRequest(w, "invalid status")
		return
	}
	command := `UPDATE print_jobs SET status=$2::print_job_status,started_at=CASE WHEN $2::text='PRINTING' AND started_at IS NULL THEN now() ELSE started_at END,updated_at=now() WHERE id=$1 AND status<>'SUCCESS'`
	tag, err := s.db.Exec(r.Context(), command, id, in.Status)
	if err != nil || tag.RowsAffected() == 0 {
		badRequest(w, "job not found or already completed")
		return
	}
	if in.Status == "PRINTING" {
		_, _ = s.db.Exec(r.Context(), `UPDATE printers SET status='PRINTING',updated_at=now() WHERE id=(SELECT printer_id FROM print_jobs WHERE id=$1)`, id)
		_, _ = s.db.Exec(r.Context(), `UPDATE orders SET status='PRINTING',updated_at=now() WHERE id=(SELECT order_id FROM print_jobs WHERE id=$1) AND status NOT IN ('CANCELLED','COMPLETED')`, id)
	}
	s.audit(r, "STATUS_CHANGE", "PrintJob", id, nil, in)
	var orderID *string
	_ = s.db.QueryRow(r.Context(), `SELECT order_id FROM print_jobs WHERE id=$1`, id).Scan(&orderID)
	if orderID != nil && in.Status == "PRINTING" {
		go s.telegram.notifyOrderStatus(*orderID)
	}
	writeJSON(w, 200, map[string]any{"id": id, "status": in.Status})
}

func (s *Server) completeJob(ctx context.Context, user authUser, id string, minutes int, grams float64, energyInput *float64) (map[string]any, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, errors.New("could not start transaction")
	}
	defer tx.Rollback(ctx)
	var status, spoolID, printerID string
	var remaining, power, rate, spoolPrice, spoolInitial, machineRate, purchase, depreciation, labourCost, post, pack, other, markup float64
	err = tx.QueryRow(ctx, `SELECT j.status,j.spool_id,j.printer_id,s.remaining_weight_grams,j.power_watts,j.electricity_price_per_kwh,s.purchase_price,s.initial_weight_grams,c.machine_rate_per_hour,p.purchase_price,p.depreciation_hours,j.labour_cost,j.post_processing_cost,j.packaging_cost,j.other_cost,j.markup_percent FROM print_jobs j JOIN filament_spools s ON s.id=j.spool_id JOIN printers p ON p.id=j.printer_id CROSS JOIN settings c WHERE j.id=$1 FOR UPDATE OF j,s`, id).Scan(&status, &spoolID, &printerID, &remaining, &power, &rate, &spoolPrice, &spoolInitial, &machineRate, &purchase, &depreciation, &labourCost, &post, &pack, &other, &markup)
	if err != nil {
		return nil, errors.New("print job not found")
	}
	if status == "SUCCESS" {
		return nil, errors.New("job already completed")
	}
	if remaining < grams {
		return nil, errors.New("not enough filament on selected spool")
	}
	hours := float64(minutes) / 60
	labourHours := 0.0
	var labourRate float64
	_ = tx.QueryRow(ctx, `SELECT labour_rate_per_hour FROM settings WHERE id=true`).Scan(&labourRate)
	if labourRate > 0 {
		labourHours = labourCost / labourRate
	}
	cost := calculateCosts(minutes, grams, power, spoolPrice, spoolInitial, rate, machineRate, purchase, depreciation, labourHours, labourRate, post, pack, other, markup)
	actualEnergy := cost.EnergyKwh
	if energyInput != nil {
		if *energyInput < 0 {
			return nil, errors.New("actual energy cannot be negative")
		}
		actualEnergy = *energyInput
		cost.EnergyKwh = actualEnergy
		cost.ElectricityCost = actualEnergy * rate
		cost.TotalCost = cost.MaterialCost + cost.ElectricityCost + cost.MachineCost + cost.LabourCost + post + pack + other
		cost.SuggestedPrice = cost.TotalCost * (1 + markup/100)
	}
	_, err = tx.Exec(ctx, `UPDATE print_jobs SET status='SUCCESS',actual_minutes=$2,actual_filament_grams=$3,actual_energy_kwh=$4,material_cost=$5,electricity_cost=$6,machine_cost=$7,total_cost=$8,suggested_price=$9,completed_at=now(),updated_at=now() WHERE id=$1`, id, minutes, grams, actualEnergy, cost.MaterialCost, cost.ElectricityCost, cost.MachineCost, cost.TotalCost, cost.SuggestedPrice)
	if err != nil {
		return nil, errors.New("could not complete job")
	}
	newBalance := remaining - grams
	_, err = tx.Exec(ctx, `UPDATE filament_spools SET remaining_weight_grams=$2,updated_at=now() WHERE id=$1`, spoolID, newBalance)
	if err != nil {
		return nil, errors.New("could not update spool")
	}
	_, err = tx.Exec(ctx, `INSERT INTO inventory_transactions(spool_id,print_job_id,type,quantity_grams,balance_after_grams,reason,created_by)VALUES($1,$2,'PRINT_USAGE',$3,$4,'Completed print job',$5)`, spoolID, id, -grams, newBalance, user.ID)
	if err != nil {
		return nil, errors.New("could not record inventory transaction")
	}
	_, _ = tx.Exec(ctx, `UPDATE printers SET status='IDLE',total_hours=total_hours+$2,updated_at=now() WHERE id=$1`, printerID, hours)
	_, _ = tx.Exec(ctx, `UPDATE orders SET status='POST_PROCESSING',updated_at=now() WHERE id=(SELECT order_id FROM print_jobs WHERE id=$1) AND status<>'CANCELLED'`, id)
	if err = tx.Commit(ctx); err != nil {
		return nil, errors.New("could not commit completion")
	}
	return map[string]any{"id": id, "status": "SUCCESS", "actualFilamentGrams": grams, "remainingSpoolGrams": newBalance, "costs": cost}, nil
}

var _ = pgx.ErrNoRows
