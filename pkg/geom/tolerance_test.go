package geom

import "testing"

func TestEpsilonRequirement(t *testing.T) {
	if !(EpsilonFeet < 0.01) {
		t.Fatalf("EpsilonFeet must be < 0.01 ft, got %f", EpsilonFeet)
	}
}

func TestNearlyEqualFeet(t *testing.T) {
	if !NearlyEqualFeet(10.0, 10.0+EpsilonFeet/2.0) {
		t.Fatal("expected points within epsilon to compare equal")
	}
	if NearlyEqualFeet(10.0, 10.0+EpsilonFeet*2.0) {
		t.Fatal("expected points outside epsilon to compare not equal")
	}
}
