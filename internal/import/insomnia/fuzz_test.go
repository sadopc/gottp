package insomnia

import "testing"

func FuzzParseInsomnia(f *testing.F) {
	// Seed: full Insomnia export with workspace, folder, requests, auth
	f.Add([]byte(`{
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
	}`))

	// Seed: minimal valid export
	f.Add([]byte(`{"_type":"export","resources":[]}`))

	// Seed: export without workspace
	f.Add([]byte(`{
		"_type": "export",
		"resources": [
			{
				"_id": "req_1",
				"_type": "request",
				"parentId": "",
				"name": "Orphan Request",
				"method": "POST",
				"url": "https://example.com/data",
				"body": {"mimeType": "application/json", "text": "{\"key\":\"val\"}"}
			}
		]
	}`))

	// Seed: request with basic auth
	f.Add([]byte(`{
		"_type": "export",
		"resources": [
			{"_id": "wrk_1", "_type": "workspace", "name": "Auth Test"},
			{
				"_id": "req_1",
				"_type": "request",
				"parentId": "wrk_1",
				"name": "Basic Auth",
				"method": "GET",
				"url": "https://example.com",
				"authentication": {"type": "basic", "username": "admin", "password": "pass123"}
			}
		]
	}`))

	// Seed: request with XML body
	f.Add([]byte(`{
		"_type": "export",
		"resources": [
			{"_id": "wrk_1", "_type": "workspace", "name": "XML Test"},
			{
				"_id": "req_1",
				"_type": "request",
				"parentId": "wrk_1",
				"name": "XML Request",
				"method": "POST",
				"url": "https://example.com/xml",
				"body": {"mimeType": "application/xml", "text": "<root><item>test</item></root>"}
			}
		]
	}`))

	// Seed: request with disabled headers/params
	f.Add([]byte(`{
		"_type": "export",
		"resources": [
			{"_id": "wrk_1", "_type": "workspace", "name": "Disabled Test"},
			{
				"_id": "req_1",
				"_type": "request",
				"parentId": "wrk_1",
				"name": "Disabled Params",
				"method": "GET",
				"url": "https://example.com",
				"headers": [{"name": "X-Debug", "value": "true", "disabled": true}],
				"parameters": [{"name": "verbose", "value": "1", "disabled": true}]
			}
		]
	}`))

	// Seed: deeply nested folders
	f.Add([]byte(`{
		"_type": "export",
		"resources": [
			{"_id": "wrk_1", "_type": "workspace", "name": "Nested"},
			{"_id": "fld_1", "_type": "request_group", "parentId": "wrk_1", "name": "Level 1"},
			{"_id": "fld_2", "_type": "request_group", "parentId": "fld_1", "name": "Level 2"},
			{"_id": "req_1", "_type": "request", "parentId": "fld_2", "name": "Deep Request", "method": "GET", "url": "https://example.com"}
		]
	}`))

	// Invalid inputs
	f.Add([]byte(`not json`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`[]`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		col, err := ParseInsomnia(data)
		if err != nil {
			return
		}
		if col == nil {
			t.Fatal("ParseInsomnia returned nil collection without error")
		}
		// Note: collection name can be empty if the workspace has no name,
		// or "Imported" if there is no workspace resource at all.
		// Both are valid parser behavior.
	})
}
