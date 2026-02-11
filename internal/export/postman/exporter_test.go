package postman

import (
	"encoding/json"
	"testing"

	"github.com/sadopc/gottp/internal/core/collection"
)

func TestExportBasic(t *testing.T) {
	col := &collection.Collection{
		Name:    "Test API",
		Version: "1.0",
		Items: []collection.Item{
			{
				Request: &collection.Request{
					ID:     "1",
					Name:   "Get Users",
					Method: "GET",
					URL:    "https://api.example.com/users",
					Headers: []collection.KVPair{
						{Key: "Accept", Value: "application/json", Enabled: true},
					},
					Params: []collection.KVPair{
						{Key: "page", Value: "1", Enabled: true},
					},
				},
			},
		},
	}

	data, err := Export(col)
	if err != nil {
		t.Fatal(err)
	}

	var pc postmanCollection
	if err := json.Unmarshal(data, &pc); err != nil {
		t.Fatal(err)
	}

	if pc.Info.Name != "Test API" {
		t.Errorf("expected name 'Test API', got %q", pc.Info.Name)
	}
	if len(pc.Item) != 1 {
		t.Fatalf("expected 1 item, got %d", len(pc.Item))
	}
	if pc.Item[0].Name != "Get Users" {
		t.Errorf("expected item name 'Get Users', got %q", pc.Item[0].Name)
	}
	if pc.Item[0].Request.Method != "GET" {
		t.Errorf("expected method GET, got %q", pc.Item[0].Request.Method)
	}
	if len(pc.Item[0].Request.Header) != 1 {
		t.Errorf("expected 1 header, got %d", len(pc.Item[0].Request.Header))
	}
	if len(pc.Item[0].Request.URL.Query) != 1 {
		t.Errorf("expected 1 query param, got %d", len(pc.Item[0].Request.URL.Query))
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
						{Request: &collection.Request{ID: "2", Name: "Logout", Method: "POST", URL: "/logout"}},
					},
				},
			},
		},
	}

	data, err := Export(col)
	if err != nil {
		t.Fatal(err)
	}

	var pc postmanCollection
	if err := json.Unmarshal(data, &pc); err != nil {
		t.Fatal(err)
	}

	if len(pc.Item) != 1 {
		t.Fatalf("expected 1 folder, got %d", len(pc.Item))
	}
	if pc.Item[0].Name != "Auth" {
		t.Errorf("expected folder 'Auth', got %q", pc.Item[0].Name)
	}
	if len(pc.Item[0].Item) != 2 {
		t.Errorf("expected 2 items in folder, got %d", len(pc.Item[0].Item))
	}
}

func TestExportWithAuth(t *testing.T) {
	col := &collection.Collection{
		Name: "Auth API",
		Items: []collection.Item{
			{
				Request: &collection.Request{
					ID: "1", Name: "Bearer", Method: "GET", URL: "/api",
					Auth: &collection.Auth{
						Type:   "bearer",
						Bearer: &collection.BearerAuth{Token: "my-token"},
					},
				},
			},
		},
	}

	data, err := Export(col)
	if err != nil {
		t.Fatal(err)
	}

	var pc postmanCollection
	if err := json.Unmarshal(data, &pc); err != nil {
		t.Fatal(err)
	}

	if pc.Item[0].Request.Auth == nil {
		t.Fatal("expected auth to be set")
	}
	if pc.Item[0].Request.Auth.Type != "bearer" {
		t.Errorf("expected auth type 'bearer', got %q", pc.Item[0].Request.Auth.Type)
	}
}

func TestExportNilCollection(t *testing.T) {
	_, err := Export(nil)
	if err == nil {
		t.Error("expected error for nil collection")
	}
}

func TestExportWithVariables(t *testing.T) {
	col := &collection.Collection{
		Name:      "Var API",
		Variables: map[string]string{"baseUrl": "https://api.example.com"},
		Items:     []collection.Item{},
	}

	data, err := Export(col)
	if err != nil {
		t.Fatal(err)
	}

	var pc postmanCollection
	if err := json.Unmarshal(data, &pc); err != nil {
		t.Fatal(err)
	}

	if len(pc.Variable) != 1 {
		t.Fatalf("expected 1 variable, got %d", len(pc.Variable))
	}
	if pc.Variable[0].Key != "baseUrl" {
		t.Errorf("expected variable key 'baseUrl', got %q", pc.Variable[0].Key)
	}
}
