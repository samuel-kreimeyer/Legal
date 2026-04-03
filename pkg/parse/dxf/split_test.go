package dxf

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/geom"
	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/model"
)

// --- intersectLineLine ---

func TestIntersectLineLine_Crossing(t *testing.T) {
	a := model.LineSegment{
		P0: geom.Point2D{XFeet: 0, YFeet: 50},
		P1: geom.Point2D{XFeet: 100, YFeet: 50},
	}
	b := model.LineSegment{
		P0: geom.Point2D{XFeet: 50, YFeet: 0},
		P1: geom.Point2D{XFeet: 50, YFeet: 100},
	}
	tsA, tsB := intersectLineLine(a, b)
	if len(tsA) != 1 || len(tsB) != 1 {
		t.Fatalf("expected 1 intersection, got tsA=%v tsB=%v", tsA, tsB)
	}
	if math.Abs(tsA[0]-0.5) > 1e-9 {
		t.Errorf("tsA[0] = %f, want 0.5", tsA[0])
	}
	if math.Abs(tsB[0]-0.5) > 1e-9 {
		t.Errorf("tsB[0] = %f, want 0.5", tsB[0])
	}
}

func TestIntersectLineLine_Parallel(t *testing.T) {
	a := model.LineSegment{P0: geom.Point2D{XFeet: 0}, P1: geom.Point2D{XFeet: 100}}
	b := model.LineSegment{
		P0: geom.Point2D{XFeet: 0, YFeet: 10},
		P1: geom.Point2D{XFeet: 100, YFeet: 10},
	}
	tsA, tsB := intersectLineLine(a, b)
	if len(tsA) != 0 || len(tsB) != 0 {
		t.Errorf("parallel lines should have no intersection, got %v %v", tsA, tsB)
	}
}

func TestIntersectLineLine_EndpointNotCounted(t *testing.T) {
	// Segments that share an endpoint — the shared point is at t=0 or t=1 and
	// should NOT be recorded as a split point.
	a := model.LineSegment{
		P0: geom.Point2D{XFeet: 0, YFeet: 0},
		P1: geom.Point2D{XFeet: 100, YFeet: 0},
	}
	b := model.LineSegment{
		P0: geom.Point2D{XFeet: 100, YFeet: 0},
		P1: geom.Point2D{XFeet: 100, YFeet: 100},
	}
	tsA, tsB := intersectLineLine(a, b)
	// t=1 on a and t=0 on b are at the boundary of inOpen, so should be excluded.
	for _, tv := range tsA {
		if !inOpen(tv) {
			t.Errorf("non-open t value on a: %f", tv)
		}
	}
	_ = tsB
}

// --- splitLine ---

func TestSplitLine_SingleSplit(t *testing.T) {
	seg := model.LineSegment{
		P0: geom.Point2D{XFeet: 0, YFeet: 0},
		P1: geom.Point2D{XFeet: 100, YFeet: 0},
	}
	parts := splitLine(seg, []float64{0.5})
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	p0 := parts[0].(model.LineSegment)
	p1 := parts[1].(model.LineSegment)
	if math.Abs(p0.P1.XFeet-50) > 1e-9 {
		t.Errorf("split point X: got %f, want 50", p0.P1.XFeet)
	}
	if geom.DistanceFeet(p0.P1, p1.P0) > geom.EpsilonFeet {
		t.Errorf("split point mismatch: %v vs %v", p0.P1, p1.P0)
	}
}

func TestSplitLine_MultiSplit(t *testing.T) {
	seg := model.LineSegment{
		P0: geom.Point2D{XFeet: 0, YFeet: 0},
		P1: geom.Point2D{XFeet: 100, YFeet: 0},
	}
	parts := splitLine(seg, []float64{0.25, 0.75})
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}
}

// --- splitArc ---

func TestSplitArc_HalfwayThrough(t *testing.T) {
	// 90° CCW arc from angle 0 to π/2.
	seg := model.ArcSegment{
		Center:        geom.Point2D{XFeet: 0, YFeet: 0},
		RadiusFeet:    10,
		StartAngleRad: 0,
		SweepRad:      math.Pi / 2,
	}
	parts := splitArc(seg, []float64{0.5})
	if len(parts) != 2 {
		t.Fatalf("expected 2 arc parts, got %d", len(parts))
	}
	sub0 := parts[0].(model.ArcSegment)
	sub1 := parts[1].(model.ArcSegment)

	// Each sub-arc should span π/4.
	if math.Abs(sub0.SweepRad-math.Pi/4) > 1e-9 {
		t.Errorf("sub0 sweep: got %f, want %f", sub0.SweepRad, math.Pi/4)
	}
	if math.Abs(sub1.SweepRad-math.Pi/4) > 1e-9 {
		t.Errorf("sub1 sweep: got %f, want %f", sub1.SweepRad, math.Pi/4)
	}
	// The end of sub0 should be the start of sub1.
	if geom.DistanceFeet(sub0.End(), sub1.Start()) > geom.EpsilonFeet {
		t.Errorf("arc sub-segment continuity failure: %v vs %v", sub0.End(), sub1.Start())
	}
}

// --- splitAtCrossLayerIntersections ---

func TestSplitAtCrossLayer_TwoCrossingLines(t *testing.T) {
	// Horizontal line on layer A, vertical line on layer B — cross at (50,50).
	byLayer := map[string][]model.Segment{
		"A": {model.LineSegment{P0: geom.Point2D{XFeet: 0, YFeet: 50}, P1: geom.Point2D{XFeet: 100, YFeet: 50}}},
		"B": {model.LineSegment{P0: geom.Point2D{XFeet: 50, YFeet: 0}, P1: geom.Point2D{XFeet: 50, YFeet: 100}}},
	}
	out := splitAtCrossLayerIntersections(byLayer, []string{"A", "B"})
	// A splits into 2, B splits into 2 → 4 segments total.
	if len(out) != 4 {
		t.Fatalf("expected 4 sub-segments, got %d", len(out))
	}
}

func TestSplitAtCrossLayer_NoIntersection(t *testing.T) {
	// Parallel horizontal lines — no cross-layer intersection.
	byLayer := map[string][]model.Segment{
		"A": {model.LineSegment{P0: geom.Point2D{XFeet: 0}, P1: geom.Point2D{XFeet: 100}}},
		"B": {model.LineSegment{P0: geom.Point2D{XFeet: 0, YFeet: 50}, P1: geom.Point2D{XFeet: 100, YFeet: 50}}},
	}
	out := splitAtCrossLayerIntersections(byLayer, []string{"A", "B"})
	// No splits: 2 segments returned unchanged.
	if len(out) != 2 {
		t.Fatalf("expected 2 segments (no splits), got %d", len(out))
	}
}

// --- buildClosedLoopsBestEffort ---

func TestBestEffort_FormsValidLoops(t *testing.T) {
	// A square of 4 lines — should form one closed loop.
	segs := []model.Segment{
		model.LineSegment{P0: geom.Point2D{XFeet: 0}, P1: geom.Point2D{XFeet: 100}},
		model.LineSegment{P0: geom.Point2D{XFeet: 100}, P1: geom.Point2D{XFeet: 100, YFeet: 100}},
		model.LineSegment{P0: geom.Point2D{XFeet: 100, YFeet: 100}, P1: geom.Point2D{XFeet: 0, YFeet: 100}},
		model.LineSegment{P0: geom.Point2D{XFeet: 0, YFeet: 100}, P1: geom.Point2D{XFeet: 0}},
	}
	loops := buildClosedLoopsBestEffort(segs)
	if len(loops) != 1 {
		t.Fatalf("expected 1 loop, got %d", len(loops))
	}
	if len(loops[0].Segments) != 4 {
		t.Errorf("expected 4 segments, got %d", len(loops[0].Segments))
	}
}

func TestBestEffort_DiscardsOrphanedSegments(t *testing.T) {
	// 4 segments of a square + 1 orphan segment that can't close.
	segs := []model.Segment{
		model.LineSegment{P0: geom.Point2D{XFeet: 0}, P1: geom.Point2D{XFeet: 100}},
		model.LineSegment{P0: geom.Point2D{XFeet: 100}, P1: geom.Point2D{XFeet: 100, YFeet: 100}},
		model.LineSegment{P0: geom.Point2D{XFeet: 100, YFeet: 100}, P1: geom.Point2D{XFeet: 0, YFeet: 100}},
		model.LineSegment{P0: geom.Point2D{XFeet: 0, YFeet: 100}, P1: geom.Point2D{XFeet: 0}},
		// Orphan — connects nowhere
		model.LineSegment{P0: geom.Point2D{XFeet: 500}, P1: geom.Point2D{XFeet: 600}},
	}
	loops := buildClosedLoopsBestEffort(segs)
	// Should find exactly the one closed square; orphan silently discarded.
	if len(loops) != 1 {
		t.Fatalf("expected 1 loop (orphan discarded), got %d", len(loops))
	}
}

// --- end-to-end: ROW line cuts through property boundary at non-corner points ---

// dxfTwoLayerCrossing builds a DXF where:
//
//	PROP layer: a 100×100 square (closed polygon via 4 LINE entities)
//	ROW layer:  a diagonal from (50,0) to (100,50) — both endpoints lie on the
//	            PROP boundary at interior (non-corner) positions
//
// After intersection splitting and multi-layer assembly the expected result is
// the triangular acquisition parcel with vertices (50,0), (100,0), (100,50).
const dxfTwoLayerCrossing = `0
SECTION
2
HEADER
9
$INSUNITS
70
2
0
ENDSEC
0
SECTION
2
ENTITIES
0
LINE
8
PROP
10
0
20
0
11
100
21
0
0
LINE
8
PROP
10
100
20
0
11
100
21
100
0
LINE
8
PROP
10
100
20
100
11
0
21
100
0
LINE
8
PROP
10
0
20
100
11
0
21
0
0
LINE
8
ROW
10
50
20
0
11
100
21
50
0
ENDSEC
0
EOF
`

func TestMultiLayer_ROWCutsThroughPropertyAtNonCorner(t *testing.T) {
	p := NewParserWithOptions(Options{LayerFilter: []string{"PROP", "ROW"}})
	parcels, err := p.Parse(context.Background(), strings.NewReader(dxfTwoLayerCrossing))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(parcels) == 0 {
		t.Fatal("expected at least 1 parcel")
	}

	// The first parcel should be the triangular acquisition area.
	// Verify it's closed.
	loop := parcels[0].Loop
	if len(loop.Segments) < 3 {
		t.Fatalf("expected at least 3 segments in acquisition parcel, got %d", len(loop.Segments))
	}
	start := loop.Segments[0].Start()
	end := loop.Segments[len(loop.Segments)-1].End()
	if geom.DistanceFeet(start, end) > geom.EpsilonFeet {
		t.Errorf("acquisition parcel is not closed: gap = %f ft", geom.DistanceFeet(start, end))
	}

	// Verify the parcel contains all three expected vertices.
	wantVertices := []geom.Point2D{
		{XFeet: 50, YFeet: 0},
		{XFeet: 100, YFeet: 0},
		{XFeet: 100, YFeet: 50},
	}
	for _, want := range wantVertices {
		found := false
		for _, seg := range loop.Segments {
			if geom.DistanceFeet(seg.Start(), want) <= geom.EpsilonFeet {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("acquisition parcel missing expected vertex %v", want)
		}
	}
}
