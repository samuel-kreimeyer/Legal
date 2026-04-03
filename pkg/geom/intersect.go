package geom

import "math"

// LineLineIntersect finds the intersection of the infinite lines through
// P0→P1 and Q0→Q1, returning parametric values t (along P) and s (along Q).
// ok is false when the lines are parallel or coincident.
// The caller should verify 0 ≤ t ≤ 1 and 0 ≤ s ≤ 1 to confirm the
// intersection lies within both segments.
func LineLineIntersect(p0, p1, q0, q1 Point2D) (t, s float64, ok bool) {
	dx1 := p1.XFeet - p0.XFeet
	dy1 := p1.YFeet - p0.YFeet
	dx2 := q1.XFeet - q0.XFeet
	dy2 := q1.YFeet - q0.YFeet

	denom := dx1*dy2 - dy1*dx2
	if math.Abs(denom) < 1e-12 {
		return 0, 0, false
	}

	rx := q0.XFeet - p0.XFeet
	ry := q0.YFeet - p0.YFeet

	t = (rx*dy2 - ry*dx2) / denom
	s = (rx*dy1 - ry*dx1) / denom
	return t, s, true
}

// LineCircleIntersectT returns up to two parametric t values (along the
// segment P0→P1) where the segment's line intersects the circle defined by
// center and radius. n is 0, 1 (tangent), or 2. The returned values are NOT
// clamped to [0,1]; the caller should apply that check.
func LineCircleIntersectT(p0, p1, center Point2D, radius float64) (t1, t2 float64, n int) {
	dx := p1.XFeet - p0.XFeet
	dy := p1.YFeet - p0.YFeet

	fx := p0.XFeet - center.XFeet
	fy := p0.YFeet - center.YFeet

	a := dx*dx + dy*dy
	if a < 1e-24 {
		return 0, 0, 0 // degenerate segment
	}

	b := 2 * (fx*dx + fy*dy)
	c := fx*fx + fy*fy - radius*radius

	disc := b*b - 4*a*c
	if disc < 0 {
		return 0, 0, 0
	}

	sqrtDisc := math.Sqrt(math.Max(0, disc))
	t1 = (-b - sqrtDisc) / (2 * a)
	t2 = (-b + sqrtDisc) / (2 * a)
	if disc < 1e-18 {
		return t1, t1, 1
	}
	return t1, t2, 2
}

// CircleCircleIntersect returns the 0, 1, or 2 points where two circles
// intersect.
func CircleCircleIntersect(c1 Point2D, r1 float64, c2 Point2D, r2 float64) []Point2D {
	dx := c2.XFeet - c1.XFeet
	dy := c2.YFeet - c1.YFeet
	d := math.Hypot(dx, dy)

	if d < 1e-9 {
		return nil // concentric
	}
	if d > r1+r2+1e-9 {
		return nil // too far apart
	}
	if d < math.Abs(r1-r2)-1e-9 {
		return nil // one circle inside the other
	}

	a := (r1*r1 - r2*r2 + d*d) / (2 * d)
	h2 := r1*r1 - a*a
	if h2 < 0 {
		if h2 > -1e-9 {
			h2 = 0
		} else {
			return nil
		}
	}
	h := math.Sqrt(h2)

	mx := c1.XFeet + a*(dx/d)
	my := c1.YFeet + a*(dy/d)

	if h < 1e-9 {
		return []Point2D{{XFeet: mx, YFeet: my}}
	}
	return []Point2D{
		{XFeet: mx + h*(dy/d), YFeet: my - h*(dx/d)},
		{XFeet: mx - h*(dy/d), YFeet: my + h*(dx/d)},
	}
}

// ArcParametricT converts a world angle (math convention, CCW from +X) to a
// parametric t ∈ [0, 1] along the arc defined by startAngle and sweepRad
// (positive sweep = CCW). Returns (t, true) if the angle falls within the arc,
// (0, false) otherwise. Endpoint tolerance of 1e-9 rad is applied.
func ArcParametricT(angle, startAngle, sweepRad float64) (float64, bool) {
	if math.Abs(sweepRad) < 1e-12 {
		return 0, false
	}

	const eps = 1e-9

	if sweepRad > 0 {
		// CCW: angular distance from startAngle to angle, going CCW.
		delta := NormalizeAngle(angle - startAngle)
		if delta > sweepRad+eps {
			return 0, false
		}
		return delta / sweepRad, true
	}

	// CW: angular distance from startAngle to angle, going CW.
	delta := NormalizeAngle(startAngle - angle)
	if delta > -sweepRad+eps {
		return 0, false
	}
	return delta / (-sweepRad), true
}
