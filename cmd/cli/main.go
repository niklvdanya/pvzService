package main

import (
	"log"
	"net"
	"os"

	server "gitlab.ozon.dev/safariproxd/homework/internal/adapter/grpc"
	"gitlab.ozon.dev/safariproxd/homework/internal/app"
	"gitlab.ozon.dev/safariproxd/homework/internal/config"
	"gitlab.ozon.dev/safariproxd/homework/internal/repository/file"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func initLogging(path string) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	log.SetOutput(file)
}

func main() {
	cfg := config.Default()
	initLogging(cfg.LogFile)

	orderRepo, err := file.NewFileOrderRepository(cfg.OrderDataFile)
	if err != nil {
		log.Fatalf("Failed to initialize file order repository: %v", err)
	}
	pvzService, err := app.NewPVZService(orderRepo, cfg.PackageConfigFile)
	if err != nil {
		log.Fatalf("Failed to init PVZ service: %v", err)
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)
	ordersServer := server.NewOrdersServer(pvzService)
	ordersServer.Register(grpcServer)

	log.Println("gRPC server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
