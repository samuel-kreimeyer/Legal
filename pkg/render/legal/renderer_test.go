package legal

import (
	"strings"
	"testing"

	"github.com/samuel-kreimeyer/Legal/pkg/geom"
	"github.com/samuel-kreimeyer/Legal/pkg/model"
)

func TestRenderParcel_UsesTangencyFromPreviousSegment(t *testing.T) {
	parcel := model.Parcel{
		ID: "p",
		Loop: model.BoundaryLoop{
			Segments: []model.Segment{
				model.LineSegment{P0: geom.Point2D{XFeet: 0, YFeet: 0}, P1: geom.Point2D{XFeet: 5, YFeet: 0}},
				model.LineSegment{P0: geom.Point2D{XFeet: 5, YFeet: 0}, P1: geom.Point2D{XFeet: 10, YFeet: 0}},
				model.LineSegment{P0: geom.Point2D{XFeet: 10, YFeet: 0}, P1: geom.Point2D{XFeet: 10, YFeet: 5}},
				model.LineSegment{P0: geom.Point2D{XFeet: 10, YFeet: 5}, P1: geom.Point2D{XFeet: 0, YFeet: 5}},
				model.LineSegment{P0: geom.Point2D{XFeet: 0, YFeet: 5}, P1: geom.Point2D{XFeet: 0, YFeet: 0}},
			},
		},
	}

	out, err := RenderParcel(parcel, Options{
		Kind:            "Test",
		County:          "Pulaski",
		State:           "Arkansas",
		AreaSquareFeet:  50.0,
		AreaDisplayUnit: "square feet",
	})
	if err != nil {
		t.Fatalf("RenderParcel failed: %v", err)
	}
	if !strings.Contains(out, "TO A POINT OF TANGENCY;") {
		t.Fatalf("expected tangency phrase in output:\n%s", out)
	}
	if !strings.Contains(out, "TO A POINT OF NON-TANGENCY;") {
		t.Fatalf("expected non-tangency phrase in output:\n%s", out)
	}
}

func TestRenderParcel_RendersArcSegment(t *testing.T) {
	arc := model.ArcSegment{
		Center:        geom.Point2D{XFeet: 5, YFeet: 0},
		RadiusFeet:    5,
		StartAngleRad: 0,
		SweepRad:      1.5707963267948966, // 90 deg
	}
	parcel := model.Parcel{
		ID: "arc-p",
		Loop: model.BoundaryLoop{
			Segments: []model.Segment{
				model.LineSegment{P0: geom.Point2D{XFeet: 0, YFeet: 0}, P1: geom.Point2D{XFeet: 10, YFeet: 0}},
				arc,
				model.LineSegment{P0: arc.End(), P1: geom.Point2D{XFeet: 0, YFeet: 0}},
			},
		},
	}
	out, err := RenderParcel(parcel, Options{
		Kind:            "Test",
		County:          "Pulaski",
		State:           "Arkansas",
		AreaSquareFeet:  10.0,
		AreaDisplayUnit: "square feet",
	})
	if err != nil {
		t.Fatalf("RenderParcel failed: %v", err)
	}
	if !strings.Contains(out, "ALONG A CURVE TO THE LEFT") {
		t.Fatalf("expected arc phrase in output:\n%s", out)
	}
	if !strings.Contains(out, "WITH TANGENT BEARING") {
		t.Fatalf("expected tangent-bearing phrase in arc output:\n%s", out)
	}
}
