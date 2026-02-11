package insomnia

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sadopc/gottp/internal/core/collection"
)

type insomniaExport struct {
	Type      string             `json:"_type"`
	Resources []insomniaResource `json:"resources"`
}

type insomniaResource struct {
	ID             string        `json:"_id"`
	Type           string        `json:"_type"`
	ParentID       string        `json:"parentId"`
	Name           string        `json:"name"`
	Method         string        `json:"method"`
	URL            string        `json:"url"`
	Body           *insomniaBody `json:"body,omitempty"`
	Headers        []insomniaKV  `json:"headers,omitempty"`
	Parameters     []insomniaKV  `json:"parameters,omitempty"`
	Authentication *insomniaAuth `json:"authentication,omitempty"`
}

type insomniaBody struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

type insomniaKV struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type insomniaAuth struct {
	Type     string `json:"type"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
}

// ParseInsomnia parses an Insomnia v4 export into a gottp Collection.
func ParseInsomnia(data []byte) (*collection.Collection, error) {
	var export insomniaExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("parsing Insomnia JSON: %w", err)
	}

	// Build parent-children map
	children := map[string][]insomniaResource{}
	var workspace *insomniaResource
	for i := range export.Resources {
		r := &export.Resources[i]
		if r.Type == "workspace" {
			workspace = r
			continue
		}
		children[r.ParentID] = append(children[r.ParentID], *r)
	}

	name := "Imported"
	rootID := ""
	if workspace != nil {
		name = workspace.Name
		rootID = workspace.ID
	}

	col := &collection.Collection{
		Name:    name,
		Version: "1.0",
	}

	col.Items = buildItems(children, rootID, nil)
	return col, nil
}

func buildItems(children map[string][]insomniaResource, parentID string, visited map[string]bool) []collection.Item {
	if visited == nil {
		visited = make(map[string]bool)
	}
	if visited[parentID] {
		return nil // break cycles
	}
	visited[parentID] = true

	var items []collection.Item
	for _, r := range children[parentID] {
		switch r.Type {
		case "request_group":
			folder := &collection.Folder{Name: r.Name}
			folder.Items = buildItems(children, r.ID, visited)
			items = append(items, collection.Item{Folder: folder})
		case "request":
			req := &collection.Request{
				ID:       uuid.New().String(),
				Name:     r.Name,
				Protocol: "http",
				Method:   strings.ToUpper(r.Method),
				URL:      r.URL,
			}
			for _, h := range r.Headers {
				req.Headers = append(req.Headers, collection.KVPair{
					Key: h.Name, Value: h.Value, Enabled: !h.Disabled,
				})
			}
			for _, p := range r.Parameters {
				req.Params = append(req.Params, collection.KVPair{
					Key: p.Name, Value: p.Value, Enabled: !p.Disabled,
				})
			}
			if r.Body != nil && r.Body.Text != "" {
				bodyType := "text"
				if strings.Contains(r.Body.MimeType, "json") {
					bodyType = "json"
				} else if strings.Contains(r.Body.MimeType, "xml") {
					bodyType = "xml"
				}
				req.Body = &collection.Body{Type: bodyType, Content: r.Body.Text}
			}
			if r.Authentication != nil {
				req.Auth = convertInsomniaAuth(r.Authentication)
			}
			items = append(items, collection.Item{Request: req})
		}
	}
	return items
}

func convertInsomniaAuth(auth *insomniaAuth) *collection.Auth {
	if auth == nil || auth.Type == "" {
		return nil
	}
	switch auth.Type {
	case "basic":
		return &collection.Auth{
			Type:  "basic",
			Basic: &collection.BasicAuth{Username: auth.Username, Password: auth.Password},
		}
	case "bearer":
		return &collection.Auth{
			Type:   "bearer",
			Bearer: &collection.BearerAuth{Token: auth.Token},
		}
	}
	return nil
}
