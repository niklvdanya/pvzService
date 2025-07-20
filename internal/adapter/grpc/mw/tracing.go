package mw

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TracingInterceptor() grpc.UnaryServerInterceptor {
	tracer := otel.Tracer("pvz-grpc")

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		ctx = otel.GetTextMapPropagator().Extract(ctx, &metadataCarrier{md})

		ctx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", "OrdersService"),
				attribute.String("rpc.method", info.FullMethod),
			),
		)
		defer span.End()

		resp, err := handler(ctx, req)

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			if st, ok := status.FromError(err); ok {
				span.SetAttributes(
					attribute.String("grpc.status_code", st.Code().String()),
					attribute.String("grpc.message", st.Message()),
				)
			}
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return resp, err
	}
}

type metadataCarrier struct {
	metadata.MD
}

func (c *metadataCarrier) Get(key string) string {
	values := c.MD.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (c *metadataCarrier) Set(key, value string) {
	c.MD.Set(key, value)
}

func (c *metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.MD))
	for k := range c.MD {
		keys = append(keys, k)
	}
	return keys
}
