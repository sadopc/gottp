package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/sadopc/gottp/internal/core/collection"
	"github.com/sadopc/gottp/internal/mock"
)

func mockCmd() {
	fs := flag.NewFlagSet("mock", flag.ExitOnError)
	portFlag := fs.Int("port", 8080, "Port to listen on")
	latencyFlag := fs.Duration("latency", 0, "Artificial response latency (e.g., 200ms, 1s)")
	errorRateFlag := fs.Float64("error-rate", 0, "Random error rate (0.0-1.0)")
	corsOriginFlag := fs.String("cors-origin", "*", "Access-Control-Allow-Origin header value")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gottp mock <collection.gottp.yaml> [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Start a mock HTTP server from a collection file.\n\n")
		fmt.Fprintf(os.Stderr, "The server matches incoming requests by method and URL path against\n")
		fmt.Fprintf(os.Stderr, "collection requests and returns canned responses. CORS headers are\n")
		fmt.Fprintf(os.Stderr, "included by default for frontend development use.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nDynamic variables in response bodies:\n")
		fmt.Fprintf(os.Stderr, "  {{$timestamp}}   Current Unix timestamp\n")
		fmt.Fprintf(os.Stderr, "  {{$uuid}}        Random UUID v4\n")
		fmt.Fprintf(os.Stderr, "  {{$randomInt}}   Random integer (0-9999)\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  gottp mock api.gottp.yaml\n")
		fmt.Fprintf(os.Stderr, "  gottp mock api.gottp.yaml --port 3000\n")
		fmt.Fprintf(os.Stderr, "  gottp mock api.gottp.yaml --latency 200ms\n")
		fmt.Fprintf(os.Stderr, "  gottp mock api.gottp.yaml --error-rate 0.1\n")
		fmt.Fprintf(os.Stderr, "  gottp mock api.gottp.yaml --cors-origin https://myapp.example.com\n")
	}

	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: collection file path is required\n\n")
		fs.Usage()
		os.Exit(2)
	}
	collectionPath := fs.Arg(0)

	// Validate error rate
	if *errorRateFlag < 0 || *errorRateFlag > 1 {
		fmt.Fprintf(os.Stderr, "Error: error-rate must be between 0.0 and 1.0\n")
		os.Exit(2)
	}

	// Validate port
	if *portFlag < 0 || *portFlag > 65535 {
		fmt.Fprintf(os.Stderr, "Error: port must be between 0 and 65535\n")
		os.Exit(2)
	}

	// Load collection
	col, err := collection.LoadFromFile(collectionPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading collection: %v\n", err)
		os.Exit(2)
	}

	// Build options
	opts := []mock.Option{
		mock.WithPort(*portFlag),
	}
	if *latencyFlag > 0 {
		opts = append(opts, mock.WithLatency(*latencyFlag))
	}
	if *errorRateFlag > 0 {
		opts = append(opts, mock.WithErrorRate(*errorRateFlag))
	}
	if *corsOriginFlag != "*" {
		opts = append(opts, mock.WithCORSOrigin(*corsOriginFlag))
	}

	srv := mock.New(col, opts...)

	if len(srv.Routes()) == 0 {
		fmt.Fprintf(os.Stderr, "Warning: no HTTP routes found in collection %q\n", col.Name)
		fmt.Fprintf(os.Stderr, "The mock server will return 404 for all requests.\n\n")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if *latencyFlag > 0 {
		fmt.Fprintf(os.Stderr, "Artificial latency: %s\n", latencyFlag.String())
	}
	if *errorRateFlag > 0 {
		fmt.Fprintf(os.Stderr, "Error rate: %.0f%%\n", *errorRateFlag*100)
	}

	if err := srv.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
