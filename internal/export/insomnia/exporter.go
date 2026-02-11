package insomnia

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/serdar/gottp/internal/core/collection"
)

type insomniaExport struct {
	Type      string             `json:"_type"`
	Format    int                `json:"__export_format"`
	Source    string             `json:"__export_source"`
	Resources []insomniaResource `json:"resources"`
}

type insomniaResource struct {
	ID             string        `json:"_id"`
	Type           string        `json:"_type"`
	ParentID       string        `json:"parentId"`
	Name           string        `json:"name"`
	Method         string        `json:"method,omitempty"`
	URL            string        `json:"url,omitempty"`
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
	Disabled bool   `json:"disabled,omitempty"`
}

type insomniaAuth struct {
	Type     string `json:"type"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
}

// Export converts a gottp Collection to Insomnia v4 export JSON.
func Export(col *collection.Collection) ([]byte, error) {
	if col == nil {
		return nil, fmt.Errorf("collection is nil")
	}

	workspaceID := "wrk_" + uuid.New().String()[:8]

	export := insomniaExport{
		Type:   "export",
		Format: 4,
		Source: "gottp",
		Resources: []insomniaResource{
			{
				ID:   workspaceID,
				Type: "workspace",
				Name: col.Name,
			},
		},
	}

	exportItems(col.Items, workspaceID, &export.Resources)

	return json.MarshalIndent(export, "", "  ")
}

func exportItems(items []collection.Item, parentID string, resources *[]insomniaResource) {
	for _, item := range items {
		if item.Folder != nil {
			folderID := "fld_" + uuid.New().String()[:8]
			*resources = append(*resources, insomniaResource{
				ID:       folderID,
				Type:     "request_group",
				ParentID: parentID,
				Name:     item.Folder.Name,
			})
			exportItems(item.Folder.Items, folderID, resources)
		}
		if item.Request != nil {
			*resources = append(*resources, exportRequest(item.Request, parentID))
		}
	}
}

func exportRequest(req *collection.Request, parentID string) insomniaResource {
	r := insomniaResource{
		ID:       "req_" + uuid.New().String()[:8],
		Type:     "request",
		ParentID: parentID,
		Name:     req.Name,
		Method:   req.Method,
		URL:      req.URL,
	}

	for _, h := range req.Headers {
		r.Headers = append(r.Headers, insomniaKV{
			Name:     h.Key,
			Value:    h.Value,
			Disabled: !h.Enabled,
		})
	}

	for _, p := range req.Params {
		r.Parameters = append(r.Parameters, insomniaKV{
			Name:     p.Key,
			Value:    p.Value,
			Disabled: !p.Enabled,
		})
	}

	if req.Body != nil && req.Body.Content != "" {
		mimeType := "text/plain"
		switch req.Body.Type {
		case "json":
			mimeType = "application/json"
		case "xml":
			mimeType = "application/xml"
		}
		r.Body = &insomniaBody{
			MimeType: mimeType,
			Text:     req.Body.Content,
		}
	}

	if req.Auth != nil {
		r.Authentication = exportAuth(req.Auth)
	}

	return r
}

func exportAuth(auth *collection.Auth) *insomniaAuth {
	if auth == nil {
		return nil
	}
	switch auth.Type {
	case "basic":
		if auth.Basic != nil {
			return &insomniaAuth{
				Type:     "basic",
				Username: auth.Basic.Username,
				Password: auth.Basic.Password,
			}
		}
	case "bearer":
		if auth.Bearer != nil {
			return &insomniaAuth{
				Type:  "bearer",
				Token: auth.Bearer.Token,
			}
		}
	}
	return nil
}


