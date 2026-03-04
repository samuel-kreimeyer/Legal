package legal

import (
	"fmt"
	"math"
	"strings"

	"github.com/samuel-kreimeyer/Legal/pkg/geom"
	"github.com/samuel-kreimeyer/Legal/pkg/model"
)

type Options struct {
	Kind            string
	Lot             string
	Block           string
	Subdivision     string
	City            string
	County          string
	State           string
	StartCorner     string
	UseCommencing   bool
	AreaSquareFeet  float64
	AreaDisplayUnit string
}

func RenderParcel(parcel model.Parcel, opts Options) (string, error) {
	if len(parcel.Loop.Segments) == 0 {
		return "", fmt.Errorf("parcel has no segments")
	}

	kind := normalizeText(opts.Kind)
	lot := normalizeText(opts.Lot)
	block := normalizeText(opts.Block)
	subdivision := normalizeText(opts.Subdivision)
	city := normalizeText(opts.City)
	county := normalizeText(opts.County)
	state := normalizeText(opts.State)
	startCorner := normalizeText(opts.StartCorner)
	if startCorner == "" {
		startCorner = "NORTHEAST"
	}

	area := opts.AreaSquareFeet
	if area <= 0.0 {
		area = parcel.AreaSquareFeet
	}
	areaUnit := normalizeText(opts.AreaDisplayUnit)
	if areaUnit == "" {
		areaUnit = "SQUARE FEET"
	}

	var b strings.Builder
	b.WriteString(kind)
	if kind != "" {
		b.WriteString(" ")
	}
	b.WriteString("DESCRIPTION:\n\n")
	b.WriteString("A PART OF ")
	if lot != "" {
		b.WriteString("LOT ")
		b.WriteString(lot)
		b.WriteString(", ")
	}
	if block != "" {
		b.WriteString("BLOCK ")
		b.WriteString(block)
		b.WriteString(", ")
	}
	b.WriteString(subdivision)
	b.WriteString(" TO ")
	if city != "" {
		b.WriteString("THE CITY OF ")
		b.WriteString(city)
		b.WriteString(", ")
	}
	b.WriteString(county)
	b.WriteString(" COUNTY, ")
	b.WriteString(state)
	b.WriteString(", BEING MORE PARTICULARLY DESCRIBED AS FOLLOWS:\n")

	if opts.UseCommencing {
		b.WriteString("COMMENCING")
	} else {
		b.WriteString("BEGINNING")
	}
	b.WriteString(" AT THE ")
	b.WriteString(startCorner)
	b.WriteString(" CORNER OF SAID LOT")
	if lot != "" {
		b.WriteString(" ")
		b.WriteString(lot)
	}
	b.WriteString(";\n")

	prevTan := parcel.Loop.Segments[0].TangentAtStart()
	for i, seg := range parcel.Loop.Segments {
		if i > 0 {
			if geom.NearlyEqualAngle(prevTan, seg.TangentAtStart()) {
				b.WriteString("TO A POINT OF TANGENCY;\n")
			} else {
				b.WriteString("TO A POINT OF NON-TANGENCY;\n")
			}
		}

		line, err := renderSegment(seg)
		if err != nil {
			return "", err
		}
		b.WriteString("THENCE ")
		b.WriteString(line)
		b.WriteString(";\n")

		prevTan = seg.TangentAtEnd()
	}

	b.WriteString("TO THE POINT OF BEGINNING, CONTAINING ")
	b.WriteString(fmt.Sprintf("%.2f", area))
	b.WriteString(" ")
	b.WriteString(areaUnit)
	b.WriteString(", MORE OR LESS.")
	return b.String(), nil
}

func renderSegment(seg model.Segment) (string, error) {
	switch s := seg.(type) {
	case model.LineSegment:
		return fmt.Sprintf("%s, A DISTANCE OF %.2f FEET", formatBearing(s.TangentAtStart()), s.LengthFeet()), nil
	case *model.LineSegment:
		return fmt.Sprintf("%s, A DISTANCE OF %.2f FEET", formatBearing(s.TangentAtStart()), s.LengthFeet()), nil
	case model.ArcSegment:
		return renderArc(s, s.TangentAtStart()), nil
	case *model.ArcSegment:
		return renderArc(*s, s.TangentAtStart()), nil
	default:
		return "", fmt.Errorf("unsupported segment type %T", seg)
	}
}

func renderArc(s model.ArcSegment, tangentStart float64) string {
	rot := "LEFT"
	if s.SweepRad < 0.0 {
		rot = "RIGHT"
	}
	central := radiansToDMS(math.Abs(s.SweepRad))
	tanBearing := formatBearing(tangentStart)
	return fmt.Sprintf(
		"ALONG A CURVE TO THE %s, WITH TANGENT BEARING %s, HAVING A RADIUS OF %.2f FEET, A CENTRAL ANGLE OF %s, AN ARC DISTANCE OF %.2f FEET",
		rot,
		tanBearing,
		s.RadiusFeet,
		central,
		s.LengthFeet(),
	)
}

func formatBearing(theta float64) string {
	az := geom.MathRadToAzimuthNorthClockwise(theta)
	const eps = 1e-9
	if math.Abs(az-360.0) <= eps {
		az = 0.0
	}

	var primary, secondary string
	var quad float64
	switch {
	case az >= 0.0 && az <= 90.0:
		primary = "N"
		secondary = "E"
		quad = az
	case az > 90.0 && az <= 180.0:
		primary = "S"
		secondary = "E"
		quad = 180.0 - az
	case az > 180.0 && az <= 270.0:
		primary = "S"
		secondary = "W"
		quad = az - 180.0
	default:
		primary = "N"
		secondary = "W"
		quad = 360.0 - az
	}
	return fmt.Sprintf("%s %s %s", primary, degreesToDMS(quad), secondary)
}

func radiansToDMS(radians float64) string {
	return degreesToDMS(radians * 180.0 / math.Pi)
}

func degreesToDMS(degrees float64) string {
	deg := math.Floor(degrees)
	minFull := (degrees - deg) * 60.0
	min := math.Floor(minFull)
	sec := (minFull - min) * 60.0

	sec = math.Round(sec*100.0) / 100.0
	if sec >= 60.0 {
		sec = 0.0
		min += 1.0
	}
	if min >= 60.0 {
		min = 0.0
		deg += 1.0
	}
	return fmt.Sprintf("%.0f°%.0f'%.2f\"", deg, min, sec)
}

func normalizeText(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
