package normalize

import (
	"testing"

	"github.com/samuel-kreimeyer/Legal/pkg/geom"
	"github.com/samuel-kreimeyer/Legal/pkg/model"
)

func TestNormalizeParcel_RepairsDirectionAndNormalizesOrientation(t *testing.T) {
	// Clockwise loop with one segment reversed.
	segments := []model.Segment{
		model.LineSegment{P0: geom.Point2D{XFeet: 0, YFeet: 0}, P1: geom.Point2D{XFeet: 0, YFeet: 10}},
		model.LineSegment{P0: geom.Point2D{XFeet: 10, YFeet: 10}, P1: geom.Point2D{XFeet: 0, YFeet: 10}}, // reversed
		model.LineSegment{P0: geom.Point2D{XFeet: 10, YFeet: 10}, P1: geom.Point2D{XFeet: 10, YFeet: 0}},
		model.LineSegment{P0: geom.Point2D{XFeet: 10, YFeet: 0}, P1: geom.Point2D{XFeet: 0, YFeet: 0}},
	}
	parcel := model.Parcel{
		ID:   "p1",
		Loop: model.BoundaryLoop{Segments: segments},
	}

	norm, err := NormalizeParcel(parcel)
	if err != nil {
		t.Fatalf("NormalizeParcel failed: %v", err)
	}
	if err := ValidateClosedLoop(norm.Loop); err != nil {
		t.Fatalf("normalized loop invalid: %v", err)
	}
	if area := signedAreaApprox(norm.Loop); area >= 0.0 {
		t.Fatalf("expected CW normalized loop with negative area, got %f", area)
	}
}

func TestNormalizeParcel_RejectsZeroAreaLoop(t *testing.T) {
	// Degenerate closed loop on a line.
	segments := []model.Segment{
		model.LineSegment{P0: geom.Point2D{XFeet: 0, YFeet: 0}, P1: geom.Point2D{XFeet: 10, YFeet: 0}},
		model.LineSegment{P0: geom.Point2D{XFeet: 10, YFeet: 0}, P1: geom.Point2D{XFeet: 20, YFeet: 0}},
		model.LineSegment{P0: geom.Point2D{XFeet: 20, YFeet: 0}, P1: geom.Point2D{XFeet: 0, YFeet: 0}},
	}
	parcel := model.Parcel{
		ID:   "p2",
		Loop: model.BoundaryLoop{Segments: segments},
	}

	_, err := NormalizeParcel(parcel)
	if err == nil {
		t.Fatal("expected zero-area loop to fail normalization")
	}
}
