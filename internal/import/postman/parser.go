package postman

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sadopc/gottp/internal/core/collection"
)

// postmanCollection represents a Postman Collection v2.1.
type postmanCollection struct {
	Info struct {
		Name   string `json:"name"`
		Schema string `json:"schema"`
	} `json:"info"`
	Item     []postmanItem `json:"item"`
	Auth     *postmanAuth  `json:"auth,omitempty"`
	Variable []postmanVar  `json:"variable,omitempty"`
}

type postmanItem struct {
	Name    string        `json:"name"`
	Item    []postmanItem `json:"item,omitempty"` // folder
	Request *postmanReq   `json:"request,omitempty"`
}

type postmanReq struct {
	Method string          `json:"method"`
	Header []postmanKV     `json:"header,omitempty"`
	Body   *postmanBody    `json:"body,omitempty"`
	URL    json.RawMessage `json:"url"`
	Auth   *postmanAuth    `json:"auth,omitempty"`
}

type postmanBody struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw"`
}

type postmanKV struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type postmanAuth struct {
	Type   string      `json:"type"`
	Basic  []postmanKV `json:"basic,omitempty"`
	Bearer []postmanKV `json:"bearer,omitempty"`
	Apikey []postmanKV `json:"apikey,omitempty"`
}

type postmanVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type postmanURLObj struct {
	Raw   string      `json:"raw"`
	Query []postmanKV `json:"query,omitempty"`
}

// ParsePostman parses a Postman Collection v2.1 JSON into a gottp Collection.
func ParsePostman(data []byte) (*collection.Collection, error) {
	var pc postmanCollection
	if err := json.Unmarshal(data, &pc); err != nil {
		return nil, fmt.Errorf("parsing Postman JSON: %w", err)
	}

	if pc.Info.Name == "" {
		return nil, fmt.Errorf("invalid Postman collection: missing info.name")
	}

	col := &collection.Collection{
		Name:    pc.Info.Name,
		Version: "1.0",
	}

	if len(pc.Variable) > 0 {
		col.Variables = make(map[string]string)
		for _, v := range pc.Variable {
			col.Variables[v.Key] = v.Value
		}
	}

	for _, item := range pc.Item {
		col.Items = append(col.Items, convertItem(item))
	}

	return col, nil
}

func convertItem(pi postmanItem) collection.Item {
	if len(pi.Item) > 0 {
		// It's a folder
		folder := &collection.Folder{Name: pi.Name}
		for _, child := range pi.Item {
			folder.Items = append(folder.Items, convertItem(child))
		}
		return collection.Item{Folder: folder}
	}

	if pi.Request != nil {
		req := &collection.Request{
			ID:       uuid.New().String(),
			Name:     pi.Name,
			Protocol: "http",
			Method:   strings.ToUpper(pi.Request.Method),
			URL:      extractURL(pi.Request.URL),
		}

		// Headers
		for _, h := range pi.Request.Header {
			req.Headers = append(req.Headers, collection.KVPair{
				Key:     h.Key,
				Value:   h.Value,
				Enabled: !h.Disabled,
			})
		}

		// Query params from URL object
		params := extractQueryParams(pi.Request.URL)
		for _, p := range params {
			req.Params = append(req.Params, collection.KVPair{
				Key:     p.Key,
				Value:   p.Value,
				Enabled: !p.Disabled,
			})
		}

		// Body
		if pi.Request.Body != nil && pi.Request.Body.Raw != "" {
			bodyType := "text"
			if pi.Request.Body.Mode == "raw" {
				bodyType = "json" // default assumption
			}
			req.Body = &collection.Body{
				Type:    bodyType,
				Content: pi.Request.Body.Raw,
			}
		}

		// Auth
		if pi.Request.Auth != nil {
			req.Auth = convertAuth(pi.Request.Auth)
		}

		return collection.Item{Request: req}
	}

	return collection.Item{}
}

func extractURL(raw json.RawMessage) string {
	// Try as string first
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	// Try as object
	var obj postmanURLObj
	if json.Unmarshal(raw, &obj) == nil {
		return obj.Raw
	}
	return ""
}

func extractQueryParams(raw json.RawMessage) []postmanKV {
	var obj postmanURLObj
	if json.Unmarshal(raw, &obj) == nil {
		return obj.Query
	}
	return nil
}

func convertAuth(pa *postmanAuth) *collection.Auth {
	if pa == nil {
		return nil
	}
	switch pa.Type {
	case "basic":
		auth := &collection.Auth{Type: "basic", Basic: &collection.BasicAuth{}}
		for _, kv := range pa.Basic {
			switch kv.Key {
			case "username":
				auth.Basic.Username = kv.Value
			case "password":
				auth.Basic.Password = kv.Value
			}
		}
		return auth
	case "bearer":
		auth := &collection.Auth{Type: "bearer", Bearer: &collection.BearerAuth{}}
		for _, kv := range pa.Bearer {
			if kv.Key == "token" {
				auth.Bearer.Token = kv.Value
			}
		}
		return auth
	case "apikey":
		auth := &collection.Auth{Type: "apikey", APIKey: &collection.APIKeyAuth{In: "header"}}
		for _, kv := range pa.Apikey {
			switch kv.Key {
			case "key":
				auth.APIKey.Key = kv.Value
			case "value":
				auth.APIKey.Value = kv.Value
			case "in":
				auth.APIKey.In = kv.Value
			}
		}
		return auth
	}
	return nil
}
