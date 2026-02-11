package har

import "testing"

func FuzzParseHAR(f *testing.F) {
	// Seed: full HAR with GET and POST entries, headers, query params, body
	f.Add([]byte(`{
		"log": {
			"version": "1.2",
			"creator": {"name": "Browser DevTools", "version": "1.0"},
			"entries": [
				{
					"startedDateTime": "2024-01-01T00:00:00.000Z",
					"time": 150,
					"request": {
						"method": "GET",
						"url": "https://api.example.com/users?page=1",
						"httpVersion": "HTTP/1.1",
						"headers": [
							{"name": "Accept", "value": "application/json"},
							{"name": "Authorization", "value": "Bearer token123"}
						],
						"queryString": [
							{"name": "page", "value": "1"}
						],
						"headersSize": -1,
						"bodySize": -1
					},
					"response": {
						"status": 200,
						"statusText": "OK",
						"httpVersion": "HTTP/1.1",
						"headers": [
							{"name": "Content-Type", "value": "application/json"}
						],
						"content": {
							"size": 42,
							"mimeType": "application/json",
							"text": "{\"users\":[]}"
						},
						"headersSize": -1,
						"bodySize": 42
					}
				},
				{
					"startedDateTime": "2024-01-01T00:00:01.000Z",
					"time": 200,
					"request": {
						"method": "POST",
						"url": "https://api.example.com/users",
						"httpVersion": "HTTP/1.1",
						"headers": [
							{"name": "Content-Type", "value": "application/json"}
						],
						"queryString": [],
						"postData": {
							"mimeType": "application/json",
							"text": "{\"name\":\"John\"}"
						},
						"headersSize": -1,
						"bodySize": 15
					},
					"response": {
						"status": 201,
						"statusText": "Created",
						"httpVersion": "HTTP/1.1",
						"headers": [],
						"content": {
							"size": 10,
							"mimeType": "application/json",
							"text": "{\"id\":1}"
						},
						"headersSize": -1,
						"bodySize": 10
					}
				}
			]
		}
	}`))

	// Seed: HAR with HTTP/2 pseudo-headers
	f.Add([]byte(`{
		"log": {
			"version": "1.2",
			"entries": [{
				"startedDateTime": "2024-01-01T00:00:00.000Z",
				"time": 100,
				"request": {
					"method": "GET",
					"url": "https://example.com/",
					"httpVersion": "HTTP/2",
					"headers": [
						{"name": ":method", "value": "GET"},
						{"name": ":path", "value": "/"},
						{"name": "Accept", "value": "*/*"}
					],
					"queryString": [],
					"headersSize": -1,
					"bodySize": -1
				},
				"response": {
					"status": 200,
					"statusText": "OK",
					"httpVersion": "HTTP/2",
					"headers": [],
					"content": {"size": 0, "mimeType": "text/html", "text": ""},
					"headersSize": -1,
					"bodySize": 0
				}
			}]
		}
	}`))

	// Seed: minimal valid HAR
	f.Add([]byte(`{
		"log": {
			"version": "1.2",
			"entries": [{
				"request": {
					"method": "GET",
					"url": "https://example.com",
					"headers": [],
					"queryString": []
				},
				"response": {
					"status": 200,
					"statusText": "OK",
					"headers": [],
					"content": {"size": 0, "mimeType": "text/html"}
				}
			}]
		}
	}`))

	// Seed: HAR with form-urlencoded body
	f.Add([]byte(`{
		"log": {
			"version": "1.2",
			"entries": [{
				"request": {
					"method": "POST",
					"url": "https://example.com/login",
					"headers": [],
					"queryString": [],
					"postData": {
						"mimeType": "application/x-www-form-urlencoded",
						"text": "username=admin&password=secret"
					}
				},
				"response": {"status": 302, "statusText": "Found", "headers": [], "content": {"size": 0, "mimeType": "text/html"}}
			}]
		}
	}`))

	// Seed: HAR with timings
	f.Add([]byte(`{
		"log": {
			"version": "1.2",
			"entries": [{
				"startedDateTime": "2024-06-15T10:30:00.000Z",
				"time": 350,
				"request": {
					"method": "GET",
					"url": "https://example.com/slow",
					"headers": [],
					"queryString": []
				},
				"response": {
					"status": 200,
					"statusText": "OK",
					"headers": [],
					"content": {"size": 1024, "mimeType": "application/octet-stream"}
				},
				"timings": {
					"dns": 10,
					"connect": 50,
					"ssl": 80,
					"send": 5,
					"wait": 150,
					"receive": 55
				}
			}]
		}
	}`))

	// Seed: HAR with no creator
	f.Add([]byte(`{
		"log": {
			"version": "1.2",
			"entries": [{
				"request": {
					"method": "DELETE",
					"url": "https://api.example.com/items/42",
					"headers": [{"name": "Authorization", "value": "Bearer tok"}],
					"queryString": []
				},
				"response": {"status": 204, "statusText": "No Content", "headers": [], "content": {"size": 0, "mimeType": ""}}
			}]
		}
	}`))

	// Seed: HAR with long URL that gets truncated in name
	f.Add([]byte(`{
		"log": {
			"version": "1.2",
			"entries": [{
				"request": {
					"method": "GET",
					"url": "https://very-long-hostname.example.com/very/long/path/to/resource/that/exceeds/sixty/characters/limit",
					"headers": [],
					"queryString": []
				},
				"response": {"status": 200, "statusText": "OK", "headers": [], "content": {"size": 0, "mimeType": "text/html"}}
			}]
		}
	}`))

	// Invalid inputs
	f.Add([]byte(`not json`))
	f.Add([]byte(`{"log":{"version":"1.2","entries":[]}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		col, err := ParseHAR(data)
		if err != nil {
			return
		}
		if col == nil {
			t.Fatal("ParseHAR returned nil collection without error")
		}
		if col.Name == "" {
			t.Fatal("ParseHAR returned collection with empty name without error")
		}
		if len(col.Items) == 0 {
			t.Fatal("ParseHAR returned collection with no items without error")
		}
	})
}
