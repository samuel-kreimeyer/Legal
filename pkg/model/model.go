package model

import (
	"math"

	"github.com/samuel-kreimeyer/Legal/pkg/geom"
)

type LinearUnit string

const (
	LinearUnitFoot LinearUnit = "ft"
)

type SourceType string

const (
	SourceDXF     SourceType = "dxf"
	SourceIFC     SourceType = "ifc"
	SourceLandXML SourceType = "landxml"
)

type SegmentKind string

const (
	SegmentKindLine SegmentKind = "line"
	SegmentKindArc  SegmentKind = "arc"
)

type Parcel struct {
	ID             string
	Source         SourceType
	Unit           LinearUnit
	AreaSquareFeet float64
	Loop           BoundaryLoop
}

type BoundaryLoop struct {
	Segments []Segment
}

type Segment interface {
	Kind() SegmentKind
	Start() geom.Point2D
	End() geom.Point2D
	LengthFeet() float64
	TangentAtStart() float64
	TangentAtEnd() float64
}

type LineSegment struct {
	P0 geom.Point2D
	P1 geom.Point2D
}

func (s LineSegment) Kind() SegmentKind {
	return SegmentKindLine
}

func (s LineSegment) Start() geom.Point2D {
	return s.P0
}

func (s LineSegment) End() geom.Point2D {
	return s.P1
}

func (s LineSegment) LengthFeet() float64 {
	return geom.DistanceFeet(s.P0, s.P1)
}

func (s LineSegment) TangentAtStart() float64 {
	return geom.PointAngle(s.P0, s.P1)
}

func (s LineSegment) TangentAtEnd() float64 {
	return s.TangentAtStart()
}

type ArcSegment struct {
	Center        geom.Point2D
	RadiusFeet    float64
	StartAngleRad float64
	SweepRad      float64 // Positive is CCW, negative is CW.
}

func (s ArcSegment) Kind() SegmentKind {
	return SegmentKindArc
}

func (s ArcSegment) Start() geom.Point2D {
	return geom.PointOnCircle(s.Center, s.RadiusFeet, s.StartAngleRad)
}

func (s ArcSegment) End() geom.Point2D {
	return geom.PointOnCircle(s.Center, s.RadiusFeet, s.StartAngleRad+s.SweepRad)
}

func (s ArcSegment) LengthFeet() float64 {
	return math.Abs(s.RadiusFeet * s.SweepRad)
}

func (s ArcSegment) TangentAtStart() float64 {
	sign := 1.0
	if s.SweepRad < 0.0 {
		sign = -1.0
	}
	return geom.NormalizeAngle(s.StartAngleRad + sign*math.Pi/2.0)
}

func (s ArcSegment) TangentAtEnd() float64 {
	sign := 1.0
	if s.SweepRad < 0.0 {
		sign = -1.0
	}
	return geom.NormalizeAngle(s.StartAngleRad + s.SweepRad + sign*math.Pi/2.0)
}
