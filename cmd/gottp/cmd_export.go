package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/internal/export"
	harexport "github.com/serdar/gottp/internal/export/har"
	insomniaexport "github.com/serdar/gottp/internal/export/insomnia"
	postmanexport "github.com/serdar/gottp/internal/export/postman"
	"github.com/serdar/gottp/internal/protocol"
)

func exportCmd() {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	formatFlag := fs.String("format", "curl", "Export format: curl, har, postman, insomnia")
	requestFlag := fs.String("request", "", "Export a single request by name")
	outputFlag := fs.String("output", "", "Output file path (default: stdout)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gottp export <collection.gottp.yaml> [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Export a collection to various formats.\n\n")
		fmt.Fprintf(os.Stderr, "Supported formats: curl, har, postman, insomnia\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  gottp export api.gottp.yaml --format curl\n")
		fmt.Fprintf(os.Stderr, "  gottp export api.gottp.yaml --format har --output api.har\n")
		fmt.Fprintf(os.Stderr, "  gottp export api.gottp.yaml --format curl --request \"Get Users\"\n")
	}

	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: collection file path is required\n\n")
		fs.Usage()
		os.Exit(1)
	}

	colPath := fs.Arg(0)
	col, err := collection.LoadFromFile(colPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading collection: %v\n", err)
		os.Exit(1)
	}

	// Collect requests to export
	requests := collectAllRequests(col.Items)
	if *requestFlag != "" {
		var filtered []*collection.Request
		for _, req := range requests {
			if strings.EqualFold(req.Name, *requestFlag) {
				filtered = append(filtered, req)
			}
		}
		if len(filtered) == 0 {
			fmt.Fprintf(os.Stderr, "Error: request %q not found in collection\n", *requestFlag)
			os.Exit(1)
		}
		requests = filtered
	}

	if len(requests) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no requests to export\n")
		os.Exit(1)
	}

	// Determine output writer
	out := os.Stdout
	if *outputFlag != "" {
		f, err := os.Create(*outputFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	}

	switch *formatFlag {
	case "curl":
		exportAsCurl(out, requests)
	case "har":
		exportAsHAR(out, requests)
	case "postman":
		exportAsPostman(out, col)
	case "insomnia":
		exportAsInsomnia(out, col)
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported format %q (use curl, har, postman, or insomnia)\n", *formatFlag)
		os.Exit(1)
	}

	if *outputFlag != "" {
		fmt.Fprintf(os.Stderr, "Exported %d requests to %s\n", len(requests), *outputFlag)
	}
}

func collectAllRequests(items []collection.Item) []*collection.Request {
	var requests []*collection.Request
	for i := range items {
		if items[i].Request != nil {
			requests = append(requests, items[i].Request)
		}
		if items[i].Folder != nil {
			requests = append(requests, collectAllRequests(items[i].Folder.Items)...)
		}
	}
	return requests
}

func exportAsCurl(out *os.File, requests []*collection.Request) {
	for i, colReq := range requests {
		req := collectionRequestToProtocol(colReq)
		curlCmd := export.AsCurl(req)
		if i > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "# %s\n", colReq.Name)
		fmt.Fprintln(out, curlCmd)
	}
}

func exportAsHAR(out *os.File, requests []*collection.Request) {
	var entries []harexport.HAREntry
	for _, colReq := range requests {
		req := collectionRequestToProtocol(colReq)
		resp := &protocol.Response{
			StatusCode:  0,
			Status:      "",
			ContentType: "",
		}
		harData, err := harexport.Export(req, resp)
		if err != nil {
			continue
		}
		var singleHAR harexport.HAR
		if json.Unmarshal(harData, &singleHAR) == nil {
			entries = append(entries, singleHAR.Log.Entries...)
		}
	}

	combined := harexport.HAR{
		Log: harexport.HARLog{
			Version: "1.2",
			Creator: harexport.HARCreator{Name: "gottp", Version: "0.4.0"},
			Entries: entries,
		},
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	_ = enc.Encode(combined)
}

func exportAsPostman(out *os.File, col *collection.Collection) {
	data, err := postmanexport.Export(col)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error exporting to Postman: %v\n", err)
		os.Exit(1)
	}
	_, _ = out.Write(data)
	fmt.Fprintln(out)
}

func exportAsInsomnia(out *os.File, col *collection.Collection) {
	data, err := insomniaexport.Export(col)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error exporting to Insomnia: %v\n", err)
		os.Exit(1)
	}
	_, _ = out.Write(data)
	fmt.Fprintln(out)
}

func collectionRequestToProtocol(colReq *collection.Request) *protocol.Request {
	req := &protocol.Request{
		Protocol: colReq.Protocol,
		Method:   colReq.Method,
		URL:      colReq.URL,
		Headers:  make(map[string]string),
		Params:   make(map[string]string),
	}
	if req.Protocol == "" {
		req.Protocol = "http"
	}
	for _, p := range colReq.Params {
		if p.Enabled && p.Key != "" {
			req.Params[p.Key] = p.Value
		}
	}
	for _, h := range colReq.Headers {
		if h.Enabled && h.Key != "" {
			req.Headers[h.Key] = h.Value
		}
	}
	if colReq.Body != nil && colReq.Body.Content != "" {
		req.Body = []byte(colReq.Body.Content)
	}
	return req
}
