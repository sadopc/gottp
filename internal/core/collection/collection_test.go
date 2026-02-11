package collection

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleYAML = `
name: Test API
version: "1"
variables:
  base_url: "https://api.example.com"
items:
  - folder:
      name: Users
      items:
        - request:
            id: "req-1"
            name: List Users
            protocol: http
            method: GET
            url: "{{base_url}}/users"
            params:
              - { key: page, value: "1", enabled: true }
            headers:
              - { key: Accept, value: application/json, enabled: true }
        - request:
            id: "req-2"
            name: Create User
            protocol: http
            method: POST
            url: "{{base_url}}/users"
            body:
              type: json
              content: '{"name":"test"}'
  - folder:
      name: Products
      items:
        - request:
            id: "req-3"
            name: List Products
            protocol: http
            method: GET
            url: "{{base_url}}/products"
`

func TestLoadFromBytes(t *testing.T) {
	col, err := LoadFromBytes([]byte(sampleYAML))
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}

	if col.Name != "Test API" {
		t.Errorf("expected name 'Test API', got %q", col.Name)
	}
	if col.Version != "1" {
		t.Errorf("expected version '1', got %q", col.Version)
	}
	if len(col.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(col.Items))
	}

	users := col.Items[0].Folder
	if users == nil || users.Name != "Users" {
		t.Fatal("first item should be Users folder")
	}
	if len(users.Items) != 2 {
		t.Errorf("expected 2 user requests, got %d", len(users.Items))
	}

	listUsers := users.Items[0].Request
	if listUsers.Method != "GET" {
		t.Errorf("expected GET, got %s", listUsers.Method)
	}
	if listUsers.URL != "{{base_url}}/users" {
		t.Errorf("unexpected URL: %s", listUsers.URL)
	}
	if len(listUsers.Params) != 1 {
		t.Errorf("expected 1 param, got %d", len(listUsers.Params))
	}
}

func TestSaveAndLoad(t *testing.T) {
	col := &Collection{
		Name:    "Roundtrip Test",
		Version: "1",
		Items: []Item{
			{Request: NewRequest("Test Request", "GET", "https://example.com")},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.gottp.yaml")

	if err := SaveToFile(col, path); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	loaded, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if loaded.Name != "Roundtrip Test" {
		t.Errorf("expected name 'Roundtrip Test', got %q", loaded.Name)
	}
	if len(loaded.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(loaded.Items))
	}
	if loaded.Items[0].Request.Name != "Test Request" {
		t.Errorf("expected 'Test Request', got %q", loaded.Items[0].Request.Name)
	}
}

func TestFlattenItems(t *testing.T) {
	col, err := LoadFromBytes([]byte(sampleYAML))
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}

	flat := FlattenItems(col.Items, 0, "")

	// Should have: Users(folder), List Users, Create User, Products(folder), List Products
	if len(flat) != 5 {
		t.Fatalf("expected 5 flat items, got %d", len(flat))
	}

	if !flat[0].IsFolder || flat[0].Folder.Name != "Users" {
		t.Error("first item should be Users folder")
	}
	if flat[0].Depth != 0 {
		t.Errorf("folder depth should be 0, got %d", flat[0].Depth)
	}
	if flat[1].IsFolder || flat[1].Request.Name != "List Users" {
		t.Error("second item should be List Users request")
	}
	if flat[1].Depth != 1 {
		t.Errorf("request depth should be 1, got %d", flat[1].Depth)
	}
}

func TestLoadFromDir(t *testing.T) {
	dir := t.TempDir()

	// Write two collection files
	for _, name := range []string{"api.gottp.yaml", "auth.gottp.yaml"} {
		col := &Collection{Name: name, Version: "1"}
		if err := SaveToFile(col, filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}

	// Also write a non-matching file
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644)

	cols, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}
	if len(cols) != 2 {
		t.Errorf("expected 2 collections, got %d", len(cols))
	}
}

func TestNewRequest(t *testing.T) {
	req := NewRequest("My Request", "POST", "https://example.com/api")
	if req.ID == "" {
		t.Error("request ID should not be empty")
	}
	if req.Protocol != "http" {
		t.Errorf("default protocol should be http, got %s", req.Protocol)
	}
	if req.Method != "POST" {
		t.Errorf("expected POST, got %s", req.Method)
	}
}
