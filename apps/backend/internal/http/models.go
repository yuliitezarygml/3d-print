package httpapi

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (s *Server) listModels(w http.ResponseWriter, r *http.Request) {
	customerID := strings.TrimSpace(r.URL.Query().Get("customerId"))
	rows, err := s.db.Query(r.Context(), `SELECT m.id,m.name,m.original_filename,m.format,m.file_size_bytes,m.dimensions_x_mm,m.dimensions_y_mm,m.dimensions_z_mm,m.volume_cm3,m.triangle_count,m.version,m.created_at,m.customer_id,c.name,m.preview_path,m.estimated_print_minutes,m.estimated_filament_grams,m.slicer_metadata FROM models m LEFT JOIN customers c ON c.id=m.customer_id WHERE ($1='' OR m.customer_id::text=$1) ORDER BY m.created_at DESC`, customerID)
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not load models"})
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, name, filename, format string
		var size int64
		var x, y, z, volume *float64
		var triangles *int64
		var customerID, customerName, previewPath *string
		var estimatedMinutes *int
		var estimatedGrams *float64
		var slicerMetadata []byte
		var version int
		var created time.Time
		if rows.Scan(&id, &name, &filename, &format, &size, &x, &y, &z, &volume, &triangles, &version, &created, &customerID, &customerName, &previewPath, &estimatedMinutes, &estimatedGrams, &slicerMetadata) == nil {
			var previewURL *string
			if previewPath != nil {
				value := "/api/models/" + id + "/preview"
				previewURL = &value
			}
			metadata := map[string]any{}
			_ = json.Unmarshal(slicerMetadata, &metadata)
			items = append(items, map[string]any{"id": id, "name": name, "originalFilename": filename, "format": format, "fileSizeBytes": size, "dimensionsXmm": x, "dimensionsYmm": y, "dimensionsZmm": z, "volumeCm3": volume, "triangleCount": triangles, "version": version, "createdAt": created, "customerId": customerID, "customerName": customerName, "previewUrl": previewURL, "fileUrl": "/api/models/" + id + "/file", "estimatedPrintMinutes": estimatedMinutes, "estimatedFilamentGrams": estimatedGrams, "slicerMetadata": metadata})
		}
	}
	writeJSON(w, 200, items)
}

func (s *Server) uploadModel(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxModelFileBytes+2*1024*1024)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		badRequest(w, "file is too large or form is invalid")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		badRequest(w, "file is required")
		return
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".stl" && ext != ".obj" && ext != ".3mf" && ext != ".gcode" && ext != ".gco" {
		badRequest(w, "only STL, OBJ, 3MF and G-code files are supported")
		return
	}
	probe := make([]byte, 512)
	n, _ := file.Read(probe)
	probe = probe[:n]
	if !validModelSignature(ext, probe) {
		badRequest(w, "file content does not match a supported 3D format")
		return
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		badRequest(w, "could not read uploaded file")
		return
	}
	storageName := uuid.NewString() + ext
	storageKey := modelObjectKey(storageName)
	temp, err := os.CreateTemp("", "printforge-model-*"+ext)
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not store file"})
		return
	}
	path := temp.Name()
	defer os.Remove(path)
	written, copyErr := io.Copy(temp, io.LimitReader(file, s.cfg.MaxModelFileBytes+1))
	closeErr := temp.Close()
	if copyErr != nil || closeErr != nil || written > s.cfg.MaxModelFileBytes {
		badRequest(w, "file exceeds configured size limit")
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(header.Filename), ext)
	}
	format := strings.ToUpper(strings.TrimPrefix(ext, "."))
	var customerID *string
	if value := strings.TrimSpace(r.FormValue("customerId")); value != "" {
		var exists bool
		if err := s.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM customers WHERE id=$1)`, value).Scan(&exists); err != nil || !exists {
			badRequest(w, "customer not found")
			return
		}
		customerID = &value
	}
	var x, y, z, volume *float64
	var triangles *int64
	if ext == ".stl" || ext == ".3mf" {
		if info, analysisErr := analyseModel(path, ext); analysisErr == nil {
			x = &info.X
			y = &info.Y
			z = &info.Z
			volume = &info.VolumeCM3
			triangles = &info.Triangles
		}
	}
	estimatedMinutes, estimatedGrams, slicerMetadata := analyseSlicerEstimate(path, ext)
	var previewPath *string
	var previewTemp string
	if ext == ".3mf" {
		if extracted, previewErr := extract3MFPreview(path, filepath.Dir(path), strings.TrimSuffix(filepath.Base(path), ext)); previewErr == nil && extracted != "" {
			previewTemp = filepath.Join(filepath.Dir(path), extracted)
			previewKey := modelObjectKey(strings.TrimSuffix(storageName, ext) + "-preview.png")
			previewPath = &previewKey
		}
	}
	stored, err := os.Open(path)
	if err != nil || s.storage.Put(r.Context(), storageKey, stored, written, modelMime(ext)) != nil {
		if stored != nil {
			_ = stored.Close()
		}
		writeJSON(w, 500, apiError{Error: "could not store file"})
		return
	}
	_ = stored.Close()
	if previewPath != nil {
		previewFile, previewErr := os.Open(previewTemp)
		if previewErr == nil {
			if info, statErr := previewFile.Stat(); statErr == nil {
				if putErr := s.storage.Put(r.Context(), *previewPath, previewFile, info.Size(), "image/png"); putErr != nil {
					previewPath = nil
				}
			}
			_ = previewFile.Close()
		}
		_ = os.Remove(previewTemp)
	}
	var id string
	err = s.db.QueryRow(r.Context(), `INSERT INTO models(name,original_filename,storage_path,mime_type,file_size_bytes,format,dimensions_x_mm,dimensions_y_mm,dimensions_z_mm,volume_cm3,triangle_count,customer_id,preview_path,estimated_print_minutes,estimated_filament_grams,slicer_metadata)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)RETURNING id`, name, filepath.Base(header.Filename), storageKey, modelMime(ext), written, format, x, y, z, volume, triangles, customerID, previewPath, estimatedMinutes, estimatedGrams, slicerMetadata).Scan(&id)
	if err != nil {
		_ = s.storage.Remove(r.Context(), storageKey)
		if previewPath != nil {
			_ = s.storage.Remove(r.Context(), *previewPath)
		}
		writeJSON(w, 500, apiError{Error: "could not register model"})
		return
	}
	s.audit(r, "UPLOAD", "Model", id, nil, map[string]any{"filename": header.Filename, "size": written})
	var previewURL *string
	if previewPath != nil {
		value := "/api/models/" + id + "/preview"
		previewURL = &value
	}
	metadata := map[string]any{}
	_ = json.Unmarshal(slicerMetadata, &metadata)
	writeJSON(w, 201, map[string]any{"id": id, "name": name, "originalFilename": filepath.Base(header.Filename), "format": format, "fileSizeBytes": written, "dimensionsXmm": x, "dimensionsYmm": y, "dimensionsZmm": z, "volumeCm3": volume, "triangleCount": triangles, "customerId": customerID, "previewUrl": previewURL, "fileUrl": "/api/models/" + id + "/file", "estimatedPrintMinutes": estimatedMinutes, "estimatedFilamentGrams": estimatedGrams, "slicerMetadata": metadata})
}

// BackfillModelAnalysis fills geometry metadata for models uploaded by older
// PrintForge versions. A failed file is skipped so one damaged upload cannot
// prevent the API from starting.
func (s *Server) BackfillModelAnalysis(ctx context.Context) error {
	rows, err := s.db.Query(ctx, `SELECT id,storage_path,format,preview_path FROM models WHERE (dimensions_x_mm IS NULL OR dimensions_y_mm IS NULL OR dimensions_z_mm IS NULL OR volume_cm3 IS NULL OR triangle_count IS NULL OR (format='3MF' AND preview_path IS NULL)) AND format IN ('STL','3MF')`)
	if err != nil {
		return fmt.Errorf("query models requiring analysis: %w", err)
	}
	type pendingModel struct {
		id, storage, format string
		preview             *string
	}
	pending := make([]pendingModel, 0)
	for rows.Next() {
		var model pendingModel
		if err := rows.Scan(&model.id, &model.storage, &model.format, &model.preview); err != nil {
			rows.Close()
			return fmt.Errorf("scan model requiring analysis: %w", err)
		}
		pending = append(pending, model)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read models requiring analysis: %w", err)
	}
	rows.Close()

	var analysisErrors []error
	for _, model := range pending {
		ext := "." + strings.ToLower(model.format)
		path, ok := s.storage.LocalPath(model.storage)
		if !ok {
			continue
		}
		info, err := analyseModel(path, ext)
		if err != nil {
			analysisErrors = append(analysisErrors, fmt.Errorf("analyse model %s: %w", model.id, err))
			continue
		}
		previewPath := model.preview
		if ext == ".3mf" && previewPath == nil {
			if extracted, previewErr := extract3MFPreview(path, filepath.Dir(path), strings.TrimSuffix(filepath.Base(model.storage), ext)); previewErr == nil && extracted != "" {
				previewPath = &extracted
			}
		}
		if _, err := s.db.Exec(ctx, `UPDATE models SET dimensions_x_mm=$2,dimensions_y_mm=$3,dimensions_z_mm=$4,volume_cm3=$5,triangle_count=$6,preview_path=$7 WHERE id=$1`, model.id, info.X, info.Y, info.Z, info.VolumeCM3, info.Triangles, previewPath); err != nil {
			analysisErrors = append(analysisErrors, fmt.Errorf("update model %s analysis: %w", model.id, err))
		}
	}
	return errors.Join(analysisErrors...)
}

func (s *Server) modelFile(w http.ResponseWriter, r *http.Request) {
	s.serveModelFile(w, r, chi.URLParam(r, "id"))
}

func (s *Server) serveModelFile(w http.ResponseWriter, r *http.Request, id string) {
	var storage, filename, mime string
	err := s.db.QueryRow(r.Context(), `SELECT storage_path,original_filename,mime_type FROM models WHERE id=$1`, id).Scan(&storage, &filename, &mime)
	if err != nil {
		writeJSON(w, 404, apiError{Error: "model not found"})
		return
	}
	file, _, _, err := s.storage.Open(r.Context(), storage)
	if err != nil {
		writeJSON(w, 404, apiError{Error: "model file not found"})
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", strings.ReplaceAll(filename, "\"", "")))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = io.Copy(w, file)
}

func (s *Server) modelPreview(w http.ResponseWriter, r *http.Request) {
	s.serveModelPreview(w, r, chi.URLParam(r, "id"))
}

func (s *Server) serveModelPreview(w http.ResponseWriter, r *http.Request, id string) {
	var storage string
	if err := s.db.QueryRow(r.Context(), `SELECT preview_path FROM models WHERE id=$1 AND preview_path IS NOT NULL`, id).Scan(&storage); err != nil {
		writeJSON(w, 404, apiError{Error: "model preview not found"})
		return
	}
	file, _, _, err := s.storage.Open(r.Context(), storage)
	if err != nil {
		writeJSON(w, 404, apiError{Error: "model preview not found"})
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", imageMime(filepath.Ext(storage)))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = io.Copy(w, file)
}

func (s *Server) uploadModelPreview(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxImageFileBytes+1024*1024)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		badRequest(w, "preview image is too large or invalid")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		badRequest(w, "preview image is required")
		return
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".webp" {
		badRequest(w, "preview must be PNG, JPEG or WebP")
		return
	}
	probe := make([]byte, 512)
	n, _ := file.Read(probe)
	if !validImageSignature(ext, probe[:n]) {
		badRequest(w, "preview content is not a supported image")
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		badRequest(w, "could not read preview image")
		return
	}
	id := chi.URLParam(r, "id")
	var oldPreview *string
	if err := s.db.QueryRow(r.Context(), `SELECT preview_path FROM models WHERE id=$1`, id).Scan(&oldPreview); err != nil {
		writeJSON(w, 404, apiError{Error: "model not found"})
		return
	}
	storage := modelObjectKey(id + "-preview" + ext)
	temp, err := os.CreateTemp("", "printforge-preview-*"+ext)
	if err != nil {
		writeJSON(w, 500, apiError{Error: "could not store preview image"})
		return
	}
	path := temp.Name()
	defer os.Remove(path)
	written, copyErr := io.Copy(temp, io.LimitReader(file, s.cfg.MaxImageFileBytes+1))
	closeErr := temp.Close()
	if copyErr != nil || closeErr != nil || written > s.cfg.MaxImageFileBytes {
		badRequest(w, "preview image exceeds configured size limit")
		return
	}
	stored, openErr := os.Open(path)
	if openErr != nil || s.storage.Put(r.Context(), storage, stored, written, imageMime(ext)) != nil {
		if stored != nil {
			_ = stored.Close()
		}
		writeJSON(w, 500, apiError{Error: "could not store preview image"})
		return
	}
	_ = stored.Close()
	if _, err := s.db.Exec(r.Context(), `UPDATE models SET preview_path=$2 WHERE id=$1`, id, storage); err != nil {
		_ = s.storage.Remove(r.Context(), storage)
		writeJSON(w, 500, apiError{Error: "could not register preview image"})
		return
	}
	if oldPreview != nil && *oldPreview != storage {
		_ = s.storage.Remove(r.Context(), *oldPreview)
	}
	writeJSON(w, 200, map[string]any{"previewUrl": "/api/models/" + id + "/preview"})
}

func validImageSignature(ext string, data []byte) bool {
	if len(data) < 12 {
		return false
	}
	switch ext {
	case ".png":
		return string(data[:8]) == "\x89PNG\r\n\x1a\n"
	case ".jpg", ".jpeg":
		return data[0] == 0xff && data[1] == 0xd8
	case ".webp":
		return string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP"
	}
	return false
}

func imageMime(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

func extract3MFPreview(modelPath, dir, basename string) (string, error) {
	archive, err := zip.OpenReader(modelPath)
	if err != nil {
		return "", err
	}
	defer archive.Close()
	priority := []string{"auxiliaries/.thumbnails/thumbnail_3mf.png", "metadata/plate_1.png", "metadata/thumbnail.png"}
	var selected *zip.File
	for _, wanted := range priority {
		for _, part := range archive.File {
			if strings.ToLower(strings.TrimPrefix(part.Name, "/")) == wanted {
				selected = part
				break
			}
		}
		if selected != nil {
			break
		}
	}
	if selected == nil || selected.UncompressedSize64 > 10<<20 {
		return "", errors.New("3MF preview is missing or too large")
	}
	reader, err := selected.Open()
	if err != nil {
		return "", err
	}
	defer reader.Close()
	storage := basename + "-preview.png"
	out, err := os.OpenFile(filepath.Join(dir, filepath.Base(storage)), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0640)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(out, io.LimitReader(reader, 10<<20))
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(filepath.Join(dir, filepath.Base(storage)))
		return "", errors.Join(copyErr, closeErr)
	}
	return storage, nil
}

func validModelSignature(ext string, b []byte) bool {
	if len(b) < 2 {
		return false
	}
	switch ext {
	case ".gcode", ".gco":
		text := strings.ToUpper(string(b))
		return strings.Contains(text, "G0") || strings.Contains(text, "G1") || strings.Contains(text, "M104") || strings.HasPrefix(strings.TrimSpace(text), ";")
	case ".3mf":
		return b[0] == 'P' && b[1] == 'K'
	case ".obj":
		s := strings.TrimSpace(string(b))
		return strings.HasPrefix(s, "v ") || strings.HasPrefix(s, "#") || strings.Contains(s, "\nv ")
	case ".stl":
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(string(b))), "solid") {
			return true
		}
		return len(b) >= 84
	}
	return false
}
func modelMime(ext string) string {
	switch ext {
	case ".stl":
		return "model/stl"
	case ".obj":
		return "model/obj"
	case ".gcode", ".gco":
		return "text/x.gcode"
	default:
		return "model/3mf"
	}
}

var (
	timeSecondsPattern = regexp.MustCompile(`(?im)^;\s*(?:TIME|total estimated time)\s*[:=]\s*([0-9]+(?:\.[0-9]+)?)\s*$`)
	timeHumanPattern   = regexp.MustCompile(`(?im)^;\s*(?:estimated printing time[^=]*|model printing time)\s*=\s*(.+)$`)
	gramsPattern       = regexp.MustCompile(`(?im)^;\s*(?:total filament used \[g\]|filament used \[g\]|filament weight|total filament weight)\s*=\s*([0-9]+(?:\.[0-9]+)?)`)
	lengthPattern      = regexp.MustCompile(`(?im)^;\s*(?:filament used|total filament used)\s*=\s*([0-9]+(?:\.[0-9]+)?)\s*mm`)
	printerPattern     = regexp.MustCompile(`(?im)^;\s*(?:printer_model|printer model)\s*=\s*(.+)$`)
	materialPattern    = regexp.MustCompile(`(?im)^;\s*(?:filament_type|filament type)\s*=\s*(.+)$`)
)

func analyseSlicerEstimate(path, ext string) (*int, *float64, []byte) {
	data, err := slicerText(path, ext)
	if err != nil || len(data) == 0 {
		return nil, nil, []byte(`{}`)
	}
	metadata := map[string]any{"source": strings.ToUpper(strings.TrimPrefix(ext, "."))}
	var minutes *int
	if match := timeSecondsPattern.FindSubmatch(data); len(match) > 1 {
		if seconds, err := strconv.ParseFloat(string(match[1]), 64); err == nil {
			value := int(math.Ceil(seconds / 60))
			minutes = &value
		}
	} else if match := timeHumanPattern.FindSubmatch(data); len(match) > 1 {
		if value := parseHumanDuration(string(match[1])); value > 0 {
			minutes = &value
		}
	}
	var grams *float64
	if match := gramsPattern.FindSubmatch(data); len(match) > 1 {
		if value, err := strconv.ParseFloat(string(match[1]), 64); err == nil {
			grams = &value
		}
	} else if match := lengthPattern.FindSubmatch(data); len(match) > 1 {
		if lengthMM, err := strconv.ParseFloat(string(match[1]), 64); err == nil {
			value := lengthMM * math.Pi * math.Pow(1.75/2, 2) / 1000 * 1.24
			grams = &value
		}
	}
	if match := printerPattern.FindSubmatch(data); len(match) > 1 {
		metadata["printer"] = strings.TrimSpace(string(match[1]))
	}
	if match := materialPattern.FindSubmatch(data); len(match) > 1 {
		metadata["material"] = strings.TrimSpace(string(match[1]))
	}
	encoded, _ := json.Marshal(metadata)
	return minutes, grams, encoded
}

func slicerText(path, ext string) ([]byte, error) {
	if ext != ".3mf" {
		return os.ReadFile(path)
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer archive.Close()
	for _, part := range archive.File {
		name := strings.ToLower(part.Name)
		if strings.HasSuffix(name, ".gcode") || strings.HasSuffix(name, "slice_info.config") {
			reader, err := part.Open()
			if err != nil {
				continue
			}
			data, readErr := io.ReadAll(io.LimitReader(reader, 16<<20))
			_ = reader.Close()
			if readErr == nil && len(data) > 0 {
				return data, nil
			}
		}
	}
	return nil, errors.New("slicer metadata not found")
}

func parseHumanDuration(value string) int {
	lower := strings.ToLower(value)
	patterns := []struct {
		re         *regexp.Regexp
		multiplier int
	}{
		{regexp.MustCompile(`([0-9]+)\s*(?:d|day)`), 24 * 60},
		{regexp.MustCompile(`([0-9]+)\s*(?:h|hour)`), 60},
		{regexp.MustCompile(`([0-9]+)\s*(?:m|min)`), 1},
	}
	total := 0
	for _, pattern := range patterns {
		if match := pattern.re.FindStringSubmatch(lower); len(match) > 1 {
			number, _ := strconv.Atoi(match[1])
			total += number * pattern.multiplier
		}
	}
	return total
}

type stlInfo struct {
	X, Y, Z, VolumeCM3 float64
	Triangles          int64
}

func analyseModel(path, ext string) (stlInfo, error) {
	switch strings.ToLower(ext) {
	case ".stl":
		return analyseSTL(path)
	case ".3mf":
		return analyse3MF(path)
	default:
		return stlInfo{}, fmt.Errorf("analysis is not supported for %s", ext)
	}
}

func analyseSTL(path string) (stlInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return stlInfo{}, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return stlInfo{}, err
	}
	header := make([]byte, 84)
	if _, err = io.ReadFull(file, header); err == nil {
		count := binary.LittleEndian.Uint32(header[80:84])
		if int64(84)+int64(count)*50 == stat.Size() {
			return analyseBinarySTL(file, int64(count))
		}
	}
	if _, err = file.Seek(0, 0); err != nil {
		return stlInfo{}, err
	}
	return analyseASCIISTL(file)
}
func analyseBinarySTL(r io.Reader, count int64) (stlInfo, error) {
	min := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	max := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	volume := 0.0
	record := make([]byte, 50)
	for i := int64(0); i < count; i++ {
		if _, err := io.ReadFull(r, record); err != nil {
			return stlInfo{}, err
		}
		var v [3][3]float64
		for a := 0; a < 3; a++ {
			for c := 0; c < 3; c++ {
				offset := 12 + a*12 + c*4
				v[a][c] = float64(math.Float32frombits(binary.LittleEndian.Uint32(record[offset : offset+4])))
				if v[a][c] < min[c] {
					min[c] = v[a][c]
				}
				if v[a][c] > max[c] {
					max[c] = v[a][c]
				}
			}
		}
		volume += signedTetra(v[0], v[1], v[2])
	}
	return stlInfo{X: max[0] - min[0], Y: max[1] - min[1], Z: max[2] - min[2], VolumeCM3: math.Abs(volume) / 1000, Triangles: count}, nil
}
func analyseASCIISTL(r io.Reader) (stlInfo, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	min := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	max := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	vertices := make([][3]float64, 0, 3)
	volume := 0.0
	triangles := int64(0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "vertex ") {
			continue
		}
		var v [3]float64
		if _, err := fmt.Sscanf(line, "vertex %f %f %f", &v[0], &v[1], &v[2]); err != nil {
			continue
		}
		for c := 0; c < 3; c++ {
			if v[c] < min[c] {
				min[c] = v[c]
			}
			if v[c] > max[c] {
				max[c] = v[c]
			}
		}
		vertices = append(vertices, v)
		if len(vertices) == 3 {
			volume += signedTetra(vertices[0], vertices[1], vertices[2])
			triangles++
			vertices = vertices[:0]
		}
	}
	if triangles == 0 {
		return stlInfo{}, errors.New("no triangles found")
	}
	return stlInfo{X: max[0] - min[0], Y: max[1] - min[1], Z: max[2] - min[2], VolumeCM3: math.Abs(volume) / 1000, Triangles: triangles}, scanner.Err()
}
func signedTetra(a, b, c [3]float64) float64 {
	return (a[0]*(b[1]*c[2]-b[2]*c[1]) - a[1]*(b[0]*c[2]-b[2]*c[0]) + a[2]*(b[0]*c[1]-b[1]*c[0])) / 6
}

const maxThreeMFModelXMLBytes = 256 << 20

type threeMFModel struct {
	Unit      string `xml:"unit,attr"`
	Resources struct {
		Objects []threeMFObject `xml:"object"`
	} `xml:"resources"`
	Build struct {
		Items []threeMFBuildItem `xml:"item"`
	} `xml:"build"`
}

type threeMFObject struct {
	ID         string             `xml:"id,attr"`
	Mesh       *threeMFMesh       `xml:"mesh"`
	Components *threeMFComponents `xml:"components"`
}

type threeMFMesh struct {
	Vertices struct {
		Items []threeMFVertex `xml:"vertex"`
	} `xml:"vertices"`
	Triangles struct {
		Items []threeMFTriangle `xml:"triangle"`
	} `xml:"triangles"`
}

type threeMFVertex struct {
	X float64 `xml:"x,attr"`
	Y float64 `xml:"y,attr"`
	Z float64 `xml:"z,attr"`
}

type threeMFTriangle struct {
	V1 int `xml:"v1,attr"`
	V2 int `xml:"v2,attr"`
	V3 int `xml:"v3,attr"`
}

type threeMFComponents struct {
	Items []threeMFComponent `xml:"component"`
}

type threeMFComponent struct {
	ObjectID  string `xml:"objectid,attr"`
	Transform string `xml:"transform,attr"`
}

type threeMFBuildItem struct {
	ObjectID  string `xml:"objectid,attr"`
	Transform string `xml:"transform,attr"`
}

// affineMatrix is a row-major 3x4 affine transform.
type affineMatrix [12]float64

func identityAffine() affineMatrix {
	return affineMatrix{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0}
}

func parseThreeMFTransform(value string) (affineMatrix, error) {
	if strings.TrimSpace(value) == "" {
		return identityAffine(), nil
	}
	parts := strings.Fields(value)
	if len(parts) != 12 {
		return affineMatrix{}, fmt.Errorf("transform must contain 12 numbers")
	}
	values := make([]float64, 12)
	for i, part := range parts {
		value, err := strconv.ParseFloat(part, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return affineMatrix{}, fmt.Errorf("invalid transform value %q", part)
		}
		values[i] = value
	}
	// 3MF serializes the three basis columns followed by translation.
	return affineMatrix{
		values[0], values[3], values[6], values[9],
		values[1], values[4], values[7], values[10],
		values[2], values[5], values[8], values[11],
	}, nil
}

func multiplyAffine(a, b affineMatrix) affineMatrix {
	var result affineMatrix
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			result[row*4+col] = a[row*4]*b[col] + a[row*4+1]*b[4+col] + a[row*4+2]*b[8+col]
		}
		result[row*4+3] = a[row*4]*b[3] + a[row*4+1]*b[7] + a[row*4+2]*b[11] + a[row*4+3]
	}
	return result
}

func transformVertex(matrix affineMatrix, vertex threeMFVertex, unitScale float64) [3]float64 {
	return [3]float64{
		(matrix[0]*vertex.X + matrix[1]*vertex.Y + matrix[2]*vertex.Z + matrix[3]) * unitScale,
		(matrix[4]*vertex.X + matrix[5]*vertex.Y + matrix[6]*vertex.Z + matrix[7]) * unitScale,
		(matrix[8]*vertex.X + matrix[9]*vertex.Y + matrix[10]*vertex.Z + matrix[11]) * unitScale,
	}
}

func threeMFUnitScale(unit string) (float64, error) {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "", "millimeter":
		return 1, nil
	case "micron":
		return 0.001, nil
	case "centimeter":
		return 10, nil
	case "meter":
		return 1000, nil
	case "inch":
		return 25.4, nil
	case "foot":
		return 304.8, nil
	default:
		return 0, fmt.Errorf("unsupported 3MF unit %q", unit)
	}
}

func analyse3MF(path string) (stlInfo, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return stlInfo{}, fmt.Errorf("open 3MF archive: %w", err)
	}
	defer archive.Close()

	var modelPart *zip.File
	for _, part := range archive.File {
		name := strings.ToLower(strings.TrimPrefix(part.Name, "/"))
		if name == "3d/3dmodel.model" {
			modelPart = part
			break
		}
		if modelPart == nil && strings.HasSuffix(name, ".model") {
			modelPart = part
		}
	}
	if modelPart == nil {
		return stlInfo{}, errors.New("3MF model part is missing")
	}
	if modelPart.UncompressedSize64 > maxThreeMFModelXMLBytes {
		return stlInfo{}, errors.New("3MF model XML is too large")
	}
	reader, err := modelPart.Open()
	if err != nil {
		return stlInfo{}, fmt.Errorf("open 3MF model part: %w", err)
	}
	defer reader.Close()
	var model threeMFModel
	decoder := xml.NewDecoder(io.LimitReader(reader, maxThreeMFModelXMLBytes+1))
	if err := decoder.Decode(&model); err != nil {
		return stlInfo{}, fmt.Errorf("decode 3MF model XML: %w", err)
	}

	unitScale, err := threeMFUnitScale(model.Unit)
	if err != nil {
		return stlInfo{}, err
	}
	objects := make(map[string]threeMFObject, len(model.Resources.Objects))
	for _, object := range model.Resources.Objects {
		if object.ID == "" {
			return stlInfo{}, errors.New("3MF object has no id")
		}
		objects[object.ID] = object
	}
	if len(model.Build.Items) == 0 {
		return stlInfo{}, errors.New("3MF build has no items")
	}

	min := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	max := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	volumeMM3 := 0.0
	triangles := int64(0)
	stack := make(map[string]bool)
	var visitObject func(string, affineMatrix, int) error
	visitObject = func(objectID string, transform affineMatrix, depth int) error {
		if depth > 64 {
			return errors.New("3MF component nesting is too deep")
		}
		if stack[objectID] {
			return fmt.Errorf("3MF component cycle at object %s", objectID)
		}
		object, ok := objects[objectID]
		if !ok {
			return fmt.Errorf("3MF object %s is missing", objectID)
		}
		stack[objectID] = true
		defer delete(stack, objectID)

		if object.Mesh != nil {
			meshVolume := 0.0
			for _, triangle := range object.Mesh.Triangles.Items {
				indices := [3]int{triangle.V1, triangle.V2, triangle.V3}
				var vertices [3][3]float64
				for i, index := range indices {
					if index < 0 || index >= len(object.Mesh.Vertices.Items) {
						return fmt.Errorf("3MF object %s has an invalid triangle index", objectID)
					}
					vertex := object.Mesh.Vertices.Items[index]
					if math.IsNaN(vertex.X) || math.IsNaN(vertex.Y) || math.IsNaN(vertex.Z) || math.IsInf(vertex.X, 0) || math.IsInf(vertex.Y, 0) || math.IsInf(vertex.Z, 0) {
						return fmt.Errorf("3MF object %s has a non-finite vertex", objectID)
					}
					vertices[i] = transformVertex(transform, vertex, unitScale)
					for coordinate := 0; coordinate < 3; coordinate++ {
						if vertices[i][coordinate] < min[coordinate] {
							min[coordinate] = vertices[i][coordinate]
						}
						if vertices[i][coordinate] > max[coordinate] {
							max[coordinate] = vertices[i][coordinate]
						}
					}
				}
				meshVolume += signedTetra(vertices[0], vertices[1], vertices[2])
				triangles++
			}
			volumeMM3 += math.Abs(meshVolume)
		}
		if object.Components != nil {
			for _, component := range object.Components.Items {
				componentTransform, err := parseThreeMFTransform(component.Transform)
				if err != nil {
					return fmt.Errorf("3MF object %s component transform: %w", objectID, err)
				}
				if err := visitObject(component.ObjectID, multiplyAffine(transform, componentTransform), depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}

	for _, item := range model.Build.Items {
		transform, err := parseThreeMFTransform(item.Transform)
		if err != nil {
			return stlInfo{}, fmt.Errorf("3MF build item transform: %w", err)
		}
		if err := visitObject(item.ObjectID, transform, 0); err != nil {
			return stlInfo{}, err
		}
	}
	if triangles == 0 {
		return stlInfo{}, errors.New("no triangles found")
	}
	return stlInfo{
		X:         max[0] - min[0],
		Y:         max[1] - min[1],
		Z:         max[2] - min[2],
		VolumeCM3: volumeMM3 / 1000,
		Triangles: triangles,
	}, nil
}
