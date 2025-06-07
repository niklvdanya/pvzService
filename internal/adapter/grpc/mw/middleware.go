package mw

import (
	"context"
	"log"
	"time"

	"github.com/ulule/limiter/v3"
	server "gitlab.ozon.dev/safariproxd/homework/internal/adapter/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		md, _ := metadata.FromIncomingContext(ctx)
		log.Printf("Request: method=%s, metadata=%v, req=%v", info.FullMethod, md, req)
		resp, err := handler(ctx, req)
		duration := time.Since(start)
		if err != nil {
			log.Printf("Response: method=%s, duration=%v, error=%v", info.FullMethod, duration, err)
		} else {
			log.Printf("Response: method=%s, duration=%v, resp=%v", info.FullMethod, duration, resp)
		}
		return resp, err
	}
}

func ErrorMappingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			return resp, server.MapErrorToGRPCStatus(err)
		}
		return resp, nil
	}
}

func RateLimiterInterceptor(limiter *limiter.Limiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		sender := "unknown"
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if s, ok := md["sender"]; ok {
				sender = s[0]
			}
		}

		limiterCtx, err := limiter.Get(ctx, sender)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if limiterCtx.Reached {
			return nil, status.Error(codes.ResourceExhausted, "rate limited")
		}

		return handler(ctx, req)
	}
}

func ValidationInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if v, ok := req.(proto.Message); ok {
			if validator, ok := v.(interface{ ValidateAll() error }); ok {
				if err := validator.ValidateAll(); err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
				}
			}
		}
		return handler(ctx, req)
	}
}
