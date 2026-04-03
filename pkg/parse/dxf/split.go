package dxf

import (
	"math"
	"sort"

	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/geom"
	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/model"
)

// splitEps is the minimum parametric distance from an endpoint (0 or 1) for a
// split to be recorded. Values closer than this are treated as coincident with
// the endpoint and ignored, preventing zero-length sub-segments.
const splitEps = 1e-9

// splitAtCrossLayerIntersections finds all intersections between segments from
// different layers, splits each segment at those intersection points, and
// returns the combined pool of sub-segments.
//
// The layers slice controls output ordering: segments from the last-listed layer
// appear first in the output. In a two-layer scenario this means the
// acquisition/ROW layer seeds loop-building before the property boundary, so
// the greedy walk finds the acquired parcel first.
func splitAtCrossLayerIntersections(byLayer map[string][]model.Segment, layers []string) []model.Segment {
	// splitTs[layer][segIdx] accumulates parametric t values where segment
	// segIdx of layer should be split.
	splitTs := make(map[string][][]float64)
	for _, layer := range layers {
		splitTs[layer] = make([][]float64, len(byLayer[layer]))
	}

	// Test every pair of distinct layers.
	for li := 0; li < len(layers); li++ {
		for lj := li + 1; lj < len(layers); lj++ {
			la, lb := layers[li], layers[lj]
			for ai, sa := range byLayer[la] {
				for bi, sb := range byLayer[lb] {
					tAs, tBs := intersectSegments(sa, sb)
					splitTs[la][ai] = append(splitTs[la][ai], tAs...)
					splitTs[lb][bi] = append(splitTs[lb][bi], tBs...)
				}
			}
		}
	}

	// Emit segments in reverse layer order so that the last-listed (acquisition)
	// layer appears first, biasing the greedy loop-builder to seed from it.
	var out []model.Segment
	for i := len(layers) - 1; i >= 0; i-- {
		layer := layers[i]
		for j, seg := range byLayer[layer] {
			out = append(out, splitSegment(seg, splitTs[layer][j])...)
		}
	}
	return out
}

// intersectSegments returns parametric t values (each in (splitEps, 1-splitEps))
// for intersections between segment a (tsA) and segment b (tsB).
func intersectSegments(a, b model.Segment) (tsA, tsB []float64) {
	switch sa := a.(type) {
	case model.LineSegment:
		switch sb := b.(type) {
		case model.LineSegment:
			tsA, tsB = intersectLineLine(sa, sb)
		case model.ArcSegment:
			tsA, tsB = intersectLineArc(sa, sb)
		}
	case model.ArcSegment:
		switch sb := b.(type) {
		case model.LineSegment:
			tb, ta := intersectLineArc(sb, sa)
			tsA, tsB = ta, tb
		case model.ArcSegment:
			tsA, tsB = intersectArcArc(sa, sb)
		}
	}
	return
}

// onSeg is the tolerance for deciding whether a parametric value is within a
// segment's [0, 1] range (distinct from inOpen which excludes endpoints).
const onSeg = 1e-9

func intersectLineLine(a, b model.LineSegment) (tsA, tsB []float64) {
	t, s, ok := geom.LineLineIntersect(a.P0, a.P1, b.P0, b.P1)
	if !ok {
		return nil, nil
	}
	// The intersection must lie within both segments (endpoints count as valid).
	if t < -onSeg || t > 1+onSeg || s < -onSeg || s > 1+onSeg {
		return nil, nil
	}
	// Record splits independently: a segment is split only if the intersection
	// falls at an interior point of *that* segment. An endpoint of segment B
	// landing on the interior of segment A is the normal ROW/property scenario
	// and must still split A.
	if inOpen(t) {
		tsA = []float64{t}
	}
	if inOpen(s) {
		tsB = []float64{s}
	}
	return
}

func intersectLineArc(line model.LineSegment, arc model.ArcSegment) (tsLine, tsArc []float64) {
	t1, t2, n := geom.LineCircleIntersectT(line.P0, line.P1, arc.Center, arc.RadiusFeet)
	candidates := []float64{t1}
	if n == 2 {
		candidates = append(candidates, t2)
	}

	for _, tLine := range candidates {
		// Intersection must be within the line segment (endpoints allowed).
		if tLine < -onSeg || tLine > 1+onSeg {
			continue
		}
		// World coordinates of the intersection point.
		dx := line.P1.XFeet - line.P0.XFeet
		dy := line.P1.YFeet - line.P0.YFeet
		px := line.P0.XFeet + tLine*dx
		py := line.P0.YFeet + tLine*dy

		// Intersection must also be within the arc.
		angle := geom.NormalizeAngle(math.Atan2(py-arc.Center.YFeet, px-arc.Center.XFeet))
		tArc, ok := geom.ArcParametricT(angle, arc.StartAngleRad, arc.SweepRad)
		if !ok {
			continue
		}
		// Record splits independently.
		if inOpen(tLine) {
			tsLine = append(tsLine, tLine)
		}
		if inOpen(tArc) {
			tsArc = append(tsArc, tArc)
		}
	}
	return
}

func intersectArcArc(a, b model.ArcSegment) (tsA, tsB []float64) {
	pts := geom.CircleCircleIntersect(a.Center, a.RadiusFeet, b.Center, b.RadiusFeet)
	for _, pt := range pts {
		angA := geom.NormalizeAngle(math.Atan2(pt.YFeet-a.Center.YFeet, pt.XFeet-a.Center.XFeet))
		angB := geom.NormalizeAngle(math.Atan2(pt.YFeet-b.Center.YFeet, pt.XFeet-b.Center.XFeet))

		tA, okA := geom.ArcParametricT(angA, a.StartAngleRad, a.SweepRad)
		tB, okB := geom.ArcParametricT(angB, b.StartAngleRad, b.SweepRad)
		if !okA || !okB {
			continue
		}
		if inOpen(tA) {
			tsA = append(tsA, tA)
		}
		if inOpen(tB) {
			tsB = append(tsB, tB)
		}
	}
	return
}

// splitSegment divides seg at the given parametric t values and returns the
// resulting sub-segments. Values outside (splitEps, 1-splitEps) are ignored.
// Nearby t values (within 10×splitEps) are deduplicated.
func splitSegment(seg model.Segment, ts []float64) []model.Segment {
	filtered := make([]float64, 0, len(ts))
	for _, t := range ts {
		if inOpen(t) {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) == 0 {
		return []model.Segment{seg}
	}

	sort.Float64s(filtered)

	deduped := []float64{filtered[0]}
	for i := 1; i < len(filtered); i++ {
		if filtered[i]-deduped[len(deduped)-1] > splitEps*10 {
			deduped = append(deduped, filtered[i])
		}
	}

	switch s := seg.(type) {
	case model.LineSegment:
		return splitLine(s, deduped)
	case *model.LineSegment:
		return splitLine(*s, deduped)
	case model.ArcSegment:
		return splitArc(s, deduped)
	case *model.ArcSegment:
		return splitArc(*s, deduped)
	default:
		return []model.Segment{seg}
	}
}

func splitLine(seg model.LineSegment, ts []float64) []model.Segment {
	dx := seg.P1.XFeet - seg.P0.XFeet
	dy := seg.P1.YFeet - seg.P0.YFeet
	interp := func(t float64) geom.Point2D {
		return geom.Point2D{XFeet: seg.P0.XFeet + t*dx, YFeet: seg.P0.YFeet + t*dy}
	}

	out := make([]model.Segment, 0, len(ts)+1)
	prev := 0.0
	for _, t := range ts {
		out = append(out, model.LineSegment{P0: interp(prev), P1: interp(t)})
		prev = t
	}
	out = append(out, model.LineSegment{P0: interp(prev), P1: seg.P1})
	return out
}

func splitArc(seg model.ArcSegment, ts []float64) []model.Segment {
	out := make([]model.Segment, 0, len(ts)+1)
	prev := 0.0
	for _, t := range ts {
		out = append(out, arcSubsegment(seg, prev, t))
		prev = t
	}
	out = append(out, arcSubsegment(seg, prev, 1.0))
	return out
}

func arcSubsegment(seg model.ArcSegment, t0, t1 float64) model.ArcSegment {
	startAngle := geom.NormalizeAngle(seg.StartAngleRad + t0*seg.SweepRad)
	return model.ArcSegment{
		Center:        seg.Center,
		RadiusFeet:    seg.RadiusFeet,
		StartAngleRad: startAngle,
		SweepRad:      (t1 - t0) * seg.SweepRad,
	}
}

func inOpen(t float64) bool {
	return t > splitEps && t < 1.0-splitEps
}
