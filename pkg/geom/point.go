package geom

import "math"

type Point2D struct {
	XFeet float64
	YFeet float64
}

func DistanceFeet(a, b Point2D) float64 {
	return math.Hypot(a.XFeet-b.XFeet, a.YFeet-b.YFeet)
}

func PointAngle(center, p Point2D) float64 {
	return NormalizeAngle(math.Atan2(p.YFeet-center.YFeet, p.XFeet-center.XFeet))
}

func PointOnCircle(center Point2D, radius, angleRad float64) Point2D {
	return Point2D{
		XFeet: center.XFeet + radius*math.Cos(angleRad),
		YFeet: center.YFeet + radius*math.Sin(angleRad),
	}
}
