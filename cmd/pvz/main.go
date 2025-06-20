package main

import (
	"log"
	"net"
	"os"
	"time"

	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	server "gitlab.ozon.dev/safariproxd/homework/internal/adapter/grpc"
	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/grpc/mw"
	"gitlab.ozon.dev/safariproxd/homework/internal/app"
	"gitlab.ozon.dev/safariproxd/homework/internal/config"
	"gitlab.ozon.dev/safariproxd/homework/internal/repository/postgres"
	"gitlab.ozon.dev/safariproxd/homework/pkg/db"
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
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	initLogging(cfg.Log.File)
	dbCfg := db.Config{
		ReadDSN:  cfg.ReadDSN(),
		WriteDSN: cfg.WriteDSN(),
		MaxOpen:  cfg.DB.Pool.MaxOpen,
		MaxIdle:  cfg.DB.Pool.MaxIdle,
		LogFile:  cfg.Log.File,
	}
	client, err := db.NewClient(dbCfg)
	if err != nil {
		log.Fatalf("db client: %v", err)
	}
	defer client.Close()
	orderRepo := postgres.NewOrderRepository(client)
	pvzService := app.NewPVZService(orderRepo)
	lis, err := net.Listen("tcp", cfg.Service.GRPCAddress)
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
			mw.TimeoutInterceptor(cfg.Service.Timeout),
			mw.LoggingInterceptor(),
			mw.ValidationInterceptor(),
			mw.ErrorMappingInterceptor(),
			mw.RateLimiterInterceptor(limiterInstance),
		),
	)

	reflection.Register(grpcServer)
	ordersServer.Register(grpcServer)

	log.Printf("gRPC server listening on %s", cfg.Service.GRPCAddress)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
