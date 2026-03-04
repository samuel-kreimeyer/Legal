#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${REPO_ROOT}"

export GOCACHE="${GOCACHE:-/tmp/go-build}"
mkdir -p "${GOCACHE}"

GOLDEN="tests/golden/square_description.txt"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

common_args=(
  -kind "Utility Easement"
  -lot 7
  -block B
  -sub "Sample Addition"
  -city "North Little Rock"
  -county Pulaski
  -state Arkansas
)

run_case() {
  local format="$1"
  local input="$2"
  local output_file="$3"

  go run ./cmd/legal -format "${format}" "${common_args[@]}" "${input}" > "${output_file}"
  diff -u "${GOLDEN}" "${output_file}"
}

echo "==> Running v2 package tests"
go test ./pkg/... >/dev/null

echo "==> Running v2 command build check"
go test ./cmd/... >/dev/null

echo "==> Verifying CLI output for LandXML"
run_case "landxml" "tests/fixtures/landxml/square.xml" "${TMP_DIR}/landxml.txt"

echo "==> Verifying CLI output for DXF"
run_case "dxf" "tests/fixtures/dxf/square_lines.dxf" "${TMP_DIR}/dxf.txt"

echo "==> Verifying CLI output for IFC"
run_case "ifc" "tests/fixtures/ifc/square_profile.ifc" "${TMP_DIR}/ifc.txt"

echo "v2 CLI smoke checks passed."
