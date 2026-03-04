.PHONY: test-v2 smoke-v2 test-api serve-api

test-v2: smoke-v2

smoke-v2:
	./scripts/smoke_cli_v2.sh

test-api:
	GOCACHE=$${GOCACHE:-/tmp/go-build} go test ./pkg/api/...

serve-api:
	go run ./cmd/legal-api -addr :8080
