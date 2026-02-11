package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/serdar/gottp/internal/app"
	"github.com/serdar/gottp/internal/config"
	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/internal/runner"
	"github.com/serdar/gottp/pkg/version"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "run":
			runCmd()
			return
		case "init":
			initCmd()
			return
		case "validate":
			validateCmd()
			return
		case "fmt":
			fmtCmd()
			return
		case "import":
			importCmd()
			return
		case "export":
			exportCmd()
			return
		case "mock":
			mockCmd()
			return
		case "completion":
			completionCmd()
			return
		case "version":
			fmt.Printf("gottp %s (%s) built %s\n", version.Version, version.Commit, version.Date)
			return
		case "help":
			printHelp()
			return
		}
	}
	tuiCmd()
}

func printHelp() {
	fmt.Fprintf(os.Stderr, `gottp - A TUI API client for the terminal

Usage:
  gottp [flags]                    Launch TUI (interactive mode)
  gottp <command> [args] [flags]   Run a subcommand

Commands:
  run       Run API requests headlessly from a collection file
  init      Create a new .gottp.yaml collection interactively
  validate  Validate collection and environment YAML files
  fmt       Format and normalize collection YAML files
  import    Import collection from cURL/Postman/Insomnia/OpenAPI/HAR
  export    Export collection to cURL/HAR format
  mock      Start a mock HTTP server from a collection file
  completion  Generate shell completion scripts (bash, zsh, fish)
  version   Print version information
  help      Show this help message

TUI Flags:
  --collection <path>  Path to a .gottp.yaml collection file
  --version            Print version and exit

Run 'gottp <command> --help' for more information about a command.
`)
}

func runCmd() {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	envFlag := fs.String("env", "", "Environment name to use")
	requestFlag := fs.String("request", "", "Run a single request by name")
	folderFlag := fs.String("folder", "", "Run all requests in a folder")
	workflowFlag := fs.String("workflow", "", "Run a named workflow")
	outputFlag := fs.String("output", "text", "Output format: text, json, junit")
	verboseFlag := fs.Bool("verbose", false, "Show response bodies and headers")
	timeoutFlag := fs.Duration("timeout", 30*time.Second, "Request timeout")
	perfSaveFlag := fs.String("perf-save", "", "Save timing results as a performance baseline file")
	perfBaselineFlag := fs.String("perf-baseline", "", "Compare timings against a baseline file")
	perfThresholdFlag := fs.Float64("perf-threshold", 20.0, "Regression threshold percentage (default 20%)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gottp run <collection.gottp.yaml> [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Run API requests headlessly from a collection file.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  gottp run api.gottp.yaml\n")
		fmt.Fprintf(os.Stderr, "  gottp run api.gottp.yaml --env Production\n")
		fmt.Fprintf(os.Stderr, "  gottp run api.gottp.yaml --request \"Get Users\"\n")
		fmt.Fprintf(os.Stderr, "  gottp run api.gottp.yaml --folder Auth --output json\n")
		fmt.Fprintf(os.Stderr, "  gottp run api.gottp.yaml --workflow \"Create and Verify\" --verbose\n")
		fmt.Fprintf(os.Stderr, "  gottp run api.gottp.yaml --output junit > results.xml\n")
		fmt.Fprintf(os.Stderr, "\nExit codes:\n")
		fmt.Fprintf(os.Stderr, "  0  All requests succeeded, all tests passed\n")
		fmt.Fprintf(os.Stderr, "  1  One or more script test assertions failed\n")
		fmt.Fprintf(os.Stderr, "  2  One or more requests had errors\n")
	}

	// Parse args after "run"
	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}

	// Collection path is the first positional argument
	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: collection file path is required\n\n")
		fs.Usage()
		os.Exit(2)
	}
	collectionPath := fs.Arg(0)

	// Validate output format
	switch *outputFlag {
	case "text", "json", "junit":
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid output format %q (must be text, json, or junit)\n", *outputFlag)
		os.Exit(2)
	}

	cfg := runner.Config{
		CollectionPath: collectionPath,
		Environment:    *envFlag,
		RequestName:    *requestFlag,
		FolderName:     *folderFlag,
		WorkflowName:   *workflowFlag,
		OutputFormat:   *outputFlag,
		Verbose:        *verboseFlag,
		Timeout:        *timeoutFlag,
	}

	r, err := runner.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Workflow mode
	if cfg.WorkflowName != "" {
		wfResult, err := r.RunWorkflow(ctx, cfg.WorkflowName, cfg.Verbose)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(2)
		}

		switch cfg.OutputFormat {
		case "json":
			if err := runner.PrintWorkflowJSON(os.Stdout, wfResult); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing JSON: %v\n", err)
				os.Exit(2)
			}
		case "junit":
			if err := runner.PrintWorkflowJUnit(os.Stdout, wfResult); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing JUnit XML: %v\n", err)
				os.Exit(2)
			}
		default:
			runner.PrintWorkflowText(os.Stdout, wfResult, cfg.Verbose)
		}

		if !wfResult.Success {
			os.Exit(1)
		}
		os.Exit(0)
	}

	results, err := r.Run(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	switch cfg.OutputFormat {
	case "json":
		if err := runner.PrintJSON(os.Stdout, results); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing JSON: %v\n", err)
			os.Exit(2)
		}
	case "junit":
		if err := runner.PrintJUnit(os.Stdout, results); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing JUnit XML: %v\n", err)
			os.Exit(2)
		}
	default:
		runner.PrintText(os.Stdout, results, cfg.Verbose)
	}

	// Performance baseline: save
	if *perfSaveFlag != "" {
		if err := runner.SavePerfBaseline(*perfSaveFlag, results); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving perf baseline: %v\n", err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "Performance baseline saved to %s\n", *perfSaveFlag)
	}

	// Performance baseline: compare
	if *perfBaselineFlag != "" {
		baseline, err := runner.LoadPerfBaseline(*perfBaselineFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading perf baseline: %v\n", err)
			os.Exit(2)
		}
		comparisons := runner.ComparePerfBaseline(results, baseline, *perfThresholdFlag)
		fmt.Fprintln(os.Stdout)
		runner.PrintPerfComparison(os.Stdout, comparisons, *perfThresholdFlag)
		if runner.HasRegressions(comparisons) {
			os.Exit(1)
		}
	}

	os.Exit(runner.ExitCode(results))
}

func tuiCmd() {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	collectionFlag := flag.String("collection", "", "Path to a .gottp.yaml collection file")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("gottp %s (%s) built %s\n", version.Version, version.Commit, version.Date)
		os.Exit(0)
	}

	// Load collection
	var col *collection.Collection
	var colPath string

	if *collectionFlag != "" {
		c, err := collection.LoadFromFile(*collectionFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading collection: %v\n", err)
			os.Exit(1)
		}
		col = c
		colPath = *collectionFlag
	} else {
		// Try to find a .gottp.yaml file in the current directory
		cwd, _ := os.Getwd()
		matches, _ := filepath.Glob(filepath.Join(cwd, "*.gottp.yaml"))
		if len(matches) > 0 {
			c, err := collection.LoadFromFile(matches[0])
			if err == nil {
				col = c
				colPath = matches[0]
			}
		}
	}

	cfg := config.Load()
	model := app.New(col, colPath, cfg)
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
