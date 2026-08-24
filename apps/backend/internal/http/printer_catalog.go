package httpapi

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed printer_catalog.json
var printerCatalogJSON []byte

type printerCatalogSource struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Repository string `json:"repository"`
	Revision   string `json:"revision"`
	License    string `json:"license"`
}

type printerCatalogModel struct {
	Key              string                 `json:"key"`
	Manufacturer     string                 `json:"manufacturer"`
	Model            string                 `json:"model"`
	FullName         string                 `json:"fullName"`
	ModelID          string                 `json:"modelId"`
	Family           string                 `json:"family"`
	Technology       string                 `json:"technology"`
	NozzleDiameters  []float64              `json:"nozzleDiameters"`
	BuildXMM         float64                `json:"buildXmm"`
	BuildYMM         float64                `json:"buildYmm"`
	BuildZMM         float64                `json:"buildZmm"`
	ImageURL         *string                `json:"imageUrl"`
	DefaultMaterials []string               `json:"defaultMaterials"`
	ProfileURL       string                 `json:"profileUrl"`
	Sources          []printerCatalogSource `json:"sources"`
}

type printerCatalog struct {
	GeneratedAt string                 `json:"generatedAt"`
	Total       int                    `json:"total"`
	Sources     []printerCatalogSource `json:"sources"`
	Models      []printerCatalogModel  `json:"models"`
	byKey       map[string]printerCatalogModel
}

func loadPrinterCatalog() (printerCatalog, error) {
	var catalog printerCatalog
	if err := json.Unmarshal(printerCatalogJSON, &catalog); err != nil {
		return printerCatalog{}, fmt.Errorf("decode embedded printer catalog: %w", err)
	}
	catalog.byKey = make(map[string]printerCatalogModel, len(catalog.Models))
	for _, model := range catalog.Models {
		catalog.byKey[model.Key] = model
	}
	return catalog, nil
}

func normalizeCatalogValue(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func (s *Server) BackfillPrinterCatalog(ctx context.Context) error {
	rows, err := s.db.Query(ctx, `SELECT id,manufacturer,model,name FROM printers WHERE catalog_key IS NULL OR image_url IS NULL`)
	if err != nil {
		return fmt.Errorf("query printers requiring catalog match: %w", err)
	}
	type existingPrinter struct{ id, manufacturer, model, name string }
	items := make([]existingPrinter, 0)
	for rows.Next() {
		var item existingPrinter
		if err := rows.Scan(&item.id, &item.manufacturer, &item.model, &item.name); err != nil {
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, item := range items {
		manufacturer := normalizeCatalogValue(item.manufacturer)
		model := normalizeCatalogValue(item.model)
		name := normalizeCatalogValue(item.name)
		for _, candidate := range s.catalog.Models {
			if normalizeCatalogValue(candidate.Manufacturer) != manufacturer {
				continue
			}
			candidateModel := normalizeCatalogValue(candidate.Model)
			candidateFull := normalizeCatalogValue(candidate.FullName)
			if candidateModel != model && candidateModel != name && candidateFull != model && candidateFull != name {
				continue
			}
			_, err = s.db.Exec(ctx, `UPDATE printers SET catalog_key=$2,image_url=$3,build_x_mm=CASE WHEN build_x_mm=0 THEN $4 ELSE build_x_mm END,build_y_mm=CASE WHEN build_y_mm=0 THEN $5 ELSE build_y_mm END,build_z_mm=CASE WHEN build_z_mm=0 THEN $6 ELSE build_z_mm END,updated_at=now() WHERE id=$1`, item.id, candidate.Key, candidate.ImageURL, candidate.BuildXMM, candidate.BuildYMM, candidate.BuildZMM)
			if err != nil {
				return fmt.Errorf("update printer catalog match: %w", err)
			}
			break
		}
	}
	return nil
}
