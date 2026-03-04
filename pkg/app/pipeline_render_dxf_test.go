package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samuel-kreimeyer/Legal/pkg/parse"
	renderlegal "github.com/samuel-kreimeyer/Legal/pkg/render/legal"
)

func TestPipeline_ParseNormalizeRender_DXFFixture(t *testing.T) {
	fixture := filepath.Join("..", "..", "tests", "fixtures", "dxf", "square_lines.dxf")
	golden := filepath.Join("..", "..", "tests", "golden", "square_description.txt")

	p := NewPipeline()
	parcels, err := p.ParseFile(context.Background(), fixture, parse.SourceFormatDXF)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if len(parcels) != 1 {
		t.Fatalf("expected exactly 1 parcel, got %d", len(parcels))
	}

	got, err := renderlegal.RenderParcel(parcels[0], renderlegal.Options{
		Kind:            "Utility Easement",
		Lot:             "7",
		Block:           "B",
		Subdivision:     "Sample Addition",
		City:            "North Little Rock",
		County:          "Pulaski",
		State:           "Arkansas",
		StartCorner:     "Northeast",
		UseCommencing:   false,
		AreaSquareFeet:  5000.0,
		AreaDisplayUnit: "square feet",
	})
	if err != nil {
		t.Fatalf("RenderParcel failed: %v", err)
	}

	wantBytes, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden failed: %v", err)
	}
	want := strings.TrimSpace(string(wantBytes))
	if strings.TrimSpace(got) != want {
		t.Fatalf("render mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
