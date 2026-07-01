package debugviz

import (
	"context"
	"strings"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const grpcTraceMetadataKey = "x-trace-id"

// UnaryServerInterceptor returns a gRPC unary server interceptor that records a root span.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if !isEnabled() {
			return handler(ctx, req)
		}

		traceID := traceIDFromMetadata(ctx)
		if traceID == "" {
			traceID = newTraceID()
		}

		entryName := formatGRPCMethod(info.FullMethod)
		ctx = withTraceID(ctx, traceID)
		ctx, end := startRootSpan(ctx, protocol.EntryKindGRPC, entryName)
		defer end()

		span := SpanFromContext(ctx)
		resp, err := handler(ctx, req)
		if span != nil && err != nil {
			span.Status = protocol.SpanStatusError
		}
		return resp, err
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor that records a root span.
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !isEnabled() {
			return handler(srv, ss)
		}

		ctx := ss.Context()
		traceID := traceIDFromMetadata(ctx)
		if traceID == "" {
			traceID = newTraceID()
		}

		entryName := formatGRPCMethod(info.FullMethod)
		ctx = withTraceID(ctx, traceID)
		ctx, end := startRootSpan(ctx, protocol.EntryKindGRPC, entryName)
		defer end()

		span := SpanFromContext(ctx)
		wrapped := &streamWithContext{ServerStream: ss, ctx: ctx}
		err := handler(srv, wrapped)
		if span != nil && err != nil {
			span.Status = protocol.SpanStatusError
		}
		return err
	}
}

func formatGRPCMethod(fullMethod string) string {
	return strings.TrimPrefix(fullMethod, "/")
}

func traceIDFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(grpcTraceMetadataKey)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

type streamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *streamWithContext) Context() context.Context {
	return s.ctx
}
