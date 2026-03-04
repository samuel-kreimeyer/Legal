package landxml

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/samuel-kreimeyer/Legal/pkg/geom"
	"github.com/samuel-kreimeyer/Legal/pkg/model"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(ctx context.Context, r io.Reader) ([]model.Parcel, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var doc landXMLDocument
	if err := xml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("failed to decode LandXML: %w", err)
	}

	linearFactor, err := linearUnitFactorFromDoc(doc.Units)
	if err != nil {
		return nil, err
	}

	parcels := make([]model.Parcel, 0, len(doc.Parcels))
	for _, srcParcel := range doc.Parcels {
		loop, err := parseCoordGeom(srcParcel.CoordGeom.InnerXML, linearFactor)
		if err != nil {
			return nil, fmt.Errorf("parcel %q parse failed: %w", srcParcel.Name, err)
		}

		parcels = append(parcels, model.Parcel{
			ID:             srcParcel.Name,
			Source:         model.SourceLandXML,
			Unit:           model.LinearUnitFoot,
			AreaSquareFeet: srcParcel.Area * linearFactor * linearFactor,
			Loop:           loop,
		})
	}

	return parcels, nil
}

type landXMLDocument struct {
	Units   landXMLUnits    `xml:"Units"`
	Parcels []landXMLParcel `xml:"Parcels>Parcel"`
}

type landXMLUnits struct {
	Imperial *landXMLImperialUnits `xml:"Imperial"`
	Metric   *landXMLMetricUnits   `xml:"Metric"`
}

type landXMLImperialUnits struct {
	LinearUnit string `xml:"linearUnit,attr"`
}

type landXMLMetricUnits struct {
	LinearUnit string `xml:"linearUnit,attr"`
}

type landXMLParcel struct {
	Name      string           `xml:"name,attr"`
	Area      float64          `xml:"area,attr"`
	CoordGeom landXMLCoordGeom `xml:"CoordGeom"`
}

type landXMLCoordGeom struct {
	InnerXML string `xml:",innerxml"`
}

type landXMLLine struct {
	Start string `xml:"Start"`
	End   string `xml:"End"`
}

type landXMLCurve struct {
	Rot    string  `xml:"rot,attr"`
	Radius float64 `xml:"radius,attr"`
	Start  string  `xml:"Start"`
	End    string  `xml:"End"`
	Center string  `xml:"Center"`
}

func parseCoordGeom(innerXML string, linearFactor float64) (model.BoundaryLoop, error) {
	dec := xml.NewDecoder(strings.NewReader("<CoordGeom>" + innerXML + "</CoordGeom>"))
	segments := make([]model.Segment, 0, 8)

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return model.BoundaryLoop{}, fmt.Errorf("coord geom decode token failed: %w", err)
		}

		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		switch start.Name.Local {
		case "Line":
			var line landXMLLine
			if err := dec.DecodeElement(&line, &start); err != nil {
				return model.BoundaryLoop{}, fmt.Errorf("line decode failed: %w", err)
			}
			seg, err := line.toSegment(linearFactor)
			if err != nil {
				return model.BoundaryLoop{}, err
			}
			segments = append(segments, seg)
		case "Curve":
			var curve landXMLCurve
			if err := dec.DecodeElement(&curve, &start); err != nil {
				return model.BoundaryLoop{}, fmt.Errorf("curve decode failed: %w", err)
			}
			seg, err := curve.toSegment(linearFactor)
			if err != nil {
				return model.BoundaryLoop{}, err
			}
			segments = append(segments, seg)
		}
	}

	if len(segments) == 0 {
		return model.BoundaryLoop{}, fmt.Errorf("coord geom has no line/curve segments")
	}

	return model.BoundaryLoop{Segments: segments}, nil
}

func (l landXMLLine) toSegment(linearFactor float64) (model.LineSegment, error) {
	start, err := parsePoint2D(l.Start, linearFactor)
	if err != nil {
		return model.LineSegment{}, fmt.Errorf("line start parse failed: %w", err)
	}
	end, err := parsePoint2D(l.End, linearFactor)
	if err != nil {
		return model.LineSegment{}, fmt.Errorf("line end parse failed: %w", err)
	}
	return model.LineSegment{P0: start, P1: end}, nil
}

func (c landXMLCurve) toSegment(linearFactor float64) (model.ArcSegment, error) {
	start, err := parsePoint2D(c.Start, linearFactor)
	if err != nil {
		return model.ArcSegment{}, fmt.Errorf("curve start parse failed: %w", err)
	}
	end, err := parsePoint2D(c.End, linearFactor)
	if err != nil {
		return model.ArcSegment{}, fmt.Errorf("curve end parse failed: %w", err)
	}
	center, err := parsePoint2D(c.Center, linearFactor)
	if err != nil {
		return model.ArcSegment{}, fmt.Errorf("curve center parse failed: %w", err)
	}

	radiusFromPoints := geom.DistanceFeet(center, start)
	radius := radiusFromPoints
	if c.Radius > 0.0 && radiusFromPoints <= geom.EpsilonFeet {
		radius = c.Radius * linearFactor
	}
	if radius <= geom.EpsilonFeet {
		return model.ArcSegment{}, fmt.Errorf("curve radius is zero or invalid")
	}

	startAngle := geom.PointAngle(center, start)
	endAngle := geom.PointAngle(center, end)
	sweep, err := sweepFromRotation(c.Rot, startAngle, endAngle)
	if err != nil {
		return model.ArcSegment{}, err
	}

	return model.ArcSegment{
		Center:        center,
		RadiusFeet:    radius,
		StartAngleRad: startAngle,
		SweepRad:      sweep,
	}, nil
}

func parsePoint2D(src string, linearFactor float64) (geom.Point2D, error) {
	fields := strings.Fields(strings.TrimSpace(src))
	if len(fields) < 2 {
		return geom.Point2D{}, fmt.Errorf("expected two coordinates, got %q", src)
	}
	x, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return geom.Point2D{}, err
	}
	y, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return geom.Point2D{}, err
	}
	return geom.Point2D{
		XFeet: x * linearFactor,
		YFeet: y * linearFactor,
	}, nil
}

func sweepFromRotation(rotation string, startAngle, endAngle float64) (float64, error) {
	switch strings.ToLower(strings.TrimSpace(rotation)) {
	case "", "ccw":
		if endAngle <= startAngle {
			endAngle += 2.0 * math.Pi
		}
		return endAngle - startAngle, nil
	case "cw":
		if endAngle >= startAngle {
			endAngle -= 2.0 * math.Pi
		}
		return endAngle - startAngle, nil
	default:
		return 0.0, fmt.Errorf("unsupported curve rotation %q", rotation)
	}
}

func linearUnitFactorFromDoc(units landXMLUnits) (float64, error) {
	if units.Imperial != nil {
		if strings.TrimSpace(units.Imperial.LinearUnit) == "" {
			return 1.0, nil
		}
		return linearUnitFactor(units.Imperial.LinearUnit)
	}
	if units.Metric != nil {
		if strings.TrimSpace(units.Metric.LinearUnit) == "" {
			return 1.0, nil
		}
		return linearUnitFactor(units.Metric.LinearUnit)
	}
	return 1.0, nil
}

func linearUnitFactor(unit string) (float64, error) {
	normalized := strings.ToLower(unit)
	normalized = strings.ReplaceAll(normalized, "_", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, " ", "")

	switch normalized {
	case "ussurveyfoot", "ussurveyfeet", "usfoot", "foot", "feet", "ft":
		return 1.0, nil
	case "meter", "meters", "metre", "metres", "m":
		return 3.280839895013123, nil
	case "millimeter", "millimeters", "mm":
		return 0.0032808398950131233, nil
	case "centimeter", "centimeters", "cm":
		return 0.03280839895013123, nil
	case "inch", "inches", "in":
		return 1.0 / 12.0, nil
	default:
		return 0.0, fmt.Errorf("unsupported linear unit %q", unit)
	}
}
