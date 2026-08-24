package httpapi

import (
	_ "embed"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/signintech/gopdf"
	qrcode "github.com/skip2/go-qrcode"
)

// Noto Sans is distributed under the SIL Open Font License 1.1.
// See assets/OFL.txt.
//
//go:embed assets/NotoSans-Regular.ttf
var receiptRegularFont []byte

//go:embed assets/NotoSans-Bold.ttf
var receiptBoldFont []byte

const (
	receiptPageWidth = 595.28
	receiptMargin    = 44.0
)

type receiptColor struct{ r, g, b uint8 }

var (
	receiptInk        = receiptColor{20, 25, 22}
	receiptMuted      = receiptColor{101, 113, 105}
	receiptPurple     = receiptColor{137, 112, 255}
	receiptPurpleInk  = receiptColor{91, 61, 224}
	receiptPurpleSoft = receiptColor{239, 235, 255}
	receiptPanel      = receiptColor{243, 246, 244}
	receiptLine       = receiptColor{220, 226, 222}
	receiptWhite      = receiptColor{255, 255, 255}
)

func (s *Server) publicOrderReceipt(w http.ResponseWriter, r *http.Request) {
	order, err := s.loadTrackedOrder(r.Context(), chi.URLParam(r, "code"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, apiError{Error: "заказ с таким кодом не найден"})
		return
	}
	s.writeOrderReceipt(w, order)
}

func (s *Server) orderReceipt(w http.ResponseWriter, r *http.Request) {
	var trackingCode string
	if err := s.db.QueryRow(r.Context(), `SELECT tracking_code FROM orders WHERE id=$1`, chi.URLParam(r, "id")).Scan(&trackingCode); err != nil {
		writeJSON(w, http.StatusNotFound, apiError{Error: "order not found"})
		return
	}
	order, err := s.loadTrackedOrder(r.Context(), trackingCode)
	if err != nil {
		writeJSON(w, http.StatusNotFound, apiError{Error: "order not found"})
		return
	}
	s.writeOrderReceipt(w, order)
}

func (s *Server) writeOrderReceipt(w http.ResponseWriter, order trackedOrder) {
	document, err := renderOrderReceipt(order, time.Now())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "could not create receipt"})
		return
	}
	filename := "receipt-" + strings.ReplaceAll(order.Number, "\"", "") + ".pdf"
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(document)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(document)
}

func renderOrderReceipt(order trackedOrder, generatedAt time.Time) ([]byte, error) {
	if order.CompanyName == "" {
		order.CompanyName = "PrintForge Studio"
	}
	if order.Currency == "" {
		order.Currency = "MDL"
	}

	var pdf gopdf.GoPdf
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	if err := pdf.AddTTFFontData("Noto", receiptRegularFont); err != nil {
		return nil, err
	}
	if err := pdf.AddTTFFontData("NotoBold", receiptBoldFont); err != nil {
		return nil, err
	}
	pdf.SetInfo(gopdf.PdfInfo{
		Title:        "Квитанция " + order.Number,
		Author:       order.CompanyName,
		Subject:      "Квитанция к заказу 3D-печати",
		Creator:      "PrintForge",
		Producer:     "PrintForge Go backend",
		CreationDate: generatedAt,
	})
	pdf.AddPage()
	if err := drawReceiptHeader(&pdf, order); err != nil {
		return nil, err
	}
	if err := drawReceiptOrderDetails(&pdf, order); err != nil {
		return nil, err
	}

	y := 445.0
	if err := drawReceiptTableHeader(&pdf, y); err != nil {
		return nil, err
	}
	y += 32
	models := order.Models
	if len(models) == 0 {
		models = []trackedModel{{Name: "Услуги 3D-печати", OriginalFilename: "Модель не прикреплена", Format: "-"}}
	}
	for index, model := range models {
		if y > 680 {
			pdf.AddPage()
			if err := drawReceiptContinuationHeader(&pdf, order); err != nil {
				return nil, err
			}
			y = 116
			if err := drawReceiptTableHeader(&pdf, y); err != nil {
				return nil, err
			}
			y += 32
		}
		if err := drawReceiptModelRow(&pdf, y, index+1, model); err != nil {
			return nil, err
		}
		y += 42
	}

	if y > 590 {
		pdf.AddPage()
		if err := drawReceiptContinuationHeader(&pdf, order); err != nil {
			return nil, err
		}
		y = 130
	} else {
		y += 22
	}
	if err := drawReceiptTotals(&pdf, y, order); err != nil {
		return nil, err
	}

	pageCount := pdf.GetNumberOfPages()
	for page := 1; page <= pageCount; page++ {
		if err := pdf.SetPage(page); err != nil {
			return nil, err
		}
		if err := drawReceiptFooter(&pdf, page, pageCount, generatedAt); err != nil {
			return nil, err
		}
	}
	return pdf.GetBytesPdfReturnErr()
}

func drawReceiptHeader(pdf *gopdf.GoPdf, order trackedOrder) error {
	pdf.SetFillColor(18, 24, 20)
	pdf.RectFromUpperLeftWithStyle(0, 0, receiptPageWidth, 218, "F")
	pdf.SetFillColor(receiptPurple.r, receiptPurple.g, receiptPurple.b)
	pdf.RectFromUpperLeftWithStyle(0, 0, 11, 218, "F")
	pdf.SetFillColor(31, 38, 34)
	pdf.Polygon([]gopdf.Point{{X: 365, Y: 0}, {X: receiptPageWidth, Y: 0}, {X: receiptPageWidth, Y: 218}, {X: 438, Y: 218}}, "F")
	pdf.SetFillColor(43, 38, 70)
	pdf.Polygon([]gopdf.Point{{X: 485, Y: 0}, {X: receiptPageWidth, Y: 0}, {X: receiptPageWidth, Y: 75}}, "F")
	if err := receiptRoundedBox(pdf, receiptMargin, 31, 40, 40, 9, receiptPurple); err != nil {
		return err
	}
	if err := receiptText(pdf, receiptMargin, 31, 40, 40, "PF", "NotoBold", 14, receiptWhite, gopdf.Center|gopdf.Middle); err != nil {
		return err
	}
	if err := receiptText(pdf, 96, 31, 280, 22, order.CompanyName, "NotoBold", 15, receiptWhite, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	if err := receiptText(pdf, 96, 53, 280, 16, "Мастерская 3D-печати", "Noto", 8.5, receiptColor{171, 183, 175}, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	if err := receiptText(pdf, receiptMargin, 93, 345, 22, "PDF-КВИТАНЦИЯ", "Noto", 8, receiptPurple, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	if err := receiptText(pdf, receiptMargin, 114, 345, 42, "Квитанция к заказу", "NotoBold", 25, receiptWhite, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	if err := receiptRoundedBox(pdf, receiptMargin, 169, 132, 25, 12, receiptPurple); err != nil {
		return err
	}
	if err := receiptText(pdf, receiptMargin, 169, 132, 25, order.StatusLabel, "NotoBold", 8.5, receiptWhite, gopdf.Center|gopdf.Middle); err != nil {
		return err
	}
	if err := receiptText(pdf, 188, 169, 188, 25, order.Number, "Noto", 10, receiptColor{194, 202, 197}, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	if err := receiptRoundedBox(pdf, 436, 29, 115, 115, 12, receiptWhite); err != nil {
		return err
	}
	if err := drawReceiptQR(pdf, 447, 40, 93, receiptTrackingURL(order)); err != nil {
		return err
	}
	if err := receiptText(pdf, 436, 151, 115, 14, "ОТКРОЙТЕ ЗАКАЗ", "Noto", 7, receiptColor{171, 183, 175}, gopdf.Center|gopdf.Middle); err != nil {
		return err
	}
	return receiptText(pdf, 421, 169, 145, 24, order.TrackingCode, "NotoBold", 12.5, receiptPurple, gopdf.Center|gopdf.Middle)
}

func drawReceiptOrderDetails(pdf *gopdf.GoPdf, order trackedOrder) error {
	created := receiptLocalTime(order.CreatedAt).Format("02.01.2006 15:04")
	deadline := "Не указан"
	if order.Deadline != nil {
		deadline = receiptLocalTime(*order.Deadline).Format("02.01.2006 15:04")
	}
	customer := "Без указанного клиента"
	if order.CustomerName != nil && strings.TrimSpace(*order.CustomerName) != "" {
		customer = *order.CustomerName
	}
	details := []struct{ label, value string }{
		{"КЛИЕНТ", customer},
		{"ДАТА ЗАКАЗА", created},
		{"СТАТУС", order.StatusLabel},
		{"ПЛАНОВЫЙ СРОК", deadline},
	}
	if err := receiptText(pdf, receiptMargin, 238, 220, 18, "ДЕТАЛИ ЗАКАЗА", "NotoBold", 8.5, receiptPurpleInk, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	for index, detail := range details {
		x := receiptMargin + float64(index%2)*258
		y := 263.0 + float64(index/2)*70
		if err := receiptRoundedBox(pdf, x, y, 247, 58, 8, receiptPanel); err != nil {
			return err
		}
		if err := receiptText(pdf, x+14, y+8, 219, 14, detail.label, "Noto", 7, receiptMuted, gopdf.Left|gopdf.Middle); err != nil {
			return err
		}
		if err := receiptText(pdf, x+14, y+25, 219, 23, truncateReceiptText(detail.value, 42), "NotoBold", 10, receiptInk, gopdf.Left|gopdf.Middle); err != nil {
			return err
		}
	}
	return receiptText(pdf, receiptMargin, 414, 220, 18, "СОСТАВ ЗАКАЗА", "NotoBold", 8.5, receiptPurpleInk, gopdf.Left|gopdf.Middle)
}

func drawReceiptContinuationHeader(pdf *gopdf.GoPdf, order trackedOrder) error {
	pdf.SetFillColor(18, 24, 20)
	pdf.RectFromUpperLeftWithStyle(0, 0, receiptPageWidth, 86, "F")
	pdf.SetFillColor(receiptPurple.r, receiptPurple.g, receiptPurple.b)
	pdf.RectFromUpperLeftWithStyle(0, 0, 9, 86, "F")
	if err := receiptText(pdf, receiptMargin, 18, 330, 24, "Квитанция "+order.Number, "NotoBold", 14, receiptWhite, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	if err := receiptText(pdf, receiptMargin, 45, 330, 16, order.CompanyName, "Noto", 8, receiptColor{171, 183, 175}, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	return receiptText(pdf, 390, 25, 161, 22, order.TrackingCode, "NotoBold", 11, receiptPurple, gopdf.Right|gopdf.Middle)
}

func drawReceiptTableHeader(pdf *gopdf.GoPdf, y float64) error {
	if err := receiptRoundedBox(pdf, receiptMargin, y, receiptPageWidth-2*receiptMargin, 30, 7, receiptPanel); err != nil {
		return err
	}
	columns := []struct {
		x, w  float64
		text  string
		align int
	}{{receiptMargin + 10, 28, "№", gopdf.Left}, {receiptMargin + 42, 210, "МОДЕЛЬ", gopdf.Left}, {receiptMargin + 258, 58, "ФОРМАТ", gopdf.Left}, {receiptMargin + 322, 175, "ФАЙЛ", gopdf.Left}}
	for _, column := range columns {
		if err := receiptText(pdf, column.x, y, column.w, 30, column.text, "NotoBold", 8, receiptMuted, column.align|gopdf.Middle); err != nil {
			return err
		}
	}
	return nil
}

func drawReceiptModelRow(pdf *gopdf.GoPdf, y float64, index int, model trackedModel) error {
	pdf.SetStrokeColor(receiptLine.r, receiptLine.g, receiptLine.b)
	pdf.SetLineWidth(.6)
	pdf.Line(receiptMargin, y+41, receiptPageWidth-receiptMargin, y+41)
	values := []struct {
		x, w  float64
		text  string
		font  string
		color receiptColor
	}{{receiptMargin + 10, 28, fmt.Sprintf("%d", index), "Noto", receiptMuted}, {receiptMargin + 42, 210, truncateReceiptText(model.Name, 42), "NotoBold", receiptInk}, {receiptMargin + 258, 58, strings.ToUpper(model.Format), "Noto", receiptInk}, {receiptMargin + 322, 175, truncateReceiptText(model.OriginalFilename, 32), "Noto", receiptMuted}}
	for _, value := range values {
		if err := receiptText(pdf, value.x, y, value.w, 41, value.text, value.font, 9, value.color, gopdf.Left|gopdf.Middle); err != nil {
			return err
		}
	}
	return nil
}

func drawReceiptTotals(pdf *gopdf.GoPdf, y float64, order trackedOrder) error {
	if err := receiptRoundedBox(pdf, receiptMargin, y, 309, 158, 11, receiptPanel); err != nil {
		return err
	}
	if err := receiptText(pdf, receiptMargin+18, y+16, 270, 18, "ОПЛАТА", "NotoBold", 8.5, receiptPurpleInk, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	if err := receiptText(pdf, receiptMargin+18, y+43, 138, 17, "Полная стоимость", "Noto", 8, receiptMuted, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	if err := receiptText(pdf, receiptMargin+151, y+40, 140, 22, receiptMoney(order.SellingPrice, order.Currency), "NotoBold", 11, receiptInk, gopdf.Right|gopdf.Middle); err != nil {
		return err
	}
	if err := receiptText(pdf, receiptMargin+18, y+73, 138, 17, "Уже оплачено", "Noto", 8, receiptMuted, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	if err := receiptText(pdf, receiptMargin+151, y+70, 140, 22, receiptMoney(order.PaidAmount, order.Currency), "NotoBold", 10, receiptInk, gopdf.Right|gopdf.Middle); err != nil {
		return err
	}
	progress := 0.0
	if order.SellingPrice > 0 {
		progress = order.PaidAmount / order.SellingPrice
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	if err := receiptRoundedBox(pdf, receiptMargin+18, y+108, 273, 8, 4, receiptLine); err != nil {
		return err
	}
	if progress > 0 {
		if err := receiptRoundedBox(pdf, receiptMargin+18, y+108, 273*progress, 8, 4, receiptPurple); err != nil {
			return err
		}
	}
	if err := receiptText(pdf, receiptMargin+18, y+125, 273, 17, fmt.Sprintf("Оплачено %.0f%% от полной стоимости", progress*100), "Noto", 8, receiptMuted, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	if err := receiptRoundedBox(pdf, 365, y, 186, 158, 11, receiptPurple); err != nil {
		return err
	}
	if err := receiptText(pdf, 383, y+18, 150, 18, "ОСТАЛОСЬ ОПЛАТИТЬ", "NotoBold", 8, receiptPurpleSoft, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	if err := receiptText(pdf, 383, y+48, 150, 42, receiptMoney(order.BalanceDue, order.Currency), "NotoBold", 18, receiptWhite, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	paymentMessage := "Ожидается оплата"
	if order.BalanceDue <= 0 {
		paymentMessage = "Заказ полностью оплачен"
	}
	if err := receiptText(pdf, 383, y+101, 150, 20, paymentMessage, "Noto", 8, receiptPurpleSoft, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	if err := receiptText(pdf, 383, y+126, 150, 18, "Код: "+order.TrackingCode, "NotoBold", 8, receiptWhite, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	if err := receiptText(pdf, receiptMargin, y+174, receiptPageWidth-2*receiptMargin, 25, "Документ подтверждает расчёты по заказу и не является фискальным кассовым чеком.", "Noto", 8, receiptMuted, gopdf.Center|gopdf.Middle); err != nil {
		return err
	}
	return receiptText(pdf, receiptMargin, y+207, receiptPageWidth-2*receiptMargin, 24, "Спасибо, что выбрали "+order.CompanyName, "NotoBold", 10, receiptInk, gopdf.Center|gopdf.Middle)
}

func drawReceiptFooter(pdf *gopdf.GoPdf, page, total int, generatedAt time.Time) error {
	pdf.SetStrokeColor(receiptLine.r, receiptLine.g, receiptLine.b)
	pdf.SetLineWidth(.6)
	pdf.Line(receiptMargin, 795, receiptPageWidth-receiptMargin, 795)
	if err := receiptText(pdf, receiptMargin, 801, 340, 18, "Сформировано PrintForge · "+receiptLocalTime(generatedAt).Format("02.01.2006 15:04"), "Noto", 7.5, receiptMuted, gopdf.Left|gopdf.Middle); err != nil {
		return err
	}
	return receiptText(pdf, 420, 801, 131, 18, fmt.Sprintf("Страница %d из %d", page, total), "Noto", 7.5, receiptMuted, gopdf.Right|gopdf.Middle)
}

func receiptText(pdf *gopdf.GoPdf, x, y, width, height float64, text, font string, size float64, color receiptColor, align int) error {
	if err := pdf.SetFont(font, "", size); err != nil {
		return err
	}
	pdf.SetTextColor(color.r, color.g, color.b)
	pdf.SetXY(x, y)
	return pdf.CellWithOption(&gopdf.Rect{W: width, H: height}, text, gopdf.CellOption{Align: align})
}

func receiptRoundedBox(pdf *gopdf.GoPdf, x, y, width, height, radius float64, color receiptColor) error {
	if width <= 0 || height <= 0 {
		return nil
	}
	if maximum := min(width/2, height/2); radius > maximum {
		radius = maximum
	}
	pdf.SetFillColor(color.r, color.g, color.b)
	return pdf.Rectangle(x, y, x+width, y+height, "F", radius, 8)
}

func drawReceiptQR(pdf *gopdf.GoPdf, x, y, size float64, target string) error {
	code, err := qrcode.New(target, qrcode.Medium)
	if err != nil {
		return err
	}
	bitmap := code.Bitmap()
	if len(bitmap) == 0 {
		return fmt.Errorf("empty QR code")
	}
	moduleSize := size / float64(len(bitmap))
	pdf.SetFillColor(receiptInk.r, receiptInk.g, receiptInk.b)
	for row := range bitmap {
		for column, filled := range bitmap[row] {
			if filled {
				pdf.RectFromUpperLeftWithStyle(x+float64(column)*moduleSize, y+float64(row)*moduleSize, moduleSize+.06, moduleSize+.06, "F")
			}
		}
	}
	return nil
}

func receiptTrackingURL(order trackedOrder) string {
	baseURL := strings.TrimRight(strings.TrimSpace(order.PublicBaseURL), "/")
	if baseURL == "" {
		baseURL = "http://localhost"
	}
	return baseURL + "/track/" + order.TrackingCode
}

func receiptMoney(value float64, currency string) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	parts := strings.Split(fmt.Sprintf("%.2f", value), ".")
	whole := parts[0]
	for index := len(whole) - 3; index > 0; index -= 3 {
		whole = whole[:index] + " " + whole[index:]
	}
	return sign + whole + "," + parts[1] + " " + currency
}

func receiptLocalTime(value time.Time) time.Time {
	location, err := time.LoadLocation("Europe/Chisinau")
	if err != nil {
		return value
	}
	return value.In(location)
}

func truncateReceiptText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:limit-3])) + "..."
}
