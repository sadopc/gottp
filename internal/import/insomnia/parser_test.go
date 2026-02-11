package insomnia

import (
	"testing"
)

func TestParseInsomnia(t *testing.T) {
	data := []byte(`{
		"_type": "export",
		"resources": [
			{
				"_id": "wrk_1",
				"_type": "workspace",
				"name": "My API"
			},
			{
				"_id": "fld_1",
				"_type": "request_group",
				"parentId": "wrk_1",
				"name": "Users"
			},
			{
				"_id": "req_1",
				"_type": "request",
				"parentId": "fld_1",
				"name": "List Users",
				"method": "GET",
				"url": "https://api.example.com/users",
				"headers": [{"name": "Accept", "value": "application/json"}],
				"parameters": [{"name": "limit", "value": "10"}]
			},
			{
				"_id": "req_2",
				"_type": "request",
				"parentId": "wrk_1",
				"name": "Health Check",
				"method": "GET",
				"url": "https://api.example.com/health",
				"authentication": {"type": "bearer", "token": "my-token"}
			}
		]
	}`)

	col, err := ParseInsomnia(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if col.Name != "My API" {
		t.Errorf("expected My API, got %s", col.Name)
	}

	if len(col.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(col.Items))
	}

	folder := col.Items[0].Folder
	if folder == nil || folder.Name != "Users" {
		t.Fatal("expected Users folder")
	}
	if len(folder.Items) != 1 {
		t.Fatalf("expected 1 item in folder, got %d", len(folder.Items))
	}

	listReq := folder.Items[0].Request
	if listReq.Method != "GET" {
		t.Errorf("expected GET, got %s", listReq.Method)
	}
	if len(listReq.Headers) != 1 {
		t.Errorf("expected 1 header, got %d", len(listReq.Headers))
	}

	healthReq := col.Items[1].Request
	if healthReq == nil {
		t.Fatal("expected health check request")
	}
	if healthReq.Auth == nil || healthReq.Auth.Type != "bearer" {
		t.Error("expected bearer auth on health check")
	}
}
