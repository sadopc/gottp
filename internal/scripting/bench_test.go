package scripting

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func BenchmarkRunPreScript(b *testing.B) {
	b.Run("Simple/LogOnly", func(b *testing.B) {
		engine := NewEngine(5 * time.Second)
		req := &ScriptRequest{
			Method:  "GET",
			URL:     "https://api.example.com/users",
			Headers: map[string]string{"Accept": "application/json"},
		}
		script := `gottp.log("pre-script executed");`
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := engine.RunPreScript(script, req, nil)
			if result.Err != nil {
				b.Fatal(result.Err)
			}
		}
	})

	b.Run("Simple/SetHeader", func(b *testing.B) {
		engine := NewEngine(5 * time.Second)
		req := &ScriptRequest{
			Method:  "GET",
			URL:     "https://api.example.com/users",
			Headers: map[string]string{},
		}
		script := `gottp.request.SetHeader("X-Timestamp", Date.now().toString());`
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := engine.RunPreScript(script, req, nil)
			if result.Err != nil {
				b.Fatal(result.Err)
			}
		}
	})

	b.Run("Medium/MutateRequestAndEnv", func(b *testing.B) {
		engine := NewEngine(5 * time.Second)
		req := &ScriptRequest{
			Method:  "POST",
			URL:     "https://api.example.com/data",
			Headers: map[string]string{},
			Params:  map[string]string{},
		}
		envVars := map[string]string{
			"base_url": "https://api.example.com",
			"token":    "old-token",
		}
		script := `
			gottp.request.SetHeader("Authorization", "Bearer " + gottp.getEnvVar("token"));
			gottp.request.SetHeader("X-Request-ID", gottp.uuid());
			gottp.request.SetParam("timestamp", Date.now().toString());
			gottp.setEnvVar("last_run", Date.now().toString());
			gottp.log("request prepared");
		`
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := engine.RunPreScript(script, req, envVars)
			if result.Err != nil {
				b.Fatal(result.Err)
			}
		}
	})

	b.Run("Complex/UtilityHeavy", func(b *testing.B) {
		engine := NewEngine(5 * time.Second)
		req := &ScriptRequest{
			Method:  "POST",
			URL:     "https://api.example.com/secure",
			Headers: map[string]string{},
			Body:    `{"data":"sensitive"}`,
		}
		script := `
			var body = gottp.request.Body;
			var hash = gottp.sha256(body);
			gottp.request.SetHeader("X-Body-Hash", hash);

			var encoded = gottp.base64encode(body);
			gottp.request.SetHeader("X-Body-Base64-Length", encoded.length.toString());

			var id = gottp.uuid();
			gottp.request.SetHeader("X-Correlation-ID", id);

			gottp.setEnvVar("last_hash", hash);
			gottp.setEnvVar("last_id", id);

			gottp.log("hash: " + hash);
			gottp.log("id: " + id);
		`
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := engine.RunPreScript(script, req, map[string]string{})
			if result.Err != nil {
				b.Fatal(result.Err)
			}
		}
	})

	b.Run("Complex/LoopAndComputation", func(b *testing.B) {
		engine := NewEngine(5 * time.Second)
		req := &ScriptRequest{
			Method:  "GET",
			URL:     "https://api.example.com",
			Headers: map[string]string{},
		}
		script := `
			var items = [];
			for (var i = 0; i < 100; i++) {
				items.push("item_" + i);
			}
			gottp.request.SetHeader("X-Item-Count", items.length.toString());
			gottp.log("processed " + items.length + " items");
		`
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := engine.RunPreScript(script, req, nil)
			if result.Err != nil {
				b.Fatal(result.Err)
			}
		}
	})
}

func BenchmarkRunPostScript(b *testing.B) {
	b.Run("Simple/StatusCheck", func(b *testing.B) {
		engine := NewEngine(5 * time.Second)
		req := &ScriptRequest{}
		resp := &ScriptResponse{
			StatusCode:  200,
			Status:      "200 OK",
			Body:        `{"ok":true}`,
			ContentType: "application/json",
			Duration:    150.5,
			Size:        11,
		}
		script := `gottp.log("status: " + gottp.response.StatusCode);`
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := engine.RunPostScript(script, req, resp, nil)
			if result.Err != nil {
				b.Fatal(result.Err)
			}
		}
	})

	b.Run("Medium/ParseAndCheck", func(b *testing.B) {
		engine := NewEngine(5 * time.Second)
		req := &ScriptRequest{}
		resp := &ScriptResponse{
			StatusCode:  200,
			Status:      "200 OK",
			Body:        `{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}],"total":2}`,
			ContentType: "application/json",
			Duration:    250.0,
			Size:        66,
		}
		script := `
			var data = JSON.parse(gottp.response.Body);
			gottp.log("total users: " + data.total);
			gottp.setEnvVar("first_user_id", data.users[0].id.toString());
		`
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := engine.RunPostScript(script, req, resp, map[string]string{})
			if result.Err != nil {
				b.Fatal(result.Err)
			}
		}
	})

	b.Run("Complex/LargeBodyParse", func(b *testing.B) {
		engine := NewEngine(5 * time.Second)
		req := &ScriptRequest{}
		// Build a large JSON response body
		var items []string
		for i := 0; i < 100; i++ {
			items = append(items, fmt.Sprintf(`{"id":%d,"name":"item_%d","value":%d}`, i, i, i*10))
		}
		bodyStr := `{"items":[` + strings.Join(items, ",") + `],"total":100}`
		resp := &ScriptResponse{
			StatusCode:  200,
			Body:        bodyStr,
			ContentType: "application/json",
			Duration:    500.0,
			Size:        int64(len(bodyStr)),
		}
		script := `
			var data = JSON.parse(gottp.response.Body);
			var sum = 0;
			for (var i = 0; i < data.items.length; i++) {
				sum += data.items[i].value;
			}
			gottp.log("sum: " + sum);
			gottp.setEnvVar("item_count", data.total.toString());
		`
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := engine.RunPostScript(script, req, resp, map[string]string{})
			if result.Err != nil {
				b.Fatal(result.Err)
			}
		}
	})
}

func BenchmarkRunPostScriptAssertions(b *testing.B) {
	engine := NewEngine(5 * time.Second)
	req := &ScriptRequest{}
	resp := &ScriptResponse{
		StatusCode:  200,
		Status:      "200 OK",
		Body:        `{"ok":true,"items":[1,2,3],"message":"success"}`,
		ContentType: "application/json",
		Duration:    100.0,
		Size:        47,
		Headers:     map[string]string{"X-Rate-Limit": "100"},
	}

	b.Run("1_Assertion", func(b *testing.B) {
		script := `
			gottp.test("status is 200", function() {
				gottp.assert(gottp.response.StatusCode === 200);
			});
		`
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := engine.RunPostScript(script, req, resp, nil)
			if result.Err != nil {
				b.Fatal(result.Err)
			}
		}
	})

	b.Run("5_Assertions", func(b *testing.B) {
		script := `
			gottp.test("status is 200", function() {
				gottp.assert(gottp.response.StatusCode === 200);
			});
			gottp.test("is json", function() {
				gottp.assert(gottp.response.ContentType === "application/json");
			});
			gottp.test("has body", function() {
				gottp.assert(gottp.response.Body.length > 0);
			});
			gottp.test("fast response", function() {
				gottp.assert(gottp.response.Duration < 5000);
			});
			gottp.test("body is valid json", function() {
				var data = JSON.parse(gottp.response.Body);
				gottp.assert(data.ok === true);
			});
		`
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := engine.RunPostScript(script, req, resp, nil)
			if result.Err != nil {
				b.Fatal(result.Err)
			}
		}
	})

	b.Run("10_Assertions", func(b *testing.B) {
		script := `
			gottp.test("status is 200", function() {
				gottp.assert(gottp.response.StatusCode === 200);
			});
			gottp.test("is json", function() {
				gottp.assert(gottp.response.ContentType === "application/json");
			});
			gottp.test("has body", function() {
				gottp.assert(gottp.response.Body.length > 0);
			});
			gottp.test("fast response", function() {
				gottp.assert(gottp.response.Duration < 5000);
			});
			gottp.test("body is valid json", function() {
				var data = JSON.parse(gottp.response.Body);
				gottp.assert(data.ok === true);
			});
			gottp.test("has items", function() {
				var data = JSON.parse(gottp.response.Body);
				gottp.assert(data.items.length === 3);
			});
			gottp.test("has message", function() {
				var data = JSON.parse(gottp.response.Body);
				gottp.assert(data.message === "success");
			});
			gottp.test("status string", function() {
				gottp.assert(gottp.response.Status === "200 OK");
			});
			gottp.test("size check", function() {
				gottp.assert(gottp.response.Size > 0);
			});
			gottp.test("content not empty", function() {
				gottp.assert(gottp.response.Body !== "");
			});
		`
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := engine.RunPostScript(script, req, resp, nil)
			if result.Err != nil {
				b.Fatal(result.Err)
			}
		}
	})

	b.Run("20_Assertions", func(b *testing.B) {
		// Build a script with 20 test assertions dynamically
		var sb strings.Builder
		for i := 0; i < 20; i++ {
			fmt.Fprintf(&sb, `
			gottp.test("test_%d", function() {
				gottp.assert(gottp.response.StatusCode === 200, "test %d failed");
			});
			`, i, i)
		}
		script := sb.String()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := engine.RunPostScript(script, req, resp, nil)
			if result.Err != nil {
				b.Fatal(result.Err)
			}
		}
	})
}

func BenchmarkNewEngine(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewEngine(5 * time.Second)
	}
}

func BenchmarkScriptEnvironmentOps(b *testing.B) {
	b.Run("SetAndGet/Few", func(b *testing.B) {
		engine := NewEngine(5 * time.Second)
		req := &ScriptRequest{}
		envVars := map[string]string{"a": "1", "b": "2", "c": "3"}
		script := `
			gottp.setEnvVar("x", gottp.getEnvVar("a") + gottp.getEnvVar("b"));
			gottp.setEnvVar("y", gottp.getEnvVar("c"));
		`
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := engine.RunPreScript(script, req, envVars)
			if result.Err != nil {
				b.Fatal(result.Err)
			}
		}
	})

	b.Run("SetAndGet/Many", func(b *testing.B) {
		engine := NewEngine(5 * time.Second)
		req := &ScriptRequest{}
		envVars := make(map[string]string)
		for i := 0; i < 50; i++ {
			envVars[fmt.Sprintf("var_%d", i)] = fmt.Sprintf("value_%d", i)
		}
		var sb strings.Builder
		for i := 0; i < 50; i++ {
			fmt.Fprintf(&sb, `gottp.setEnvVar("out_%d", gottp.getEnvVar("var_%d") + "_processed");`+"\n", i, i)
		}
		script := sb.String()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := engine.RunPreScript(script, req, envVars)
			if result.Err != nil {
				b.Fatal(result.Err)
			}
		}
	})
}
