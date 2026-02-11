package grpc

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fullstorydev/grpcurl"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/serdar/gottp/internal/protocol"
)

// Client implements the gRPC protocol using server reflection and grpcurl
// for dynamic invocation without compiled protobuf stubs.
type Client struct {
	mu    sync.Mutex
	conns map[string]*grpc.ClientConn

	// Streaming state for client-streaming and bidi-streaming RPCs.
	streamMu    sync.Mutex
	streamInput chan string // channel for feeding messages to the request supplier
	streamDone  chan struct{}
}

// New creates a new gRPC client.
func New() *Client {
	return &Client{
		conns: make(map[string]*grpc.ClientConn),
	}
}

func (c *Client) Name() string { return "grpc" }

func (c *Client) Validate(req *protocol.Request) error {
	if req.URL == "" {
		return fmt.Errorf("URL (host:port) is required")
	}
	if req.GRPCService == "" {
		return fmt.Errorf("gRPC service name is required")
	}
	if req.GRPCMethod == "" {
		return fmt.Errorf("gRPC method name is required")
	}
	return nil
}

func (c *Client) Execute(ctx context.Context, req *protocol.Request) (*protocol.Response, error) {
	if err := c.Validate(req); err != nil {
		return nil, err
	}

	// Get or create a connection for this address.
	conn, err := c.getConn(req.URL)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", req.URL, err)
	}

	// Build the full method name: "package.Service/Method"
	fullMethod := req.GRPCService + "/" + req.GRPCMethod

	// Create reflection client and descriptor source.
	refClient := grpcreflect.NewClientAuto(ctx, conn)
	defer refClient.Reset()

	descSource := grpcurl.DescriptorSourceFromServer(ctx, refClient)

	// Build metadata from req.Metadata and auth.
	md := buildMetadata(req)

	// Convert metadata to header strings for grpcurl.
	var headers []string
	for k, vals := range md {
		for _, v := range vals {
			headers = append(headers, k+": "+v)
		}
	}

	// Prepare the request body as a reader. grpcurl expects JSON input.
	var requestBody io.Reader
	if len(req.Body) > 0 {
		requestBody = bytes.NewReader(req.Body)
	} else {
		requestBody = bytes.NewReader([]byte("{}"))
	}

	// Create request parser and response formatter.
	requestParser := grpcurl.NewJSONRequestParser(requestBody, nil)
	var responseBuf bytes.Buffer
	formatter := grpcurl.NewJSONFormatter(true, nil)

	// Create event handler to capture response data.
	handler := &responseHandler{
		out:       &responseBuf,
		formatter: formatter,
	}

	// Set timeout.
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	invokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Invoke the RPC.
	start := time.Now()
	err = grpcurl.InvokeRPC(
		invokeCtx,
		descSource,
		conn,
		fullMethod,
		headers,
		handler,
		requestParser.Next,
	)
	duration := time.Since(start)

	// Even if err is non-nil, the handler may have received trailers with a status.
	// grpcurl returns nil for gRPC errors (the status is in the handler).
	// A non-nil err here means something went wrong outside gRPC status handling.
	if err != nil {
		return nil, fmt.Errorf("invoking %s: %w", fullMethod, err)
	}

	// Map gRPC status to response.
	grpcStatus := handler.status
	if grpcStatus == nil {
		grpcStatus = status.New(codes.OK, "")
	}

	httpCode := grpcStatusToHTTP(grpcStatus.Code())
	statusText := fmt.Sprintf("%d %s (grpc: %s)", httpCode, http.StatusText(httpCode), grpcStatus.Code().String())

	// Build response body. If the gRPC call returned an error status,
	// include the error message in the body.
	respBody := responseBuf.Bytes()
	if grpcStatus.Code() != codes.OK && len(respBody) == 0 {
		errBody := fmt.Sprintf(`{"grpc_code": "%s", "message": %q}`,
			grpcStatus.Code().String(), grpcStatus.Message())
		respBody = []byte(errBody)
	}

	// Build response headers from received metadata.
	respHeaders := make(http.Header)
	for k, vals := range handler.responseHeaders {
		for _, v := range vals {
			respHeaders.Add(k, v)
		}
	}
	for k, vals := range handler.responseTrailers {
		for _, v := range vals {
			respHeaders.Add("trailer-"+k, v)
		}
	}
	respHeaders.Set("grpc-status", grpcStatus.Code().String())
	if grpcStatus.Message() != "" {
		respHeaders.Set("grpc-message", grpcStatus.Message())
	}

	return &protocol.Response{
		StatusCode:  httpCode,
		Status:      statusText,
		Headers:     respHeaders,
		Body:        respBody,
		ContentType: "application/json",
		Duration:    duration,
		Size:        int64(len(respBody)),
		Proto:       "gRPC",
		TLS:         strings.HasPrefix(req.URL, "dns:///") || strings.Contains(req.URL, ":443"),
	}, nil
}

// Close closes all cached gRPC connections.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for addr, conn := range c.conns {
		conn.Close()
		delete(c.conns, addr)
	}
}

// IsStreaming uses server reflection to detect whether the given method uses
// client-streaming, server-streaming, or both (bidirectional).
func (c *Client) IsStreaming(ctx context.Context, req *protocol.Request) (serverStream, clientStream bool, err error) {
	if err = c.Validate(req); err != nil {
		return false, false, err
	}

	conn, err := c.getConn(req.URL)
	if err != nil {
		return false, false, fmt.Errorf("connecting to %s: %w", req.URL, err)
	}

	refClient := grpcreflect.NewClientAuto(ctx, conn)
	defer refClient.Reset()

	descSource := grpcurl.DescriptorSourceFromServer(ctx, refClient)

	fullMethod := req.GRPCService + "/" + req.GRPCMethod

	// Find the method descriptor via the service descriptor.
	dsc, err := descSource.FindSymbol(req.GRPCService)
	if err != nil {
		return false, false, fmt.Errorf("finding service %s: %w", req.GRPCService, err)
	}

	sd, ok := dsc.(*desc.ServiceDescriptor)
	if !ok {
		return false, false, fmt.Errorf("symbol %s is not a service descriptor", req.GRPCService)
	}

	for _, md := range sd.GetMethods() {
		if md.GetFullyQualifiedName() == req.GRPCMethod || md.GetName() == req.GRPCMethod ||
			req.GRPCService+"/"+md.GetName() == fullMethod {
			return md.IsServerStreaming(), md.IsClientStreaming(), nil
		}
	}

	return false, false, fmt.Errorf("method %s not found in service %s", req.GRPCMethod, req.GRPCService)
}

// StreamExecute executes a streaming gRPC RPC, sending each received message
// to msgChan as it arrives. For server-streaming RPCs, InvokeRPC is called
// with the request body and the handler streams responses to the channel.
// For client-streaming or bidi-streaming RPCs, messages are read from an
// internal input channel (fed via SendStreamMessage) and sent as the request
// supplier. The caller should close msgChan only after StreamExecute returns.
// The channel is closed by this method when the RPC completes.
func (c *Client) StreamExecute(ctx context.Context, req *protocol.Request, msgChan chan<- protocol.StreamMessage) error {
	if err := c.Validate(req); err != nil {
		return err
	}

	conn, err := c.getConn(req.URL)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", req.URL, err)
	}

	fullMethod := req.GRPCService + "/" + req.GRPCMethod

	refClient := grpcreflect.NewClientAuto(ctx, conn)
	defer refClient.Reset()

	descSource := grpcurl.DescriptorSourceFromServer(ctx, refClient)

	md := buildMetadata(req)
	var headers []string
	for k, vals := range md {
		for _, v := range vals {
			headers = append(headers, k+": "+v)
		}
	}

	formatter := grpcurl.NewJSONFormatter(true, nil)

	handler := &responseHandler{
		out:       io.Discard,
		formatter: formatter,
		streaming: true,
		msgChan:   msgChan,
	}

	// Detect streaming direction to decide request supplier strategy.
	svrStream, cliStream, detectErr := c.IsStreaming(ctx, req)
	_ = svrStream // server streaming is handled by the handler

	timeout := req.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute // longer timeout for streaming
	}
	invokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if cliStream && detectErr == nil {
		// Client-streaming or bidi: use an input channel for the request supplier.
		c.streamMu.Lock()
		c.streamInput = make(chan string, 16)
		c.streamDone = make(chan struct{})
		c.streamMu.Unlock()

		// Send the initial body if provided.
		if len(req.Body) > 0 {
			msgChan <- protocol.StreamMessage{
				Content:   string(req.Body),
				IsJSON:    true,
				Timestamp: time.Now(),
				Direction: "sent",
			}
		}

		// Build a request supplier that reads from the input channel.
		// The supplier is called repeatedly by grpcurl; returning io.EOF signals
		// the end of the request stream.
		firstMsg := true
		requestSupplier := func(msg proto.Message) error {
			var jsonData string
			if firstMsg && len(req.Body) > 0 {
				firstMsg = false
				jsonData = string(req.Body)
			} else {
				select {
				case data, ok := <-c.streamInput:
					if !ok {
						return io.EOF
					}
					jsonData = data
				case <-invokeCtx.Done():
					return invokeCtx.Err()
				}
			}

			// Parse the JSON into the proto message using a temporary parser.
			parser := grpcurl.NewJSONRequestParser(bytes.NewReader([]byte(jsonData)), nil)
			return parser.Next(msg)
		}

		go func() {
			defer close(msgChan)
			defer func() {
				c.streamMu.Lock()
				if c.streamDone != nil {
					close(c.streamDone)
					c.streamDone = nil
				}
				c.streamInput = nil
				c.streamMu.Unlock()
			}()

			rpcErr := grpcurl.InvokeRPC(invokeCtx, descSource, conn, fullMethod, headers, handler, requestSupplier)
			if rpcErr != nil {
				msgChan <- protocol.StreamMessage{
					Timestamp: time.Now(),
					Direction: "received",
					Err:       rpcErr,
				}
			}
		}()

		return nil
	}

	// Server-streaming or unary-but-called-as-stream: use the body as a single request.
	var requestBody io.Reader
	if len(req.Body) > 0 {
		requestBody = bytes.NewReader(req.Body)
	} else {
		requestBody = bytes.NewReader([]byte("{}"))
	}
	requestParser := grpcurl.NewJSONRequestParser(requestBody, nil)

	go func() {
		defer close(msgChan)

		rpcErr := grpcurl.InvokeRPC(invokeCtx, descSource, conn, fullMethod, headers, handler, requestParser.Next)
		if rpcErr != nil {
			msgChan <- protocol.StreamMessage{
				Timestamp: time.Now(),
				Direction: "received",
				Err:       rpcErr,
			}
		}
	}()

	return nil
}

// SendStreamMessage sends a message on an open client-streaming or bidi-streaming
// RPC. The message should be valid JSON matching the method's input type.
func (c *Client) SendStreamMessage(message string) error {
	c.streamMu.Lock()
	input := c.streamInput
	c.streamMu.Unlock()

	if input == nil {
		return fmt.Errorf("no active client stream")
	}

	select {
	case input <- message:
		return nil
	default:
		return fmt.Errorf("stream input channel full")
	}
}

// CloseStream closes the client-side of a client-streaming or bidi-streaming
// RPC, signalling that no more messages will be sent. This causes the request
// supplier to return io.EOF, which completes the client half of the stream.
func (c *Client) CloseStream() error {
	c.streamMu.Lock()
	input := c.streamInput
	done := c.streamDone
	c.streamMu.Unlock()

	if input == nil {
		return fmt.Errorf("no active client stream")
	}

	close(input)

	// Wait for the RPC to finish (the goroutine will signal streamDone).
	if done != nil {
		<-done
	}
	return nil
}

// getConn returns an existing connection or creates a new one for the given address.
func (c *Client) getConn(addr string) (*grpc.ClientConn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if conn, ok := c.conns[addr]; ok {
		return conn, nil
	}

	// Strip any scheme prefix for the dial target.
	target := addr
	target = strings.TrimPrefix(target, "http://")
	target = strings.TrimPrefix(target, "https://")
	target = strings.TrimPrefix(target, "grpc://")

	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(50*1024*1024)),
	)
	if err != nil {
		return nil, err
	}

	c.conns[addr] = conn
	return conn, nil
}

// buildMetadata constructs gRPC metadata from the request's Metadata map
// and Auth configuration.
func buildMetadata(req *protocol.Request) metadata.MD {
	md := metadata.MD{}

	// Add user-defined metadata.
	for k, v := range req.Metadata {
		md.Append(strings.ToLower(k), v)
	}

	// Add headers as metadata (gRPC metadata is the equivalent of HTTP headers).
	for k, v := range req.Headers {
		key := strings.ToLower(k)
		if key == "content-type" {
			continue // gRPC manages its own content-type
		}
		md.Append(key, v)
	}

	// Apply auth.
	if req.Auth != nil {
		switch req.Auth.Type {
		case "bearer":
			if req.Auth.Token != "" {
				md.Set("authorization", "Bearer "+req.Auth.Token)
			}
		case "basic":
			if req.Auth.Username != "" || req.Auth.Password != "" {
				encoded := base64.StdEncoding.EncodeToString(
					[]byte(req.Auth.Username + ":" + req.Auth.Password),
				)
				md.Set("authorization", "Basic "+encoded)
			}
		case "apikey":
			if req.Auth.APIKey != "" && req.Auth.APIValue != "" {
				md.Set(strings.ToLower(req.Auth.APIKey), req.Auth.APIValue)
			}
		case "oauth2":
			if req.Auth.OAuth2 != nil && req.Auth.OAuth2.AccessToken != "" {
				md.Set("authorization", "Bearer "+req.Auth.OAuth2.AccessToken)
			}
		}
	}

	return md
}

// responseHandler implements grpcurl.InvocationEventHandler to capture
// gRPC response data.
type responseHandler struct {
	out              io.Writer
	formatter        grpcurl.Formatter
	responseHeaders  metadata.MD
	responseTrailers metadata.MD
	status           *status.Status
	numResponses     int

	// Streaming support: when streaming is true, OnReceiveResponse sends
	// each formatted message to msgChan instead of writing to out.
	streaming bool
	msgChan   chan<- protocol.StreamMessage
}

func (h *responseHandler) OnResolveMethod(_ *desc.MethodDescriptor) {}

func (h *responseHandler) OnSendHeaders(_ metadata.MD) {}

func (h *responseHandler) OnReceiveHeaders(md metadata.MD) {
	h.responseHeaders = md
}

func (h *responseHandler) OnReceiveResponse(resp proto.Message) {
	h.numResponses++
	respStr, err := h.formatter(resp)
	if err != nil {
		if h.streaming && h.msgChan != nil {
			h.msgChan <- protocol.StreamMessage{
				Timestamp: time.Now(),
				Direction: "received",
				Err:       fmt.Errorf("formatting response: %w", err),
			}
		}
		return
	}

	if h.streaming && h.msgChan != nil {
		content := strings.TrimSpace(respStr)
		isJSON := len(content) > 0 && (content[0] == '{' || content[0] == '[')
		h.msgChan <- protocol.StreamMessage{
			Content:   content,
			IsJSON:    isJSON,
			Timestamp: time.Now(),
			Direction: "received",
		}
	} else {
		fmt.Fprint(h.out, respStr)
	}
}

func (h *responseHandler) OnReceiveTrailers(stat *status.Status, md metadata.MD) {
	h.status = stat
	h.responseTrailers = md
}

// grpcStatusToHTTP maps gRPC status codes to HTTP status codes for display purposes.
func grpcStatusToHTTP(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return 499 // Client Closed Request (nginx convention)
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusBadRequest
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
