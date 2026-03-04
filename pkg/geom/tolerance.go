package geom

import "math"

// EpsilonFeet is the global linear tolerance for geometry comparisons.
// This remains strictly below 0.01 ft per project requirements.
const EpsilonFeet = 0.001

// EpsilonAngleRad is the angular tolerance used for tangent comparisons.
const EpsilonAngleRad = 1e-6

func NearlyEqualFeet(a, b float64) bool {
	return math.Abs(a-b) <= EpsilonFeet
}

func NearlyZeroFeet(v float64) bool {
	return math.Abs(v) <= EpsilonFeet
}

func NearlyEqualAngle(a, b float64) bool {
	return AngleDiffAbs(a, b) <= EpsilonAngleRad
}
