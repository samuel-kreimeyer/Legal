package parse

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/samuel-kreimeyer/Legal/pkg/model"
)

type SourceFormat string

const (
	SourceFormatUnknown SourceFormat = "unknown"
	SourceFormatDXF     SourceFormat = "dxf"
	SourceFormatIFC     SourceFormat = "ifc"
	SourceFormatLandXML SourceFormat = "landxml"
)

type Parser interface {
	Parse(ctx context.Context, r io.Reader) ([]model.Parcel, error)
}

func DetectSourceFormat(path string) (SourceFormat, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".dxf":
		return SourceFormatDXF, nil
	case ".ifc":
		return SourceFormatIFC, nil
	case ".xml", ".landxml":
		return SourceFormatLandXML, nil
	default:
		return SourceFormatUnknown, fmt.Errorf("unsupported file extension: %s", ext)
	}
}
