package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/samuel-kreimeyer/Legal/pkg/app"
	"github.com/samuel-kreimeyer/Legal/pkg/parse"
	renderlegal "github.com/samuel-kreimeyer/Legal/pkg/render/legal"
)

type Handler struct {
	pipeline *app.Pipeline
}

func NewHandler() *Handler {
	return &Handler{
		pipeline: app.NewPipeline(),
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.handleIndex)
	mux.HandleFunc("/healthz", h.handleHealthz)
	mux.HandleFunc("/openapi.json", h.handleOpenAPI)
	mux.HandleFunc("/docs", h.handleDocs)
	mux.HandleFunc("/v1/describe", h.handleDescribe)
	return mux
}

type describeRequest struct {
	Format        string               `json:"format"`
	Filename      string               `json:"filename"`
	ContentBase64 string               `json:"content_base64"`
	Parcel        int                  `json:"parcel"`
	All           bool                 `json:"all"`
	RenderOptions renderOptionsRequest `json:"render_options"`
}

type renderOptionsRequest struct {
	Kind           string  `json:"kind"`
	Lot            string  `json:"lot"`
	Block          string  `json:"block"`
	Subdivision    string  `json:"subdivision"`
	City           string  `json:"city"`
	County         string  `json:"county"`
	State          string  `json:"state"`
	StartCorner    string  `json:"start_corner"`
	UseCommencing  bool    `json:"use_commencing"`
	AreaSquareFeet float64 `json:"area_square_feet"`
	AreaUnit       string  `json:"area_unit"`
}

type describeResponse struct {
	Format       string           `json:"format"`
	ParcelsTotal int              `json:"parcels_total"`
	Results      []describeResult `json:"results"`
}

type describeResult struct {
	ParcelIndex    int     `json:"parcel_index"`
	ParcelID       string  `json:"parcel_id"`
	SegmentCount   int     `json:"segment_count"`
	AreaSquareFeet float64 `json:"area_square_feet"`
	Description    string  `json:"description"`
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":    "legal-description-api",
		"version": "0.1.0",
		"links": map[string]string{
			"healthz":      "/healthz",
			"openapi_json": "/openapi.json",
			"docs":         "/docs",
			"describe":     "/v1/describe",
		},
	})
}

func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	spec, err := openAPIJSON()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(spec)
}

func (h *Handler) handleDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	content := `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>Legal Description API Docs</title>
    <style>
      body { font-family: sans-serif; margin: 2rem auto; max-width: 920px; line-height: 1.4; }
      pre { background: #f6f8fa; padding: 1rem; overflow-x: auto; border-radius: 6px; }
      code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
      .route { margin-bottom: 1.2rem; }
    </style>
  </head>
  <body>
    <h1>Legal Description API</h1>
    <p>Machine-readable OpenAPI schema: <a href="/openapi.json">/openapi.json</a></p>
    <h2>Endpoints</h2>
    <div class="route"><code>GET /</code> - service index with discoverable links.</div>
    <div class="route"><code>GET /healthz</code> - health probe.</div>
    <div class="route"><code>GET /openapi.json</code> - OpenAPI 3.1 document.</div>
    <div class="route"><code>POST /v1/describe</code> - render legal descriptions from encoded geometry payload.</div>
    <h2>Example</h2>
    <pre><code>curl -sS -X POST http://localhost:8080/v1/describe \
  -H "Content-Type: application/json" \
  -d '{
    "format": "landxml",
    "filename": "parcel.xml",
    "content_base64": "&lt;base64 bytes&gt;",
    "render_options": {
      "kind": "Utility Easement",
      "lot": "7",
      "block": "B",
      "subdivision": "Sample Addition",
      "city": "North Little Rock",
      "county": "Pulaski",
      "state": "Arkansas"
    }
  }'</code></pre>
  </body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}

func (h *Handler) handleDescribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req describeRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid json payload: %v", err))
		return
	}
	if strings.TrimSpace(req.ContentBase64) == "" {
		writeError(w, http.StatusBadRequest, "content_base64 is required")
		return
	}

	content, err := base64.StdEncoding.DecodeString(req.ContentBase64)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("content_base64 decode failed: %v", err))
		return
	}

	sourceFormat, err := parseRequestFormat(req.Format, req.Filename)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	parcels, err := h.pipeline.ParseReader(context.Background(), bytes.NewReader(content), sourceFormat)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("parse failed: %v", err))
		return
	}
	if len(parcels) == 0 {
		writeError(w, http.StatusBadRequest, "no parcels parsed from payload")
		return
	}

	indices, err := selectedIndices(len(parcels), req.Parcel, req.All)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	renderOpts := renderlegal.Options{
		Kind:            req.RenderOptions.Kind,
		Lot:             req.RenderOptions.Lot,
		Block:           req.RenderOptions.Block,
		Subdivision:     req.RenderOptions.Subdivision,
		City:            req.RenderOptions.City,
		County:          req.RenderOptions.County,
		State:           req.RenderOptions.State,
		StartCorner:     req.RenderOptions.StartCorner,
		UseCommencing:   req.RenderOptions.UseCommencing,
		AreaSquareFeet:  req.RenderOptions.AreaSquareFeet,
		AreaDisplayUnit: req.RenderOptions.AreaUnit,
	}

	results := make([]describeResult, 0, len(indices))
	for _, idx := range indices {
		text, err := renderlegal.RenderParcel(parcels[idx], renderOpts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("render failed for parcel %d: %v", idx+1, err))
			return
		}
		results = append(results, describeResult{
			ParcelIndex:    idx + 1,
			ParcelID:       parcels[idx].ID,
			SegmentCount:   len(parcels[idx].Loop.Segments),
			AreaSquareFeet: parcels[idx].AreaSquareFeet,
			Description:    text,
		})
	}

	writeJSON(w, http.StatusOK, describeResponse{
		Format:       formatLabel(sourceFormat),
		ParcelsTotal: len(parcels),
		Results:      results,
	})
}

func parseRequestFormat(requestFormat, filename string) (parse.SourceFormat, error) {
	switch strings.ToLower(strings.TrimSpace(requestFormat)) {
	case "", "auto":
		if strings.TrimSpace(filename) == "" {
			return parse.SourceFormatUnknown, fmt.Errorf("filename is required when format is auto")
		}
		f, err := parse.DetectSourceFormat(filename)
		if err != nil {
			return parse.SourceFormatUnknown, err
		}
		return f, nil
	case "dxf":
		return parse.SourceFormatDXF, nil
	case "ifc":
		return parse.SourceFormatIFC, nil
	case "landxml", "xml":
		return parse.SourceFormatLandXML, nil
	default:
		return parse.SourceFormatUnknown, fmt.Errorf("invalid format %q", requestFormat)
	}
}

func selectedIndices(total, parcel int, all bool) ([]int, error) {
	if all {
		out := make([]int, 0, total)
		for i := 0; i < total; i++ {
			out = append(out, i)
		}
		return out, nil
	}
	if parcel == 0 {
		parcel = 1
	}
	if parcel < 1 || parcel > total {
		return nil, fmt.Errorf("parcel must be 1..%d", total)
	}
	return []int{parcel - 1}, nil
}

func formatLabel(f parse.SourceFormat) string {
	switch f {
	case parse.SourceFormatDXF:
		return "dxf"
	case parse.SourceFormatIFC:
		return "ifc"
	case parse.SourceFormatLandXML:
		return "landxml"
	default:
		return "unknown"
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": html.EscapeString(message)})
}
