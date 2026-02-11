package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sadopc/gottp/internal/protocol"
)

func TestGraphQLExecute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type")
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		query, ok := body["query"].(string)
		if !ok || query == "" {
			t.Error("expected query in body")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"countries": []map[string]string{
					{"name": "Germany"},
					{"name": "France"},
				},
			},
		})
	}))
	defer server.Close()

	client := New()
	req := &protocol.Request{
		Protocol:     "graphql",
		URL:          server.URL,
		Headers:      map[string]string{},
		GraphQLQuery: `{ countries { name } }`,
	}

	resp, err := client.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if resp.ContentType != "application/json" {
		t.Errorf("expected application/json, got %s", resp.ContentType)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if _, ok := result["data"]; !ok {
		t.Error("expected data in response")
	}
}

func TestGraphQLWithVariables(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if _, ok := body["variables"]; !ok {
			t.Error("expected variables in body")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": nil})
	}))
	defer server.Close()

	client := New()
	req := &protocol.Request{
		Protocol:         "graphql",
		URL:              server.URL,
		Headers:          map[string]string{},
		GraphQLQuery:     `query ($id: ID!) { user(id: $id) { name } }`,
		GraphQLVariables: `{"id": "123"}`,
	}

	resp, err := client.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGraphQLValidate(t *testing.T) {
	client := New()

	// Missing URL
	err := client.Validate(&protocol.Request{GraphQLQuery: "{ test }"})
	if err == nil {
		t.Error("expected error for missing URL")
	}

	// Missing query
	err = client.Validate(&protocol.Request{URL: "http://test.com"})
	if err == nil {
		t.Error("expected error for missing query")
	}

	// Valid
	err = client.Validate(&protocol.Request{URL: "http://test.com", GraphQLQuery: "{ test }"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestIntrospection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"__schema": map[string]interface{}{
					"types": []map[string]interface{}{
						{
							"name": "Query",
							"kind": "OBJECT",
							"fields": []map[string]interface{}{
								{
									"name": "users",
									"type": map[string]interface{}{
										"name": nil,
										"kind": "LIST",
										"ofType": map[string]interface{}{
											"name": "User",
											"kind": "OBJECT",
										},
									},
								},
							},
						},
						{
							"name":   "__Schema",
							"kind":   "OBJECT",
							"fields": nil,
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	schema, err := RunIntrospection(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(schema.Types) != 1 {
		t.Fatalf("expected 1 type (internal types filtered), got %d", len(schema.Types))
	}
	if schema.Types[0].Name != "Query" {
		t.Errorf("expected Query, got %s", schema.Types[0].Name)
	}
	if len(schema.Types[0].Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(schema.Types[0].Fields))
	}
	if schema.Types[0].Fields[0].Type != "[User]" {
		t.Errorf("expected [User], got %s", schema.Types[0].Fields[0].Type)
	}
}

func TestGraphQLWithBearerAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", auth)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": nil})
	}))
	defer server.Close()

	client := New()
	req := &protocol.Request{
		Protocol:     "graphql",
		URL:          server.URL,
		Headers:      map[string]string{},
		GraphQLQuery: `{ me { name } }`,
		Auth: &protocol.AuthConfig{
			Type:  "bearer",
			Token: "test-token",
		},
	}

	resp, err := client.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGraphQLWithCustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "value" {
			t.Errorf("expected custom header")
		}
		// User-provided Content-Type should override default
		if r.Header.Get("Content-Type") != "application/graphql+json" {
			t.Errorf("expected custom content type, got %s", r.Header.Get("Content-Type"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": nil})
	}))
	defer server.Close()

	client := New()
	req := &protocol.Request{
		Protocol: "graphql",
		URL:      server.URL,
		Headers: map[string]string{
			"X-Custom":     "value",
			"Content-Type": "application/graphql+json",
		},
		GraphQLQuery: `{ test }`,
	}

	_, err := client.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphQLName(t *testing.T) {
	client := New()
	if client.Name() != "graphql" {
		t.Errorf("expected graphql, got %s", client.Name())
	}
}
