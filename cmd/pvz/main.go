package main

import (
	"log"
	"net"
	"os"
	"time"

	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	server "gitlab.ozon.dev/safariproxd/homework/internal/adapter/grpc"
	"gitlab.ozon.dev/safariproxd/homework/internal/app"
	"gitlab.ozon.dev/safariproxd/homework/internal/config"
	"gitlab.ozon.dev/safariproxd/homework/internal/repository/file"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// TODO исправить нейминг пакетов и директорий
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
	lis, err := net.Listen("tcp", cfg.GRPCAddress)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	limiterInstance := limiter.New(memory.NewStore(), limiter.Rate{
		Period: time.Second,
		Limit:  5,
	})
	ordersServer := server.NewOrdersServer(pvzService)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			server.LoggingInterceptor(),
			server.ValidationInterceptor(),
			server.ErrorMappingInterceptor(),
			server.RateLimiterInterceptor(limiterInstance),
		),
	)

	reflection.Register(grpcServer)
	ordersServer.Register(grpcServer)

	log.Printf("gRPC server listening on %s", cfg.GRPCAddress)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
