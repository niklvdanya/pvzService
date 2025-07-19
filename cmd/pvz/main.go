package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	server "gitlab.ozon.dev/safariproxd/homework/internal/adapter/grpc"
	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/grpc/mw"
	"gitlab.ozon.dev/safariproxd/homework/internal/app"
	"gitlab.ozon.dev/safariproxd/homework/internal/config"
	"gitlab.ozon.dev/safariproxd/homework/internal/infra"
	"gitlab.ozon.dev/safariproxd/homework/internal/repository/postgres"
	"gitlab.ozon.dev/safariproxd/homework/internal/workerpool"
	"gitlab.ozon.dev/safariproxd/homework/pkg/cache"
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

	var orderRepo app.OrderRepository
	var outboxRepo app.OutboxRepository
	var cacheManager infra.CacheManager

	if cfg.Cache.Enabled {
		cacheConfig := cache.Config{
			MaxSize: cfg.Cache.MaxSize,
			TTL:     cfg.Cache.TTL,
		}

		cachedRepo := postgres.NewCachedOrderRepository(client, cacheConfig)
		orderRepo = cachedRepo
		cacheManager = cachedRepo

		go func() {
			ticker := time.NewTicker(cfg.Cache.CleanupInterval)
			defer ticker.Stop()

			for range ticker.C {
				cachedRepo.CleanupExpired()
				slog.Debug("Cache cleanup completed", "stats", cachedRepo.GetCacheStats())
			}
		}()

		slog.Info("Cache enabled",
			"max_size", cfg.Cache.MaxSize,
			"ttl", cfg.Cache.TTL,
			"cleanup_interval", cfg.Cache.CleanupInterval)
	} else {
		orderRepo = postgres.NewOrderRepository(client)
		slog.Info("Cache disabled")
	}

	outboxRepo = postgres.NewOutboxRepository(client)
	pvzService := app.NewPVZService(orderRepo, outboxRepo, client, time.Now, cfg.Service.WorkerLimit)

	pool := workerpool.New(cfg.Service.WorkerLimit, cfg.Service.QueueSize)

	limiterInstance := limiter.New(memory.NewStore(), limiter.Rate{Period: cfg.Service.Timeout, Limit: 5})

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			mw.RateLimiterInterceptor(limiterInstance),
			mw.TimeoutInterceptor(2*time.Second),
			mw.LoggingInterceptor(),
			mw.ValidationInterceptor(),
			mw.ErrorMappingInterceptor(),
			mw.PoolInterceptor(pool),
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

	admin := infra.NewAdmin(cfg.Service.AdminAddress, pool, cacheManager)
	admin.Start()
	slog.Info("admin HTTP listening", "addr", cfg.Service.AdminAddress)

	// curl -XPOST 'http://localhost:6060/resize?workers=11'
	// curl -XGET 'http://localhost:6060/cache/stats'
	// curl -XPOST 'http://localhost:6060/cache/clear'
	// curl -XPOST 'http://localhost:6060/cache/cleanup'

	infra.Graceful(
		func(ctx context.Context) { grpcServer.GracefulStop() },
		admin.Shutdown,
		func(ctx context.Context) { pool.Close() },
	)
}
