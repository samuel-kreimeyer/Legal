package survey_test

import (
	"strings"
	"testing"

	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/geom"
	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/model"
	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/survey"
)

// --- ParseSurveyFile ---

func TestParseSurveyFile_CSV(t *testing.T) {
	input := `PT,CODE,NORTHING,EASTING,ELEVATION,DESCRIPTION
200,IP,568234.567,1234567.890,456.789,FOUND IRON PIN
201,MBD,568300.000,1234650.000,457.100,SET BDY MON
`
	pts, err := survey.ParseSurveyFile(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pts) != 2 {
		t.Fatalf("expected 2 points, got %d", len(pts))
	}

	p := pts[0]
	if p.Number != 200 || p.Code != "IP" {
		t.Errorf("point 0: got number=%d code=%q", p.Number, p.Code)
	}
	if p.Northing != 568234.567 || p.Easting != 1234567.890 {
		t.Errorf("point 0 coords: N=%f E=%f", p.Northing, p.Easting)
	}

	p2 := pts[1]
	if p2.Number != 201 || p2.Code != "MBD" {
		t.Errorf("point 1: got number=%d code=%q", p2.Number, p2.Code)
	}
}

func TestParseSurveyFile_CSV_AlternateHeaders(t *testing.T) {
	input := `NUMBER,FEATURE,N,E,Z,DESC
300,MRW,100.0,200.0,50.0,ROW MON
`
	pts, err := survey.ParseSurveyFile(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pts) != 1 {
		t.Fatalf("expected 1 point, got %d", len(pts))
	}
	if pts[0].Number != 300 || pts[0].Code != "MRW" {
		t.Errorf("unexpected point: %+v", pts[0])
	}
}

func TestParseSurveyFile_SpaceDelimited_CodeFirst(t *testing.T) {
	// PT CODE N E Z DESC
	input := `200 IP 568234.567 1234567.890 456.789 FOUND IRON PIN
201 MBD 568300.000 1234650.000 457.100 SET BDY MON
`
	pts, err := survey.ParseSurveyFile(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pts) != 2 {
		t.Fatalf("expected 2 points, got %d", len(pts))
	}
	if pts[0].Number != 200 || pts[0].Code != "IP" {
		t.Errorf("point 0: %+v", pts[0])
	}
	if pts[1].Code != "MBD" {
		t.Errorf("point 1 code: %q", pts[1].Code)
	}
}

func TestParseSurveyFile_SpaceDelimited_CoordsFirst(t *testing.T) {
	// PT N E Z CODE DESC
	input := `200 568234.567 1234567.890 456.789 IP FOUND IRON PIN`
	pts, err := survey.ParseSurveyFile(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pts) != 1 {
		t.Fatalf("expected 1 point, got %d", len(pts))
	}
	if pts[0].Code != "IP" || pts[0].Northing != 568234.567 {
		t.Errorf("unexpected point: %+v", pts[0])
	}
}

func TestParseSurveyFile_SkipsComments(t *testing.T) {
	input := `# This is a comment
PT,CODE,NORTHING,EASTING
# Another comment
200,IP,100.0,200.0
`
	pts, err := survey.ParseSurveyFile(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pts) != 1 {
		t.Fatalf("expected 1 point, got %d", len(pts))
	}
}

func TestParseSurveyFile_Empty(t *testing.T) {
	pts, err := survey.ParseSurveyFile(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pts) != 0 {
		t.Fatalf("expected 0 points, got %d", len(pts))
	}
}

// --- DescribeCode ---

func TestDescribeCode_KnownCodes(t *testing.T) {
	cases := []struct {
		code    survey.FeatureCode
		wantSub string
		wantKind survey.MonumentKind
	}{
		{"IP", "found iron pin", survey.KindFound},
		{"MBD", "standard boundary monument", survey.KindSet},
		{"MRW", "standard right-of-way monument", survey.KindSet},
		{"LC", "calculated point", survey.KindCalc},
		{"RE", "existing right-of-way", survey.KindROW},
	}
	for _, c := range cases {
		desc, kind := survey.DescribeCode(c.code)
		if !strings.Contains(desc, c.wantSub) {
			t.Errorf("DescribeCode(%q) = %q; want substring %q", c.code, desc, c.wantSub)
		}
		if kind != c.wantKind {
			t.Errorf("DescribeCode(%q) kind = %v; want %v", c.code, kind, c.wantKind)
		}
	}
}

func TestDescribeCode_Unknown(t *testing.T) {
	desc, kind := survey.DescribeCode("ZZZZ")
	if desc == "" {
		t.Error("expected non-empty description for unknown code")
	}
	if kind != survey.KindCalc {
		t.Errorf("expected KindCalc for unknown code, got %v", kind)
	}
}

// --- MatchToLoop ---

func TestMatchToLoop_MatchesByProximity(t *testing.T) {
	loop := model.BoundaryLoop{
		Segments: []model.Segment{
			model.LineSegment{P0: geom.Point2D{XFeet: 0, YFeet: 0}, P1: geom.Point2D{XFeet: 100, YFeet: 0}},
			model.LineSegment{P0: geom.Point2D{XFeet: 100, YFeet: 0}, P1: geom.Point2D{XFeet: 100, YFeet: 100}},
			model.LineSegment{P0: geom.Point2D{XFeet: 100, YFeet: 100}, P1: geom.Point2D{XFeet: 0, YFeet: 0}},
		},
	}
	// Survey points: XFeet=Easting, YFeet=Northing.
	points := []survey.Point{
		{Number: 200, Code: "IP", Easting: 0.1, Northing: 0.1},   // near vertex 0 (0,0)
		{Number: 201, Code: "MBD", Easting: 100.2, Northing: 0.0}, // near vertex 1 (100,0)
		// No point near vertex 2 (100,100)
	}

	mons := survey.MatchToLoop(points, loop, 0.5)
	if len(mons) != 3 {
		t.Fatalf("expected 3 slots, got %d", len(mons))
	}
	if mons[0] == nil || mons[0].PointNumber != 200 {
		t.Errorf("vertex 0: expected monument 200, got %v", mons[0])
	}
	if mons[1] == nil || mons[1].PointNumber != 201 {
		t.Errorf("vertex 1: expected monument 201, got %v", mons[1])
	}
	if mons[2] != nil {
		t.Errorf("vertex 2: expected nil monument, got %v", mons[2])
	}
}

func TestMatchToLoop_OutsideTolerance(t *testing.T) {
	loop := model.BoundaryLoop{
		Segments: []model.Segment{
			model.LineSegment{P0: geom.Point2D{XFeet: 0, YFeet: 0}, P1: geom.Point2D{XFeet: 100, YFeet: 0}},
		},
	}
	points := []survey.Point{
		{Number: 200, Code: "IP", Easting: 5.0, Northing: 0.0}, // 5 ft away from vertex
	}

	mons := survey.MatchToLoop(points, loop, 0.5)
	if len(mons) != 1 {
		t.Fatalf("expected 1 slot, got %d", len(mons))
	}
	if mons[0] != nil {
		t.Errorf("expected nil (outside tolerance), got %v", mons[0])
	}
}

func TestMatchToLoop_EmptyInputs(t *testing.T) {
	loop := model.BoundaryLoop{
		Segments: []model.Segment{
			model.LineSegment{P0: geom.Point2D{XFeet: 0, YFeet: 0}, P1: geom.Point2D{XFeet: 10, YFeet: 0}},
		},
	}
	if mons := survey.MatchToLoop(nil, loop, 0.5); mons != nil {
		t.Errorf("expected nil for empty points, got %v", mons)
	}
	if mons := survey.MatchToLoop([]survey.Point{{Number: 1, Code: "IP"}}, model.BoundaryLoop{}, 0.5); mons != nil {
		t.Errorf("expected nil for empty loop, got %v", mons)
	}
}
