package postman

import (
	"testing"
)

func TestParsePostman(t *testing.T) {
	data := []byte(`{
		"info": {
			"name": "Test Collection",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "Users",
				"item": [
					{
						"name": "Get Users",
						"request": {
							"method": "GET",
							"header": [{"key": "Accept", "value": "application/json"}],
							"url": {
								"raw": "https://api.example.com/users?page=1",
								"query": [{"key": "page", "value": "1"}]
							}
						}
					},
					{
						"name": "Create User",
						"request": {
							"method": "POST",
							"header": [{"key": "Content-Type", "value": "application/json"}],
							"body": {"mode": "raw", "raw": "{\"name\":\"John\"}"},
							"url": "https://api.example.com/users",
							"auth": {
								"type": "bearer",
								"bearer": [{"key": "token", "value": "abc123"}]
							}
						}
					}
				]
			}
		],
		"variable": [
			{"key": "base_url", "value": "https://api.example.com"}
		]
	}`)

	col, err := ParsePostman(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if col.Name != "Test Collection" {
		t.Errorf("expected Test Collection, got %s", col.Name)
	}

	if len(col.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(col.Items))
	}

	folder := col.Items[0].Folder
	if folder == nil {
		t.Fatal("expected folder")
	}
	if folder.Name != "Users" {
		t.Errorf("expected Users folder, got %s", folder.Name)
	}
	if len(folder.Items) != 2 {
		t.Fatalf("expected 2 items in folder, got %d", len(folder.Items))
	}

	getReq := folder.Items[0].Request
	if getReq.Method != "GET" {
		t.Errorf("expected GET, got %s", getReq.Method)
	}
	if len(getReq.Params) != 1 {
		t.Errorf("expected 1 param, got %d", len(getReq.Params))
	}

	postReq := folder.Items[1].Request
	if postReq.Auth == nil || postReq.Auth.Type != "bearer" {
		t.Error("expected bearer auth")
	}
	if postReq.Body == nil || postReq.Body.Content == "" {
		t.Error("expected body content")
	}

	if col.Variables["base_url"] != "https://api.example.com" {
		t.Error("expected base_url variable")
	}
}

func TestParsePostmanInvalid(t *testing.T) {
	_, err := ParsePostman([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
