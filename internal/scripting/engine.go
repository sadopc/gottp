package scripting

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"
)

// Engine executes JavaScript pre/post-request scripts.
type Engine struct {
	timeout time.Duration
}

// NewEngine creates a new scripting engine with the given timeout.
func NewEngine(timeout time.Duration) *Engine {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &Engine{timeout: timeout}
}

// Result holds script execution results.
type Result struct {
	Logs        []string
	TestResults []TestResult
	EnvChanges  map[string]string
	Err         error
}

// RunPreScript executes a pre-request script that can mutate the request.
func (e *Engine) RunPreScript(script string, req *ScriptRequest, envVars map[string]string) *Result {
	api := newScriptAPI(req, nil, envVars)
	err := e.run(script, api)
	return &Result{
		Logs:        api.logs,
		TestResults: api.testResults,
		EnvChanges:  api.envChanges,
		Err:         err,
	}
}

// RunPostScript executes a post-request script with access to the response.
func (e *Engine) RunPostScript(script string, req *ScriptRequest, resp *ScriptResponse, envVars map[string]string) *Result {
	api := newScriptAPI(req, resp, envVars)
	err := e.run(script, api)
	return &Result{
		Logs:        api.logs,
		TestResults: api.testResults,
		EnvChanges:  api.envChanges,
		Err:         err,
	}
}

func (e *Engine) run(script string, api *ScriptAPI) error {
	vm := goja.New()
	api.registerOnRuntime(vm)

	// Set up timeout via context
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	// Interrupt VM on timeout
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			vm.Interrupt("script timeout exceeded")
		case <-done:
		}
	}()

	_, err := vm.RunString(script)
	close(done)

	if err != nil {
		return fmt.Errorf("script error: %w", err)
	}
	return nil
}
