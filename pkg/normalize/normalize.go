package normalize

import (
	"fmt"
	"math"

	"github.com/samuel-kreimeyer/Legal/pkg/geom"
	"github.com/samuel-kreimeyer/Legal/pkg/model"
)

func NormalizeParcel(parcel model.Parcel) (model.Parcel, error) {
	loop := model.BoundaryLoop{
		Segments: make([]model.Segment, 0, len(parcel.Loop.Segments)),
	}
	for _, seg := range parcel.Loop.Segments {
		if geom.NearlyZeroFeet(seg.LengthFeet()) {
			continue
		}
		loop.Segments = append(loop.Segments, seg)
	}

	if len(loop.Segments) == 0 {
		return parcel, fmt.Errorf("parcel has no non-zero segments")
	}

	loop, err := normalizeSegmentDirections(loop)
	if err != nil {
		return parcel, err
	}
	if err := ValidateClosedLoop(loop); err != nil {
		return parcel, err
	}

	area := signedAreaApprox(loop)
	if math.Abs(area) <= geom.EpsilonFeet*geom.EpsilonFeet {
		return parcel, fmt.Errorf("loop area is zero or below tolerance")
	}

	// Normalize loop orientation to CW for deterministic rendering.
	if area > 0.0 {
		loop, err = reverseLoop(loop)
		if err != nil {
			return parcel, err
		}
		loop, err = normalizeSegmentDirections(loop)
		if err != nil {
			return parcel, err
		}
		if err := ValidateClosedLoop(loop); err != nil {
			return parcel, err
		}
	}
	finalArea := math.Abs(signedAreaApprox(loop))
	if parcel.AreaSquareFeet <= 0.0 {
		parcel.AreaSquareFeet = finalArea
	}

	parcel.Unit = model.LinearUnitFoot
	parcel.Loop = loop
	return parcel, nil
}

func ValidateClosedLoop(loop model.BoundaryLoop) error {
	if len(loop.Segments) < 3 {
		return fmt.Errorf("closed loop requires at least 3 segments, got %d", len(loop.Segments))
	}
	for i := 0; i < len(loop.Segments)-1; i++ {
		if !pointsNear(loop.Segments[i].End(), loop.Segments[i+1].Start()) {
			return fmt.Errorf("segment %d endpoint does not connect to segment %d startpoint", i, i+1)
		}
	}
	last := loop.Segments[len(loop.Segments)-1].End()
	first := loop.Segments[0].Start()
	if !pointsNear(last, first) {
		return fmt.Errorf("loop is not closed: last segment end does not match first segment start")
	}
	return nil
}

func pointsNear(a, b geom.Point2D) bool {
	return geom.DistanceFeet(a, b) <= geom.EpsilonFeet
}

func normalizeSegmentDirections(loop model.BoundaryLoop) (model.BoundaryLoop, error) {
	if len(loop.Segments) < 2 {
		return loop, nil
	}
	out := model.BoundaryLoop{
		Segments: make([]model.Segment, len(loop.Segments)),
	}
	copy(out.Segments, loop.Segments)

	for i := 1; i < len(out.Segments); i++ {
		prevEnd := out.Segments[i-1].End()
		curr := out.Segments[i]

		if pointsNear(prevEnd, curr.Start()) {
			continue
		}
		if pointsNear(prevEnd, curr.End()) {
			rev, err := reverseSegment(curr)
			if err != nil {
				return out, err
			}
			out.Segments[i] = rev
			continue
		}
		return out, fmt.Errorf("segment %d does not connect to previous segment", i)
	}
	return out, nil
}

func reverseLoop(loop model.BoundaryLoop) (model.BoundaryLoop, error) {
	reversed := model.BoundaryLoop{
		Segments: make([]model.Segment, 0, len(loop.Segments)),
	}
	for i := len(loop.Segments) - 1; i >= 0; i-- {
		seg, err := reverseSegment(loop.Segments[i])
		if err != nil {
			return reversed, err
		}
		reversed.Segments = append(reversed.Segments, seg)
	}
	return reversed, nil
}

func reverseSegment(seg model.Segment) (model.Segment, error) {
	switch s := seg.(type) {
	case model.LineSegment:
		return model.LineSegment{P0: s.P1, P1: s.P0}, nil
	case *model.LineSegment:
		return model.LineSegment{P0: s.P1, P1: s.P0}, nil
	case model.ArcSegment:
		return model.ArcSegment{
			Center:        s.Center,
			RadiusFeet:    s.RadiusFeet,
			StartAngleRad: geom.NormalizeAngle(s.StartAngleRad + s.SweepRad),
			SweepRad:      -s.SweepRad,
		}, nil
	case *model.ArcSegment:
		return model.ArcSegment{
			Center:        s.Center,
			RadiusFeet:    s.RadiusFeet,
			StartAngleRad: geom.NormalizeAngle(s.StartAngleRad + s.SweepRad),
			SweepRad:      -s.SweepRad,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported segment type %T", seg)
	}
}

func signedAreaApprox(loop model.BoundaryLoop) float64 {
	if len(loop.Segments) == 0 {
		return 0.0
	}
	points := make([]geom.Point2D, 0, len(loop.Segments)+1)
	for _, seg := range loop.Segments {
		points = append(points, seg.Start())
	}
	points = append(points, loop.Segments[len(loop.Segments)-1].End())

	var sum float64
	for i := 0; i < len(points)-1; i++ {
		sum += points[i].XFeet*points[i+1].YFeet - points[i+1].XFeet*points[i].YFeet
	}
	return 0.5 * sum
}
