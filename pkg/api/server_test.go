package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenAPIEndpoint(t *testing.T) {
	handler := NewHandler().Routes()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var doc map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&doc); err != nil {
		t.Fatalf("decode openapi failed: %v", err)
	}
	if doc["openapi"] != "3.1.0" {
		t.Fatalf("unexpected openapi version: %v", doc["openapi"])
	}
}

func TestDiscoverableDocsEndpoints(t *testing.T) {
	handler := NewHandler().Routes()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from /, got %d", rr.Code)
	}
	var idx map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&idx); err != nil {
		t.Fatalf("decode index failed: %v", err)
	}
	links, ok := idx["links"].(map[string]any)
	if !ok {
		t.Fatalf("index links missing")
	}
	if links["openapi_json"] != "/openapi.json" || links["docs"] != "/docs" {
		t.Fatalf("index links do not expose docs/openapi endpoints: %v", links)
	}

	docsReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	docsRR := httptest.NewRecorder()
	handler.ServeHTTP(docsRR, docsReq)
	if docsRR.Code != http.StatusOK {
		t.Fatalf("expected 200 from /docs, got %d", docsRR.Code)
	}
	if !strings.Contains(docsRR.Body.String(), "/openapi.json") {
		t.Fatalf("docs page does not mention /openapi.json")
	}
}

func TestDescribeLandXMLSuccess(t *testing.T) {
	fixture := filepath.Join("..", "..", "tests", "fixtures", "landxml", "square.xml")
	src, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture failed: %v", err)
	}

	reqBody := map[string]any{
		"format":         "landxml",
		"filename":       "square.xml",
		"content_base64": base64.StdEncoding.EncodeToString(src),
		"render_options": map[string]any{
			"kind":        "Utility Easement",
			"lot":         "7",
			"block":       "B",
			"subdivision": "Sample Addition",
			"city":        "North Little Rock",
			"county":      "Pulaski",
			"state":       "Arkansas",
		},
	}
	b, _ := json.Marshal(reqBody)

	handler := NewHandler().Routes()
	req := httptest.NewRequest(http.MethodPost, "/v1/describe", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		raw, _ := io.ReadAll(rr.Body)
		t.Fatalf("expected 200, got %d: %s", rr.Code, string(raw))
	}

	var out describeResponse
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if out.ParcelsTotal != 1 || len(out.Results) != 1 {
		t.Fatalf("unexpected parcel counts: %+v", out)
	}
	if !strings.Contains(out.Results[0].Description, "THENCE") {
		t.Fatalf("response description missing THENCE lines")
	}
}

func TestDescribeAutoWithoutFilenameReturns400(t *testing.T) {
	reqBody := map[string]any{
		"format":         "auto",
		"content_base64": base64.StdEncoding.EncodeToString([]byte("dummy")),
	}
	b, _ := json.Marshal(reqBody)

	handler := NewHandler().Routes()
	req := httptest.NewRequest(http.MethodPost, "/v1/describe", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
