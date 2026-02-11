package har

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/serdar/gottp/internal/core/collection"
)

// HAR represents the HAR 1.2 format.
type HAR struct {
	Log HARLog `json:"log"`
}

// HARLog is the top-level log object.
type HARLog struct {
	Version string     `json:"version"`
	Creator *Creator   `json:"creator,omitempty"`
	Entries []HAREntry `json:"entries"`
}

// Creator identifies the tool that created the HAR.
type Creator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// HAREntry represents a single request/response pair.
type HAREntry struct {
	StartedDateTime string      `json:"startedDateTime"`
	Time            float64     `json:"time"`
	Request         HARRequest  `json:"request"`
	Response        HARResponse `json:"response"`
	Timings         *HARTimings `json:"timings,omitempty"`
}

// HARRequest is the request portion of an entry.
type HARRequest struct {
	Method      string       `json:"method"`
	URL         string       `json:"url"`
	HTTPVersion string       `json:"httpVersion"`
	Headers     []HARHeader  `json:"headers"`
	QueryString []HARQuery   `json:"queryString"`
	PostData    *HARPostData `json:"postData,omitempty"`
	HeadersSize int          `json:"headersSize"`
	BodySize    int          `json:"bodySize"`
}

// HARResponse is the response portion of an entry.
type HARResponse struct {
	Status      int         `json:"status"`
	StatusText  string      `json:"statusText"`
	HTTPVersion string      `json:"httpVersion"`
	Headers     []HARHeader `json:"headers"`
	Content     HARContent  `json:"content"`
	HeadersSize int         `json:"headersSize"`
	BodySize    int         `json:"bodySize"`
}

// HARHeader is a name/value pair for headers.
type HARHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HARQuery is a name/value pair for query string parameters.
type HARQuery struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HARPostData is the body of a request.
type HARPostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

// HARContent is the body of a response.
type HARContent struct {
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

// HARTimings holds timing info for an entry.
type HARTimings struct {
	DNS     float64 `json:"dns"`
	Connect float64 `json:"connect"`
	SSL     float64 `json:"ssl"`
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
}

// ParseHAR parses a HAR file and returns a collection.
func ParseHAR(data []byte) (*collection.Collection, error) {
	var har HAR
	if err := json.Unmarshal(data, &har); err != nil {
		return nil, fmt.Errorf("parsing HAR: %w", err)
	}

	if len(har.Log.Entries) == 0 {
		return nil, fmt.Errorf("HAR file contains no entries")
	}

	name := "HAR Import"
	if har.Log.Creator != nil && har.Log.Creator.Name != "" {
		name = fmt.Sprintf("HAR Import (%s)", har.Log.Creator.Name)
	}

	col := &collection.Collection{
		Name:    name,
		Version: "1.0",
	}

	for _, entry := range har.Log.Entries {
		item := convertEntry(entry)
		col.Items = append(col.Items, item)
	}

	return col, nil
}

func convertEntry(entry HAREntry) collection.Item {
	method := strings.ToUpper(entry.Request.Method)
	reqURL := entry.Request.URL

	// Parse URL to extract base URL without query string
	parsedURL, err := url.Parse(reqURL)
	if err == nil && len(parsedURL.Query()) > 0 {
		// Remove query string from URL; we store params separately
		parsedURL.RawQuery = ""
		reqURL = parsedURL.String()
	}

	// Build a name from the URL path
	name := method + " " + reqURL
	if parsedURL != nil && parsedURL.Path != "" {
		name = method + " " + parsedURL.Path
	}
	if len(name) > 60 {
		name = name[:57] + "..."
	}

	req := &collection.Request{
		ID:       uuid.New().String(),
		Name:     name,
		Protocol: "http",
		Method:   method,
		URL:      reqURL,
	}

	// Headers (skip pseudo-headers and common browser headers)
	for _, h := range entry.Request.Headers {
		lowerName := strings.ToLower(h.Name)
		if strings.HasPrefix(lowerName, ":") {
			continue // skip HTTP/2 pseudo-headers
		}
		req.Headers = append(req.Headers, collection.KVPair{
			Key:     h.Name,
			Value:   h.Value,
			Enabled: true,
		})
	}

	// Query params
	for _, q := range entry.Request.QueryString {
		req.Params = append(req.Params, collection.KVPair{
			Key:     q.Name,
			Value:   q.Value,
			Enabled: true,
		})
	}

	// Body
	if entry.Request.PostData != nil && entry.Request.PostData.Text != "" {
		bodyType := detectBodyType(entry.Request.PostData.MimeType)
		req.Body = &collection.Body{
			Type:    bodyType,
			Content: entry.Request.PostData.Text,
		}
	}

	return collection.Item{Request: req}
}

func detectBodyType(mimeType string) string {
	lower := strings.ToLower(mimeType)
	switch {
	case strings.Contains(lower, "json"):
		return "json"
	case strings.Contains(lower, "xml"):
		return "xml"
	case strings.Contains(lower, "form-urlencoded"):
		return "form"
	case strings.Contains(lower, "multipart"):
		return "multipart"
	default:
		return "text"
	}
}
