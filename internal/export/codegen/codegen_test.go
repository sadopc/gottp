package codegen

import (
	"strings"
	"testing"

	"github.com/serdar/gottp/internal/protocol"
)

func testRequest() *protocol.Request {
	return &protocol.Request{
		Method:  "POST",
		URL:     "https://api.example.com/users",
		Headers: map[string]string{"Content-Type": "application/json"},
		Params:  map[string]string{},
		Body:    []byte(`{"name":"John","email":"john@example.com"}`),
	}
}

func TestGenerateAllLanguages(t *testing.T) {
	req := testRequest()
	for _, lang := range Languages() {
		t.Run(string(lang), func(t *testing.T) {
			code, err := Generate(req, lang)
			if err != nil {
				t.Fatalf("Generate(%s) error: %v", lang, err)
			}
			if code == "" {
				t.Fatalf("Generate(%s) returned empty code", lang)
			}
			// All should contain the URL
			if !strings.Contains(code, "api.example.com") {
				t.Errorf("Generate(%s) missing URL in output", lang)
			}
			// All should reference POST
			if !strings.Contains(strings.ToLower(code), "post") {
				t.Errorf("Generate(%s) missing method in output", lang)
			}
		})
	}
}

func TestGenerateGo(t *testing.T) {
	req := testRequest()
	code, _ := Generate(req, LangGo)
	if !strings.Contains(code, "http.NewRequest") {
		t.Error("Go code missing http.NewRequest")
	}
	if !strings.Contains(code, "Content-Type") {
		t.Error("Go code missing header")
	}
}

func TestGeneratePython(t *testing.T) {
	req := testRequest()
	code, _ := Generate(req, LangPython)
	if !strings.Contains(code, "import requests") {
		t.Error("Python code missing import")
	}
	if !strings.Contains(code, "requests.post") {
		t.Error("Python code missing method call")
	}
}

func TestGenerateJavaScript(t *testing.T) {
	req := testRequest()
	code, _ := Generate(req, LangJavaScript)
	if !strings.Contains(code, "fetch(") {
		t.Error("JS code missing fetch")
	}
}

func TestGenerateWithAuth(t *testing.T) {
	req := testRequest()
	req.Auth = &protocol.AuthConfig{
		Type:     "bearer",
		Token:    "my-token-123",
	}

	for _, lang := range Languages() {
		t.Run(string(lang), func(t *testing.T) {
			code, err := Generate(req, lang)
			if err != nil {
				t.Fatalf("Generate(%s) error: %v", lang, err)
			}
			if !strings.Contains(code, "my-token-123") {
				t.Errorf("Generate(%s) missing auth token", lang)
			}
		})
	}
}

func TestGenerateWithParams(t *testing.T) {
	req := &protocol.Request{
		Method:  "GET",
		URL:     "https://api.example.com/search",
		Headers: map[string]string{},
		Params:  map[string]string{"q": "hello", "page": "1"},
	}

	code, _ := Generate(req, LangCurl)
	if !strings.Contains(code, "page=1") {
		t.Error("cURL code missing query param")
	}
	if !strings.Contains(code, "q=hello") {
		t.Error("cURL code missing query param")
	}
}

func TestGenerateUnsupportedLanguage(t *testing.T) {
	req := testRequest()
	_, err := Generate(req, Language("cobol"))
	if err == nil {
		t.Error("expected error for unsupported language")
	}
}

func TestLanguages(t *testing.T) {
	langs := Languages()
	if len(langs) != 8 {
		t.Errorf("expected 8 languages, got %d", len(langs))
	}
}
