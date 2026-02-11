package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Schema holds introspection results.
type Schema struct {
	Types []SchemaType
}

// SchemaType represents a GraphQL type.
type SchemaType struct {
	Name   string
	Kind   string
	Fields []SchemaField
}

// SchemaField represents a field on a type.
type SchemaField struct {
	Name string
	Type string
}

const introspectionQuery = `{
  __schema {
    types {
      name
      kind
      fields {
        name
        type {
          name
          kind
          ofType {
            name
            kind
          }
        }
      }
    }
  }
}`

type introspectionResponse struct {
	Data struct {
		Schema struct {
			Types []struct {
				Name   string `json:"name"`
				Kind   string `json:"kind"`
				Fields []struct {
					Name string `json:"name"`
					Type struct {
						Name   *string `json:"name"`
						Kind   string  `json:"kind"`
						OfType *struct {
							Name *string `json:"name"`
							Kind string  `json:"kind"`
						} `json:"ofType"`
					} `json:"type"`
				} `json:"fields"`
			} `json:"types"`
		} `json:"__schema"`
	} `json:"data"`
}

// RunIntrospection sends an introspection query and parses the result.
func RunIntrospection(ctx context.Context, url string, headers map[string]string) (*Schema, error) {
	body, _ := json.Marshal(map[string]string{"query": introspectionQuery})
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating introspection request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("introspection request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading introspection response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("introspection returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result introspectionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing introspection: %w", err)
	}

	schema := &Schema{}
	for _, t := range result.Data.Schema.Types {
		// Skip internal types
		if len(t.Name) > 0 && t.Name[0] == '_' {
			continue
		}
		st := SchemaType{
			Name: t.Name,
			Kind: t.Kind,
		}
		for _, f := range t.Fields {
			typeName := "unknown"
			if f.Type.Name != nil {
				typeName = *f.Type.Name
			} else if f.Type.OfType != nil && f.Type.OfType.Name != nil {
				typeName = "[" + *f.Type.OfType.Name + "]"
			}
			st.Fields = append(st.Fields, SchemaField{
				Name: f.Name,
				Type: typeName,
			})
		}
		schema.Types = append(schema.Types, st)
	}

	return schema, nil
}
