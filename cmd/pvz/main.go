package main

import (
	"log/slog"
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

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		slog.Error("Config load failed", "error", err)
		os.Exit(1)
	}

	dbCfg := db.Config{
		ReadDSN:  cfg.ReadDSN(),
		WriteDSN: cfg.WriteDSN(),
		MaxOpen:  cfg.DB.Pool.MaxOpen,
		MaxIdle:  cfg.DB.Pool.MaxIdle,
	}
	client, err := db.NewClient(dbCfg)
	if err != nil {
		slog.Error("DB client creation failed", "error", err)
		os.Exit(1)
	}
	defer client.Close()
	orderRepo := postgres.NewOrderRepository(client)
	pvzService := app.NewPVZService(orderRepo, time.Now)
	lis, err := net.Listen("tcp", cfg.Service.GRPCAddress)
	if err != nil {
		slog.Error("Failed to listen", "error", err)
		os.Exit(1)
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

	slog.Info("gRPC server listening on", "address", cfg.Service.GRPCAddress)
	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("Failed to serve", "error", err)
		os.Exit(1)
	}
}
