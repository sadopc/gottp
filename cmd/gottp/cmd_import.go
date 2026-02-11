package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sadopc/gottp/internal/core/collection"
	importutil "github.com/sadopc/gottp/internal/import"
	curlimport "github.com/sadopc/gottp/internal/import/curl"
	"github.com/sadopc/gottp/internal/import/har"
	"github.com/sadopc/gottp/internal/import/insomnia"
	"github.com/sadopc/gottp/internal/import/openapi"
	"github.com/sadopc/gottp/internal/import/postman"
	"github.com/sadopc/gottp/internal/protocol"
)

func importCmd() {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	formatFlag := fs.String("format", "", "Force format: curl, postman, insomnia, openapi, har (default: auto-detect)")
	outputFlag := fs.String("output", "", "Output .gottp.yaml file path (default: imported.gottp.yaml)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gottp import <file> [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Import a collection from various formats.\n\n")
		fmt.Fprintf(os.Stderr, "Supported formats: cURL, Postman, Insomnia, OpenAPI, HAR.\n")
		fmt.Fprintf(os.Stderr, "Format is auto-detected from file content unless --format is specified.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  gottp import postman-collection.json\n")
		fmt.Fprintf(os.Stderr, "  gottp import openapi.yaml --output api.gottp.yaml\n")
		fmt.Fprintf(os.Stderr, "  gottp import request.har --format har\n")
		fmt.Fprintf(os.Stderr, "  echo 'curl -X GET https://api.example.com' | gottp import -\n")
	}

	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: file path is required\n\n")
		fs.Usage()
		os.Exit(1)
	}

	inputPath := fs.Arg(0)

	// Read input
	var data []byte
	var err error
	if inputPath == "-" {
		// Read from stdin
		data, err = readStdin()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
	} else {
		data, err = os.ReadFile(inputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", inputPath, err)
			os.Exit(1)
		}
	}

	if len(data) == 0 {
		fmt.Fprintf(os.Stderr, "Error: input is empty\n")
		os.Exit(1)
	}

	// Detect or use specified format
	format := *formatFlag
	if format == "" {
		format = importutil.DetectFormat(data)
		if format == "unknown" {
			fmt.Fprintf(os.Stderr, "Error: unable to detect format. Use --format to specify.\n")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Detected format: %s\n", format)
	}

	// Parse based on format
	var col *collection.Collection
	switch format {
	case "curl":
		req, parseErr := curlimport.ParseCurl(string(data))
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "Error parsing cURL: %v\n", parseErr)
			os.Exit(1)
		}
		col = curlRequestToCollection(req)
	case "postman":
		col, err = postman.ParsePostman(data)
	case "insomnia":
		col, err = insomnia.ParseInsomnia(data)
	case "openapi":
		col, err = openapi.ParseOpenAPI(data)
	case "har":
		col, err = har.ParseHAR(data)
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported format %q (use curl, postman, insomnia, openapi, or har)\n", format)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", format, err)
		os.Exit(1)
	}

	if col == nil {
		fmt.Fprintf(os.Stderr, "Error: no data imported\n")
		os.Exit(1)
	}

	// Determine output path
	output := *outputFlag
	if output == "" {
		if inputPath != "-" {
			base := filepath.Base(inputPath)
			ext := filepath.Ext(base)
			output = base[:len(base)-len(ext)] + ".gottp.yaml"
		} else {
			output = "imported.gottp.yaml"
		}
	}

	if err := collection.SaveToFile(col, output); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing collection: %v\n", err)
		os.Exit(1)
	}

	requestCount := countRequests(col.Items)
	fmt.Printf("Imported %d requests from %s -> %s\n", requestCount, format, output)
}

func readStdin() ([]byte, error) {
	var data []byte
	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	return data, nil
}

func curlRequestToCollection(req *protocol.Request) *collection.Collection {
	colReq := collection.NewRequest("Imported Request", req.Method, req.URL)
	for k, v := range req.Headers {
		colReq.Headers = append(colReq.Headers, collection.KVPair{Key: k, Value: v, Enabled: true})
	}
	for k, v := range req.Params {
		colReq.Params = append(colReq.Params, collection.KVPair{Key: k, Value: v, Enabled: true})
	}
	if len(req.Body) > 0 {
		colReq.Body = &collection.Body{Type: "json", Content: string(req.Body)}
	}

	return &collection.Collection{
		Name:    "cURL Import",
		Version: "1",
		Items: []collection.Item{
			{Request: colReq},
		},
	}
}
