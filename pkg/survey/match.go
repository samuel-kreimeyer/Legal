package survey

import (
	"math"

	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/geom"
	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/model"
)

// DefaultMatchToleranceFeet is the maximum distance between a survey point and
// a parcel vertex for them to be considered co-located when no tolerance is
// provided by the caller.
const DefaultMatchToleranceFeet = 0.5

// MatchToLoop correlates survey points with parcel boundary vertices by
// proximity. It returns a slice of length len(loop.Segments), where element i
// is the monument (if any) located within toleranceFeet of Segments[i].Start().
// A nil element means no matching survey point was found for that vertex.
//
// Survey coordinates use the surveying convention (Northing = Y, Easting = X)
// which is mapped to geom.Point2D (YFeet = Northing, XFeet = Easting).
//
// Pass toleranceFeet <= 0 to use DefaultMatchToleranceFeet.
func MatchToLoop(points []Point, loop model.BoundaryLoop, toleranceFeet float64) []*model.Monument {
	if toleranceFeet <= 0 {
		toleranceFeet = DefaultMatchToleranceFeet
	}
	if len(loop.Segments) == 0 || len(points) == 0 {
		return nil
	}

	monuments := make([]*model.Monument, len(loop.Segments))
	for i, seg := range loop.Segments {
		vertex := seg.Start()
		var closest *Point
		closestDist := math.MaxFloat64

		for j := range points {
			sp := geom.Point2D{XFeet: points[j].Easting, YFeet: points[j].Northing}
			d := geom.DistanceFeet(vertex, sp)
			if d < closestDist {
				closestDist = d
				closest = &points[j]
			}
		}

		if closest != nil && closestDist <= toleranceFeet {
			desc, _ := DescribeCode(closest.Code)
			monuments[i] = &model.Monument{
				PointNumber: closest.Number,
				FeatureCode: string(closest.Code),
				Description: desc,
			}
		}
	}
	return monuments
}
