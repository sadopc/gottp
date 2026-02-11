package collection

import "github.com/google/uuid"

// Collection represents a collection of API requests.
type Collection struct {
	Name      string            `yaml:"name"`
	Version   string            `yaml:"version"`
	Auth      *Auth             `yaml:"auth,omitempty"`
	Variables map[string]string `yaml:"variables,omitempty"`
	Items     []Item            `yaml:"items"`
}

// Item is a union type: either a Folder or a Request.
type Item struct {
	Folder  *Folder  `yaml:"folder,omitempty"`
	Request *Request `yaml:"request,omitempty"`
}

// Folder groups related requests.
type Folder struct {
	Name  string `yaml:"name"`
	Items []Item `yaml:"items,omitempty"`
}

// Request represents an API request.
type Request struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Protocol string `yaml:"protocol"` // http, graphql, grpc, websocket
	Method   string `yaml:"method"`
	URL      string `yaml:"url"`

	Params  []KVPair `yaml:"params,omitempty"`
	Headers []KVPair `yaml:"headers,omitempty"`
	Auth    *Auth    `yaml:"auth,omitempty"`
	Body    *Body    `yaml:"body,omitempty"`

	GraphQL   *GraphQLConfig   `yaml:"graphql,omitempty"`
	WebSocket *WebSocketConfig `yaml:"websocket,omitempty"`
	GRPC      *GRPCConfig      `yaml:"grpc,omitempty"`

	PreScript  string `yaml:"pre_script,omitempty"`
	PostScript string `yaml:"post_script,omitempty"`
}

// NewRequest creates a new request with defaults.
func NewRequest(name, method, url string) *Request {
	return &Request{
		ID:       uuid.New().String(),
		Name:     name,
		Protocol: "http",
		Method:   method,
		URL:      url,
	}
}

// KVPair represents a key-value pair (header, param, etc.)
type KVPair struct {
	Key     string `yaml:"key"`
	Value   string `yaml:"value"`
	Enabled bool   `yaml:"enabled"`
}

// Auth represents authentication configuration.
type Auth struct {
	Type   string      `yaml:"type"` // none, basic, bearer, apikey, oauth2, awsv4
	Basic  *BasicAuth  `yaml:"basic,omitempty"`
	Bearer *BearerAuth `yaml:"bearer,omitempty"`
	APIKey *APIKeyAuth `yaml:"apikey,omitempty"`
	OAuth2 *OAuth2Auth `yaml:"oauth2,omitempty"`
	AWSAuth *AWSAuth   `yaml:"awsv4,omitempty"`
}

// BasicAuth holds basic auth credentials.
type BasicAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// BearerAuth holds a bearer token.
type BearerAuth struct {
	Token string `yaml:"token"`
}

// APIKeyAuth holds an API key configuration.
type APIKeyAuth struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
	In    string `yaml:"in"` // header, query
}

// OAuth2Auth holds OAuth2 auth configuration in collection files.
type OAuth2Auth struct {
	GrantType    string `yaml:"grant_type"` // authorization_code, client_credentials, password
	AuthURL      string `yaml:"auth_url,omitempty"`
	TokenURL     string `yaml:"token_url"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret,omitempty"`
	Scope        string `yaml:"scope,omitempty"`
	Username     string `yaml:"username,omitempty"`
	Password     string `yaml:"password,omitempty"`
	UsePKCE      bool   `yaml:"use_pkce,omitempty"`
}

// AWSAuth holds AWS Signature v4 auth configuration in collection files.
type AWSAuth struct {
	AccessKeyID    string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	SessionToken   string `yaml:"session_token,omitempty"`
	Region         string `yaml:"region"`
	Service        string `yaml:"service"`
}

// Body represents a request body.
type Body struct {
	Type    string `yaml:"type"` // none, json, xml, text, form, multipart
	Content string `yaml:"content"`
}

// GraphQLConfig holds GraphQL-specific settings.
type GraphQLConfig struct {
	Query     string `yaml:"query"`
	Variables string `yaml:"variables,omitempty"`
}

// WebSocketConfig holds WebSocket-specific settings.
type WebSocketConfig struct {
	Messages []WSMessage `yaml:"messages,omitempty"`
}

// WSMessage represents a pre-defined WebSocket message.
type WSMessage struct {
	Name    string `yaml:"name"`
	Content string `yaml:"content"`
	IsJSON  bool   `yaml:"is_json"`
}

// GRPCConfig holds gRPC-specific settings.
type GRPCConfig struct {
	Service  string   `yaml:"service"`
	Method   string   `yaml:"method"`
	Metadata []KVPair `yaml:"metadata,omitempty"`
}

// FlatItem represents a flattened tree item for display.
type FlatItem struct {
	Request  *Request
	Folder   *Folder
	Depth    int
	IsFolder bool
	Expanded bool
	Path     string // "Collection/Folder/Request"
}

// FlattenItems flattens the tree for display.
func FlattenItems(items []Item, depth int, parentPath string) []FlatItem {
	var result []FlatItem
	for i := range items {
		item := &items[i]
		if item.Folder != nil {
			path := parentPath + "/" + item.Folder.Name
			result = append(result, FlatItem{
				Folder:   item.Folder,
				Depth:    depth,
				IsFolder: true,
				Expanded: true,
				Path:     path,
			})
			result = append(result, FlattenItems(item.Folder.Items, depth+1, path)...)
		}
		if item.Request != nil {
			path := parentPath + "/" + item.Request.Name
			result = append(result, FlatItem{
				Request: item.Request,
				Depth:   depth,
				Path:    path,
			})
		}
	}
	return result
}
