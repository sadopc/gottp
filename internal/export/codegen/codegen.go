package codegen

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/serdar/gottp/internal/protocol"
)

// Language represents a target programming language.
type Language string

const (
	LangGo         Language = "go"
	LangPython     Language = "python"
	LangJavaScript Language = "javascript"
	LangCurl       Language = "curl"
	LangRuby       Language = "ruby"
	LangJava       Language = "java"
	LangRust       Language = "rust"
	LangPHP        Language = "php"
)

// Languages returns all supported languages.
func Languages() []Language {
	return []Language{LangGo, LangPython, LangJavaScript, LangCurl, LangRuby, LangJava, LangRust, LangPHP}
}

// Generate generates a code snippet for the given request in the specified language.
func Generate(req *protocol.Request, lang Language) (string, error) {
	switch lang {
	case LangGo:
		return generateGo(req), nil
	case LangPython:
		return generatePython(req), nil
	case LangJavaScript:
		return generateJavaScript(req), nil
	case LangCurl:
		return generateCurl(req), nil
	case LangRuby:
		return generateRuby(req), nil
	case LangJava:
		return generateJava(req), nil
	case LangRust:
		return generateRust(req), nil
	case LangPHP:
		return generatePHP(req), nil
	default:
		return "", fmt.Errorf("unsupported language: %s", lang)
	}
}

func buildFullURL(req *protocol.Request) string {
	u := req.URL
	if len(req.Params) > 0 {
		params := url.Values{}
		keys := sortedKeys(req.Params)
		for _, k := range keys {
			params.Set(k, req.Params[k])
		}
		sep := "?"
		if strings.Contains(u, "?") {
			sep = "&"
		}
		u += sep + params.Encode()
	}
	return u
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func generateGo(req *protocol.Request) string {
	var b strings.Builder
	fullURL := buildFullURL(req)
	hasBody := len(req.Body) > 0

	b.WriteString("package main\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"fmt\"\n")
	b.WriteString("\t\"io\"\n")
	b.WriteString("\t\"net/http\"\n")
	if hasBody {
		b.WriteString("\t\"strings\"\n")
	}
	b.WriteString(")\n\n")
	b.WriteString("func main() {\n")

	if hasBody {
		b.WriteString(fmt.Sprintf("\tbody := strings.NewReader(`%s`)\n", string(req.Body)))
		b.WriteString(fmt.Sprintf("\treq, err := http.NewRequest(%q, %q, body)\n", req.Method, fullURL))
	} else {
		b.WriteString(fmt.Sprintf("\treq, err := http.NewRequest(%q, %q, nil)\n", req.Method, fullURL))
	}
	b.WriteString("\tif err != nil {\n\t\tpanic(err)\n\t}\n\n")

	for _, k := range sortedKeys(req.Headers) {
		b.WriteString(fmt.Sprintf("\treq.Header.Set(%q, %q)\n", k, req.Headers[k]))
	}
	if len(req.Headers) > 0 {
		b.WriteString("\n")
	}

	// Auth
	if req.Auth != nil {
		switch req.Auth.Type {
		case "basic":
			b.WriteString(fmt.Sprintf("\treq.SetBasicAuth(%q, %q)\n\n", req.Auth.Username, req.Auth.Password))
		case "bearer":
			b.WriteString(fmt.Sprintf("\treq.Header.Set(\"Authorization\", \"Bearer %s\")\n\n", req.Auth.Token))
		}
	}

	b.WriteString("\tresp, err := http.DefaultClient.Do(req)\n")
	b.WriteString("\tif err != nil {\n\t\tpanic(err)\n\t}\n")
	b.WriteString("\tdefer resp.Body.Close()\n\n")
	b.WriteString("\tdata, _ := io.ReadAll(resp.Body)\n")
	b.WriteString("\tfmt.Println(resp.Status)\n")
	b.WriteString("\tfmt.Println(string(data))\n")
	b.WriteString("}\n")

	return b.String()
}

func generatePython(req *protocol.Request) string {
	var b strings.Builder
	fullURL := buildFullURL(req)

	b.WriteString("import requests\n\n")

	// Headers
	if len(req.Headers) > 0 {
		b.WriteString("headers = {\n")
		for _, k := range sortedKeys(req.Headers) {
			b.WriteString(fmt.Sprintf("    %q: %q,\n", k, req.Headers[k]))
		}
		b.WriteString("}\n\n")
	}

	// Body
	if len(req.Body) > 0 {
		b.WriteString(fmt.Sprintf("data = '''%s'''\n\n", string(req.Body)))
	}

	// Auth
	authStr := ""
	if req.Auth != nil {
		switch req.Auth.Type {
		case "basic":
			authStr = fmt.Sprintf(", auth=(%q, %q)", req.Auth.Username, req.Auth.Password)
		case "bearer":
			if len(req.Headers) == 0 {
				b.WriteString(fmt.Sprintf("headers = {\"Authorization\": \"Bearer %s\"}\n\n", req.Auth.Token))
			} else {
				// Insert into existing headers block - already written above, so add it separately
				b.WriteString(fmt.Sprintf("headers[\"Authorization\"] = \"Bearer %s\"\n\n", req.Auth.Token))
			}
		}
	}

	method := strings.ToLower(req.Method)
	args := fmt.Sprintf("%q", fullURL)
	if len(req.Headers) > 0 {
		args += ", headers=headers"
	}
	if len(req.Body) > 0 {
		args += ", data=data"
	}
	args += authStr

	b.WriteString(fmt.Sprintf("response = requests.%s(%s)\n", method, args))
	b.WriteString("print(response.status_code)\n")
	b.WriteString("print(response.text)\n")

	return b.String()
}

func generateJavaScript(req *protocol.Request) string {
	var b strings.Builder
	fullURL := buildFullURL(req)

	b.WriteString(fmt.Sprintf("const response = await fetch(%q, {\n", fullURL))
	b.WriteString(fmt.Sprintf("  method: %q,\n", req.Method))

	if len(req.Headers) > 0 || (req.Auth != nil && req.Auth.Type == "bearer") {
		b.WriteString("  headers: {\n")
		for _, k := range sortedKeys(req.Headers) {
			b.WriteString(fmt.Sprintf("    %q: %q,\n", k, req.Headers[k]))
		}
		if req.Auth != nil && req.Auth.Type == "bearer" {
			b.WriteString(fmt.Sprintf("    \"Authorization\": \"Bearer %s\",\n", req.Auth.Token))
		}
		b.WriteString("  },\n")
	}

	if len(req.Body) > 0 {
		b.WriteString(fmt.Sprintf("  body: `%s`,\n", string(req.Body)))
	}

	b.WriteString("});\n\n")
	b.WriteString("const data = await response.text();\n")
	b.WriteString("console.log(response.status, data);\n")

	return b.String()
}

func generateCurl(req *protocol.Request) string {
	var parts []string
	parts = append(parts, "curl")

	if req.Method != "GET" {
		parts = append(parts, "-X", req.Method)
	}

	for _, k := range sortedKeys(req.Headers) {
		parts = append(parts, "-H", fmt.Sprintf("'%s: %s'", k, req.Headers[k]))
	}

	if req.Auth != nil {
		switch req.Auth.Type {
		case "basic":
			parts = append(parts, "-u", fmt.Sprintf("'%s:%s'", req.Auth.Username, req.Auth.Password))
		case "bearer":
			parts = append(parts, "-H", fmt.Sprintf("'Authorization: Bearer %s'", req.Auth.Token))
		}
	}

	if len(req.Body) > 0 {
		body := strings.ReplaceAll(string(req.Body), "'", "'\\''")
		parts = append(parts, "-d", fmt.Sprintf("'%s'", body))
	}

	parts = append(parts, fmt.Sprintf("'%s'", buildFullURL(req)))
	return strings.Join(parts, " \\\n  ")
}

func generateRuby(req *protocol.Request) string {
	var b strings.Builder
	fullURL := buildFullURL(req)

	b.WriteString("require 'net/http'\n")
	b.WriteString("require 'uri'\n")
	b.WriteString("require 'json'\n\n")

	b.WriteString(fmt.Sprintf("uri = URI.parse(%q)\n", fullURL))
	b.WriteString("http = Net::HTTP.new(uri.host, uri.port)\n")
	b.WriteString("http.use_ssl = uri.scheme == 'https'\n\n")

	methodClass := "Get"
	switch req.Method {
	case "POST":
		methodClass = "Post"
	case "PUT":
		methodClass = "Put"
	case "PATCH":
		methodClass = "Patch"
	case "DELETE":
		methodClass = "Delete"
	case "HEAD":
		methodClass = "Head"
	case "OPTIONS":
		methodClass = "Options"
	}

	b.WriteString(fmt.Sprintf("request = Net::HTTP::%s.new(uri.request_uri)\n", methodClass))

	for _, k := range sortedKeys(req.Headers) {
		b.WriteString(fmt.Sprintf("request[%q] = %q\n", k, req.Headers[k]))
	}

	if req.Auth != nil {
		switch req.Auth.Type {
		case "basic":
			b.WriteString(fmt.Sprintf("request.basic_auth(%q, %q)\n", req.Auth.Username, req.Auth.Password))
		case "bearer":
			b.WriteString(fmt.Sprintf("request['Authorization'] = 'Bearer %s'\n", req.Auth.Token))
		}
	}

	if len(req.Body) > 0 {
		b.WriteString(fmt.Sprintf("request.body = '%s'\n", strings.ReplaceAll(string(req.Body), "'", "\\'")))
	}

	b.WriteString("\nresponse = http.request(request)\n")
	b.WriteString("puts response.code\n")
	b.WriteString("puts response.body\n")

	return b.String()
}

func generateJava(req *protocol.Request) string {
	var b strings.Builder
	fullURL := buildFullURL(req)

	b.WriteString("import java.net.URI;\n")
	b.WriteString("import java.net.http.HttpClient;\n")
	b.WriteString("import java.net.http.HttpRequest;\n")
	b.WriteString("import java.net.http.HttpResponse;\n\n")

	b.WriteString("public class Main {\n")
	b.WriteString("    public static void main(String[] args) throws Exception {\n")
	b.WriteString("        HttpClient client = HttpClient.newHttpClient();\n\n")

	b.WriteString("        HttpRequest request = HttpRequest.newBuilder()\n")
	b.WriteString(fmt.Sprintf("            .uri(URI.create(%q))\n", fullURL))

	if len(req.Body) > 0 {
		b.WriteString(fmt.Sprintf("            .method(%q, HttpRequest.BodyPublishers.ofString(%q))\n",
			req.Method, string(req.Body)))
	} else {
		switch req.Method {
		case "GET":
			b.WriteString("            .GET()\n")
		case "DELETE":
			b.WriteString("            .DELETE()\n")
		default:
			b.WriteString(fmt.Sprintf("            .method(%q, HttpRequest.BodyPublishers.noBody())\n", req.Method))
		}
	}

	for _, k := range sortedKeys(req.Headers) {
		b.WriteString(fmt.Sprintf("            .header(%q, %q)\n", k, req.Headers[k]))
	}

	if req.Auth != nil && req.Auth.Type == "bearer" {
		b.WriteString(fmt.Sprintf("            .header(\"Authorization\", \"Bearer %s\")\n", req.Auth.Token))
	}

	b.WriteString("            .build();\n\n")
	b.WriteString("        HttpResponse<String> response = client.send(request,\n")
	b.WriteString("            HttpResponse.BodyHandlers.ofString());\n\n")
	b.WriteString("        System.out.println(response.statusCode());\n")
	b.WriteString("        System.out.println(response.body());\n")
	b.WriteString("    }\n")
	b.WriteString("}\n")

	return b.String()
}

func generateRust(req *protocol.Request) string {
	var b strings.Builder
	fullURL := buildFullURL(req)

	b.WriteString("// Add to Cargo.toml: reqwest = { version = \"0.12\", features = [\"blocking\"] }\n\n")
	b.WriteString("use reqwest;\n\n")
	b.WriteString("fn main() -> Result<(), Box<dyn std::error::Error>> {\n")
	b.WriteString("    let client = reqwest::blocking::Client::new();\n\n")

	method := strings.ToLower(req.Method)
	b.WriteString(fmt.Sprintf("    let response = client.%s(%q)\n", method, fullURL))

	for _, k := range sortedKeys(req.Headers) {
		b.WriteString(fmt.Sprintf("        .header(%q, %q)\n", k, req.Headers[k]))
	}

	if req.Auth != nil {
		switch req.Auth.Type {
		case "basic":
			b.WriteString(fmt.Sprintf("        .basic_auth(%q, Some(%q))\n", req.Auth.Username, req.Auth.Password))
		case "bearer":
			b.WriteString(fmt.Sprintf("        .bearer_auth(%q)\n", req.Auth.Token))
		}
	}

	if len(req.Body) > 0 {
		b.WriteString(fmt.Sprintf("        .body(%q)\n", string(req.Body)))
	}

	b.WriteString("        .send()?;\n\n")
	b.WriteString("    println!(\"{}\", response.status());\n")
	b.WriteString("    println!(\"{}\", response.text()?);\n")
	b.WriteString("    Ok(())\n")
	b.WriteString("}\n")

	return b.String()
}

func generatePHP(req *protocol.Request) string {
	var b strings.Builder
	fullURL := buildFullURL(req)

	b.WriteString("<?php\n\n")
	b.WriteString("$ch = curl_init();\n\n")
	b.WriteString(fmt.Sprintf("curl_setopt($ch, CURLOPT_URL, %q);\n", fullURL))
	b.WriteString("curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);\n")

	if req.Method != "GET" {
		b.WriteString(fmt.Sprintf("curl_setopt($ch, CURLOPT_CUSTOMREQUEST, %q);\n", req.Method))
	}

	if len(req.Headers) > 0 || (req.Auth != nil && req.Auth.Type == "bearer") {
		b.WriteString("curl_setopt($ch, CURLOPT_HTTPHEADER, [\n")
		for _, k := range sortedKeys(req.Headers) {
			b.WriteString(fmt.Sprintf("    '%s: %s',\n", k, req.Headers[k]))
		}
		if req.Auth != nil && req.Auth.Type == "bearer" {
			b.WriteString(fmt.Sprintf("    'Authorization: Bearer %s',\n", req.Auth.Token))
		}
		b.WriteString("]);\n")
	}

	if req.Auth != nil && req.Auth.Type == "basic" {
		b.WriteString(fmt.Sprintf("curl_setopt($ch, CURLOPT_USERPWD, '%s:%s');\n", req.Auth.Username, req.Auth.Password))
	}

	if len(req.Body) > 0 {
		b.WriteString(fmt.Sprintf("curl_setopt($ch, CURLOPT_POSTFIELDS, '%s');\n",
			strings.ReplaceAll(string(req.Body), "'", "\\'")))
	}

	b.WriteString("\n$response = curl_exec($ch);\n")
	b.WriteString("$httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);\n")
	b.WriteString("curl_close($ch);\n\n")
	b.WriteString("echo $httpCode . \"\\n\";\n")
	b.WriteString("echo $response . \"\\n\";\n")

	return b.String()
}
