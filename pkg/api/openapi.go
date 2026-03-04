package api

import "encoding/json"

func openAPISpec() map[string]any {
	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "Legal Description API",
			"version":     "0.1.0",
			"description": "Generate legal descriptions from DXF, IFC, and LandXML geometry.",
		},
		"servers": []map[string]any{
			{"url": "http://localhost:8080"},
		},
		"paths": map[string]any{
			"/": map[string]any{
				"get": map[string]any{
					"summary":     "Service index",
					"description": "Discover entrypoints for health, docs, and OpenAPI.",
					"responses": map[string]any{
						"200": map[string]any{"description": "OK"},
					},
				},
			},
			"/healthz": map[string]any{
				"get": map[string]any{
					"summary": "Health check",
					"responses": map[string]any{
						"200": map[string]any{"description": "Healthy"},
					},
				},
			},
			"/openapi.json": map[string]any{
				"get": map[string]any{
					"summary": "Machine-readable OpenAPI document",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "OpenAPI schema",
							"content": map[string]any{
								"application/json": map[string]any{},
							},
						},
					},
				},
			},
			"/docs": map[string]any{
				"get": map[string]any{
					"summary": "Human-readable API docs",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Documentation page",
							"content": map[string]any{
								"text/html": map[string]any{},
							},
						},
					},
				},
			},
			"/v1/describe": map[string]any{
				"post": map[string]any{
					"summary": "Render legal descriptions from geometry payload",
					"requestBody": map[string]any{
						"required": true,
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{
									"$ref": "#/components/schemas/DescribeRequest",
								},
							},
						},
					},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Rendered description output",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"$ref": "#/components/schemas/DescribeResponse",
									},
								},
							},
						},
						"400": map[string]any{"description": "Bad request"},
						"500": map[string]any{"description": "Server error"},
					},
				},
			},
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"DescribeRequest": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"format": map[string]any{
							"type":        "string",
							"description": "Input format: auto|dxf|ifc|landxml",
							"enum":        []string{"auto", "dxf", "ifc", "landxml"},
						},
						"filename": map[string]any{
							"type":        "string",
							"description": "Optional filename hint used for auto format detection",
						},
						"content_base64": map[string]any{
							"type":        "string",
							"description": "Base64-encoded source file bytes",
						},
						"parcel": map[string]any{
							"type":        "integer",
							"description": "1-based parcel index when returning a single parcel",
							"minimum":     1,
						},
						"all": map[string]any{
							"type":        "boolean",
							"description": "Render all parcels from the source",
						},
						"render_options": map[string]any{
							"$ref": "#/components/schemas/RenderOptions",
						},
					},
					"required": []string{"content_base64"},
				},
				"RenderOptions": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"kind":             map[string]any{"type": "string"},
						"lot":              map[string]any{"type": "string"},
						"block":            map[string]any{"type": "string"},
						"subdivision":      map[string]any{"type": "string"},
						"city":             map[string]any{"type": "string"},
						"county":           map[string]any{"type": "string"},
						"state":            map[string]any{"type": "string"},
						"start_corner":     map[string]any{"type": "string"},
						"use_commencing":   map[string]any{"type": "boolean"},
						"area_square_feet": map[string]any{"type": "number"},
						"area_unit":        map[string]any{"type": "string"},
					},
				},
				"DescribeResponse": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"format":        map[string]any{"type": "string"},
						"parcels_total": map[string]any{"type": "integer"},
						"results": map[string]any{
							"type": "array",
							"items": map[string]any{
								"$ref": "#/components/schemas/DescribeResult",
							},
						},
					},
				},
				"DescribeResult": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"parcel_index":     map[string]any{"type": "integer"},
						"parcel_id":        map[string]any{"type": "string"},
						"segment_count":    map[string]any{"type": "integer"},
						"area_square_feet": map[string]any{"type": "number"},
						"description":      map[string]any{"type": "string"},
					},
				},
			},
		},
	}
}

func openAPIJSON() ([]byte, error) {
	return json.MarshalIndent(openAPISpec(), "", "  ")
}
