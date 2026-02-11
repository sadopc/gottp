package scripting

// ScriptContext holds the data exposed to scripts.
type ScriptContext struct {
	Request  *ScriptRequest
	Response *ScriptResponse
}

// ScriptRequest is the mutable request object exposed to pre-scripts.
type ScriptRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Params  map[string]string
	Body    string
}

// SetHeader sets a request header.
func (r *ScriptRequest) SetHeader(key, value string) {
	if r.Headers == nil {
		r.Headers = map[string]string{}
	}
	r.Headers[key] = value
}

// SetParam sets a query parameter.
func (r *ScriptRequest) SetParam(key, value string) {
	if r.Params == nil {
		r.Params = map[string]string{}
	}
	r.Params[key] = value
}

// SetBody sets the request body.
func (r *ScriptRequest) SetBody(body string) {
	r.Body = body
}

// SetURL sets the request URL.
func (r *ScriptRequest) SetURL(url string) {
	r.URL = url
}

// ScriptResponse is the read-only response object exposed to post-scripts.
type ScriptResponse struct {
	StatusCode  int
	Status      string
	Body        string
	Headers     map[string]string
	Duration    float64 // milliseconds
	Size        int64
	ContentType string
}
