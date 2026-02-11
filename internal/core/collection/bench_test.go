package collection

import (
	"fmt"
	"strings"
	"testing"
)

const smallCollectionYAML = `
name: Small API
version: "1"
items:
  - request:
      id: "req-1"
      name: Get Status
      protocol: http
      method: GET
      url: "https://api.example.com/status"
      headers:
        - { key: Accept, value: application/json, enabled: true }
`

const mediumCollectionYAML = `
name: Medium API
version: "1"
variables:
  base_url: "https://api.example.com"
  api_key: "sk-test-1234"
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
              - { key: limit, value: "50", enabled: true }
            headers:
              - { key: Accept, value: application/json, enabled: true }
              - { key: X-API-Key, value: "{{api_key}}", enabled: true }
        - request:
            id: "req-2"
            name: Create User
            protocol: http
            method: POST
            url: "{{base_url}}/users"
            headers:
              - { key: Content-Type, value: application/json, enabled: true }
            body:
              type: json
              content: '{"name":"test","email":"test@example.com"}'
        - request:
            id: "req-3"
            name: Get User
            protocol: http
            method: GET
            url: "{{base_url}}/users/1"
        - request:
            id: "req-4"
            name: Update User
            protocol: http
            method: PUT
            url: "{{base_url}}/users/1"
            body:
              type: json
              content: '{"name":"updated"}'
        - request:
            id: "req-5"
            name: Delete User
            protocol: http
            method: DELETE
            url: "{{base_url}}/users/1"
  - folder:
      name: Products
      items:
        - request:
            id: "req-6"
            name: List Products
            protocol: http
            method: GET
            url: "{{base_url}}/products"
        - request:
            id: "req-7"
            name: Create Product
            protocol: http
            method: POST
            url: "{{base_url}}/products"
            body:
              type: json
              content: '{"name":"Widget","price":9.99}'
  - folder:
      name: Auth
      items:
        - request:
            id: "req-8"
            name: Login
            protocol: http
            method: POST
            url: "{{base_url}}/auth/login"
            body:
              type: json
              content: '{"email":"admin@example.com","password":"secret"}'
            auth:
              type: none
        - request:
            id: "req-9"
            name: Refresh Token
            protocol: http
            method: POST
            url: "{{base_url}}/auth/refresh"
`

// generateLargeCollectionYAML creates a YAML string with the specified number
// of folders, each containing the specified number of requests.
func generateLargeCollectionYAML(folders, requestsPerFolder int) string {
	var sb strings.Builder
	sb.WriteString("name: Large API\nversion: \"1\"\nvariables:\n  base_url: \"https://api.example.com\"\nitems:\n")
	for f := 0; f < folders; f++ {
		fmt.Fprintf(&sb, "  - folder:\n      name: Folder_%d\n      items:\n", f)
		for r := 0; r < requestsPerFolder; r++ {
			id := f*requestsPerFolder + r
			fmt.Fprintf(&sb, "        - request:\n")
			fmt.Fprintf(&sb, "            id: \"req-%d\"\n", id)
			fmt.Fprintf(&sb, "            name: Request_%d_%d\n", f, r)
			fmt.Fprintf(&sb, "            protocol: http\n")
			if r%3 == 0 {
				fmt.Fprintf(&sb, "            method: GET\n")
			} else if r%3 == 1 {
				fmt.Fprintf(&sb, "            method: POST\n")
				fmt.Fprintf(&sb, "            body:\n")
				fmt.Fprintf(&sb, "              type: json\n")
				fmt.Fprintf(&sb, "              content: '{\"key\":\"value_%d\"}'\n", id)
			} else {
				fmt.Fprintf(&sb, "            method: PUT\n")
			}
			fmt.Fprintf(&sb, "            url: \"{{base_url}}/folder_%d/resource_%d\"\n", f, r)
			fmt.Fprintf(&sb, "            headers:\n")
			fmt.Fprintf(&sb, "              - { key: Accept, value: application/json, enabled: true }\n")
			fmt.Fprintf(&sb, "            params:\n")
			fmt.Fprintf(&sb, "              - { key: page, value: \"1\", enabled: true }\n")
		}
	}
	return sb.String()
}

func BenchmarkLoadFromBytes(b *testing.B) {
	b.Run("Small/1_request", func(b *testing.B) {
		data := []byte(smallCollectionYAML)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := LoadFromBytes(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Medium/9_requests", func(b *testing.B) {
		data := []byte(mediumCollectionYAML)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := LoadFromBytes(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Large/50_requests", func(b *testing.B) {
		data := []byte(generateLargeCollectionYAML(5, 10))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := LoadFromBytes(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Large/200_requests", func(b *testing.B) {
		data := []byte(generateLargeCollectionYAML(10, 20))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := LoadFromBytes(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Large/500_requests", func(b *testing.B) {
		data := []byte(generateLargeCollectionYAML(25, 20))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := LoadFromBytes(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkFlattenItems(b *testing.B) {
	b.Run("Small/1_request", func(b *testing.B) {
		col, err := LoadFromBytes([]byte(smallCollectionYAML))
		if err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = FlattenItems(col.Items, 0, "")
		}
	})

	b.Run("Medium/9_requests_3_folders", func(b *testing.B) {
		col, err := LoadFromBytes([]byte(mediumCollectionYAML))
		if err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = FlattenItems(col.Items, 0, "")
		}
	})

	b.Run("Large/50_requests_5_folders", func(b *testing.B) {
		col, err := LoadFromBytes([]byte(generateLargeCollectionYAML(5, 10)))
		if err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = FlattenItems(col.Items, 0, "")
		}
	})

	b.Run("Large/200_requests_10_folders", func(b *testing.B) {
		col, err := LoadFromBytes([]byte(generateLargeCollectionYAML(10, 20)))
		if err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = FlattenItems(col.Items, 0, "")
		}
	})

	b.Run("Large/500_requests_25_folders", func(b *testing.B) {
		col, err := LoadFromBytes([]byte(generateLargeCollectionYAML(25, 20)))
		if err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = FlattenItems(col.Items, 0, "")
		}
	})
}

func BenchmarkFlattenItemsDeepNesting(b *testing.B) {
	// Build a deeply nested structure: folder > folder > folder > ... > request
	buildNested := func(depth int) []Item {
		req := Item{Request: &Request{
			ID:       "deep-req",
			Name:     "Deep Request",
			Protocol: "http",
			Method:   "GET",
			URL:      "https://example.com",
		}}
		items := []Item{req}
		for d := 0; d < depth; d++ {
			items = []Item{
				{Folder: &Folder{
					Name:  fmt.Sprintf("Level_%d", depth-d),
					Items: items,
				}},
			}
		}
		return items
	}

	depths := []int{3, 5, 10, 20}
	for _, d := range depths {
		b.Run(fmt.Sprintf("Depth_%d", d), func(b *testing.B) {
			items := buildNested(d)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = FlattenItems(items, 0, "")
			}
		})
	}
}

func BenchmarkAssignIDs(b *testing.B) {
	b.Run("NoIDsNeeded", func(b *testing.B) {
		col, err := LoadFromBytes([]byte(mediumCollectionYAML))
		if err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			assignIDs(col.Items)
		}
	})

	b.Run("AllNeedIDs/50_requests", func(b *testing.B) {
		// Build items without IDs
		makeItems := func() []Item {
			var items []Item
			for f := 0; f < 5; f++ {
				folder := &Folder{Name: fmt.Sprintf("Folder_%d", f)}
				for r := 0; r < 10; r++ {
					folder.Items = append(folder.Items, Item{
						Request: &Request{
							Name:     fmt.Sprintf("Req_%d_%d", f, r),
							Protocol: "http",
							Method:   "GET",
							URL:      "https://example.com",
						},
					})
				}
				items = append(items, Item{Folder: folder})
			}
			return items
		}
		items := makeItems()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Reset IDs before each iteration
			for fi := range items {
				if items[fi].Folder != nil {
					for ri := range items[fi].Folder.Items {
						if items[fi].Folder.Items[ri].Request != nil {
							items[fi].Folder.Items[ri].Request.ID = ""
						}
					}
				}
			}
			assignIDs(items)
		}
	})
}
