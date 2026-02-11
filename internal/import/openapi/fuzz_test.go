package openapi

import "testing"

func FuzzParseOpenAPI(f *testing.F) {
	// Seed: valid OpenAPI 3.0 JSON with tags, params, request body
	f.Add([]byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Pet Store", "version": "1.0.0"},
		"paths": {
			"/pets": {
				"get": {
					"summary": "List Pets",
					"tags": ["Pets"],
					"parameters": [
						{"name": "limit", "in": "query", "required": false, "example": 10}
					]
				},
				"post": {
					"summary": "Create Pet",
					"tags": ["Pets"],
					"requestBody": {
						"content": {
							"application/json": {
								"example": {"name": "Fido", "type": "dog"}
							}
						}
					}
				}
			},
			"/health": {
				"get": {
					"summary": "Health Check"
				}
			}
		}
	}`))

	// Seed: valid OpenAPI YAML
	f.Add([]byte(`openapi: "3.0.0"
info:
  title: "YAML API"
  version: "1.0"
paths:
  /items:
    get:
      summary: "List Items"
`))

	// Seed: minimal valid OpenAPI JSON
	f.Add([]byte(`{"openapi":"3.0.0","info":{"title":"Min","version":"1"},"paths":{}}`))

	// Seed: OpenAPI with header parameters
	f.Add([]byte(`{
		"openapi": "3.1.0",
		"info": {"title": "Header API", "version": "2.0"},
		"paths": {
			"/data": {
				"get": {
					"summary": "Get Data",
					"parameters": [
						{"name": "X-Request-ID", "in": "header", "required": true, "example": "abc-123"},
						{"name": "q", "in": "query", "required": false}
					]
				}
			}
		}
	}`))

	// Seed: OpenAPI with operationId instead of summary
	f.Add([]byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Op ID API", "version": "1.0"},
		"paths": {
			"/items/{id}": {
				"get": {
					"operationId": "getItemById",
					"parameters": [
						{"name": "id", "in": "path", "required": true}
					]
				},
				"delete": {
					"operationId": "deleteItem"
				}
			}
		}
	}`))

	// Seed: OpenAPI with XML request body
	f.Add([]byte(`{
		"openapi": "3.0.0",
		"info": {"title": "XML API", "version": "1.0"},
		"paths": {
			"/data": {
				"post": {
					"summary": "Submit XML",
					"requestBody": {
						"content": {
							"application/xml": {}
						}
					}
				}
			}
		}
	}`))

	// Seed: OpenAPI with multiple tags
	f.Add([]byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Multi Tag", "version": "1.0"},
		"paths": {
			"/a": {"get": {"summary": "A", "tags": ["Alpha"]}},
			"/b": {"post": {"summary": "B", "tags": ["Beta"]}},
			"/c": {"put": {"summary": "C", "tags": ["Alpha"]}}
		}
	}`))

	// Seed: OpenAPI with all HTTP methods
	f.Add([]byte(`{
		"openapi": "3.0.0",
		"info": {"title": "All Methods", "version": "1.0"},
		"paths": {
			"/resource": {
				"get": {"summary": "Get"},
				"post": {"summary": "Create"},
				"put": {"summary": "Replace"},
				"patch": {"summary": "Update"},
				"delete": {"summary": "Remove"},
				"head": {"summary": "Head"},
				"options": {"summary": "Options"}
			}
		}
	}`))

	// Invalid inputs
	f.Add([]byte(`not valid`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"info":{"title":"No openapi field"}}`))
	f.Add([]byte(`null`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		col, err := ParseOpenAPI(data)
		if err != nil {
			return
		}
		if col == nil {
			t.Fatal("ParseOpenAPI returned nil collection without error")
		}
	})
}
