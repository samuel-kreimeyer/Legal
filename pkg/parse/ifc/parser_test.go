package ifc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samuel-kreimeyer/Legal/pkg/geom"
	"github.com/samuel-kreimeyer/Legal/pkg/model"
)

func TestParseSquareProfileIFC(t *testing.T) {
	fixture := filepath.Join("..", "..", "..", "tests", "fixtures", "ifc", "square_profile.ifc")
	f, err := os.Open(fixture)
	if err != nil {
		t.Fatalf("open fixture failed: %v", err)
	}
	defer f.Close()

	p := NewParser()
	parcels, err := p.Parse(context.Background(), f)
	if err != nil {
		t.Fatalf("IFC parse failed: %v", err)
	}
	if len(parcels) != 1 {
		t.Fatalf("expected 1 parcel, got %d", len(parcels))
	}
	if parcels[0].Source != model.SourceIFC {
		t.Fatalf("expected IFC source, got %q", parcels[0].Source)
	}
	if len(parcels[0].Loop.Segments) != 4 {
		t.Fatalf("expected 4 segments, got %d", len(parcels[0].Loop.Segments))
	}

	start := parcels[0].Loop.Segments[0].Start()
	end := parcels[0].Loop.Segments[len(parcels[0].Loop.Segments)-1].End()
	if geom.DistanceFeet(start, end) > geom.EpsilonFeet {
		t.Fatalf("expected closed loop within epsilon; distance=%f", geom.DistanceFeet(start, end))
	}
}

func TestParseIndexedPolyCurveWithExplicitSegmentsReturnsError(t *testing.T) {
	src := `ISO-10303-21;
HEADER;
FILE_SCHEMA(('IFC4'));
ENDSEC;
DATA;
#1=IFCSIUNIT(*,.LENGTHUNIT.,$,.METRE.);
#10=IFCCARTESIANPOINTLIST2D(((0.,0.),(30.48,0.),(30.48,15.24),(0.,15.24)));
#11=IFCLINEINDEX((1,2));
#12=IFCLINEINDEX((2,3));
#13=IFCLINEINDEX((3,4));
#14=IFCLINEINDEX((4,1));
#20=IFCINDEXEDPOLYCURVE(#10,(#11,#12,#13,#14),.F.);
#30=IFCARBITRARYCLOSEDPROFILEDEF(.AREA.,'Profile',#20);
ENDSEC;
END-ISO-10303-21;`

	p := NewParser()
	_, err := p.Parse(context.Background(), strings.NewReader(src))
	if err == nil {
		t.Fatal("expected parse error for indexed polycurve with explicit segments")
	}
	if !strings.Contains(err.Error(), "explicit segments") {
		t.Fatalf("expected explicit-segments error, got: %v", err)
	}
}
