package dxf

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samuel-kreimeyer/Legal/pkg/geom"
	"github.com/samuel-kreimeyer/Legal/pkg/model"
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
