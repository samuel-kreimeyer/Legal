package ifc

import (
	"context"
	"fmt"
	"io"
	"regexp"
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

	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("IFC read failed: %w", err)
	}
	entities, err := parseEntities(string(raw))
	if err != nil {
		return nil, err
	}
	if len(entities) == 0 {
		return nil, fmt.Errorf("no IFC entities found")
	}

	factor := detectLengthFactor(entities)
	points, err := parseCartesianPoints(entities, factor)
	if err != nil {
		return nil, err
	}
	pointLists2D, err := parsePointLists2D(entities, factor)
	if err != nil {
		return nil, err
	}
	polylines, err := parsePolylines(entities, points)
	if err != nil {
		return nil, err
	}
	indexedCurves, err := parseIndexedPolyCurves(entities, pointLists2D)
	if err != nil {
		return nil, err
	}

	closedCandidates := closedCurveReferences(entities)
	seenCurves := map[int]bool{}
	parcels := make([]model.Parcel, 0, 4)
	count := 0

	emit := func(curveID int, pts []geom.Point2D, allowImplicitClose bool) error {
		if seenCurves[curveID] {
			return nil
		}
		loop, err := pointsToLoop(pts, allowImplicitClose)
		if err != nil {
			return err
		}
		seenCurves[curveID] = true
		count++
		parcels = append(parcels, model.Parcel{
			ID:     fmt.Sprintf("IFC-%d", count),
			Source: model.SourceIFC,
			Unit:   model.LinearUnitFoot,
			Loop:   loop,
		})
		return nil
	}

	for _, curveID := range closedCandidates {
		if pts, ok := polylines[curveID]; ok {
			if err := emit(curveID, pts, true); err != nil {
				return nil, fmt.Errorf("curve #%d: %w", curveID, err)
			}
			continue
		}
		if pts, ok := indexedCurves[curveID]; ok {
			if err := emit(curveID, pts, true); err != nil {
				return nil, fmt.Errorf("curve #%d: %w", curveID, err)
			}
			continue
		}
	}

	// Fallback: standalone closed polylines if no profile references were found.
	if len(parcels) == 0 {
		ids := make([]int, 0, len(polylines))
		for id := range polylines {
			ids = append(ids, id)
		}
		sortInts(ids)
		for _, id := range ids {
			if err := emit(id, polylines[id], false); err != nil {
				continue
			}
		}
	}

	if len(parcels) == 0 {
		return nil, fmt.Errorf("no closed polygon geometry extracted from IFC")
	}
	return parcels, nil
}

type ifcEntity struct {
	ID   int
	Type string
	Args string
}

func parseEntities(src string) (map[int]ifcEntity, error) {
	entities := map[int]ifcEntity{}
	chunks := strings.Split(src, ";")
	for _, chunk := range chunks {
		line := strings.TrimSpace(chunk)
		if line == "" || !strings.HasPrefix(line, "#") {
			continue
		}
		id, typ, args, ok := parseEntityLine(line)
		if !ok {
			continue
		}
		entities[id] = ifcEntity{ID: id, Type: typ, Args: args}
	}
	return entities, nil
}

func parseEntityLine(line string) (int, string, string, bool) {
	eq := strings.Index(line, "=")
	if eq < 0 {
		return 0, "", "", false
	}
	idPart := strings.TrimSpace(line[:eq])
	if !strings.HasPrefix(idPart, "#") {
		return 0, "", "", false
	}
	id, err := strconv.Atoi(strings.TrimPrefix(idPart, "#"))
	if err != nil {
		return 0, "", "", false
	}

	rest := strings.TrimSpace(line[eq+1:])
	lp := strings.Index(rest, "(")
	rp := strings.LastIndex(rest, ")")
	if lp <= 0 || rp <= lp {
		return 0, "", "", false
	}
	typ := strings.ToUpper(strings.TrimSpace(rest[:lp]))
	args := strings.TrimSpace(rest[lp+1 : rp])
	return id, typ, args, true
}

func detectLengthFactor(entities map[int]ifcEntity) float64 {
	for _, e := range entities {
		if e.Type != "IFCSIUNIT" {
			continue
		}
		parts := splitTopLevel(e.Args, ',')
		if len(parts) < 4 {
			continue
		}
		if !strings.Contains(strings.ToUpper(parts[1]), "LENGTHUNIT") {
			continue
		}
		prefix := normalizeEnum(parts[2])
		name := normalizeEnum(parts[3])
		return ifcSIToFeet(prefix, name)
	}

	for _, e := range entities {
		if e.Type != "IFCCONVERSIONBASEDUNIT" {
			continue
		}
		if strings.Contains(strings.ToUpper(e.Args), "'FOOT'") || strings.Contains(strings.ToUpper(e.Args), "'FEET'") {
			return 1.0
		}
	}
	return 1.0
}

func ifcSIToFeet(prefix, name string) float64 {
	unitMeters := 1.0
	switch prefix {
	case "MILLI":
		unitMeters = 0.001
	case "CENTI":
		unitMeters = 0.01
	case "DECI":
		unitMeters = 0.1
	case "", "$":
		unitMeters = 1.0
	case "KILO":
		unitMeters = 1000.0
	}

	switch name {
	case "METRE", "METER":
		return unitMeters * 3.280839895013123
	case "FOOT", "FEET":
		return 1.0
	case "INCH":
		return 1.0 / 12.0
	default:
		return unitMeters * 3.280839895013123
	}
}

func parseCartesianPoints(entities map[int]ifcEntity, factor float64) (map[int]geom.Point2D, error) {
	out := map[int]geom.Point2D{}
	for id, e := range entities {
		if e.Type != "IFCCARTESIANPOINT" {
			continue
		}
		values := parseNumberList(strings.TrimSpace(e.Args))
		if len(values) < 2 {
			return nil, fmt.Errorf("IFCCARTESIANPOINT #%d has <2 coordinates", id)
		}
		out[id] = geom.Point2D{XFeet: values[0] * factor, YFeet: values[1] * factor}
	}
	return out, nil
}

func parsePointLists2D(entities map[int]ifcEntity, factor float64) (map[int][]geom.Point2D, error) {
	out := map[int][]geom.Point2D{}
	for id, e := range entities {
		if e.Type != "IFCCARTESIANPOINTLIST2D" {
			continue
		}
		points, err := parsePointList2DArg(e.Args, factor)
		if err != nil {
			return nil, fmt.Errorf("IFCCARTESIANPOINTLIST2D #%d: %w", id, err)
		}
		out[id] = points
	}
	return out, nil
}

func parsePolylines(entities map[int]ifcEntity, points map[int]geom.Point2D) (map[int][]geom.Point2D, error) {
	out := map[int][]geom.Point2D{}
	for id, e := range entities {
		if e.Type != "IFCPOLYLINE" {
			continue
		}
		refs := extractRefs(e.Args)
		if len(refs) < 2 {
			continue
		}
		pts := make([]geom.Point2D, 0, len(refs))
		for _, ref := range refs {
			p, ok := points[ref]
			if !ok {
				return nil, fmt.Errorf("IFCPOLYLINE #%d references unknown point #%d", id, ref)
			}
			pts = append(pts, p)
		}
		out[id] = pts
	}
	return out, nil
}

func parseIndexedPolyCurves(entities map[int]ifcEntity, pointLists map[int][]geom.Point2D) (map[int][]geom.Point2D, error) {
	out := map[int][]geom.Point2D{}
	for id, e := range entities {
		if e.Type != "IFCINDEXEDPOLYCURVE" {
			continue
		}
		parts := splitTopLevel(e.Args, ',')
		if len(parts) < 1 {
			continue
		}
		pointListID, ok := parseRef(parts[0])
		if !ok {
			return nil, fmt.Errorf("IFCINDEXEDPOLYCURVE #%d missing point list ref", id)
		}
		pts, ok := pointLists[pointListID]
		if !ok {
			return nil, fmt.Errorf("IFCINDEXEDPOLYCURVE #%d references unknown point list #%d", id, pointListID)
		}
		// Initial version supports implicit-segment curves only.
		if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "$" {
			return nil, fmt.Errorf("IFCINDEXEDPOLYCURVE #%d with explicit segments is not supported yet", id)
		}
		out[id] = append([]geom.Point2D{}, pts...)
	}
	return out, nil
}

func closedCurveReferences(entities map[int]ifcEntity) []int {
	ids := make([]int, 0, 8)
	seen := map[int]bool{}

	add := func(ref int) {
		if !seen[ref] {
			seen[ref] = true
			ids = append(ids, ref)
		}
	}

	for _, e := range entities {
		switch e.Type {
		case "IFCARBITRARYCLOSEDPROFILEDEF":
			parts := splitTopLevel(e.Args, ',')
			if len(parts) >= 3 {
				if ref, ok := parseRef(parts[2]); ok {
					add(ref)
				}
			}
		case "IFCARBITRARYPROFILEDEFWITHVOIDS":
			parts := splitTopLevel(e.Args, ',')
			if len(parts) >= 3 {
				if ref, ok := parseRef(parts[2]); ok {
					add(ref)
				}
			}
			if len(parts) >= 4 {
				for _, ref := range extractRefs(parts[3]) {
					add(ref)
				}
			}
		}
	}

	sortInts(ids)
	return ids
}

func pointsToLoop(points []geom.Point2D, allowImplicitClose bool) (model.BoundaryLoop, error) {
	if len(points) < 3 {
		return model.BoundaryLoop{}, fmt.Errorf("polygon has fewer than 3 points")
	}
	pts := append([]geom.Point2D{}, points...)

	if !pointsNear(pts[0], pts[len(pts)-1]) {
		if !allowImplicitClose {
			return model.BoundaryLoop{}, fmt.Errorf("polyline is not explicitly closed")
		}
		pts = append(pts, pts[0])
	}
	if len(pts) < 4 {
		return model.BoundaryLoop{}, fmt.Errorf("closed polygon has insufficient points")
	}

	segs := make([]model.Segment, 0, len(pts)-1)
	for i := 0; i < len(pts)-1; i++ {
		if geom.DistanceFeet(pts[i], pts[i+1]) <= geom.EpsilonFeet {
			continue
		}
		segs = append(segs, model.LineSegment{P0: pts[i], P1: pts[i+1]})
	}
	if len(segs) < 3 {
		return model.BoundaryLoop{}, fmt.Errorf("polygon collapsed after removing zero-length edges")
	}
	return model.BoundaryLoop{Segments: segs}, nil
}

func parseNumberList(arg string) []float64 {
	trimmed := strings.TrimSpace(arg)
	for strings.HasPrefix(trimmed, "(") && strings.HasSuffix(trimmed, ")") {
		trimmed = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	}
	parts := splitTopLevel(trimmed, ',')
	out := make([]float64, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err == nil {
			out = append(out, v)
		}
	}
	return out
}

var pointPairRe = regexp.MustCompile(`\(\s*([\-+]?\d*\.?\d+(?:[eE][\-+]?\d+)?)\s*,\s*([\-+]?\d*\.?\d+(?:[eE][\-+]?\d+)?)`)

func parsePointList2DArg(arg string, factor float64) ([]geom.Point2D, error) {
	matches := pointPairRe.FindAllStringSubmatch(arg, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no 2D points found")
	}
	out := make([]geom.Point2D, 0, len(matches))
	for _, m := range matches {
		x, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return nil, err
		}
		y, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			return nil, err
		}
		out = append(out, geom.Point2D{XFeet: x * factor, YFeet: y * factor})
	}
	return out, nil
}

func splitTopLevel(s string, sep rune) []string {
	var parts []string
	start := 0
	depth := 0
	inString := false

	for i, r := range s {
		if r == '\'' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if r == sep && depth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(s[start:]))
	return parts
}

func extractRefs(s string) []int {
	matches := regexp.MustCompile(`#\d+`).FindAllString(s, -1)
	out := make([]int, 0, len(matches))
	for _, m := range matches {
		if id, err := strconv.Atoi(strings.TrimPrefix(m, "#")); err == nil {
			out = append(out, id)
		}
	}
	return out
}

func parseRef(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "#") {
		return 0, false
	}
	id, err := strconv.Atoi(strings.TrimPrefix(s, "#"))
	return id, err == nil
}

func normalizeEnum(s string) string {
	s = strings.TrimSpace(strings.ToUpper(s))
	s = strings.Trim(s, ".")
	return s
}

func pointsNear(a, b geom.Point2D) bool {
	return geom.DistanceFeet(a, b) <= geom.EpsilonFeet
}

func sortInts(v []int) {
	for i := 1; i < len(v); i++ {
		j := i
		for j > 0 && v[j-1] > v[j] {
			v[j-1], v[j] = v[j], v[j-1]
			j--
		}
	}
}
