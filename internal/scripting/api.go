package scripting

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/dop251/goja"
	"github.com/google/uuid"
)

// ScriptAPI is the `gottp` global object exposed to scripts.
type ScriptAPI struct {
	envVars     map[string]string
	envChanges  map[string]string
	logs        []string
	testResults []TestResult
	request     *ScriptRequest
	response    *ScriptResponse
}

// TestResult holds the result of a gottp.test() call.
type TestResult struct {
	Name   string
	Passed bool
	Error  string
}

func newScriptAPI(req *ScriptRequest, resp *ScriptResponse, envVars map[string]string) *ScriptAPI {
	env := map[string]string{}
	for k, v := range envVars {
		env[k] = v
	}
	return &ScriptAPI{
		envVars:    env,
		envChanges: map[string]string{},
		request:    req,
		response:   resp,
	}
}

func (a *ScriptAPI) registerOnRuntime(vm *goja.Runtime) {
	gottpObj := vm.NewObject()

	// Environment variables
	gottpObj.Set("setEnvVar", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		value := call.Argument(1).String()
		a.envVars[key] = value
		a.envChanges[key] = value
		return goja.Undefined()
	})
	gottpObj.Set("getEnvVar", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		if v, ok := a.envVars[key]; ok {
			return vm.ToValue(v)
		}
		return goja.Undefined()
	})

	// Logging
	gottpObj.Set("log", func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		a.logs = append(a.logs, fmt.Sprint(args...))
		return goja.Undefined()
	})

	// Testing
	gottpObj.Set("test", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		fn, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			a.testResults = append(a.testResults, TestResult{Name: name, Passed: false, Error: "invalid test function"})
			return goja.Undefined()
		}

		result := TestResult{Name: name, Passed: true}
		_, err := fn(goja.Undefined())
		if err != nil {
			result.Passed = false
			result.Error = err.Error()
		}
		a.testResults = append(a.testResults, result)
		return goja.Undefined()
	})

	gottpObj.Set("assert", func(call goja.FunctionCall) goja.Value {
		val := call.Argument(0).ToBoolean()
		if !val {
			msg := "assertion failed"
			if len(call.Arguments) > 1 {
				msg = call.Argument(1).String()
			}
			panic(vm.NewGoError(fmt.Errorf("%s", msg)))
		}
		return goja.Undefined()
	})

	// Utility functions
	gottpObj.Set("base64encode", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(base64.StdEncoding.EncodeToString([]byte(call.Argument(0).String())))
	})
	gottpObj.Set("base64decode", func(call goja.FunctionCall) goja.Value {
		decoded, err := base64.StdEncoding.DecodeString(call.Argument(0).String())
		if err != nil {
			return vm.ToValue("")
		}
		return vm.ToValue(string(decoded))
	})
	gottpObj.Set("sha256", func(call goja.FunctionCall) goja.Value {
		h := sha256.Sum256([]byte(call.Argument(0).String()))
		return vm.ToValue(hex.EncodeToString(h[:]))
	})
	gottpObj.Set("uuid", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(uuid.New().String())
	})

	// Request/Response objects
	gottpObj.Set("request", a.request)
	gottpObj.Set("response", a.response)

	vm.Set("gottp", gottpObj)
}
