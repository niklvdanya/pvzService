// cmd/pvz/main.go
package main

import (
	"context"
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
	"gitlab.ozon.dev/safariproxd/homework/internal/infra"
	"gitlab.ozon.dev/safariproxd/homework/internal/repository/postgres"
	"gitlab.ozon.dev/safariproxd/homework/internal/workerpool"
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

	pool := workerpool.New(cfg.Service.WorkerLimit, cfg.Service.QueueSize)

	limiterInstance := limiter.New(memory.NewStore(), limiter.Rate{Period: cfg.Service.Timeout, Limit: 5})

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			mw.PoolInterceptor(pool),
			mw.TimeoutInterceptor(cfg.Service.Timeout),
			mw.LoggingInterceptor(),
			mw.ValidationInterceptor(),
			mw.ErrorMappingInterceptor(),
			mw.RateLimiterInterceptor(limiterInstance),
		),
	)

	ordersServer := server.NewOrdersServer(pvzService)
	reflection.Register(grpcServer)
	ordersServer.Register(grpcServer)

	lis, err := net.Listen("tcp", cfg.Service.GRPCAddress)
	if err != nil {
		slog.Error("failed to listen", "error", err)
		os.Exit(1)
	}
	go func() {
		slog.Info("gRPC listening", "addr", cfg.Service.GRPCAddress)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC serve error", "error", err)
			os.Exit(1)
		}
	}()

	admin := infra.NewAdmin(cfg.Service.AdminAddress, pool)
	admin.Start()
	slog.Info("admin HTTP listening", "addr", cfg.Service.AdminAddress)
	// curl -XPOST 'http://localhost:6060/resize?workers=11'
	// реализовал через http запросы, возможно надо было добавлять grpc ручку
	infra.Graceful(
		func(ctx context.Context) { grpcServer.GracefulStop() },
		admin.Shutdown,
		func(ctx context.Context) { pool.Close() },
	)
}
