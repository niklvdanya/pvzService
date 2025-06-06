package main

import (
	"context"
	"log"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"gitlab.ozon.dev/safariproxd/homework/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	grpcAddress = "localhost:50051"
	httpAddress = "localhost:8081"
)

func main() {
	ctx := context.Background()
	mux := runtime.NewServeMux()
	err := api.RegisterOrdersServiceHandlerFromEndpoint(ctx, mux, grpcAddress, []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	})
	if err != nil {
		log.Fatalf("RegisterOrdersServiceHandlerFromEndpoint err: %v", err)
	}

	log.Printf("http server running on %v", httpAddress)
	if err := http.ListenAndServe(httpAddress, mux); err != nil {
		log.Fatalf("http server running err: %v", err)
	}
}
