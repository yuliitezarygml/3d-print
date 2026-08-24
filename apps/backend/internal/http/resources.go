package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type printer struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Manufacturer      string  `json:"manufacturer"`
	Model             string  `json:"model"`
	SerialNumber      *string `json:"serialNumber"`
	Status            string  `json:"status"`
	BuildXMM          float64 `json:"buildXmm"`
	BuildYMM          float64 `json:"buildYmm"`
	BuildZMM          float64 `json:"buildZmm"`
	NozzleMM          float64 `json:"nozzleMm"`
	PowerWatts        float64 `json:"powerWatts"`
	PurchasePrice     float64 `json:"purchasePrice"`
	DepreciationHours float64 `json:"depreciationHours"`
	TotalHours        float64 `json:"totalHours"`
	Location          *string `json:"location"`
	CatalogKey        *string `json:"catalogKey"`
	ImageURL          *string `json:"imageUrl"`
}

func scanPrinter(scanner interface{ Scan(...any) error }) (printer, error) {
	var p printer
	err := scanner.Scan(&p.ID, &p.Name, &p.Manufacturer, &p.Model, &p.SerialNumber, &p.Status, &p.BuildXMM, &p.BuildYMM, &p.BuildZMM, &p.NozzleMM, &p.PowerWatts, &p.PurchasePrice, &p.DepreciationHours, &p.TotalHours, &p.Location, &p.CatalogKey, &p.ImageURL)
	return p, err
}

const printerColumns = `id, name, manufacturer, model, serial_number, status, build_x_mm, build_y_mm, build_z_mm, nozzle_mm, power_watts, purchase_price, depreciation_hours, total_hours, location, catalog_key, image_url`

func (s *Server) listPrinterCatalog(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.catalog)
}

func (s *Server) listPrinters(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT `+printerColumns+` FROM printers ORDER BY created_at`)
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not load printers"})
		return
	}
	defer rows.Close()
	items := make([]printer, 0)
	for rows.Next() {
		item, err := scanPrinter(rows)
		if err == nil {
			items = append(items, item)
		}
	}
	writeJSON(w, 200, items)
}

func (s *Server) getPrinter(w http.ResponseWriter, r *http.Request) {
	item, err := scanPrinter(s.db.QueryRow(r.Context(), `SELECT `+printerColumns+` FROM printers WHERE id=$1`, chi.URLParam(r, "id")))
	if err != nil {
		writeJSON(w, 404, apiError{Error: "printer not found"})
		return
	}
	writeJSON(w, 200, item)
}

type printerInput struct {
	Name              string  `json:"name"`
	Manufacturer      string  `json:"manufacturer"`
	Model             string  `json:"model"`
	SerialNumber      *string `json:"serialNumber"`
	Status            string  `json:"status"`
	BuildXMM          float64 `json:"buildXmm"`
	BuildYMM          float64 `json:"buildYmm"`
	BuildZMM          float64 `json:"buildZmm"`
	NozzleMM          float64 `json:"nozzleMm"`
	PowerWatts        float64 `json:"powerWatts"`
	PurchasePrice     float64 `json:"purchasePrice"`
	DepreciationHours float64 `json:"depreciationHours"`
	Location          *string `json:"location"`
	CatalogKey        *string `json:"catalogKey"`
}

func (s *Server) createPrinter(w http.ResponseWriter, r *http.Request) {
	var in printerInput
	if decodeJSON(r, &in) != nil || in.PowerWatts < 0 {
		badRequest(w, "name, manufacturer, model and valid powerWatts are required")
		return
	}
	var imageURL *string
	if in.CatalogKey != nil && *in.CatalogKey != "" {
		catalogModel, ok := s.catalog.byKey[*in.CatalogKey]
		if !ok {
			badRequest(w, "printer catalog entry not found")
			return
		}
		in.Manufacturer = catalogModel.Manufacturer
		in.Model = catalogModel.Model
		if strings.TrimSpace(in.Name) == "" {
			in.Name = catalogModel.Model
		}
		if in.BuildXMM == 0 {
			in.BuildXMM = catalogModel.BuildXMM
		}
		if in.BuildYMM == 0 {
			in.BuildYMM = catalogModel.BuildYMM
		}
		if in.BuildZMM == 0 {
			in.BuildZMM = catalogModel.BuildZMM
		}
		if in.NozzleMM == 0 && len(catalogModel.NozzleDiameters) > 0 {
			in.NozzleMM = catalogModel.NozzleDiameters[0]
		}
		imageURL = catalogModel.ImageURL
	}
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Manufacturer) == "" || strings.TrimSpace(in.Model) == "" {
		badRequest(w, "name, manufacturer and model are required")
		return
	}
	if in.Status == "" {
		in.Status = "IDLE"
	}
	if in.NozzleMM == 0 {
		in.NozzleMM = .4
	}
	if in.DepreciationHours == 0 {
		in.DepreciationHours = 5000
	}
	item, err := scanPrinter(s.db.QueryRow(r.Context(), `INSERT INTO printers (name,manufacturer,model,serial_number,status,build_x_mm,build_y_mm,build_z_mm,nozzle_mm,power_watts,purchase_price,depreciation_hours,location,catalog_key,image_url) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING `+printerColumns, in.Name, in.Manufacturer, in.Model, in.SerialNumber, in.Status, in.BuildXMM, in.BuildYMM, in.BuildZMM, in.NozzleMM, in.PowerWatts, in.PurchasePrice, in.DepreciationHours, in.Location, in.CatalogKey, imageURL))
	if err != nil {
		badRequest(w, "could not create printer: check status and serial number")
		return
	}
	s.audit(r, "CREATE", "Printer", item.ID, nil, item)
	writeJSON(w, 201, item)
}

func (s *Server) updatePrinter(w http.ResponseWriter, r *http.Request) {
	var in printerInput
	if decodeJSON(r, &in) != nil {
		badRequest(w, "invalid printer data")
		return
	}
	id := chi.URLParam(r, "id")
	item, err := scanPrinter(s.db.QueryRow(r.Context(), `UPDATE printers SET name=COALESCE(NULLIF($2,''),name), manufacturer=COALESCE(NULLIF($3,''),manufacturer), model=COALESCE(NULLIF($4,''),model), status=COALESCE(NULLIF($5,''),status::text)::printer_status, power_watts=CASE WHEN $6>=0 THEN $6 ELSE power_watts END, location=COALESCE($7,location), updated_at=now() WHERE id=$1 RETURNING `+printerColumns, id, in.Name, in.Manufacturer, in.Model, in.Status, in.PowerWatts, in.Location))
	if err != nil {
		badRequest(w, "could not update printer")
		return
	}
	s.audit(r, "UPDATE", "Printer", id, nil, item)
	writeJSON(w, 200, item)
}

type spool struct {
	ID              string  `json:"id"`
	Code            string  `json:"code"`
	Manufacturer    string  `json:"manufacturer"`
	ProductName     string  `json:"productName"`
	Material        string  `json:"material"`
	ColorName       string  `json:"colorName"`
	ColorHex        string  `json:"colorHex"`
	InitialWeight   float64 `json:"initialWeightGrams"`
	RemainingWeight float64 `json:"remainingWeightGrams"`
	PurchasePrice   float64 `json:"purchasePrice"`
	PricePerGram    float64 `json:"pricePerGram"`
	StockValue      float64 `json:"stockValue"`
	Status          string  `json:"status"`
}

func (s *Server) listSpools(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT id,code,manufacturer,product_name,material,color_name,color_hex,initial_weight_grams,remaining_weight_grams,purchase_price,round(purchase_price/initial_weight_grams,4),round(remaining_weight_grams*(purchase_price/initial_weight_grams),2),status FROM filament_spools ORDER BY created_at DESC`)
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not load spools"})
		return
	}
	defer rows.Close()
	items := make([]spool, 0)
	for rows.Next() {
		var x spool
		if rows.Scan(&x.ID, &x.Code, &x.Manufacturer, &x.ProductName, &x.Material, &x.ColorName, &x.ColorHex, &x.InitialWeight, &x.RemainingWeight, &x.PurchasePrice, &x.PricePerGram, &x.StockValue, &x.Status) == nil {
			items = append(items, x)
		}
	}
	writeJSON(w, 200, items)
}

func (s *Server) createSpool(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code          string  `json:"code"`
		Manufacturer  string  `json:"manufacturer"`
		ProductName   string  `json:"productName"`
		Material      string  `json:"material"`
		ColorName     string  `json:"colorName"`
		ColorHex      string  `json:"colorHex"`
		InitialWeight float64 `json:"initialWeightGrams"`
		PurchasePrice float64 `json:"purchasePrice"`
		Supplier      string  `json:"supplier"`
	}
	if decodeJSON(r, &in) != nil || in.Code == "" || in.Manufacturer == "" || in.Material == "" || in.InitialWeight <= 0 || in.PurchasePrice < 0 {
		badRequest(w, "code, manufacturer, material, weight and price are required")
		return
	}
	if in.ColorHex == "" {
		in.ColorHex = "#808080"
	}
	if in.ProductName == "" {
		in.ProductName = in.Material
	}
	if in.ColorName == "" {
		in.ColorName = "Unknown"
	}
	var x spool
	err := s.db.QueryRow(r.Context(), `INSERT INTO filament_spools(code,manufacturer,product_name,material,color_name,color_hex,initial_weight_grams,remaining_weight_grams,purchase_price,supplier)VALUES($1,$2,$3,$4,$5,$6,$7,$7,$8,$9) RETURNING id,code,manufacturer,product_name,material,color_name,color_hex,initial_weight_grams,remaining_weight_grams,purchase_price,round(purchase_price/initial_weight_grams,4),round(remaining_weight_grams*(purchase_price/initial_weight_grams),2),status`, in.Code, in.Manufacturer, in.ProductName, in.Material, in.ColorName, in.ColorHex, in.InitialWeight, in.PurchasePrice, in.Supplier).Scan(&x.ID, &x.Code, &x.Manufacturer, &x.ProductName, &x.Material, &x.ColorName, &x.ColorHex, &x.InitialWeight, &x.RemainingWeight, &x.PurchasePrice, &x.PricePerGram, &x.StockValue, &x.Status)
	if err != nil {
		badRequest(w, "could not create spool; code must be unique")
		return
	}
	user := currentUser(r)
	_, _ = s.db.Exec(r.Context(), `INSERT INTO inventory_transactions(spool_id,type,quantity_grams,balance_after_grams,reason,created_by)VALUES($1,'PURCHASE',$2,$2,'Initial purchase',$3)`, x.ID, x.InitialWeight, user.ID)
	s.audit(r, "CREATE", "FilamentSpool", x.ID, nil, x)
	writeJSON(w, 201, x)
}

func (s *Server) listInventoryTransactions(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT t.id,t.type,t.quantity_grams,t.balance_after_grams,COALESCE(t.reason,''),t.created_at,s.code,s.material,s.color_name FROM inventory_transactions t JOIN filament_spools s ON s.id=t.spool_id ORDER BY t.created_at DESC LIMIT 200`)
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not load transactions"})
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, typ, reason, code, material, color string
		var qty, balance float64
		var at time.Time
		if rows.Scan(&id, &typ, &qty, &balance, &reason, &at, &code, &material, &color) == nil {
			items = append(items, map[string]any{"id": id, "type": typ, "quantityGrams": qty, "balanceAfterGrams": balance, "reason": reason, "createdAt": at, "spoolCode": code, "material": material, "colorName": color})
		}
	}
	writeJSON(w, 200, items)
}

type customer struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Company     *string `json:"company"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
	OrderCount  int     `json:"orderCount"`
	TotalAmount float64 `json:"totalAmount"`
	ModelCount  int     `json:"modelCount"`
}

func (s *Server) listCustomers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT c.id,c.name,c.company,c.phone,c.email,(SELECT count(*) FROM orders o WHERE o.customer_id=c.id),(SELECT COALESCE(sum(o.selling_price),0) FROM orders o WHERE o.customer_id=c.id),(SELECT count(*) FROM models m WHERE m.customer_id=c.id) FROM customers c ORDER BY c.name`)
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not load customers"})
		return
	}
	defer rows.Close()
	items := make([]customer, 0)
	for rows.Next() {
		var x customer
		if rows.Scan(&x.ID, &x.Name, &x.Company, &x.Phone, &x.Email, &x.OrderCount, &x.TotalAmount, &x.ModelCount) == nil {
			items = append(items, x)
		}
	}
	writeJSON(w, 200, items)
}
func (s *Server) createCustomer(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name    string  `json:"name"`
		Company *string `json:"company"`
		Phone   *string `json:"phone"`
		Email   *string `json:"email"`
	}
	if decodeJSON(r, &in) != nil || strings.TrimSpace(in.Name) == "" {
		badRequest(w, "customer name is required")
		return
	}
	var x customer
	err := s.db.QueryRow(r.Context(), `INSERT INTO customers(name,company,phone,email)VALUES($1,$2,$3,$4)RETURNING id,name,company,phone,email,0,0,0`, in.Name, in.Company, in.Phone, in.Email).Scan(&x.ID, &x.Name, &x.Company, &x.Phone, &x.Email, &x.OrderCount, &x.TotalAmount, &x.ModelCount)
	if err != nil {
		badRequest(w, "could not create customer")
		return
	}
	s.audit(r, "CREATE", "Customer", x.ID, nil, x)
	writeJSON(w, 201, x)
}

func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT o.id,o.number,o.tracking_code,o.status,o.selling_price,o.paid_amount,o.deadline,o.created_at,c.id,c.name,COALESCE(json_agg(json_build_object('id',m.id,'name',m.name,'originalFilename',m.original_filename,'previewUrl',CASE WHEN m.preview_path IS NOT NULL THEN '/api/models/'||m.id||'/preview' END)) FILTER (WHERE m.id IS NOT NULL),'[]'::json) FROM orders o LEFT JOIN customers c ON c.id=o.customer_id LEFT JOIN order_models om ON om.order_id=o.id LEFT JOIN models m ON m.id=om.model_id GROUP BY o.id,c.id ORDER BY o.created_at DESC`)
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not load orders"})
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, num, trackingCode, status string
		var price, paid float64
		var deadline *time.Time
		var created time.Time
		var cid, cname *string
		var modelsJSON []byte
		if rows.Scan(&id, &num, &trackingCode, &status, &price, &paid, &deadline, &created, &cid, &cname, &modelsJSON) == nil {
			items = append(items, map[string]any{"id": id, "number": num, "trackingCode": trackingCode, "status": status, "statusLabel": orderStatusLabels[status], "sellingPrice": price, "paidAmount": paid, "balanceDue": price - paid, "deadline": deadline, "createdAt": created, "customerId": cid, "customerName": cname, "models": decodeModelIDs(modelsJSON)})
		}
	}
	writeJSON(w, 200, items)
}
func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CustomerID   *string    `json:"customerId"`
		Deadline     *time.Time `json:"deadline"`
		SellingPrice float64    `json:"sellingPrice"`
		PaidAmount   float64    `json:"paidAmount"`
		Notes        string     `json:"notes"`
		ModelIDs     []string   `json:"modelIds"`
	}
	if decodeJSON(r, &in) != nil || in.SellingPrice < 0 || in.PaidAmount < 0 {
		badRequest(w, "invalid order data")
		return
	}
	trackingCode, err := newTrackingCode()
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not generate tracking code"})
		return
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not create order"})
		return
	}
	defer tx.Rollback(r.Context())
	var id, number string
	err = tx.QueryRow(r.Context(), `INSERT INTO orders(number,tracking_code,customer_id,deadline,selling_price,paid_amount,notes)VALUES('ORD-'||to_char(now(),'YYYY')||'-'||lpad(nextval('order_number_seq')::text,5,'0'),$1,$2,$3,$4,$5,$6)RETURNING id,number`, trackingCode, in.CustomerID, in.Deadline, in.SellingPrice, in.PaidAmount, in.Notes).Scan(&id, &number)
	if err != nil {
		badRequest(w, "could not create order")
		return
	}
	for _, modelID := range in.ModelIDs {
		var valid bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM models WHERE id=$1 AND (($2::uuid IS NULL AND customer_id IS NULL) OR ($2::uuid IS NOT NULL AND (customer_id=$2 OR customer_id IS NULL))))`, modelID, in.CustomerID).Scan(&valid); err != nil || !valid {
			badRequest(w, "selected model does not belong to this customer")
			return
		}
		if _, err := tx.Exec(r.Context(), `INSERT INTO order_models(order_id,model_id)VALUES($1,$2)`, id, modelID); err != nil {
			badRequest(w, "could not attach model to order")
			return
		}
		if in.CustomerID != nil {
			_, _ = tx.Exec(r.Context(), `UPDATE models SET customer_id=$2 WHERE id=$1 AND customer_id IS NULL`, modelID, in.CustomerID)
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, 500, apiError{Error: "could not create order"})
		return
	}
	s.audit(r, "CREATE", "Order", id, nil, in)
	writeJSON(w, 201, map[string]any{"id": id, "number": number, "trackingCode": trackingCode, "status": "NEW"})
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	var company, currency, publicBaseURL string
	var telegramUsername *string
	var telegramConfigured, telegramEnabled bool
	var electricity, machine, labour, markup, threshold float64
	err := s.db.QueryRow(r.Context(), `SELECT company_name,currency,electricity_price_per_kwh,machine_rate_per_hour,labour_rate_per_hour,default_markup_percent,low_stock_threshold_grams,public_base_url,telegram_bot_token IS NOT NULL,telegram_bot_username,telegram_bot_enabled FROM settings WHERE id=true`).Scan(&company, &currency, &electricity, &machine, &labour, &markup, &threshold, &publicBaseURL, &telegramConfigured, &telegramUsername, &telegramEnabled)
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not load settings"})
		return
	}
	writeJSON(w, 200, map[string]any{"companyName": company, "currency": currency, "electricityPricePerKwh": electricity, "machineRatePerHour": machine, "labourRatePerHour": labour, "defaultMarkupPercent": markup, "lowStockThresholdGrams": threshold, "publicBaseUrl": publicBaseURL, "telegramBotConfigured": telegramConfigured, "telegramBotUsername": telegramUsername, "telegramBotEnabled": telegramEnabled})
}
func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	if currentUser(r).Role != "ADMIN" {
		writeJSON(w, 403, apiError{Error: "administrator access required"})
		return
	}
	var in struct {
		CompanyName string  `json:"companyName"`
		Currency    string  `json:"currency"`
		Electricity float64 `json:"electricityPricePerKwh"`
		Machine     float64 `json:"machineRatePerHour"`
		Labour      float64 `json:"labourRatePerHour"`
		Markup      float64 `json:"defaultMarkupPercent"`
		Threshold   float64 `json:"lowStockThresholdGrams"`
	}
	if decodeJSON(r, &in) != nil || in.CompanyName == "" || len(in.Currency) != 3 || in.Electricity < 0 || in.Machine < 0 || in.Labour < 0 || in.Markup < 0 || in.Threshold < 0 {
		badRequest(w, "invalid settings")
		return
	}
	_, err := s.db.Exec(r.Context(), `UPDATE settings SET company_name=$1,currency=upper($2),electricity_price_per_kwh=$3,machine_rate_per_hour=$4,labour_rate_per_hour=$5,default_markup_percent=$6,low_stock_threshold_grams=$7,updated_at=now()WHERE id=true`, in.CompanyName, in.Currency, in.Electricity, in.Machine, in.Labour, in.Markup, in.Threshold)
	if err != nil {
		badRequest(w, "could not update settings")
		return
	}
	s.audit(r, "UPDATE", "Settings", uuid.Nil.String(), nil, in)
	s.getSettings(w, r)
}

func (s *Server) audit(r *http.Request, action, entityType, entityID string, oldValue, newValue any) {
	id, err := uuid.Parse(entityID)
	if err != nil {
		id = uuid.Nil
	}
	user := currentUser(r)
	_, _ = s.db.Exec(r.Context(), `INSERT INTO audit_logs(user_id,action,entity_type,entity_id,old_values,new_values)VALUES($1,$2,$3,$4,$5,$6)`, user.ID, action, entityType, id, oldValue, newValue)
}
