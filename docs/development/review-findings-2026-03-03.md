# Project Review Findings (2026-03-03)

## Scope
- Reviewed project structure and core code paths.
- Ran tests with local Go build cache:
  - `GOCACHE=/tmp/go-build go test ./...`
- Ran CLI against provided sample:
  - `GOCACHE=/tmp/go-build go run ./cmd/legal ... example.txt`

## Findings

### 1. Arc text parsing is not implemented in CLI input flow (High)
- The CLI treats all `T...` lines as `LinearMete` and calls `LinearMete.FromString`.
- Arc-style `THENCE` text in `example.txt` fails with `Invalid mete description`.
- Evidence:
  - `cmd/legal/main.go` line 64
  - `pkg/legal/legal.go` line 255

### 2. Tangency preamble state is not updated across metes (Medium)
- Description template initializes a previous tangent variable but never updates it while iterating.
- This causes tangency/non-tangency preamble text to not reflect the actual previous segment.
- Evidence:
  - `pkg/legal/legal.go` line 390

### 3. Exact float equality causes unstable geometric checks (Medium)
- Code compares floating-point angles with `==` for tangency and direction transitions.
- Tests fail due to tiny floating precision drift.
- Evidence:
  - `pkg/legal/legal.go` line 247
  - `pkg/legal/legal.go` line 358
  - `Test/legal_test.go` line 103
  - `Test/legal_test.go` line 115

### 4. Test suite is currently failing (Low)
- `go test` currently fails in `Test` package.
- One test uses a fixed impossible expected value (`ALWAYS FAIL`), which keeps suite red.
- Evidence:
  - `Test/legal_test.go` line 152

## Notes
- `cmd/legalxml` currently has only a README and no executable parser implementation.
- `README.md` is placeholder content and does not describe actual runtime behavior.
