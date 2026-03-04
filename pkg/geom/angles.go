package geom

import "math"

func NormalizeAngle(theta float64) float64 {
	theta = math.Mod(theta, 2.0*math.Pi)
	if theta < 0.0 {
		theta += 2.0 * math.Pi
	}
	return theta
}

// AngleDiffAbs returns the shortest absolute angular distance between two angles.
func AngleDiffAbs(a, b float64) float64 {
	diff := math.Abs(NormalizeAngle(a) - NormalizeAngle(b))
	if diff > math.Pi {
		diff = 2.0*math.Pi - diff
	}
	return diff
}

// AzimuthNorthClockwiseToMathRad converts an azimuth in degrees measured clockwise
// from north to a math angle in radians measured counterclockwise from +X.
func AzimuthNorthClockwiseToMathRad(azimuthDegrees float64) float64 {
	return NormalizeAngle((90.0 - azimuthDegrees) * math.Pi / 180.0)
}

func MathRadToAzimuthNorthClockwise(theta float64) float64 {
	deg := 90.0 - NormalizeAngle(theta)*180.0/math.Pi
	deg = math.Mod(deg, 360.0)
	if deg < 0.0 {
		deg += 360.0
	}
	return deg
}
