package har

import (
	"encoding/json"
	"time"

	"github.com/sadopc/gottp/internal/protocol"
)

// HAR represents the HAR 1.2 format for export.
type HAR struct {
	Log HARLog `json:"log"`
}

// HARLog is the top-level log object.
type HARLog struct {
	Version string     `json:"version"`
	Creator HARCreator `json:"creator"`
	Entries []HAREntry `json:"entries"`
}

// HARCreator identifies the tool that created the HAR.
type HARCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// HAREntry represents a single request/response pair.
type HAREntry struct {
	StartedDateTime string      `json:"startedDateTime"`
	Time            float64     `json:"time"`
	Request         HARRequest  `json:"request"`
	Response        HARResponse `json:"response"`
	Timings         HARTimings  `json:"timings"`
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

// Export creates a HAR 1.2 JSON from a request/response pair.
func Export(req *protocol.Request, resp *protocol.Response) ([]byte, error) {
	harReq := buildHARRequest(req)
	harResp := buildHARResponse(resp)
	timings := buildHARTimings(resp)

	har := HAR{
		Log: HARLog{
			Version: "1.2",
			Creator: HARCreator{Name: "gottp", Version: "0.1.0"},
			Entries: []HAREntry{
				{
					StartedDateTime: time.Now().UTC().Format(time.RFC3339),
					Time:            float64(resp.Duration.Milliseconds()),
					Request:         harReq,
					Response:        harResp,
					Timings:         timings,
				},
			},
		},
	}

	return json.MarshalIndent(har, "", "  ")
}

func buildHARRequest(req *protocol.Request) HARRequest {
	harReq := HARRequest{
		Method:      req.Method,
		URL:         req.URL,
		HTTPVersion: "HTTP/1.1",
		HeadersSize: -1,
		BodySize:    len(req.Body),
	}

	// Headers
	for k, v := range req.Headers {
		harReq.Headers = append(harReq.Headers, HARHeader{Name: k, Value: v})
	}

	// Query params
	for k, v := range req.Params {
		harReq.QueryString = append(harReq.QueryString, HARQuery{Name: k, Value: v})
	}

	// Body
	if len(req.Body) > 0 {
		mimeType := "text/plain"
		if ct, ok := req.Headers["Content-Type"]; ok {
			mimeType = ct
		}
		harReq.PostData = &HARPostData{
			MimeType: mimeType,
			Text:     string(req.Body),
		}
	}

	return harReq
}

func buildHARResponse(resp *protocol.Response) HARResponse {
	harResp := HARResponse{
		Status:      resp.StatusCode,
		StatusText:  resp.Status,
		HTTPVersion: resp.Proto,
		HeadersSize: -1,
		BodySize:    int(resp.Size),
		Content: HARContent{
			Size:     int(resp.Size),
			MimeType: resp.ContentType,
			Text:     string(resp.Body),
		},
	}

	for k, vals := range resp.Headers {
		for _, v := range vals {
			harResp.Headers = append(harResp.Headers, HARHeader{Name: k, Value: v})
		}
	}

	return harResp
}

func buildHARTimings(resp *protocol.Response) HARTimings {
	if resp.Timing == nil {
		total := float64(resp.Duration.Milliseconds())
		return HARTimings{
			DNS:     -1,
			Connect: -1,
			SSL:     -1,
			Send:    0,
			Wait:    total,
			Receive: 0,
		}
	}

	td := resp.Timing
	return HARTimings{
		DNS:     float64(td.DNSLookup.Milliseconds()),
		Connect: float64(td.TCPConnect.Milliseconds()),
		SSL:     float64(td.TLSHandshake.Milliseconds()),
		Send:    0,
		Wait:    float64(td.TTFB.Milliseconds()),
		Receive: float64(td.Transfer.Milliseconds()),
	}
}
