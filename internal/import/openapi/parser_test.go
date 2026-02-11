package openapi

import (
	"testing"
)

func TestParseOpenAPIJSON(t *testing.T) {
	data := []byte(`{
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
	}`)

	col, err := ParseOpenAPI(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if col.Name != "Pet Store" {
		t.Errorf("expected Pet Store, got %s", col.Name)
	}

	// Should have Pets folder + health check
	if len(col.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(col.Items))
	}

	petsFolder := col.Items[0].Folder
	if petsFolder == nil || petsFolder.Name != "Pets" {
		t.Fatal("expected Pets folder")
	}
	if len(petsFolder.Items) != 2 {
		t.Fatalf("expected 2 items in Pets folder, got %d", len(petsFolder.Items))
	}

	getReq := petsFolder.Items[0].Request
	if getReq.Method != "GET" {
		t.Errorf("expected GET, got %s", getReq.Method)
	}
	if len(getReq.Params) != 1 {
		t.Errorf("expected 1 param, got %d", len(getReq.Params))
	}

	postReq := petsFolder.Items[1].Request
	if postReq.Body == nil {
		t.Error("expected body on POST request")
	}
}

func TestParseOpenAPIYAML(t *testing.T) {
	data := []byte(`
openapi: "3.0.0"
info:
  title: "YAML API"
  version: "1.0"
paths:
  /items:
    get:
      summary: "List Items"
`)

	col, err := ParseOpenAPI(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if col.Name != "YAML API" {
		t.Errorf("expected YAML API, got %s", col.Name)
	}
}

func TestParseOpenAPIInvalid(t *testing.T) {
	_, err := ParseOpenAPI([]byte("not valid"))
	if err == nil {
		t.Error("expected error")
	}
}
