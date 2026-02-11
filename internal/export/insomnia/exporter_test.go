package insomnia

import (
	"encoding/json"
	"testing"

	"github.com/serdar/gottp/internal/core/collection"
)

func TestExportBasic(t *testing.T) {
	col := &collection.Collection{
		Name: "Test API",
		Items: []collection.Item{
			{
				Request: &collection.Request{
					ID: "1", Name: "Get Users", Method: "GET",
					URL: "https://api.example.com/users",
				},
			},
		},
	}

	data, err := Export(col)
	if err != nil {
		t.Fatal(err)
	}

	var export insomniaExport
	if err := json.Unmarshal(data, &export); err != nil {
		t.Fatal(err)
	}

	if export.Type != "export" {
		t.Errorf("expected type 'export', got %q", export.Type)
	}
	if export.Format != 4 {
		t.Errorf("expected format 4, got %d", export.Format)
	}
	// Should have workspace + 1 request = 2 resources
	if len(export.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(export.Resources))
	}
	if export.Resources[0].Type != "workspace" {
		t.Errorf("expected first resource to be workspace, got %q", export.Resources[0].Type)
	}
	if export.Resources[1].Type != "request" {
		t.Errorf("expected second resource to be request, got %q", export.Resources[1].Type)
	}
	if export.Resources[1].Method != "GET" {
		t.Errorf("expected method GET, got %q", export.Resources[1].Method)
	}
}

func TestExportWithFolders(t *testing.T) {
	col := &collection.Collection{
		Name: "Nested API",
		Items: []collection.Item{
			{
				Folder: &collection.Folder{
					Name: "Auth",
					Items: []collection.Item{
						{Request: &collection.Request{ID: "1", Name: "Login", Method: "POST", URL: "/login"}},
					},
				},
			},
		},
	}

	data, err := Export(col)
	if err != nil {
		t.Fatal(err)
	}

	var export insomniaExport
	if err := json.Unmarshal(data, &export); err != nil {
		t.Fatal(err)
	}

	// workspace + request_group + request = 3
	if len(export.Resources) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(export.Resources))
	}

	// Find the request_group
	var folder *insomniaResource
	for i := range export.Resources {
		if export.Resources[i].Type == "request_group" {
			folder = &export.Resources[i]
			break
		}
	}
	if folder == nil {
		t.Fatal("expected a request_group resource")
	}
	if folder.Name != "Auth" {
		t.Errorf("expected folder name 'Auth', got %q", folder.Name)
	}
}

func TestExportWithAuth(t *testing.T) {
	col := &collection.Collection{
		Name: "Auth API",
		Items: []collection.Item{
			{
				Request: &collection.Request{
					ID: "1", Name: "Basic", Method: "GET", URL: "/api",
					Auth: &collection.Auth{
						Type:  "basic",
						Basic: &collection.BasicAuth{Username: "admin", Password: "secret"},
					},
				},
			},
		},
	}

	data, err := Export(col)
	if err != nil {
		t.Fatal(err)
	}

	var export insomniaExport
	if err := json.Unmarshal(data, &export); err != nil {
		t.Fatal(err)
	}

	req := export.Resources[1]
	if req.Authentication == nil {
		t.Fatal("expected authentication to be set")
	}
	if req.Authentication.Type != "basic" {
		t.Errorf("expected auth type 'basic', got %q", req.Authentication.Type)
	}
	if req.Authentication.Username != "admin" {
		t.Errorf("expected username 'admin', got %q", req.Authentication.Username)
	}
}

func TestExportNilCollection(t *testing.T) {
	_, err := Export(nil)
	if err == nil {
		t.Error("expected error for nil collection")
	}
}

func TestExportWithBody(t *testing.T) {
	col := &collection.Collection{
		Name: "Body API",
		Items: []collection.Item{
			{
				Request: &collection.Request{
					ID: "1", Name: "Create", Method: "POST", URL: "/api",
					Body: &collection.Body{Type: "json", Content: `{"key":"value"}`},
				},
			},
		},
	}

	data, err := Export(col)
	if err != nil {
		t.Fatal(err)
	}

	var export insomniaExport
	if err := json.Unmarshal(data, &export); err != nil {
		t.Fatal(err)
	}

	req := export.Resources[1]
	if req.Body == nil {
		t.Fatal("expected body to be set")
	}
	if req.Body.MimeType != "application/json" {
		t.Errorf("expected mimeType 'application/json', got %q", req.Body.MimeType)
	}
	if req.Body.Text != `{"key":"value"}` {
		t.Errorf("expected body text to match, got %q", req.Body.Text)
	}
}
