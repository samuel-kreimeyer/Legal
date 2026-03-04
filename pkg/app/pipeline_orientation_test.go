package app

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/samuel-kreimeyer/Legal/pkg/model"
	"github.com/samuel-kreimeyer/Legal/pkg/parse"
)

func TestPipeline_NormalizesClockwiseAcrossFormats(t *testing.T) {
	cases := []struct {
		name   string
		format parse.SourceFormat
		path   string
	}{
		{name: "landxml", format: parse.SourceFormatLandXML, path: filepath.Join("..", "..", "tests", "fixtures", "landxml", "square.xml")},
		{name: "dxf", format: parse.SourceFormatDXF, path: filepath.Join("..", "..", "tests", "fixtures", "dxf", "square_lines.dxf")},
		{name: "ifc", format: parse.SourceFormatIFC, path: filepath.Join("..", "..", "tests", "fixtures", "ifc", "square_profile.ifc")},
	}

	p := NewPipeline()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			parcels, err := p.ParseFile(context.Background(), tc.path, tc.format)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}
			if len(parcels) == 0 {
				t.Fatal("no parcels parsed")
			}
			area := signedArea(parcels[0].Loop)
			if area >= 0.0 {
				t.Fatalf("expected clockwise normalized loop (negative signed area), got %.6f", area)
			}
		})
	}
}

func signedArea(loop model.BoundaryLoop) float64 {
	if len(loop.Segments) == 0 {
		return 0.0
	}
	points := make([][2]float64, 0, len(loop.Segments)+1)
	for _, seg := range loop.Segments {
		p := seg.Start()
		points = append(points, [2]float64{p.XFeet, p.YFeet})
	}
	end := loop.Segments[len(loop.Segments)-1].End()
	points = append(points, [2]float64{end.XFeet, end.YFeet})

	var sum float64
	for i := 0; i < len(points)-1; i++ {
		sum += points[i][0]*points[i+1][1] - points[i+1][0]*points[i][1]
	}
	return 0.5 * sum
}
