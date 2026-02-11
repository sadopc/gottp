package main

import (
	"strings"
	"testing"
)

func TestGenerateBashCompletion(t *testing.T) {
	output := generateBashCompletion()

	if !strings.Contains(output, "_gottp") {
		t.Error("bash completion should contain _gottp function name")
	}
	if !strings.Contains(output, "complete -F _gottp gottp") {
		t.Error("bash completion should register the completion function")
	}
	if !strings.Contains(output, "commands=") {
		t.Error("bash completion should define commands list")
	}

	// Verify all subcommands are listed
	subcommands := []string{"run", "init", "validate", "fmt", "import", "export", "mock", "completion", "version", "help"}
	for _, cmd := range subcommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("bash completion should contain subcommand %q", cmd)
		}
	}

	// Verify run flags are included
	runFlags := []string{"--env", "--request", "--folder", "--workflow", "--output", "--verbose", "--timeout"}
	for _, flag := range runFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("bash completion should contain run flag %q", flag)
		}
	}

	// Verify output format values
	outputFormats := []string{"text", "json", "junit"}
	for _, fmt := range outputFormats {
		if !strings.Contains(output, fmt) {
			t.Errorf("bash completion should contain output format %q", fmt)
		}
	}

	// Verify export format values
	exportFormats := []string{"curl", "har", "postman", "insomnia"}
	for _, fmt := range exportFormats {
		if !strings.Contains(output, fmt) {
			t.Errorf("bash completion should contain export format %q", fmt)
		}
	}
}

func TestGenerateZshCompletion(t *testing.T) {
	output := generateZshCompletion()

	if !strings.Contains(output, "#compdef gottp") {
		t.Error("zsh completion should contain #compdef gottp directive")
	}
	if !strings.Contains(output, "_gottp") {
		t.Error("zsh completion should contain _gottp function")
	}
	if !strings.Contains(output, "_arguments") {
		t.Error("zsh completion should use _arguments for flag completion")
	}
	if !strings.Contains(output, "_describe") {
		t.Error("zsh completion should use _describe for command completion")
	}
	if !strings.Contains(output, `_files -g "*.gottp.yaml"`) {
		t.Error("zsh completion should complete .gottp.yaml files")
	}

	// Verify subcommands with descriptions
	subcommands := []string{"run:", "init:", "validate:", "fmt:", "import:", "export:", "completion:", "version:", "help:"}
	for _, cmd := range subcommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("zsh completion should contain subcommand description for %q", cmd)
		}
	}

	// Verify run flags
	runFlags := []string{"--env", "--request", "--folder", "--workflow", "--output", "--verbose", "--timeout"}
	for _, flag := range runFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("zsh completion should contain run flag %q", flag)
		}
	}

	// Verify format value completions
	if !strings.Contains(output, "(text json junit)") {
		t.Error("zsh completion should provide output format values")
	}
	if !strings.Contains(output, "(curl postman insomnia openapi har)") {
		t.Error("zsh completion should provide import format values")
	}
	if !strings.Contains(output, "(curl har postman insomnia)") {
		t.Error("zsh completion should provide export format values")
	}
}

func TestGenerateFishCompletion(t *testing.T) {
	output := generateFishCompletion()

	if !strings.Contains(output, "complete -c gottp") {
		t.Error("fish completion should contain complete -c gottp commands")
	}
	if !strings.Contains(output, "__fish_use_subcommand") {
		t.Error("fish completion should use __fish_use_subcommand for top-level completions")
	}
	if !strings.Contains(output, "__fish_seen_subcommand_from") {
		t.Error("fish completion should use __fish_seen_subcommand_from for subcommand flags")
	}

	// Verify all subcommands are registered with descriptions
	subcommands := map[string]string{
		"run":        "Run API requests",
		"init":       "Create a new",
		"validate":   "Validate collection",
		"fmt":        "Format and normalize",
		"import":     "Import collection",
		"export":     "Export collection",
		"mock":       "Start a mock server",
		"completion": "Generate shell completion",
		"version":    "Print version",
		"help":       "Show help",
	}
	for cmd, desc := range subcommands {
		if !strings.Contains(output, "-a "+cmd) {
			t.Errorf("fish completion should register subcommand %q", cmd)
		}
		if !strings.Contains(output, desc) {
			t.Errorf("fish completion should have description containing %q for subcommand %q", desc, cmd)
		}
	}

	// Verify run flags
	runFlags := []string{"env", "request", "folder", "workflow", "output", "verbose", "timeout"}
	for _, flag := range runFlags {
		if !strings.Contains(output, "-l "+flag) {
			t.Errorf("fish completion should contain run long flag %q", flag)
		}
	}

	// Verify format completions
	if !strings.Contains(output, "'text json junit'") {
		t.Error("fish completion should provide output format values for run")
	}
	if !strings.Contains(output, "'curl har postman insomnia'") {
		t.Error("fish completion should provide export format values")
	}
	if !strings.Contains(output, "'curl postman insomnia openapi har'") {
		t.Error("fish completion should provide import format values")
	}
}

func TestGenerateBashCompletionShellFormat(t *testing.T) {
	output := generateBashCompletion()

	// Should be a valid shell script starting with a comment
	if !strings.HasPrefix(output, "#") {
		t.Error("bash completion should start with a comment")
	}

	// Should end with the complete command
	trimmed := strings.TrimSpace(output)
	if !strings.HasSuffix(trimmed, "complete -F _gottp gottp") {
		t.Error("bash completion should end with complete registration")
	}
}

func TestGenerateZshCompletionShellFormat(t *testing.T) {
	output := generateZshCompletion()

	// Must start with #compdef
	if !strings.HasPrefix(output, "#compdef gottp") {
		t.Error("zsh completion must start with #compdef gottp")
	}

	// Should end with calling the function
	trimmed := strings.TrimSpace(output)
	if !strings.HasSuffix(trimmed, `_gottp "$@"`) {
		t.Error("zsh completion should end with _gottp \"$@\" call")
	}
}

func TestGenerateFishCompletionShellFormat(t *testing.T) {
	output := generateFishCompletion()

	// Should start with a comment
	if !strings.HasPrefix(output, "#") {
		t.Error("fish completion should start with a comment")
	}

	// Every non-comment, non-empty line should start with "complete"
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "complete ") {
			t.Errorf("fish completion non-comment line should start with 'complete': %q", line)
		}
	}
}
