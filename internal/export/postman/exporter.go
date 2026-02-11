package postman

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/serdar/gottp/internal/core/collection"
)

// postmanCollection represents a Postman Collection v2.1.
type postmanCollection struct {
	Info     postmanInfo   `json:"info"`
	Item     []postmanItem `json:"item"`
	Variable []postmanVar  `json:"variable,omitempty"`
}

type postmanInfo struct {
	PostmanID string `json:"_postman_id"`
	Name      string `json:"name"`
	Schema    string `json:"schema"`
}

type postmanItem struct {
	Name    string        `json:"name"`
	Item    []postmanItem `json:"item,omitempty"`
	Request *postmanReq   `json:"request,omitempty"`
}

type postmanReq struct {
	Method string       `json:"method"`
	Header []postmanKV  `json:"header,omitempty"`
	Body   *postmanBody `json:"body,omitempty"`
	URL    postmanURL   `json:"url"`
	Auth   *postmanAuth `json:"auth,omitempty"`
}

type postmanBody struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw"`
}

type postmanKV struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled,omitempty"`
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

type postmanURL struct {
	Raw   string      `json:"raw"`
	Query []postmanKV `json:"query,omitempty"`
}

// Export converts a gottp Collection to Postman Collection v2.1 JSON.
func Export(col *collection.Collection) ([]byte, error) {
	if col == nil {
		return nil, fmt.Errorf("collection is nil")
	}

	pc := postmanCollection{
		Info: postmanInfo{
			PostmanID: uuid.New().String(),
			Name:      col.Name,
			Schema:    "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
	}

	if len(col.Variables) > 0 {
		for k, v := range col.Variables {
			pc.Variable = append(pc.Variable, postmanVar{Key: k, Value: v})
		}
	}

	for _, item := range col.Items {
		pc.Item = append(pc.Item, exportItem(item))
	}

	return json.MarshalIndent(pc, "", "  ")
}

func exportItem(item collection.Item) postmanItem {
	if item.Folder != nil {
		pi := postmanItem{Name: item.Folder.Name}
		for _, child := range item.Folder.Items {
			pi.Item = append(pi.Item, exportItem(child))
		}
		return pi
	}

	if item.Request != nil {
		return postmanItem{
			Name:    item.Request.Name,
			Request: exportRequest(item.Request),
		}
	}

	return postmanItem{}
}

func exportRequest(req *collection.Request) *postmanReq {
	pr := &postmanReq{
		Method: req.Method,
		URL:    postmanURL{Raw: req.URL},
	}

	for _, h := range req.Headers {
		pr.Header = append(pr.Header, postmanKV{
			Key:      h.Key,
			Value:    h.Value,
			Disabled: !h.Enabled,
		})
	}

	for _, p := range req.Params {
		pr.URL.Query = append(pr.URL.Query, postmanKV{
			Key:      p.Key,
			Value:    p.Value,
			Disabled: !p.Enabled,
		})
	}

	if req.Body != nil && req.Body.Content != "" {
		pr.Body = &postmanBody{
			Mode: "raw",
			Raw:  req.Body.Content,
		}
	}

	if req.Auth != nil {
		pr.Auth = exportAuth(req.Auth)
	}

	return pr
}

func exportAuth(auth *collection.Auth) *postmanAuth {
	if auth == nil {
		return nil
	}
	switch auth.Type {
	case "basic":
		if auth.Basic != nil {
			return &postmanAuth{
				Type: "basic",
				Basic: []postmanKV{
					{Key: "username", Value: auth.Basic.Username},
					{Key: "password", Value: auth.Basic.Password},
				},
			}
		}
	case "bearer":
		if auth.Bearer != nil {
			return &postmanAuth{
				Type: "bearer",
				Bearer: []postmanKV{
					{Key: "token", Value: auth.Bearer.Token},
				},
			}
		}
	case "apikey":
		if auth.APIKey != nil {
			return &postmanAuth{
				Type: "apikey",
				Apikey: []postmanKV{
					{Key: "key", Value: auth.APIKey.Key},
					{Key: "value", Value: auth.APIKey.Value},
					{Key: "in", Value: auth.APIKey.In},
				},
			}
		}
	}
	return nil
}
