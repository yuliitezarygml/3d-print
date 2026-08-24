package httpapi

import (
	"bytes"
	"testing"
	"time"
)

func TestRenderOrderReceipt(t *testing.T) {
	customer := "Иван Петров"
	order := trackedOrder{
		Number:        "ORD-2026-00042",
		TrackingCode:  "PDF42CHECK",
		Status:        "READY",
		StatusLabel:   "Готов к выдаче",
		SellingPrice:  1234.5,
		PaidAmount:    1000,
		BalanceDue:    234.5,
		Currency:      "MDL",
		CustomerName:  &customer,
		CreatedAt:     time.Date(2026, time.August, 24, 10, 30, 0, 0, time.UTC),
		CompanyName:   "PrintForge Studio",
		PublicBaseURL: "https://print.example",
		Models: []trackedModel{{
			Name:             "Корпус электроники",
			OriginalFilename: "korpus.3mf",
			Format:           "3MF",
		}},
	}
	document, err := renderOrderReceipt(order, time.Date(2026, time.August, 24, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("render receipt: %v", err)
	}
	if len(document) < 10_000 {
		t.Fatalf("receipt is unexpectedly small: %d bytes", len(document))
	}
	if !bytes.HasPrefix(document, []byte("%PDF-")) {
		t.Fatal("receipt is not a PDF document")
	}
}

func TestReceiptTrackingURL(t *testing.T) {
	order := trackedOrder{TrackingCode: "PDF42CHECK", PublicBaseURL: "https://print.example/"}
	if got := receiptTrackingURL(order); got != "https://print.example/track/PDF42CHECK" {
		t.Fatalf("unexpected tracking URL: %q", got)
	}
}

func TestReceiptMoney(t *testing.T) {
	if got := receiptMoney(1234567.8, "MDL"); got != "1 234 567,80 MDL" {
		t.Fatalf("unexpected money format: %q", got)
	}
}
