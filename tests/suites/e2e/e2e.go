package e2e

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	server "gitlab.ozon.dev/safariproxd/homework/internal/adapter/grpc"
	mw "gitlab.ozon.dev/safariproxd/homework/internal/adapter/grpc/mw"
	apppkg "gitlab.ozon.dev/safariproxd/homework/internal/app"
	repopg "gitlab.ozon.dev/safariproxd/homework/internal/repository/postgres"
	api "gitlab.ozon.dev/safariproxd/homework/pkg/api"
	dbpkg "gitlab.ozon.dev/safariproxd/homework/pkg/db"

	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

const (
	postgresImage = "postgres:16-alpine"
	postgresUser  = "user"
	postgresPass  = "password"
	postgresDB    = "testdb"
)

type testEnv struct {
	ctx          context.Context
	container    testcontainers.Container
	dbClient     *dbpkg.Client
	grpcConn     *grpc.ClientConn
	ordersClient api.OrdersServiceClient
	stopServer   func()
}

func (e *testEnv) Close() {
	if e.grpcConn != nil {
		_ = e.grpcConn.Close()
	}
	if e.stopServer != nil {
		e.stopServer()
	}
	if e.dbClient != nil {
		_ = e.dbClient.Close()
	}
	if e.container != nil {
		_ = e.container.Terminate(e.ctx)
	}
}

func setupEnv(t *testing.T) *testEnv {
	t.Helper()

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        postgresImage,
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     postgresUser,
			"POSTGRES_PASSWORD": postgresPass,
			"POSTGRES_DB":       postgresDB,
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPass, host, port.Port(), postgresDB)

	sqlDB, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return sqlDB.PingContext(ctx) == nil }, 15*time.Second, time.Second)

	goose.SetDialect("postgres")
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	migrationsDir := filepath.Join(repoRoot, "migrations")
	require.NoError(t, goose.Up(sqlDB, migrationsDir))

	dbCfg := dbpkg.Config{ReadDSN: dsn, WriteDSN: dsn, MaxOpen: 5, MaxIdle: 5}
	dbClient, err := dbpkg.NewClient(dbCfg)
	require.NoError(t, err)

	repo := repopg.NewOrderRepository(dbClient)
	svc := apppkg.NewPVZService(repo, time.Now)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	limiterInstance := limiter.New(memory.NewStore(), limiter.Rate{Period: time.Second, Limit: 5})
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			mw.TimeoutInterceptor(2*time.Second),
			mw.LoggingInterceptor(),
			mw.ValidationInterceptor(),
			mw.ErrorMappingInterceptor(),
			mw.RateLimiterInterceptor(limiterInstance),
		),
	)
	server.NewOrdersServer(svc).Register(grpcServer)

	go func() { _ = grpcServer.Serve(lis) }()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	env := &testEnv{
		ctx:          ctx,
		container:    container,
		dbClient:     dbClient,
		grpcConn:     conn,
		ordersClient: api.NewOrdersServiceClient(conn),
		stopServer:   grpcServer.Stop,
	}

	t.Cleanup(env.Close)
	return env
}

type OrdersE2ESuite struct {
	suite.Suite
	env *testEnv
}

func (s *OrdersE2ESuite) SetupTest() {
	s.env = setupEnv(s.T())
}

func (s *OrdersE2ESuite) TearDownTest() {
	if s.env != nil {
		s.env.Close()
	}
}

func (s *OrdersE2ESuite) TestAcceptAndListOrders() {
	orderID := uint64(1)
	userID := uint64(42)

	pkgType := api.PackageType_PACKAGE_TYPE_BOX

	_, err := s.env.ordersClient.AcceptOrder(s.env.ctx, &api.AcceptOrderRequest{
		OrderId:   orderID,
		UserId:    userID,
		ExpiresAt: timestamppb.New(time.Now().Add(24 * time.Hour)),
		Package:   &pkgType,
		Weight:    2.5,
		Price:     100,
	})
	s.Require().NoError(err)

	listResp, err := s.env.ordersClient.ListOrders(s.env.ctx, &api.ListOrdersRequest{
		UserId:     userID,
		InPvz:      true,
		Pagination: &api.Pagination{Page: 1, CountOnPage: 10},
	})
	s.Require().NoError(err)
	s.Require().Len(listResp.Orders, 1)

	got := listResp.Orders[0]
	s.Equal(orderID, got.OrderId)
	s.Equal(api.OrderStatus_ORDER_STATUS_EXPECTS, got.Status)
}

func (s *OrdersE2ESuite) TestAcceptAndIssueOrders() {
	orderID := uint64(2)
	userID := uint64(99)

	pkgType := api.PackageType_PACKAGE_TYPE_BOX

	_, err := s.env.ordersClient.AcceptOrder(s.env.ctx, &api.AcceptOrderRequest{
		OrderId:   orderID,
		UserId:    userID,
		ExpiresAt: timestamppb.New(time.Now().Add(24 * time.Hour)),
		Package:   &pkgType,
		Weight:    1.1,
		Price:     55.5,
	})
	s.Require().NoError(err)

	procResp, err := s.env.ordersClient.ProcessOrders(s.env.ctx, &api.ProcessOrdersRequest{
		UserId:   userID,
		Action:   api.ActionType_ACTION_TYPE_ISSUE,
		OrderIds: []uint64{orderID},
	})
	s.Require().NoError(err)
	s.ElementsMatch([]uint64{orderID}, procResp.Processed)
	s.Empty(procResp.Errors)

	listResp, err := s.env.ordersClient.ListOrders(s.env.ctx, &api.ListOrdersRequest{
		UserId:     userID,
		Pagination: &api.Pagination{Page: 1, CountOnPage: 10},
	})
	s.Require().NoError(err)
	s.Require().Len(listResp.Orders, 1)
	s.Equal(api.OrderStatus_ORDER_STATUS_ACCEPTED, listResp.Orders[0].Status)
}
