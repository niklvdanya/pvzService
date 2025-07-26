package mw

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	grpcCodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func PanicInterceptor() grpc.UnaryServerInterceptor {
	tracer := otel.Tracer("pvz-panic-recovery")

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		ctx, span := tracer.Start(ctx, "panic.recovery",
			trace.WithSpanKind(trace.SpanKindInternal),
			trace.WithAttributes(
				attribute.String("grpc.method", info.FullMethod),
			),
		)
		defer span.End()

		defer func() {
			if r := recover(); r != nil {
				stackTrace := string(debug.Stack())
				panicMsg := fmt.Sprintf("gRPC handler panic: %v", r)

				slog.Error("gRPC handler panic recovered",
					"panic", r,
					"method", info.FullMethod,
					"stack_trace", stackTrace,
				)

				span.SetStatus(codes.Error, panicMsg)
				span.SetAttributes(
					attribute.String("panic.value", fmt.Sprintf("%v", r)),
					attribute.String("panic.type", fmt.Sprintf("%T", r)),
					attribute.String("panic.stack_trace", stackTrace),
					attribute.Bool("panic.recovered", true),
				)

				span.AddEvent("panic.recovered", trace.WithAttributes(
					attribute.String("panic.message", panicMsg),
				))

				err = status.Error(grpcCodes.Internal, "internal server error")
			} else {
				span.SetStatus(codes.Ok, "")
			}
		}()

		resp, err = handler(ctx, req)
		return resp, err
	}
}
