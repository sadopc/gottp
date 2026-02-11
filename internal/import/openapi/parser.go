package openapi

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/sadopc/gottp/internal/core/collection"
	"gopkg.in/yaml.v3"
)

type openAPISpec struct {
	OpenAPI string              `json:"openapi" yaml:"openapi"`
	Info    openAPIInfo         `json:"info" yaml:"info"`
	Paths   map[string]pathItem `json:"paths" yaml:"paths"`
}

type openAPIInfo struct {
	Title   string `json:"title" yaml:"title"`
	Version string `json:"version" yaml:"version"`
}

type pathItem map[string]operation // method -> operation

type operation struct {
	Summary     string       `json:"summary" yaml:"summary"`
	OperationID string       `json:"operationId" yaml:"operationId"`
	Tags        []string     `json:"tags" yaml:"tags"`
	Parameters  []parameter  `json:"parameters" yaml:"parameters"`
	RequestBody *requestBody `json:"requestBody" yaml:"requestBody"`
}

type parameter struct {
	Name     string      `json:"name" yaml:"name"`
	In       string      `json:"in" yaml:"in"` // query, header, path
	Required bool        `json:"required" yaml:"required"`
	Schema   *schemaObj  `json:"schema" yaml:"schema"`
	Example  interface{} `json:"example" yaml:"example"`
}

type requestBody struct {
	Content map[string]mediaType `json:"content" yaml:"content"`
}

type mediaType struct {
	Schema  *schemaObj  `json:"schema" yaml:"schema"`
	Example interface{} `json:"example" yaml:"example"`
}

type schemaObj struct {
	Type    string      `json:"type" yaml:"type"`
	Example interface{} `json:"example" yaml:"example"`
}

// ParseOpenAPI parses an OpenAPI 3.0 spec (JSON or YAML) into a gottp Collection.
func ParseOpenAPI(data []byte) (*collection.Collection, error) {
	var spec openAPISpec

	// Try JSON first
	if err := json.Unmarshal(data, &spec); err != nil {
		// Try YAML
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("parsing OpenAPI spec: not valid JSON or YAML")
		}
	}

	if spec.OpenAPI == "" {
		return nil, fmt.Errorf("not a valid OpenAPI spec: missing openapi version")
	}

	col := &collection.Collection{
		Name:    spec.Info.Title,
		Version: "1.0",
	}

	// Group by tags -> folders
	tagMap := map[string][]collection.Item{}
	var untagged []collection.Item

	// Sort paths for deterministic output
	paths := make([]string, 0, len(spec.Paths))
	for p := range spec.Paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	methods := []string{"get", "post", "put", "patch", "delete", "head", "options"}

	for _, path := range paths {
		pathOps := spec.Paths[path]
		for _, method := range methods {
			op, ok := pathOps[method]
			if !ok {
				continue
			}

			name := op.Summary
			if name == "" {
				name = op.OperationID
			}
			if name == "" {
				name = strings.ToUpper(method) + " " + path
			}

			req := &collection.Request{
				ID:       uuid.New().String(),
				Name:     name,
				Protocol: "http",
				Method:   strings.ToUpper(method),
				URL:      path, // relative, user adds base URL
			}

			// Parameters
			for _, p := range op.Parameters {
				switch p.In {
				case "query":
					val := ""
					if p.Example != nil {
						val = fmt.Sprintf("%v", p.Example)
					}
					req.Params = append(req.Params, collection.KVPair{
						Key: p.Name, Value: val, Enabled: p.Required,
					})
				case "header":
					val := ""
					if p.Example != nil {
						val = fmt.Sprintf("%v", p.Example)
					}
					req.Headers = append(req.Headers, collection.KVPair{
						Key: p.Name, Value: val, Enabled: true,
					})
				}
			}

			// Request body example
			if op.RequestBody != nil {
				for ct, mt := range op.RequestBody.Content {
					bodyType := "text"
					if strings.Contains(ct, "json") {
						bodyType = "json"
					} else if strings.Contains(ct, "xml") {
						bodyType = "xml"
					}
					content := ""
					if mt.Example != nil {
						if b, err := json.MarshalIndent(mt.Example, "", "  "); err == nil {
							content = string(b)
						}
					}
					req.Body = &collection.Body{Type: bodyType, Content: content}
					req.Headers = append(req.Headers, collection.KVPair{
						Key: "Content-Type", Value: ct, Enabled: true,
					})
					break // use first content type
				}
			}

			item := collection.Item{Request: req}

			if len(op.Tags) > 0 {
				tagMap[op.Tags[0]] = append(tagMap[op.Tags[0]], item)
			} else {
				untagged = append(untagged, item)
			}
		}
	}

	// Build folders from tags
	tags := make([]string, 0, len(tagMap))
	for t := range tagMap {
		tags = append(tags, t)
	}
	sort.Strings(tags)

	for _, tag := range tags {
		col.Items = append(col.Items, collection.Item{
			Folder: &collection.Folder{
				Name:  tag,
				Items: tagMap[tag],
			},
		})
	}
	col.Items = append(col.Items, untagged...)

	return col, nil
}
