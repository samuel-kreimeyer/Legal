# REST API

The API server is provided by `cmd/legal-api`.

## Run

```bash
go run ./cmd/legal-api -addr :8080
```

## Discoverability

- Service index: `GET /`
- Health check: `GET /healthz`
- Machine-readable OpenAPI: `GET /openapi.json`
- Human-readable docs: `GET /docs`

## Describe Endpoint

`POST /v1/describe`

### Request JSON

```json
{
  "format": "landxml",
  "filename": "parcel.xml",
  "content_base64": "BASE64_ENCODED_FILE_BYTES",
  "parcel": 1,
  "all": false,
  "render_options": {
    "kind": "Utility Easement",
    "lot": "7",
    "block": "B",
    "subdivision": "Sample Addition",
    "city": "North Little Rock",
    "county": "Pulaski",
    "state": "Arkansas",
    "start_corner": "northeast",
    "use_commencing": false,
    "area_square_feet": 0,
    "area_unit": "square feet"
  }
}
```

### Notes

- `format` accepts `auto|dxf|ifc|landxml`.
- If `format` is `auto`, `filename` with an extension is required.
- `content_base64` is required.
