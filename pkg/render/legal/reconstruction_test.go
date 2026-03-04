package legal

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/samuel-kreimeyer/Legal/pkg/geom"
	"github.com/samuel-kreimeyer/Legal/pkg/model"
)

var (
	lineRe = regexp.MustCompile(`^THENCE ([NS]) ([0-9]+)°([0-9]+)'([0-9]+(?:\.[0-9]+)?)" ([EW]), A DISTANCE OF ([0-9]+(?:\.[0-9]+)?) FEET;$`)
	arcRe  = regexp.MustCompile(`^THENCE ALONG A CURVE TO THE (LEFT|RIGHT), WITH TANGENT BEARING ([NS]) ([0-9]+)°([0-9]+)'([0-9]+(?:\.[0-9]+)?)" ([EW]), HAVING A RADIUS OF ([0-9]+(?:\.[0-9]+)?) FEET, A CENTRAL ANGLE OF ([0-9]+)°([0-9]+)'([0-9]+(?:\.[0-9]+)?)", AN ARC DISTANCE OF ([0-9]+(?:\.[0-9]+)?) FEET;$`)
)

func TestRenderParcel_ReconstructsGeometryFromKnownStart(t *testing.T) {
	parcel := model.Parcel{
		ID: "square",
		Loop: model.BoundaryLoop{
			Segments: []model.Segment{
				model.LineSegment{P0: geom.Point2D{XFeet: 0, YFeet: 0}, P1: geom.Point2D{XFeet: 0, YFeet: 50}},
				model.LineSegment{P0: geom.Point2D{XFeet: 0, YFeet: 50}, P1: geom.Point2D{XFeet: 100, YFeet: 50}},
				model.LineSegment{P0: geom.Point2D{XFeet: 100, YFeet: 50}, P1: geom.Point2D{XFeet: 100, YFeet: 0}},
				model.LineSegment{P0: geom.Point2D{XFeet: 100, YFeet: 0}, P1: geom.Point2D{XFeet: 0, YFeet: 0}},
			},
		},
	}
	out, err := RenderParcel(parcel, Options{
		Kind:            "Test",
		County:          "Pulaski",
		State:           "Arkansas",
		AreaSquareFeet:  5000.0,
		AreaDisplayUnit: "square feet",
	})
	if err != nil {
		t.Fatalf("RenderParcel failed: %v", err)
	}

	recon, err := reconstructEndpointsFromDescription(out, parcel.Loop.Segments[0].Start())
	if err != nil {
		t.Fatalf("reconstruction failed: %v\n%s", err, out)
	}
	assertEndpointsMatch(t, recon, parcel.Loop.Segments, 0.01)
}

func TestRenderParcel_ReconstructsWhenFirstSegmentIsArc(t *testing.T) {
	arc := model.ArcSegment{
		Center:        geom.Point2D{XFeet: 0, YFeet: 5},
		RadiusFeet:    5,
		StartAngleRad: -math.Pi / 2.0,
		SweepRad:      math.Pi / 2.0,
	}
	parcel := model.Parcel{
		ID: "arc-first",
		Loop: model.BoundaryLoop{
			Segments: []model.Segment{
				arc,
				model.LineSegment{P0: arc.End(), P1: geom.Point2D{XFeet: 10, YFeet: 5}},
				model.LineSegment{P0: geom.Point2D{XFeet: 10, YFeet: 5}, P1: geom.Point2D{XFeet: 10, YFeet: 0}},
				model.LineSegment{P0: geom.Point2D{XFeet: 10, YFeet: 0}, P1: geom.Point2D{XFeet: 0, YFeet: 0}},
			},
		},
	}
	out, err := RenderParcel(parcel, Options{
		Kind:            "Test",
		County:          "Pulaski",
		State:           "Arkansas",
		AreaSquareFeet:  62.5,
		AreaDisplayUnit: "square feet",
	})
	if err != nil {
		t.Fatalf("RenderParcel failed: %v", err)
	}
	if !strings.Contains(out, "WITH TANGENT BEARING") {
		t.Fatalf("expected tangent-bearing data in arc output:\n%s", out)
	}

	recon, err := reconstructEndpointsFromDescription(out, parcel.Loop.Segments[0].Start())
	if err != nil {
		t.Fatalf("reconstruction failed: %v\n%s", err, out)
	}
	assertEndpointsMatch(t, recon, parcel.Loop.Segments, 0.01)
}

func reconstructEndpointsFromDescription(description string, start geom.Point2D) ([]geom.Point2D, error) {
	lines := strings.Split(description, "\n")
	endpoints := make([]geom.Point2D, 0, 8)
	curr := start

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "THENCE ") {
			continue
		}

		if m := lineRe.FindStringSubmatch(line); m != nil {
			theta, err := parseBearingToAngle(m[1], m[2], m[3], m[4], m[5])
			if err != nil {
				return nil, err
			}
			dist, err := strconv.ParseFloat(m[6], 64)
			if err != nil {
				return nil, err
			}
			curr = geom.Point2D{
				XFeet: curr.XFeet + dist*math.Cos(theta),
				YFeet: curr.YFeet + dist*math.Sin(theta),
			}
			endpoints = append(endpoints, curr)
			continue
		}

		if m := arcRe.FindStringSubmatch(line); m != nil {
			turnLeft := m[1] == "LEFT"
			tan, err := parseBearingToAngle(m[2], m[3], m[4], m[5], m[6])
			if err != nil {
				return nil, err
			}
			radius, err := strconv.ParseFloat(m[7], 64)
			if err != nil {
				return nil, err
			}
			centralDeg, err := parseDMS(m[8], m[9], m[10])
			if err != nil {
				return nil, err
			}
			arcDist, err := strconv.ParseFloat(m[11], 64)
			if err != nil {
				return nil, err
			}

			sweep := centralDeg * math.Pi / 180.0
			if !turnLeft {
				sweep = -sweep
			}
			expectedArc := math.Abs(radius * sweep)
			if math.Abs(expectedArc-arcDist) > 0.01 {
				return nil, fmt.Errorf("arc distance mismatch: expected %.4f got %.4f", expectedArc, arcDist)
			}

			leftX := -math.Sin(tan)
			leftY := math.Cos(tan)
			center := geom.Point2D{
				XFeet: curr.XFeet + leftX*radius,
				YFeet: curr.YFeet + leftY*radius,
			}
			if !turnLeft {
				center = geom.Point2D{
					XFeet: curr.XFeet - leftX*radius,
					YFeet: curr.YFeet - leftY*radius,
				}
			}

			startRad := geom.PointAngle(center, curr)
			endRad := startRad + sweep
			curr = geom.PointOnCircle(center, radius, endRad)
			endpoints = append(endpoints, curr)
			continue
		}

		return nil, fmt.Errorf("unrecognized THENCE line format: %s", line)
	}

	return endpoints, nil
}

func assertEndpointsMatch(t *testing.T, reconstructed []geom.Point2D, segments []model.Segment, maxErrorFeet float64) {
	t.Helper()
	if len(reconstructed) != len(segments) {
		t.Fatalf("endpoint count mismatch: reconstructed=%d expected=%d", len(reconstructed), len(segments))
	}
	for i, p := range reconstructed {
		want := segments[i].End()
		errFeet := geom.DistanceFeet(p, want)
		if errFeet > maxErrorFeet {
			t.Fatalf("segment %d endpoint mismatch: error=%.6f ft, got=(%.6f,%.6f), want=(%.6f,%.6f)", i+1, errFeet, p.XFeet, p.YFeet, want.XFeet, want.YFeet)
		}
	}
}

func parseBearingToAngle(primary, deg, min, sec, secondary string) (float64, error) {
	d, err := parseDMS(deg, min, sec)
	if err != nil {
		return 0.0, err
	}
	var az float64
	switch primary + secondary {
	case "NE":
		az = d
	case "NW":
		az = 360.0 - d
	case "SE":
		az = 180.0 - d
	case "SW":
		az = 180.0 + d
	default:
		return 0.0, fmt.Errorf("invalid bearing quadrant %s%s", primary, secondary)
	}
	return geom.AzimuthNorthClockwiseToMathRad(az), nil
}

func parseDMS(deg, min, sec string) (float64, error) {
	d, err := strconv.ParseFloat(deg, 64)
	if err != nil {
		return 0.0, err
	}
	m, err := strconv.ParseFloat(min, 64)
	if err != nil {
		return 0.0, err
	}
	s, err := strconv.ParseFloat(sec, 64)
	if err != nil {
		return 0.0, err
	}
	return d + m/60.0 + s/3600.0, nil
}
