package grpc

import (
	"context"
	"fmt"
	"strings"

	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ServiceInfo describes a gRPC service discovered via server reflection.
type ServiceInfo struct {
	Name    string
	Methods []MethodInfo
}

// MethodInfo describes a single method within a gRPC service.
type MethodInfo struct {
	Name           string
	FullName       string
	InputType      string
	OutputType     string
	IsClientStream bool
	IsServerStream bool
}

// DiscoverServices connects to a gRPC server at the given address, uses
// server reflection to enumerate all services and their methods, and returns
// the result. Internal reflection services (grpc.reflection.*) are excluded.
func DiscoverServices(ctx context.Context, addr string) ([]ServiceInfo, error) {
	// Strip scheme prefixes for the dial target.
	target := addr
	target = strings.TrimPrefix(target, "http://")
	target = strings.TrimPrefix(target, "https://")
	target = strings.TrimPrefix(target, "grpc://")

	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", addr, err)
	}
	defer conn.Close()

	refClient := grpcreflect.NewClientAuto(ctx, conn)
	defer refClient.Reset()

	descSource := grpcurl.DescriptorSourceFromServer(ctx, refClient)

	// List all services.
	svcNames, err := grpcurl.ListServices(descSource)
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	var services []ServiceInfo
	for _, svcName := range svcNames {
		// Skip internal reflection services.
		if isInternalService(svcName) {
			continue
		}

		svcInfo := ServiceInfo{Name: svcName}

		// Get the service descriptor to enumerate methods.
		dsc, err := descSource.FindSymbol(svcName)
		if err != nil {
			// If we cannot resolve a service, include it without methods.
			services = append(services, svcInfo)
			continue
		}

		sd, ok := dsc.(*desc.ServiceDescriptor)
		if !ok {
			services = append(services, svcInfo)
			continue
		}

		for _, md := range sd.GetMethods() {
			mi := MethodInfo{
				Name:           md.GetName(),
				FullName:       md.GetFullyQualifiedName(),
				IsClientStream: md.IsClientStreaming(),
				IsServerStream: md.IsServerStreaming(),
			}
			if md.GetInputType() != nil {
				mi.InputType = md.GetInputType().GetFullyQualifiedName()
			}
			if md.GetOutputType() != nil {
				mi.OutputType = md.GetOutputType().GetFullyQualifiedName()
			}
			svcInfo.Methods = append(svcInfo.Methods, mi)
		}

		services = append(services, svcInfo)
	}

	return services, nil
}

// isInternalService returns true for gRPC-internal services that should not
// be shown to the user (reflection, health, channelz, etc.).
func isInternalService(name string) bool {
	return strings.HasPrefix(name, "grpc.reflection.") ||
		strings.HasPrefix(name, "grpc.health.") ||
		strings.HasPrefix(name, "grpc.channelz.") ||
		strings.HasPrefix(name, "grpc.binarylog.")
}
