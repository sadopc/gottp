package templates

import (
	"github.com/serdar/gottp/internal/core/collection"
)

// Template represents a pre-built request template.
type Template struct {
	Name        string
	Description string
	Category    string
	Request     *collection.Request
}

// Categories returns all template categories.
func Categories() []string {
	return []string{"REST", "GraphQL", "Auth", "WebSocket"}
}

// All returns all built-in templates.
func All() []Template {
	return []Template{
		// REST templates
		{
			Name:        "GET JSON API",
			Description: "Standard GET request with JSON accept header",
			Category:    "REST",
			Request: &collection.Request{
				Name:     "GET JSON API",
				Protocol: "http",
				Method:   "GET",
				URL:      "https://api.example.com/resource",
				Headers: []collection.KVPair{
					{Key: "Accept", Value: "application/json", Enabled: true},
				},
			},
		},
		{
			Name:        "POST JSON",
			Description: "POST request with JSON body",
			Category:    "REST",
			Request: &collection.Request{
				Name:     "POST JSON",
				Protocol: "http",
				Method:   "POST",
				URL:      "https://api.example.com/resource",
				Headers: []collection.KVPair{
					{Key: "Content-Type", Value: "application/json", Enabled: true},
					{Key: "Accept", Value: "application/json", Enabled: true},
				},
				Body: &collection.Body{
					Type:    "json",
					Content: "{\n  \"key\": \"value\"\n}",
				},
			},
		},
		{
			Name:        "PUT Update",
			Description: "PUT request for updating a resource",
			Category:    "REST",
			Request: &collection.Request{
				Name:     "PUT Update",
				Protocol: "http",
				Method:   "PUT",
				URL:      "https://api.example.com/resource/1",
				Headers: []collection.KVPair{
					{Key: "Content-Type", Value: "application/json", Enabled: true},
				},
				Body: &collection.Body{
					Type:    "json",
					Content: "{\n  \"key\": \"updated_value\"\n}",
				},
			},
		},
		{
			Name:        "PATCH Partial Update",
			Description: "PATCH request for partial resource update",
			Category:    "REST",
			Request: &collection.Request{
				Name:     "PATCH Partial Update",
				Protocol: "http",
				Method:   "PATCH",
				URL:      "https://api.example.com/resource/1",
				Headers: []collection.KVPair{
					{Key: "Content-Type", Value: "application/json", Enabled: true},
				},
				Body: &collection.Body{
					Type:    "json",
					Content: "{\n  \"field\": \"new_value\"\n}",
				},
			},
		},
		{
			Name:        "DELETE Resource",
			Description: "DELETE request to remove a resource",
			Category:    "REST",
			Request: &collection.Request{
				Name:     "DELETE Resource",
				Protocol: "http",
				Method:   "DELETE",
				URL:      "https://api.example.com/resource/1",
				Headers: []collection.KVPair{
					{Key: "Accept", Value: "application/json", Enabled: true},
				},
			},
		},
		{
			Name:        "Form POST",
			Description: "POST request with form-urlencoded body",
			Category:    "REST",
			Request: &collection.Request{
				Name:     "Form POST",
				Protocol: "http",
				Method:   "POST",
				URL:      "https://api.example.com/submit",
				Headers: []collection.KVPair{
					{Key: "Content-Type", Value: "application/x-www-form-urlencoded", Enabled: true},
				},
				Body: &collection.Body{
					Type:    "form",
					Content: "field1=value1&field2=value2",
				},
			},
		},
		{
			Name:        "Paginated List",
			Description: "GET request with pagination query params",
			Category:    "REST",
			Request: &collection.Request{
				Name:     "Paginated List",
				Protocol: "http",
				Method:   "GET",
				URL:      "https://api.example.com/items",
				Params: []collection.KVPair{
					{Key: "page", Value: "1", Enabled: true},
					{Key: "limit", Value: "20", Enabled: true},
					{Key: "sort", Value: "created_at", Enabled: true},
					{Key: "order", Value: "desc", Enabled: true},
				},
				Headers: []collection.KVPair{
					{Key: "Accept", Value: "application/json", Enabled: true},
				},
			},
		},

		// GraphQL templates
		{
			Name:        "GraphQL Query",
			Description: "Basic GraphQL query",
			Category:    "GraphQL",
			Request: &collection.Request{
				Name:     "GraphQL Query",
				Protocol: "graphql",
				Method:   "POST",
				URL:      "https://api.example.com/graphql",
				GraphQL: &collection.GraphQLConfig{
					Query: "query {\n  users {\n    id\n    name\n    email\n  }\n}",
				},
			},
		},
		{
			Name:        "GraphQL Mutation",
			Description: "GraphQL mutation with variables",
			Category:    "GraphQL",
			Request: &collection.Request{
				Name:     "GraphQL Mutation",
				Protocol: "graphql",
				Method:   "POST",
				URL:      "https://api.example.com/graphql",
				GraphQL: &collection.GraphQLConfig{
					Query:     "mutation CreateUser($input: CreateUserInput!) {\n  createUser(input: $input) {\n    id\n    name\n  }\n}",
					Variables: "{\n  \"input\": {\n    \"name\": \"John Doe\",\n    \"email\": \"john@example.com\"\n  }\n}",
				},
			},
		},
		{
			Name:        "GraphQL Subscription",
			Description: "GraphQL subscription for real-time data",
			Category:    "GraphQL",
			Request: &collection.Request{
				Name:     "GraphQL Subscription",
				Protocol: "graphql",
				Method:   "POST",
				URL:      "wss://api.example.com/graphql",
				GraphQL: &collection.GraphQLConfig{
					Query: "subscription {\n  messageAdded {\n    id\n    content\n    author\n  }\n}",
				},
			},
		},

		// Auth templates
		{
			Name:        "OAuth2 Token Request",
			Description: "Request an OAuth2 access token (client credentials)",
			Category:    "Auth",
			Request: &collection.Request{
				Name:     "OAuth2 Token Request",
				Protocol: "http",
				Method:   "POST",
				URL:      "https://auth.example.com/oauth/token",
				Headers: []collection.KVPair{
					{Key: "Content-Type", Value: "application/x-www-form-urlencoded", Enabled: true},
				},
				Body: &collection.Body{
					Type:    "form",
					Content: "grant_type=client_credentials&client_id={{client_id}}&client_secret={{client_secret}}",
				},
			},
		},
		{
			Name:        "JWT Login",
			Description: "Login endpoint that returns a JWT token",
			Category:    "Auth",
			Request: &collection.Request{
				Name:     "JWT Login",
				Protocol: "http",
				Method:   "POST",
				URL:      "https://api.example.com/auth/login",
				Headers: []collection.KVPair{
					{Key: "Content-Type", Value: "application/json", Enabled: true},
				},
				Body: &collection.Body{
					Type:    "json",
					Content: "{\n  \"email\": \"{{email}}\",\n  \"password\": \"{{password}}\"\n}",
				},
			},
		},

		// WebSocket templates
		{
			Name:        "WebSocket Echo",
			Description: "WebSocket connection for echo testing",
			Category:    "WebSocket",
			Request: &collection.Request{
				Name:     "WebSocket Echo",
				Protocol: "websocket",
				Method:   "GET",
				URL:      "wss://echo.websocket.org",
			},
		},
		{
			Name:        "WebSocket Chat",
			Description: "WebSocket connection with JSON message format",
			Category:    "WebSocket",
			Request: &collection.Request{
				Name:     "WebSocket Chat",
				Protocol: "websocket",
				Method:   "GET",
				URL:      "wss://api.example.com/ws/chat",
				Headers: []collection.KVPair{
					{Key: "Authorization", Value: "Bearer {{token}}", Enabled: true},
				},
			},
		},
	}
}

// ByCategory returns templates filtered by category.
func ByCategory(category string) []Template {
	var filtered []Template
	for _, t := range All() {
		if t.Category == category {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// ByName finds a template by name.
func ByName(name string) *Template {
	for _, t := range All() {
		if t.Name == name {
			return &t
		}
	}
	return nil
}
