package postman

import "testing"

func FuzzParsePostman(f *testing.F) {
	// Seed: valid Postman collection with folders, requests, auth, variables
	f.Add([]byte(`{
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
	}`))

	// Seed: minimal valid collection
	f.Add([]byte(`{"info":{"name":"Minimal"},"item":[]}`))

	// Seed: collection with basic auth
	f.Add([]byte(`{
		"info": {"name": "Auth Test"},
		"item": [{
			"name": "Basic Auth Request",
			"request": {
				"method": "GET",
				"url": "https://example.com",
				"auth": {
					"type": "basic",
					"basic": [
						{"key": "username", "value": "admin"},
						{"key": "password", "value": "secret"}
					]
				}
			}
		}]
	}`))

	// Seed: collection with apikey auth
	f.Add([]byte(`{
		"info": {"name": "API Key Test"},
		"item": [{
			"name": "API Key Request",
			"request": {
				"method": "GET",
				"url": "https://example.com",
				"auth": {
					"type": "apikey",
					"apikey": [
						{"key": "key", "value": "X-API-Key"},
						{"key": "value", "value": "my-key-123"},
						{"key": "in", "value": "header"}
					]
				}
			}
		}]
	}`))

	// Seed: collection with URL as string (not object)
	f.Add([]byte(`{
		"info": {"name": "Simple URL"},
		"item": [{"name": "Req", "request": {"method": "GET", "url": "https://example.com/path"}}]
	}`))

	// Seed: item with no request (empty item)
	f.Add([]byte(`{
		"info": {"name": "Empty Item"},
		"item": [{"name": "Ghost"}]
	}`))

	// Seed: collection with disabled headers
	f.Add([]byte(`{
		"info": {"name": "Disabled Headers"},
		"item": [{
			"name": "Req",
			"request": {
				"method": "GET",
				"url": "https://example.com",
				"header": [{"key": "X-Debug", "value": "true", "disabled": true}]
			}
		}]
	}`))

	// Invalid inputs
	f.Add([]byte(`not json`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"info":{},"item":[]}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`[]`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		col, err := ParsePostman(data)
		if err != nil {
			return
		}
		if col == nil {
			t.Fatal("ParsePostman returned nil collection without error")
		}
		if col.Name == "" {
			t.Fatal("ParsePostman returned collection with empty name without error")
		}
	})
}
