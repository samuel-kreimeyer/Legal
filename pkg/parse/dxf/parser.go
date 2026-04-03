package dxf

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/geom"
	"github.com/samuel-kreimeyer/Curatores-Viarum/packages/legal-description/pkg/model"
)

// Options configures the DXF parser.
type Options struct {
	// LayerFilter restricts parsing to entities on the named layers.
	// Names are compared case-insensitively. An empty slice means all layers.
	//
	// When two or more layer names are provided the parser operates in
	// multi-layer assembly mode: segments from all named layers are pooled
	// together before loop-building, so a closed boundary can be formed from
	// linework that spans two layers (e.g. a property-line layer that supplies
	// three sides and a ROW layer that supplies the fourth).
	LayerFilter []string
}

type Parser struct {
	opts Options
}

func NewParser() *Parser {
	return &Parser{}
}

func NewParserWithOptions(opts Options) *Parser {
	return &Parser{opts: opts}
}

func (p *Parser) Parse(ctx context.Context, r io.Reader) ([]model.Parcel, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	pairs, err := readGroupPairs(r)
	if err != nil {
		return nil, err
	}
	sections, err := splitSections(pairs)
	if err != nil {
		return nil, err
	}

	unitFactor := 1.0
	if header, ok := sections["HEADER"]; ok {
		unitFactor = parseLinearUnitFactor(header)
	}

	entities, ok := sections["ENTITIES"]
	if !ok {
		return nil, fmt.Errorf("DXF file missing ENTITIES section")
	}

	filter := buildFilterSet(p.opts.LayerFilter)
	directLoops, chainables, err := parseEntities(entities, unitFactor, filter)
	if err != nil {
		return nil, err
	}
	if len(directLoops) == 0 && len(chainables) == 0 {
		return nil, fmt.Errorf("no supported closed geometry found in DXF")
	}

	// Multi-layer assembly: dissolve all direct loops into their constituent
	// segments and merge with the open chainables, then build closed loops from
	// the combined pool. Independent boundaries on each layer will still form
	// separate loops; segments from different layers that share endpoints will
	// be chained together into a single closed boundary.
	if len(p.opts.LayerFilter) > 1 {
		return buildMergedParcels(directLoops, chainables, p.opts.LayerFilter)
	}

	// Single-layer or unfiltered mode: keep existing per-layer behaviour.
	parcels := make([]model.Parcel, 0, len(directLoops)+1)
	layerCounts := map[string]int{}

	for _, loop := range directLoops {
		layer := normalizedLayer(loop.layer)
		layerCounts[layer]++
		parcels = append(parcels, model.Parcel{
			ID:     fmt.Sprintf("DXF-%s-%d", layer, layerCounts[layer]),
			Source: model.SourceDXF,
			Unit:   model.LinearUnitFoot,
			Loop:   loop.loop,
		})
	}

	byLayer := map[string][]model.Segment{}
	for _, seg := range chainables {
		layer := normalizedLayer(seg.layer)
		byLayer[layer] = append(byLayer[layer], seg.segment)
	}

	layers := make([]string, 0, len(byLayer))
	for layer := range byLayer {
		layers = append(layers, layer)
	}
	sort.Strings(layers)

	for _, layer := range layers {
		loops, err := buildClosedLoops(byLayer[layer])
		if err != nil {
			return nil, fmt.Errorf("layer %q: %w", layer, err)
		}
		for _, loop := range loops {
			layerCounts[layer]++
			parcels = append(parcels, model.Parcel{
				ID:     fmt.Sprintf("DXF-%s-%d", layer, layerCounts[layer]),
				Source: model.SourceDXF,
				Unit:   model.LinearUnitFoot,
				Loop:   loop,
			})
		}
	}

	return parcels, nil
}

// buildFilterSet converts a slice of layer names into a normalised lookup set.
// An empty slice returns a nil map, which callers treat as "accept all".
func buildFilterSet(layers []string) map[string]bool {
	if len(layers) == 0 {
		return nil
	}
	m := make(map[string]bool, len(layers))
	for _, l := range layers {
		m[normalizedLayer(l)] = true
	}
	return m
}

// layerAllowed reports whether layer passes the filter. A nil filter accepts all.
func layerAllowed(filter map[string]bool, layer string) bool {
	if filter == nil {
		return true
	}
	return filter[normalizedLayer(layer)]
}

// buildMergedParcels pools segments from all provided direct loops and open
// chainables, splits them at cross-layer intersection points, and builds closed
// loops from the combined set.
//
// The layers slice controls seed ordering: segments from the last-listed layer
// appear first so the greedy loop-builder finds the acquisition area (which
// uses those segments) before attempting to reconstruct the residual property
// boundary, which is intentionally incomplete after splitting.
//
// Independent closed boundaries on different layers remain as separate parcels.
func buildMergedParcels(directLoops []loopWithLayer, chainables []layerSegment, layers []string) ([]model.Parcel, error) {
	// Group segments by layer.
	byLayer := make(map[string][]model.Segment, len(layers))
	for _, lw := range directLoops {
		layer := normalizedLayer(lw.layer)
		byLayer[layer] = append(byLayer[layer], lw.loop.Segments...)
	}
	for _, ls := range chainables {
		layer := normalizedLayer(ls.layer)
		byLayer[layer] = append(byLayer[layer], ls.segment)
	}

	// Normalise layer keys to match the filter list.
	normLayers := make([]string, len(layers))
	for i, l := range layers {
		normLayers[i] = normalizedLayer(l)
	}

	// Only include layers that actually have geometry.
	activeLayers := normLayers[:0:len(normLayers)]
	seen := map[string]bool{}
	for _, l := range normLayers {
		if _, ok := byLayer[l]; ok && !seen[l] {
			activeLayers = append(activeLayers, l)
			seen[l] = true
		}
	}

	all := splitAtCrossLayerIntersections(byLayer, activeLayers)
	if len(all) == 0 {
		return nil, fmt.Errorf("multi-layer assembly produced no segments")
	}

	loops := buildClosedLoopsBestEffort(all)
	if len(loops) == 0 {
		return nil, fmt.Errorf("multi-layer assembly: no closed loops could be formed from the combined geometry")
	}

	parcels := make([]model.Parcel, len(loops))
	for i, loop := range loops {
		parcels[i] = model.Parcel{
			ID:     fmt.Sprintf("DXF-COMBINED-%d", i+1),
			Source: model.SourceDXF,
			Unit:   model.LinearUnitFoot,
			Loop:   loop,
		}
	}
	return parcels, nil
}

// buildClosedLoopsBestEffort is like buildClosedLoops but silently discards any
// chain of segments that cannot be closed rather than returning an error. This
// is correct behaviour for multi-layer assembly where the combined pool may
// contain residual property-boundary segments that are intentionally incomplete
// after intersection splitting (the shared ROW edge was already consumed by the
// acquisition-area loop).
func buildClosedLoopsBestEffort(segments []model.Segment) []model.BoundaryLoop {
	used := make([]bool, len(segments))
	var loops []model.BoundaryLoop

	for {
		seed := -1
		for i := range segments {
			if !used[i] {
				seed = i
				break
			}
		}
		if seed == -1 {
			break
		}

		used[seed] = true
		chain := []model.Segment{segments[seed]}
		start := chain[0].Start()
		cursor := chain[0].End()
		closed := false

		for range segments {
			if len(chain) >= 3 && pointsNear(cursor, start) {
				closed = true
				break
			}

			found := -1
			reversed := false
			for j := range segments {
				if used[j] {
					continue
				}
				if pointsNear(cursor, segments[j].Start()) {
					found = j
					break
				}
				if pointsNear(cursor, segments[j].End()) {
					found = j
					reversed = true
					break
				}
			}
			if found == -1 {
				break
			}

			next := segments[found]
			if reversed {
				rev, err := reverseSegment(next)
				if err != nil {
					break
				}
				next = rev
			}
			used[found] = true
			chain = append(chain, next)
			cursor = next.End()
		}

		if closed {
			loops = append(loops, model.BoundaryLoop{Segments: chain})
		}
		// Segments consumed by a failed chain are marked used and discarded —
		// they are the residual property boundary that can't close without the
		// shared acquisition edge.
	}

	return loops
}

type groupPair struct {
	code  int
	value string
}

type loopWithLayer struct {
	layer string
	loop  model.BoundaryLoop
}

type layerSegment struct {
	layer   string
	segment model.Segment
}

func readGroupPairs(r io.Reader) ([]groupPair, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024), 4*1024*1024)
	lines := make([]string, 0, 4096)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("DXF read failed: %w", err)
	}
	if len(lines)%2 != 0 {
		return nil, fmt.Errorf("invalid DXF: odd number of group lines")
	}

	pairs := make([]groupPair, 0, len(lines)/2)
	for i := 0; i+1 < len(lines); i += 2 {
		codeLine := strings.TrimSpace(strings.TrimPrefix(lines[i], "\uFEFF"))
		code, err := strconv.Atoi(codeLine)
		if err != nil {
			return nil, fmt.Errorf("invalid DXF group code %q: %w", codeLine, err)
		}
		pairs = append(pairs, groupPair{
			code:  code,
			value: strings.TrimSpace(lines[i+1]),
		})
	}
	return pairs, nil
}

func splitSections(pairs []groupPair) (map[string][]groupPair, error) {
	sections := map[string][]groupPair{}
	for i := 0; i < len(pairs)-1; i++ {
		if pairs[i].code != 0 || !strings.EqualFold(pairs[i].value, "SECTION") {
			continue
		}
		if pairs[i+1].code != 2 {
			return nil, fmt.Errorf("malformed DXF SECTION at pair index %d", i)
		}
		name := strings.ToUpper(strings.TrimSpace(pairs[i+1].value))
		start := i + 2
		j := start
		for j < len(pairs) {
			if pairs[j].code == 0 && strings.EqualFold(pairs[j].value, "ENDSEC") {
				break
			}
			j++
		}
		if j >= len(pairs) {
			return nil, fmt.Errorf("section %q missing ENDSEC", name)
		}
		sections[name] = pairs[start:j]
		i = j
	}
	return sections, nil
}

func parseLinearUnitFactor(header []groupPair) float64 {
	for i := 0; i < len(header)-1; i++ {
		if header[i].code == 9 && strings.EqualFold(header[i].value, "$INSUNITS") {
			for j := i + 1; j < len(header); j++ {
				if header[j].code == 9 {
					break
				}
				if header[j].code == 70 {
					n, err := strconv.Atoi(strings.TrimSpace(header[j].value))
					if err != nil {
						return 1.0
					}
					return insUnitsToFeet(n)
				}
			}
		}
	}
	return 1.0
}

func insUnitsToFeet(ins int) float64 {
	switch ins {
	case 1: // inches
		return 1.0 / 12.0
	case 2: // feet
		return 1.0
	case 4: // mm
		return 0.0032808398950131233
	case 5: // cm
		return 0.03280839895013123
	case 6: // m
		return 3.280839895013123
	case 7: // km
		return 3280.839895013123
	default:
		return 1.0
	}
}

func parseEntities(pairs []groupPair, factor float64, filter map[string]bool) ([]loopWithLayer, []layerSegment, error) {
	directLoops := make([]loopWithLayer, 0, 4)
	chainables := make([]layerSegment, 0, 32)

	for i := 0; i < len(pairs); {
		if pairs[i].code != 0 {
			i++
			continue
		}
		entityType := strings.ToUpper(strings.TrimSpace(pairs[i].value))

		switch entityType {
		case "POLYLINE":
			j := i + 1
			for j < len(pairs) {
				if pairs[j].code == 0 && strings.EqualFold(pairs[j].value, "SEQEND") {
					break
				}
				j++
			}
			if j >= len(pairs) {
				return nil, nil, fmt.Errorf("POLYLINE missing SEQEND")
			}
			loops, segs, err := parsePolylineEntity(pairs[i+1:j], factor)
			if err != nil {
				return nil, nil, err
			}
			for _, lw := range loops {
				if layerAllowed(filter, lw.layer) {
					directLoops = append(directLoops, lw)
				}
			}
			for _, ls := range segs {
				if layerAllowed(filter, ls.layer) {
					chainables = append(chainables, ls)
				}
			}
			i = j + 1
			continue
		}

		j := i + 1
		for j < len(pairs) && pairs[j].code != 0 {
			j++
		}
		data := pairs[i+1 : j]

		switch entityType {
		case "LINE":
			layer, seg, err := parseLineEntity(data, factor)
			if err != nil {
				return nil, nil, err
			}
			if layerAllowed(filter, layer) {
				chainables = append(chainables, layerSegment{layer: layer, segment: seg})
			}
		case "ARC":
			layer, seg, err := parseArcEntity(data, factor)
			if err != nil {
				return nil, nil, err
			}
			if layerAllowed(filter, layer) {
				chainables = append(chainables, layerSegment{layer: layer, segment: seg})
			}
		case "LWPOLYLINE":
			loops, segs, err := parseLWPolylineEntity(data, factor)
			if err != nil {
				return nil, nil, err
			}
			for _, lw := range loops {
				if layerAllowed(filter, lw.layer) {
					directLoops = append(directLoops, lw)
				}
			}
			for _, ls := range segs {
				if layerAllowed(filter, ls.layer) {
					chainables = append(chainables, ls)
				}
			}
		}
		i = j
	}
	return directLoops, chainables, nil
}

func parseLineEntity(data []groupPair, factor float64) (string, model.LineSegment, error) {
	layer := "0"
	var x1, y1, x2, y2 float64
	var ok10, ok20, ok11, ok21 bool

	for _, gp := range data {
		switch gp.code {
		case 8:
			layer = gp.value
		case 10:
			v, err := parseFloat(gp.value)
			if err != nil {
				return "", model.LineSegment{}, err
			}
			x1 = v
			ok10 = true
		case 20:
			v, err := parseFloat(gp.value)
			if err != nil {
				return "", model.LineSegment{}, err
			}
			y1 = v
			ok20 = true
		case 11:
			v, err := parseFloat(gp.value)
			if err != nil {
				return "", model.LineSegment{}, err
			}
			x2 = v
			ok11 = true
		case 21:
			v, err := parseFloat(gp.value)
			if err != nil {
				return "", model.LineSegment{}, err
			}
			y2 = v
			ok21 = true
		}
	}
	if !(ok10 && ok20 && ok11 && ok21) {
		return "", model.LineSegment{}, fmt.Errorf("LINE entity missing required coordinates")
	}

	return layer, model.LineSegment{
		P0: geom.Point2D{XFeet: x1 * factor, YFeet: y1 * factor},
		P1: geom.Point2D{XFeet: x2 * factor, YFeet: y2 * factor},
	}, nil
}

func parseArcEntity(data []groupPair, factor float64) (string, model.ArcSegment, error) {
	layer := "0"
	var cx, cy, radius, startDeg, endDeg float64
	var ok10, ok20, ok40, ok50, ok51 bool

	for _, gp := range data {
		switch gp.code {
		case 8:
			layer = gp.value
		case 10:
			v, err := parseFloat(gp.value)
			if err != nil {
				return "", model.ArcSegment{}, err
			}
			cx = v
			ok10 = true
		case 20:
			v, err := parseFloat(gp.value)
			if err != nil {
				return "", model.ArcSegment{}, err
			}
			cy = v
			ok20 = true
		case 40:
			v, err := parseFloat(gp.value)
			if err != nil {
				return "", model.ArcSegment{}, err
			}
			radius = v
			ok40 = true
		case 50:
			v, err := parseFloat(gp.value)
			if err != nil {
				return "", model.ArcSegment{}, err
			}
			startDeg = v
			ok50 = true
		case 51:
			v, err := parseFloat(gp.value)
			if err != nil {
				return "", model.ArcSegment{}, err
			}
			endDeg = v
			ok51 = true
		}
	}
	if !(ok10 && ok20 && ok40 && ok50 && ok51) {
		return "", model.ArcSegment{}, fmt.Errorf("ARC entity missing required parameters")
	}
	if radius <= 0 {
		return "", model.ArcSegment{}, fmt.Errorf("ARC entity has invalid radius %.6f", radius)
	}

	startRad := geom.NormalizeAngle(startDeg * math.Pi / 180.0)
	endRad := geom.NormalizeAngle(endDeg * math.Pi / 180.0)
	sweep := endRad - startRad
	if sweep <= 0.0 {
		sweep += 2.0 * math.Pi
	}

	return layer, model.ArcSegment{
		Center:        geom.Point2D{XFeet: cx * factor, YFeet: cy * factor},
		RadiusFeet:    radius * factor,
		StartAngleRad: startRad,
		SweepRad:      sweep,
	}, nil
}

type dxfVertex struct {
	x     float64
	y     float64
	bulge float64
	hasX  bool
	hasY  bool
}

func parseLWPolylineEntity(data []groupPair, factor float64) ([]loopWithLayer, []layerSegment, error) {
	layer := "0"
	flags := 0
	vertices := make([]dxfVertex, 0, 8)
	current := -1

	for _, gp := range data {
		switch gp.code {
		case 8:
			layer = gp.value
		case 70:
			n, err := strconv.Atoi(strings.TrimSpace(gp.value))
			if err == nil {
				flags = n
			}
		case 10:
			x, err := parseFloat(gp.value)
			if err != nil {
				return nil, nil, err
			}
			vertices = append(vertices, dxfVertex{x: x, hasX: true})
			current = len(vertices) - 1
		case 20:
			if current < 0 {
				return nil, nil, fmt.Errorf("LWPOLYLINE has Y coordinate before X")
			}
			y, err := parseFloat(gp.value)
			if err != nil {
				return nil, nil, err
			}
			vertices[current].y = y
			vertices[current].hasY = true
		case 42:
			if current < 0 {
				return nil, nil, fmt.Errorf("LWPOLYLINE has bulge before first vertex")
			}
			b, err := parseFloat(gp.value)
			if err != nil {
				return nil, nil, err
			}
			vertices[current].bulge = b
		}
	}

	closed := (flags & 1) == 1
	return verticesToLoopSegments(layer, vertices, closed, factor)
}

func parsePolylineEntity(data []groupPair, factor float64) ([]loopWithLayer, []layerSegment, error) {
	layer := "0"
	flags := 0
	i := 0

	for i < len(data) && data[i].code != 0 {
		switch data[i].code {
		case 8:
			layer = data[i].value
		case 70:
			n, err := strconv.Atoi(strings.TrimSpace(data[i].value))
			if err == nil {
				flags = n
			}
		}
		i++
	}

	vertices := make([]dxfVertex, 0, 8)
	for i < len(data) {
		if data[i].code != 0 || !strings.EqualFold(data[i].value, "VERTEX") {
			i++
			continue
		}
		i++
		v := dxfVertex{}
		for i < len(data) && data[i].code != 0 {
			switch data[i].code {
			case 10:
				x, err := parseFloat(data[i].value)
				if err != nil {
					return nil, nil, err
				}
				v.x = x
				v.hasX = true
			case 20:
				y, err := parseFloat(data[i].value)
				if err != nil {
					return nil, nil, err
				}
				v.y = y
				v.hasY = true
			case 42:
				b, err := parseFloat(data[i].value)
				if err != nil {
					return nil, nil, err
				}
				v.bulge = b
			}
			i++
		}
		if v.hasX && v.hasY {
			vertices = append(vertices, v)
		}
	}

	closed := (flags & 1) == 1
	return verticesToLoopSegments(layer, vertices, closed, factor)
}

func verticesToLoopSegments(layer string, vertices []dxfVertex, closed bool, factor float64) ([]loopWithLayer, []layerSegment, error) {
	for i, v := range vertices {
		if !v.hasX || !v.hasY {
			return nil, nil, fmt.Errorf("polyline vertex %d missing coordinate", i)
		}
	}
	if len(vertices) < 2 {
		return nil, nil, fmt.Errorf("polyline requires at least 2 vertices")
	}

	count := len(vertices) - 1
	if closed {
		count = len(vertices)
	}

	segments := make([]model.Segment, 0, count)
	for i := 0; i < count; i++ {
		start := vertices[i]
		end := vertices[(i+1)%len(vertices)]
		seg, err := segmentFromVertexPair(start, end, factor)
		if err != nil {
			return nil, nil, err
		}
		segments = append(segments, seg)
	}

	if closed {
		return []loopWithLayer{{
			layer: layer,
			loop:  model.BoundaryLoop{Segments: segments},
		}}, nil, nil
	}

	out := make([]layerSegment, 0, len(segments))
	for _, seg := range segments {
		out = append(out, layerSegment{layer: layer, segment: seg})
	}
	return nil, out, nil
}

func segmentFromVertexPair(start, end dxfVertex, factor float64) (model.Segment, error) {
	p0 := geom.Point2D{XFeet: start.x * factor, YFeet: start.y * factor}
	p1 := geom.Point2D{XFeet: end.x * factor, YFeet: end.y * factor}

	if math.Abs(start.bulge) <= 1e-12 {
		return model.LineSegment{P0: p0, P1: p1}, nil
	}

	theta := 4.0 * math.Atan(start.bulge)
	if math.Abs(theta) <= 1e-12 {
		return model.LineSegment{P0: p0, P1: p1}, nil
	}

	chord := geom.DistanceFeet(p0, p1)
	if chord <= geom.EpsilonFeet {
		return model.LineSegment{P0: p0, P1: p1}, nil
	}

	radius := chord / (2.0 * math.Sin(math.Abs(theta)/2.0))
	if radius <= 0.0 || math.IsNaN(radius) || math.IsInf(radius, 0) {
		return nil, fmt.Errorf("invalid bulge-derived radius")
	}

	mid := geom.Point2D{
		XFeet: (p0.XFeet + p1.XFeet) / 2.0,
		YFeet: (p0.YFeet + p1.YFeet) / 2.0,
	}
	ux := (p1.XFeet - p0.XFeet) / chord
	uy := (p1.YFeet - p0.YFeet) / chord
	leftNormalX := -uy
	leftNormalY := ux

	h2 := radius*radius - (chord*chord)/4.0
	if h2 < 0.0 && math.Abs(h2) < 1e-9 {
		h2 = 0.0
	}
	if h2 < 0.0 {
		return nil, fmt.Errorf("invalid bulge geometry")
	}
	h := math.Sqrt(h2)

	c1 := geom.Point2D{
		XFeet: mid.XFeet + leftNormalX*h,
		YFeet: mid.YFeet + leftNormalY*h,
	}
	c2 := geom.Point2D{
		XFeet: mid.XFeet - leftNormalX*h,
		YFeet: mid.YFeet - leftNormalY*h,
	}

	cand1 := arcCandidate(c1, p0, p1, theta)
	cand2 := arcCandidate(c2, p0, p1, theta)
	best := cand1
	center := c1
	if cand2.delta < cand1.delta {
		best = cand2
		center = c2
	}

	return model.ArcSegment{
		Center:        center,
		RadiusFeet:    radius,
		StartAngleRad: best.startAngle,
		SweepRad:      best.sweep,
	}, nil
}

type arcSweepCandidate struct {
	startAngle float64
	sweep      float64
	delta      float64
}

func arcCandidate(center, start, end geom.Point2D, desiredSweep float64) arcSweepCandidate {
	startAngle := geom.PointAngle(center, start)
	endAngle := geom.PointAngle(center, end)

	ccw := endAngle - startAngle
	if ccw < 0.0 {
		ccw += 2.0 * math.Pi
	}
	cw := ccw - 2.0*math.Pi

	chosen := ccw
	if desiredSweep < 0.0 {
		chosen = cw
	}
	return arcSweepCandidate{
		startAngle: startAngle,
		sweep:      chosen,
		delta:      math.Abs(math.Abs(chosen) - math.Abs(desiredSweep)),
	}
}

func buildClosedLoops(segments []model.Segment) ([]model.BoundaryLoop, error) {
	if len(segments) == 0 {
		return nil, nil
	}

	used := make([]bool, len(segments))
	loops := make([]model.BoundaryLoop, 0, 2)

	for {
		seed := -1
		for i := range segments {
			if !used[i] {
				seed = i
				break
			}
		}
		if seed == -1 {
			break
		}

		used[seed] = true
		chain := []model.Segment{segments[seed]}
		start := chain[0].Start()
		cursor := chain[0].End()
		closed := false

		for iter := 0; iter < len(segments)*2; iter++ {
			if len(chain) >= 3 && pointsNear(cursor, start) {
				closed = true
				break
			}

			found := -1
			reversed := false
			for j := range segments {
				if used[j] {
					continue
				}
				if pointsNear(cursor, segments[j].Start()) {
					found = j
					break
				}
				if pointsNear(cursor, segments[j].End()) {
					found = j
					reversed = true
					break
				}
			}
			if found == -1 {
				return nil, fmt.Errorf("unable to close loop from segment chain of length %d", len(chain))
			}

			next := segments[found]
			if reversed {
				rev, err := reverseSegment(next)
				if err != nil {
					return nil, err
				}
				next = rev
			}
			used[found] = true
			chain = append(chain, next)
			cursor = next.End()
		}

		if !closed {
			return nil, fmt.Errorf("loop did not close before search limit")
		}
		loops = append(loops, model.BoundaryLoop{Segments: chain})
	}

	return loops, nil
}

func reverseSegment(seg model.Segment) (model.Segment, error) {
	switch s := seg.(type) {
	case model.LineSegment:
		return model.LineSegment{P0: s.P1, P1: s.P0}, nil
	case *model.LineSegment:
		return model.LineSegment{P0: s.P1, P1: s.P0}, nil
	case model.ArcSegment:
		return model.ArcSegment{
			Center:        s.Center,
			RadiusFeet:    s.RadiusFeet,
			StartAngleRad: geom.NormalizeAngle(s.StartAngleRad + s.SweepRad),
			SweepRad:      -s.SweepRad,
		}, nil
	case *model.ArcSegment:
		return model.ArcSegment{
			Center:        s.Center,
			RadiusFeet:    s.RadiusFeet,
			StartAngleRad: geom.NormalizeAngle(s.StartAngleRad + s.SweepRad),
			SweepRad:      -s.SweepRad,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported segment type %T", seg)
	}
}

func pointsNear(a, b geom.Point2D) bool {
	return geom.DistanceFeet(a, b) <= geom.EpsilonFeet
}

func parseFloat(s string) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0.0, fmt.Errorf("invalid numeric value %q", s)
	}
	return v, nil
}

func normalizedLayer(s string) string {
	layer := strings.TrimSpace(strings.ToUpper(s))
	if layer == "" {
		return "0"
	}
	return layer
}
