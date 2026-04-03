package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/app"
	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/parse"
	renderlegal "github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/render/legal"
	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/survey"
)

func main() {
	usage := `legal

	Reads parcel geometry from DXF, IFC, or LandXML and prints legal description text.

	basic usage:
	legal -format=auto -kind="Utility Easement" -lot=7 -block=B -sub="Sample Addition" INPUTFILE`

	format := flag.String("format", "auto", "Input format: auto|dxf|ifc|landxml")
	kind := flag.String("kind", "", "Type of entity described, such as 'Utility Easement'")
	lot := flag.String("lot", "", "Lot number (or letter)")
	block := flag.String("block", "", "Block number (or letter)")
	start := flag.String("start", "northeast", "Starting corner label for description (for example: northeast)")
	sub := flag.String("sub", "", "Subdivision name")
	city := flag.String("city", "North Little Rock", "City name")
	county := flag.String("county", "Pulaski", "County name")
	state := flag.String("state", "Arkansas", "State name")
	commencing := flag.Bool("commencing", false, "Use 'COMMENCING' preamble instead of 'BEGINNING'")
	area := flag.Float64("area", 0.0, "Override area value in square feet (0 uses parser-derived area when available)")
	areaUnit := flag.String("area-unit", "square feet", "Area display units text")
	parcel := flag.Int("parcel", 1, "1-based parcel index when input contains multiple parcels")
	allParcels := flag.Bool("all", false, "Render all parsed parcels")
	surveyFile := flag.String("survey-file", "", "Survey point file (CSV or space-delimited) for monument annotations")
	surveyTolerance := flag.Float64("survey-tolerance", 0.5, "Maximum distance in feet to match survey points to parcel vertices")
	var layers stringSlice
	flag.Var(&layers, "layer", "DXF layer name to include (repeat for multiple layers; two or more triggers multi-layer assembly)")

	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Println(usage)
		fmt.Println("Arguments:")
		flag.PrintDefaults()
		return
	}

	inputPath := flag.Args()[0]
	sourceFormat, err := parseFormatFlag(*format)
	if err != nil {
		failf("%v\n", err)
	}

	pipeline := app.NewPipeline()
	parseOpts := app.ParseOptions{LayerFilter: []string(layers)}
	parcels, err := pipeline.ParseFile(context.Background(), inputPath, sourceFormat, parseOpts)
	if err != nil {
		failf("parse failed: %v\n", err)
	}
	if len(parcels) == 0 {
		failf("no parcels parsed from input\n")
	}

	if *surveyFile != "" {
		sf, err := os.Open(*surveyFile)
		if err != nil {
			failf("survey file open failed: %v\n", err)
		}
		surveyPoints, err := survey.ParseSurveyFile(sf)
		sf.Close()
		if err != nil {
			failf("survey file parse failed: %v\n", err)
		}
		parcels = pipeline.AnnotateWithSurvey(parcels, surveyPoints, *surveyTolerance)
	}

	indices, err := selectedParcelIndices(len(parcels), *parcel, *allParcels)
	if err != nil {
		failf("%v\n", err)
	}

	renderOpts := renderlegal.Options{
		Kind:            *kind,
		Lot:             *lot,
		Block:           *block,
		Subdivision:     *sub,
		City:            *city,
		County:          *county,
		State:           *state,
		StartCorner:     *start,
		UseCommencing:   *commencing,
		AreaSquareFeet:  *area,
		AreaDisplayUnit: *areaUnit,
	}

	for i, idx := range indices {
		out, err := renderlegal.RenderParcel(parcels[idx], renderOpts)
		if err != nil {
			failf("render failed for parcel %d (%s): %v\n", idx+1, parcels[idx].ID, err)
		}
		if len(indices) > 1 {
			fmt.Printf("PARCEL %d (%s)\n\n", idx+1, parcels[idx].ID)
		}
		fmt.Println(out)
		if i < len(indices)-1 {
			fmt.Println()
		}
	}
}

func parseFormatFlag(v string) (parse.SourceFormat, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "auto":
		return parse.SourceFormatUnknown, nil
	case "dxf":
		return parse.SourceFormatDXF, nil
	case "ifc":
		return parse.SourceFormatIFC, nil
	case "landxml", "xml":
		return parse.SourceFormatLandXML, nil
	default:
		return parse.SourceFormatUnknown, fmt.Errorf("invalid -format value %q (expected auto|dxf|ifc|landxml)", v)
	}
}

func selectedParcelIndices(total, singleIndex int, all bool) ([]int, error) {
	if all {
		indices := make([]int, 0, total)
		for i := 0; i < total; i++ {
			indices = append(indices, i)
		}
		return indices, nil
	}
	if singleIndex < 1 || singleIndex > total {
		return nil, fmt.Errorf("invalid -parcel value %d (must be 1..%d)", singleIndex, total)
	}
	return []int{singleIndex - 1}, nil
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

// stringSlice is a flag.Value that accumulates repeated -layer flags.
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}
