package landxml

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/samuel-kreimeyer/Legal/pkg/geom"
)

func TestParseExampleLandXML(t *testing.T) {
	examplePath := filepath.Join("..", "..", "..", "example.xml")
	f, err := os.Open(examplePath)
	if err != nil {
		t.Fatalf("open example.xml failed: %v", err)
	}
	defer f.Close()

	parser := NewParser()
	parcels, err := parser.Parse(context.Background(), f)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(parcels) == 0 {
		t.Fatal("expected parcels from example.xml")
	}

	first := parcels[0]
	if len(first.Loop.Segments) == 0 {
		t.Fatal("expected first parcel to contain loop segments")
	}

	start := first.Loop.Segments[0].Start()
	end := first.Loop.Segments[len(first.Loop.Segments)-1].End()
	if geom.DistanceFeet(start, end) > geom.EpsilonFeet {
		t.Fatalf("expected closed loop within epsilon; distance=%f", geom.DistanceFeet(start, end))
	}
}
