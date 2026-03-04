# Legal Description Overhaul v2 Plan

## Agreed Requirements
- Supported geometry sources: `DXF`, `IFC`, `LandXML`.
- Input source of truth: closed polygon geometry, not freeform narrative text.
- Use a canonical internal representation shared by all parsers.
- Keep geometric tolerance strict: epsilon must be less than `0.01 ft`.
- Move formatting logic from fragile template state to deterministic renderer logic.
- Rebuild tests around fixtures and enforce green CI.

## v2 Architecture

### Top-Level Packages
- `cmd/legal`:
  - Main CLI entrypoint.
  - Accepts file path, source type autodetect/override, output options.
- `pkg/geom`:
  - Core geometry primitives, unit normalization, tolerances.
- `pkg/model`:
  - Canonical domain model for parcel boundary segments and metadata.
- `pkg/parse/dxf`:
  - Extract closed boundary loops from DXF entities.
- `pkg/parse/ifc`:
  - Extract 2D boundary loop(s) from IFC geometric representations.
- `pkg/parse/landxml`:
  - Extract parcel `CoordGeom` into canonical segments.
- `pkg/normalize`:
  - Orientation normalization, loop closure checks, segment chaining, dedupe.
- `pkg/render/legal`:
  - Deterministic legal-description sentence generation.
- `pkg/app`:
  - Orchestration pipeline (`parse -> normalize -> validate -> render`).

### Canonical Internal Representation
- `Parcel`:
  - `ID string`
  - `Source string` (`dxf|ifc|landxml`)
  - `Unit LinearUnit`
  - `Loop BoundaryLoop`
- `BoundaryLoop`:
  - `Segments []Segment`
  - Invariant: first point equals last point within epsilon.
- `Segment` (interface):
  - `Kind() SegmentKind` (`line|arc`)
  - `Start() Point2D`
  - `End() Point2D`
  - `LengthFeet() float64`
  - `TangentAtStart() float64` (radians)
  - `TangentAtEnd() float64` (radians)
- `LineSegment`:
  - `P0, P1 Point2D`
- `ArcSegment`:
  - `Center Point2D`
  - `RadiusFeet float64`
  - `StartAngleRad float64`
  - `SweepRad float64` (signed, CW/CCW)
- `Point2D`:
  - Stored in feet after parser-level unit conversion.

### Tolerance Policy
- `pkg/geom/tolerance.go`:
  - `const EpsilonFeet = 0.001` (strict and `< 0.01 ft`)
  - `func NearlyEqualFeet(a, b float64) bool`
  - `func NearlyEqualAngle(a, b float64) bool`
- Rule:
  - All closure, tangency, and continuity checks use tolerance helpers.
  - Never compare floats directly with `==` in geometry logic.

## Parsing Strategy by Source

### DXF
- Read entities that define boundary candidates (`LINE`, `LWPOLYLINE`, `POLYLINE`, `ARC`).
- Build graph/chain by coincident endpoints within epsilon.
- Extract simple closed loop(s), then map to canonical `Segment`s.

### IFC
- Resolve site/building element geometry references.
- Read curve loops from profile/boundary representations.
- Convert local/project coordinates to a consistent 2D plane in feet.
- Map curve primitives to canonical `LineSegment`/`ArcSegment`.

### LandXML
- Parse parcel `CoordGeom` (`Line`, `Curve`) directly.
- Use existing sample structure as fixture seed.
- Preserve rotation semantics (`cw`/`ccw`) in signed sweep.

## Pipeline
1. Parse source file into raw loop candidates.
2. Convert units to feet.
3. Normalize orientation and segment continuity.
4. Validate:
   - closed loop
   - non-zero area
   - no broken segment links
5. Render:
   - beginning/commencing preamble
   - segment-by-segment legal narrative
   - area summary

## Rendering Approach (No Template State Bugs)
- Renderer iterates normalized segments in order.
- Previous tangent is explicit local state in Go code.
- Tangency and non-tangency are computed with tolerance helpers.
- Output casing/style options handled centrally in renderer config.

## Testing Strategy

### Unit Tests
- `pkg/geom`:
  - tolerance, angle normalization, bearing conversion.
- `pkg/normalize`:
  - closure, continuity, loop orientation.
- `pkg/render/legal`:
  - line and arc sentence generation, tangency transitions.

### Fixture Tests
- `tests/fixtures/dxf/*.dxf`
- `tests/fixtures/ifc/*.ifc`
- `tests/fixtures/landxml/*.xml`
- Golden outputs in `tests/golden/*.txt`.

### End-to-End Tests
- CLI parse+render for each source type.
- Failure fixtures for non-closed boundaries and malformed geometry.

## Migration Plan
1. Create new package skeleton and canonical model.
2. Move reusable math/bearing logic into `pkg/geom` and add tolerance helpers.
3. Implement `LandXML` parser first (lowest friction based on current sample).
4. Implement normalize+validate pipeline and hook to renderer.
5. Implement renderer v2 and add golden tests.
6. Implement DXF parser.
7. Implement IFC parser.
8. Retire old text-line parser path after parity checks.

## Deliverables by Phase
- Phase 1:
  - Internal model + tolerance primitives + parser interface + LandXML parser.
- Phase 2:
  - Normalization/validation + renderer v2 + fixtures + golden tests.
- Phase 3:
  - DXF + IFC parsers + CLI source autodetect + full e2e coverage.

## Phase 1 Progress (Current)
- Implemented canonical model (`pkg/model`) with `LineSegment` and `ArcSegment`.
- Implemented strict geometry tolerance helpers with `EpsilonFeet = 0.001`.
- Added parser contract and source format detection (`pkg/parse`).
- Added LandXML parser that extracts ordered `Line`/`Curve` geometry into canonical segments (`pkg/parse/landxml`).
- Added parser stubs for DXF and IFC (`pkg/parse/dxf`, `pkg/parse/ifc`).
- Added normalization/closure validation and pipeline orchestration (`pkg/normalize`, `pkg/app`).
- Added initial tests for tolerance and LandXML fixture parsing.

## Phase 2 Progress (Current)
- Implemented deterministic renderer in code (no template-driven implicit state): `pkg/render/legal`.
- Implemented explicit tangency/non-tangency transitions using angular tolerance.
- Added tangent-bearing output for curve segments so geometry can be reconstructed from description text with a known start point.
- Expanded normalization to include:
  - segment direction repair
  - closed-loop validation
  - zero-area rejection
  - deterministic CW orientation normalization
- Added fixture/golden testing:
  - LandXML fixture: `tests/fixtures/landxml/square.xml`
  - Golden output: `tests/golden/square_description.txt`
  - End-to-end pipeline render test: `pkg/app/pipeline_render_test.go`

## Phase 3 Progress (Current)
- Implemented ASCII DXF parser in `pkg/parse/dxf` with support for:
  - `LINE`
  - `ARC`
  - `LWPOLYLINE` (including bulge-to-arc conversion)
  - `POLYLINE`/`VERTEX`/`SEQEND`
- Added DXF unit conversion via `$INSUNITS` header handling.
- Added loop assembly for unordered line/arc entities by endpoint chaining within tolerance.
- Added DXF fixtures and tests:
  - `tests/fixtures/dxf/square_lines.dxf`
  - `pkg/parse/dxf/parser_test.go`
  - `pkg/app/pipeline_render_dxf_test.go`
- Implemented IFC parser in `pkg/parse/ifc` with support for:
  - `IFCCARTESIANPOINT`
  - `IFCPOLYLINE`
  - `IFCCARTESIANPOINTLIST2D`
  - `IFCINDEXEDPOLYCURVE` (implicit segments)
  - profile references via `IFCARBITRARYCLOSEDPROFILEDEF` and `IFCARBITRARYPROFILEDEFWITHVOIDS`
- Added IFC unit handling for SI length units and conversion-based foot fallback.
- Added IFC fixtures and tests:
  - `tests/fixtures/ifc/square_profile.ifc`
  - `pkg/parse/ifc/parser_test.go`
  - `pkg/app/pipeline_render_ifc_test.go`
- Wired `cmd/legal` to v2 pipeline (`parse -> normalize -> render`) with:
  - format selection: `-format auto|dxf|ifc|landxml`
  - parcel selection: `-parcel` and `-all`
  - metadata and rendering options (`-kind`, `-lot`, `-block`, `-sub`, `-city`, `-county`, `-state`, `-start`, `-commencing`, `-area`, `-area-unit`)
- Added normalization area fallback to compute parcel area from loop geometry when parser area is missing (notably DXF/IFC linework).
- Added repeatable v2 smoke workflow:
  - Script: `scripts/smoke_cli_v2.sh`
  - Make target: `make test-v2`
- Expanded edge-case coverage for v2 behavior:
  - reconstruction tests verify rendered `THENCE` lines can regenerate segment endpoints from a known start point
  - orientation tests verify normalized loop direction is consistently clockwise across DXF/IFC/LandXML
  - parser error tests for non-closed DXF linework and unsupported explicit-segment IFC indexed polycurves

## API Progress (Current)
- Added REST API server command: `cmd/legal-api`.
- Added discoverable and machine-readable docs endpoints:
  - `GET /` (index with links)
  - `GET /docs` (human-readable docs)
  - `GET /openapi.json` (OpenAPI 3.1 schema)
- Added description generation endpoint:
  - `POST /v1/describe` with base64 source payload (`DXF`, `IFC`, `LandXML`) and renderer options.
- Added API tests for docs discoverability, OpenAPI delivery, success path, and bad-request path.
