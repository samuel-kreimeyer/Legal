package app

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/samuel-kreimeyer/Legal/pkg/model"
	"github.com/samuel-kreimeyer/Legal/pkg/normalize"
	"github.com/samuel-kreimeyer/Legal/pkg/parse"
	"github.com/samuel-kreimeyer/Legal/pkg/parse/dxf"
	"github.com/samuel-kreimeyer/Legal/pkg/parse/ifc"
	"github.com/samuel-kreimeyer/Legal/pkg/parse/landxml"
)

type Pipeline struct {
	dxf     parse.Parser
	ifc     parse.Parser
	landxml parse.Parser
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		dxf:     dxf.NewParser(),
		ifc:     ifc.NewParser(),
		landxml: landxml.NewParser(),
	}
}

func (p *Pipeline) ParseFile(ctx context.Context, path string, format parse.SourceFormat) ([]model.Parcel, error) {
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

	return p.ParseReader(ctx, f, format)
}

func (p *Pipeline) ParseReader(ctx context.Context, reader io.Reader, format parse.SourceFormat) ([]model.Parcel, error) {
	parser, err := p.parserForFormat(format)
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

func (p *Pipeline) parserForFormat(format parse.SourceFormat) (parse.Parser, error) {
	switch format {
	case parse.SourceFormatLandXML:
		return p.landxml, nil
	case parse.SourceFormatDXF:
		return p.dxf, nil
	case parse.SourceFormatIFC:
		return p.ifc, nil
	default:
		return nil, fmt.Errorf("unsupported source format: %q", format)
	}
}
