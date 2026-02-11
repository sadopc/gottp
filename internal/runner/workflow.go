package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/serdar/gottp/internal/core/collection"
)

// WorkflowResult holds the results of a workflow execution.
type WorkflowResult struct {
	Name    string   `json:"name"`
	Steps   []Result `json:"steps"`
	Success bool     `json:"success"`
	Error   string   `json:"error,omitempty"`
}

// RunWorkflow executes a named workflow from the collection.
func (r *Runner) RunWorkflow(ctx context.Context, workflowName string, verbose bool) (*WorkflowResult, error) {
	if r.collection == nil {
		return nil, fmt.Errorf("no collection loaded")
	}

	// Find the workflow
	var wf *collection.Workflow
	for i := range r.collection.Workflows {
		if strings.EqualFold(r.collection.Workflows[i].Name, workflowName) {
			wf = &r.collection.Workflows[i]
			break
		}
	}
	if wf == nil {
		// List available workflows
		var names []string
		for _, w := range r.collection.Workflows {
			names = append(names, w.Name)
		}
		if len(names) == 0 {
			return nil, fmt.Errorf("no workflows defined in collection")
		}
		return nil, fmt.Errorf("workflow %q not found (available: %s)", workflowName, strings.Join(names, ", "))
	}

	result := &WorkflowResult{
		Name:    wf.Name,
		Success: true,
	}

	// Build a lookup map of request name -> collection.Request
	requestMap := r.buildRequestMap()

	for i, step := range wf.Steps {
		colReq, ok := requestMap[strings.ToLower(step.Request)]
		if !ok {
			result.Success = false
			result.Error = fmt.Sprintf("step %d: request %q not found", i+1, step.Request)
			return result, nil
		}

		// Execute the request
		stepResult := r.executeRequest(ctx, colReq, verbose)
		result.Steps = append(result.Steps, stepResult)

		if stepResult.Error != nil {
			result.Success = false
			result.Error = fmt.Sprintf("step %d (%s) failed: %v", i+1, step.Request, stepResult.Error)
			return result, nil
		}

		// Extract variables from response
		if len(step.Extracts) > 0 {
			body := stepResult.Body
			if body == nil && stepResult.BodyString != "" {
				body = []byte(stepResult.BodyString)
			}
			if body != nil {
				for varName, expr := range step.Extracts {
					value := extractValue(body, expr)
					if value != "" {
						r.envVars[varName] = value
					}
				}
			}
		}

		// Check condition
		if step.Condition != "" {
			if !evaluateCondition(step.Condition, stepResult) {
				result.Success = false
				result.Error = fmt.Sprintf("step %d (%s): condition failed: %s", i+1, step.Request, step.Condition)
				return result, nil
			}
		}

		if !stepResult.TestsPassed {
			result.Success = false
		}
	}

	return result, nil
}

// buildRequestMap creates a lowercase name -> *collection.Request map.
func (r *Runner) buildRequestMap() map[string]*collection.Request {
	m := make(map[string]*collection.Request)
	r.walkItems(r.collection.Items, "", func(req *collection.Request, folder string) {
		m[strings.ToLower(req.Name)] = req
	})
	return m
}

// extractValue extracts a value from JSON response body using a simple JSONPath-like expression.
// Supports: $.field, $.field.nested, $.array[0].field
func extractValue(body []byte, expr string) string {
	expr = strings.TrimSpace(expr)

	// Simple JSONPath: $.key or $.key.nested
	if strings.HasPrefix(expr, "$.") {
		path := strings.TrimPrefix(expr, "$.")
		return jsonPathExtract(body, path)
	}

	// If it's just a key name, treat it as a top-level field
	return jsonPathExtract(body, expr)
}

// jsonPathExtract does simple dot-notation JSON extraction.
func jsonPathExtract(body []byte, path string) string {
	parts := strings.Split(path, ".")
	var current interface{}

	if err := json.Unmarshal(body, &current); err != nil {
		return ""
	}

	for _, part := range parts {
		// Handle array indexing: field[0]
		if idx := strings.Index(part, "["); idx > 0 {
			field := part[:idx]
			indexStr := strings.TrimSuffix(part[idx+1:], "]")
			var arrayIdx int
			_, _ = fmt.Sscanf(indexStr, "%d", &arrayIdx)

			obj, ok := current.(map[string]interface{})
			if !ok {
				return ""
			}
			arr, ok := obj[field].([]interface{})
			if !ok || arrayIdx >= len(arr) {
				return ""
			}
			current = arr[arrayIdx]
			continue
		}

		obj, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current, ok = obj[part]
		if !ok {
			return ""
		}
	}

	switch v := current.(type) {
	case string:
		return v
	case float64:
		if v == float64(int(v)) {
			return fmt.Sprintf("%d", int(v))
		}
		return fmt.Sprintf("%g", v)
	case bool:
		return fmt.Sprintf("%v", v)
	case nil:
		return ""
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

// evaluateCondition checks simple conditions against a result.
// Supports: status == 200, status < 400, status >= 200
func evaluateCondition(condition string, result Result) bool {
	condition = strings.TrimSpace(condition)

	// status code checks
	if strings.HasPrefix(condition, "status") {
		parts := strings.Fields(condition)
		if len(parts) == 3 {
			op := parts[1]
			var expected int
			_, _ = fmt.Sscanf(parts[2], "%d", &expected)
			actual := result.StatusCode

			switch op {
			case "==":
				return actual == expected
			case "!=":
				return actual != expected
			case "<":
				return actual < expected
			case "<=":
				return actual <= expected
			case ">":
				return actual > expected
			case ">=":
				return actual >= expected
			}
		}
	}

	// success check
	if condition == "success" {
		return result.StatusCode >= 200 && result.StatusCode < 400
	}

	// Default: true (unknown conditions pass)
	return true
}

// ListWorkflows returns all workflow names in the collection.
func (r *Runner) ListWorkflows() []string {
	if r.collection == nil {
		return nil
	}
	var names []string
	for _, w := range r.collection.Workflows {
		names = append(names, w.Name)
	}
	return names
}
