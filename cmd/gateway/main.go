package main

import (
	"context"
	"log"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"gitlab.ozon.dev/safariproxd/homework/internal/config"
	"gitlab.ozon.dev/safariproxd/homework/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	ctx := context.Background()
	mux := runtime.NewServeMux()
	err = api.RegisterOrdersServiceHandlerFromEndpoint(ctx, mux, cfg.Service.GRPCAddress, []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	})
	if err != nil {
		log.Fatalf("RegisterOrdersServiceHandlerFromEndpoint err: %v", err)
	}

	log.Printf("http server running on %v", cfg.Service.HTTPAddress)
	if err := http.ListenAndServe(cfg.Service.HTTPAddress, mux); err != nil {
		log.Fatalf("http server running err: %v", err)
	}
}
