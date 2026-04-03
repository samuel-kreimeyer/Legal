package dxf

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/geom"
	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/model"
)

func TestParseSquareLinesDXF(t *testing.T) {
	fixture := filepath.Join("..", "..", "..", "tests", "fixtures", "dxf", "square_lines.dxf")
	f, err := os.Open(fixture)
	if err != nil {
		t.Fatalf("open fixture failed: %v", err)
	}
	defer f.Close()

	p := NewParser()
	parcels, err := p.Parse(context.Background(), f)
	if err != nil {
		t.Fatalf("DXF parse failed: %v", err)
	}
	if len(parcels) != 1 {
		t.Fatalf("expected 1 parcel, got %d", len(parcels))
	}
	if parcels[0].Source != model.SourceDXF {
		t.Fatalf("expected DXF source, got %q", parcels[0].Source)
	}
	if len(parcels[0].Loop.Segments) != 4 {
		t.Fatalf("expected 4 segments, got %d", len(parcels[0].Loop.Segments))
	}

	start := parcels[0].Loop.Segments[0].Start()
	end := parcels[0].Loop.Segments[len(parcels[0].Loop.Segments)-1].End()
	if geom.DistanceFeet(start, end) > geom.EpsilonFeet {
		t.Fatalf("expected closed loop within epsilon; distance=%f", geom.DistanceFeet(start, end))
	}
}

func TestParseOpenLineworkReturnsError(t *testing.T) {
	src := `
0
SECTION
2
ENTITIES
0
LINE
8
BOUNDARY
10
0
20
0
11
10
21
0
0
LINE
8
BOUNDARY
10
10
20
0
11
10
21
10
0
ENDSEC
0
EOF
`
	p := NewParser()
	_, err := p.Parse(context.Background(), strings.NewReader(src))
	if err == nil {
		t.Fatal("expected parse error for non-closed linework")
	}
}

// dxfSquare builds a minimal DXF string containing four LINE entities that form
// a square. Each edge can be assigned an arbitrary layer name.
func dxfSquare(bottom, right, top, left string) string {
	return "0\nSECTION\n2\nHEADER\n9\n$INSUNITS\n70\n2\n0\nENDSEC\n" +
		"0\nSECTION\n2\nENTITIES\n" +
		"0\nLINE\n8\n" + bottom + "\n10\n0\n20\n0\n11\n100\n21\n0\n" +
		"0\nLINE\n8\n" + right + "\n10\n100\n20\n0\n11\n100\n21\n100\n" +
		"0\nLINE\n8\n" + top + "\n10\n100\n20\n100\n11\n0\n21\n100\n" +
		"0\nLINE\n8\n" + left + "\n10\n0\n20\n100\n11\n0\n21\n0\n" +
		"0\nENDSEC\n0\nEOF\n"
}

func TestLayerFilter_SingleLayer_IncludesMatchingExcludesOther(t *testing.T) {
	// Two independent closed squares on different layers.
	src := dxfSquare("PROP", "PROP", "PROP", "PROP") +
		// append a second complete square on ROW layer — build manually to avoid
		// duplicate SECTION headers; instead put all entities in one file.
		""
	// Build a file with two squares on different layers.
	twoLayer := "0\nSECTION\n2\nHEADER\n9\n$INSUNITS\n70\n2\n0\nENDSEC\n" +
		"0\nSECTION\n2\nENTITIES\n" +
		// Square 1 on PROP
		"0\nLINE\n8\nPROP\n10\n0\n20\n0\n11\n100\n21\n0\n" +
		"0\nLINE\n8\nPROP\n10\n100\n20\n0\n11\n100\n21\n100\n" +
		"0\nLINE\n8\nPROP\n10\n100\n20\n100\n11\n0\n21\n100\n" +
		"0\nLINE\n8\nPROP\n10\n0\n20\n100\n11\n0\n21\n0\n" +
		// Square 2 on ROW (offset to avoid shared vertices)
		"0\nLINE\n8\nROW\n10\n200\n20\n0\n11\n300\n21\n0\n" +
		"0\nLINE\n8\nROW\n10\n300\n20\n0\n11\n300\n21\n100\n" +
		"0\nLINE\n8\nROW\n10\n300\n20\n100\n11\n200\n21\n100\n" +
		"0\nLINE\n8\nROW\n10\n200\n20\n100\n11\n200\n21\n0\n" +
		"0\nENDSEC\n0\nEOF\n"
	_ = src

	// No filter: expect 2 parcels.
	p := NewParser()
	parcels, err := p.Parse(context.Background(), strings.NewReader(twoLayer))
	if err != nil {
		t.Fatalf("unfiltered parse failed: %v", err)
	}
	if len(parcels) != 2 {
		t.Fatalf("unfiltered: expected 2 parcels, got %d", len(parcels))
	}

	// Filter to PROP only: expect 1 parcel.
	pFiltered := NewParserWithOptions(Options{LayerFilter: []string{"PROP"}})
	parcels, err = pFiltered.Parse(context.Background(), strings.NewReader(twoLayer))
	if err != nil {
		t.Fatalf("filtered parse failed: %v", err)
	}
	if len(parcels) != 1 {
		t.Fatalf("PROP filter: expected 1 parcel, got %d", len(parcels))
	}
}

func TestLayerFilter_CaseInsensitive(t *testing.T) {
	src := dxfSquare("Boundary", "Boundary", "Boundary", "Boundary")

	pLower := NewParserWithOptions(Options{LayerFilter: []string{"boundary"}})
	parcels, err := pLower.Parse(context.Background(), strings.NewReader(src))
	if err != nil {
		t.Fatalf("case-insensitive filter failed: %v", err)
	}
	if len(parcels) != 1 {
		t.Fatalf("expected 1 parcel, got %d", len(parcels))
	}
}

func TestLayerFilter_MultiLayer_AssemblesAcrossLayers(t *testing.T) {
	// Three sides of a square on PROP, one side on ROW.
	// Neither layer alone can close, but together they form a complete square.
	split := "0\nSECTION\n2\nHEADER\n9\n$INSUNITS\n70\n2\n0\nENDSEC\n" +
		"0\nSECTION\n2\nENTITIES\n" +
		// PROP contributes bottom, right, top
		"0\nLINE\n8\nPROP\n10\n0\n20\n0\n11\n100\n21\n0\n" +
		"0\nLINE\n8\nPROP\n10\n100\n20\n0\n11\n100\n21\n100\n" +
		"0\nLINE\n8\nPROP\n10\n100\n20\n100\n11\n0\n21\n100\n" +
		// ROW contributes the closing left edge
		"0\nLINE\n8\nROW\n10\n0\n20\n100\n11\n0\n21\n0\n" +
		"0\nENDSEC\n0\nEOF\n"

	p := NewParserWithOptions(Options{LayerFilter: []string{"PROP", "ROW"}})
	parcels, err := p.Parse(context.Background(), strings.NewReader(split))
	if err != nil {
		t.Fatalf("multi-layer parse failed: %v", err)
	}
	if len(parcels) != 1 {
		t.Fatalf("expected 1 merged parcel, got %d", len(parcels))
	}
	if len(parcels[0].Loop.Segments) != 4 {
		t.Fatalf("expected 4 segments in merged loop, got %d", len(parcels[0].Loop.Segments))
	}
	start := parcels[0].Loop.Segments[0].Start()
	end := parcels[0].Loop.Segments[len(parcels[0].Loop.Segments)-1].End()
	if geom.DistanceFeet(start, end) > geom.EpsilonFeet {
		t.Fatalf("merged loop is not closed; gap = %f ft", geom.DistanceFeet(start, end))
	}
}

func TestLayerFilter_MultiLayer_IndependentLoopsStaySeparate(t *testing.T) {
	// Each layer has its own complete square. Multi-layer mode should still
	// produce two independent parcels.
	twoComplete := "0\nSECTION\n2\nHEADER\n9\n$INSUNITS\n70\n2\n0\nENDSEC\n" +
		"0\nSECTION\n2\nENTITIES\n" +
		"0\nLINE\n8\nPROP\n10\n0\n20\n0\n11\n100\n21\n0\n" +
		"0\nLINE\n8\nPROP\n10\n100\n20\n0\n11\n100\n21\n100\n" +
		"0\nLINE\n8\nPROP\n10\n100\n20\n100\n11\n0\n21\n100\n" +
		"0\nLINE\n8\nPROP\n10\n0\n20\n100\n11\n0\n21\n0\n" +
		"0\nLINE\n8\nROW\n10\n200\n20\n0\n11\n300\n21\n0\n" +
		"0\nLINE\n8\nROW\n10\n300\n20\n0\n11\n300\n21\n100\n" +
		"0\nLINE\n8\nROW\n10\n300\n20\n100\n11\n200\n21\n100\n" +
		"0\nLINE\n8\nROW\n10\n200\n20\n100\n11\n200\n21\n0\n" +
		"0\nENDSEC\n0\nEOF\n"

	p := NewParserWithOptions(Options{LayerFilter: []string{"PROP", "ROW"}})
	parcels, err := p.Parse(context.Background(), strings.NewReader(twoComplete))
	if err != nil {
		t.Fatalf("multi-layer separate parse failed: %v", err)
	}
	if len(parcels) != 2 {
		t.Fatalf("expected 2 independent parcels, got %d", len(parcels))
	}
}

func TestSegmentFromVertexPair_BulgeCreatesArc(t *testing.T) {
	seg, err := segmentFromVertexPair(
		dxfVertex{x: 0, y: 0, bulge: 0.5, hasX: true, hasY: true},
		dxfVertex{x: 10, y: 0, hasX: true, hasY: true},
		1.0,
	)
	if err != nil {
		t.Fatalf("segmentFromVertexPair failed: %v", err)
	}

	arc, ok := seg.(model.ArcSegment)
	if !ok {
		t.Fatalf("expected arc segment from bulge, got %T", seg)
	}
	if math.Abs(arc.SweepRad) <= 1e-9 {
		t.Fatalf("expected non-zero arc sweep, got %f", arc.SweepRad)
	}
	if geom.DistanceFeet(arc.Start(), geom.Point2D{XFeet: 0, YFeet: 0}) > geom.EpsilonFeet {
		t.Fatalf("arc start does not match input vertex")
	}
	if geom.DistanceFeet(arc.End(), geom.Point2D{XFeet: 10, YFeet: 0}) > geom.EpsilonFeet {
		t.Fatalf("arc end does not match input vertex")
	}
}
