package app

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/model"
	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/normalize"
	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/parse"
	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/parse/dxf"
	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/parse/ifc"
	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/parse/landxml"
	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/survey"
)

// ParseOptions carries optional parameters for ParseFile and ParseReader.
type ParseOptions struct {
	// LayerFilter restricts DXF parsing to the named layers (case-insensitive).
	// When two or more names are given the parser operates in multi-layer
	// assembly mode — see dxf.Options.LayerFilter for details.
	// Has no effect on IFC or LandXML input.
	LayerFilter []string
}

type Pipeline struct {
	ifc     parse.Parser
	landxml parse.Parser
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		ifc:     ifc.NewParser(),
		landxml: landxml.NewParser(),
	}
}

func (p *Pipeline) ParseFile(ctx context.Context, path string, format parse.SourceFormat, opts ...ParseOptions) ([]model.Parcel, error) {
	if format == parse.SourceFormatUnknown {
		detected, err := parse.DetectSourceFormat(path)
		if err != nil {
			return nil, err
		}
		format = detected
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return p.ParseReader(ctx, f, format, opts...)
}

func (p *Pipeline) ParseReader(ctx context.Context, reader io.Reader, format parse.SourceFormat, opts ...ParseOptions) ([]model.Parcel, error) {
	var o ParseOptions
	if len(opts) > 0 {
		o = opts[0]
	}

	parser, err := p.parserForFormat(format, o)
	if err != nil {
		return nil, err
	}
	parcels, err := parser.Parse(ctx, reader)
	if err != nil {
		return nil, err
	}

	normalized := make([]model.Parcel, 0, len(parcels))
	for _, parcel := range parcels {
		norm, err := normalize.NormalizeParcel(parcel)
		if err != nil {
			return nil, fmt.Errorf("parcel %q failed normalization: %w", parcel.ID, err)
		}
		normalized = append(normalized, norm)
	}
	return normalized, nil
}

// AnnotateWithSurvey matches survey points to parcel boundary vertices and
// attaches monument annotations to each parcel's BoundaryLoop. Parcels without
// any matching points are returned unchanged.
//
// Pass toleranceFeet <= 0 to use survey.DefaultMatchToleranceFeet (0.5 ft).
func (p *Pipeline) AnnotateWithSurvey(parcels []model.Parcel, points []survey.Point, toleranceFeet float64) []model.Parcel {
	out := make([]model.Parcel, len(parcels))
	for i, parcel := range parcels {
		mons := survey.MatchToLoop(points, parcel.Loop, toleranceFeet)
		parcel.Loop.Monuments = mons
		out[i] = parcel
	}
	return out
}

func (p *Pipeline) parserForFormat(format parse.SourceFormat, opts ParseOptions) (parse.Parser, error) {
	switch format {
	case parse.SourceFormatLandXML:
		return p.landxml, nil
	case parse.SourceFormatDXF:
		return dxf.NewParserWithOptions(dxf.Options{LayerFilter: opts.LayerFilter}), nil
	case parse.SourceFormatIFC:
		return p.ifc, nil
	default:
		return nil, fmt.Errorf("unsupported source format: %q", format)
	}
}
