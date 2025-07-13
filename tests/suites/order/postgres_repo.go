package postgres_repo

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/lib/pq"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/internal/repository/postgres"
	dbpkg "gitlab.ozon.dev/safariproxd/homework/pkg/db"
)

// надо вынести в конфиг наверное
const (
	postgresImage = "postgres:16-alpine"
	postgresUser  = "user"
	postgresPass  = "password"
	postgresDB    = "testdb"
)

type OrderRepositorySuite struct {
	suite.Suite
	ctx                context.Context
	pgContainer        testcontainers.Container
	dsn                string
	sqlDB              *sql.DB
	dbClient           *dbpkg.Client
	orderRepo          *postgres.OrderRepository
	migrationsLocation string
}

func (s *OrderRepositorySuite) SetupSuite() {
	s.ctx = context.Background()

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
	container, err := testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(s.T(), err)
	s.pgContainer = container

	host, err := container.Host(s.ctx)
	require.NoError(s.T(), err)
	port, err := container.MappedPort(s.ctx, "5432/tcp")
	require.NoError(s.T(), err)
	s.dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPass, host, port.Port(), postgresDB)

	sqlDB, err := sql.Open("postgres", s.dsn)
	require.NoError(s.T(), err)
	s.sqlDB = sqlDB
	require.Eventually(s.T(), func() bool { return sqlDB.PingContext(s.ctx) == nil }, 10*time.Second, time.Second)

	_ = goose.SetDialect("postgres")
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(s.T(), err)

	s.migrationsLocation = filepath.Join(root, "migrations")
	require.NoError(s.T(), goose.Up(sqlDB, s.migrationsLocation))

	cfg := dbpkg.Config{
		ReadDSN:  s.dsn,
		WriteDSN: s.dsn,
		MaxOpen:  5,
		MaxIdle:  5,
	}
	s.dbClient, err = dbpkg.NewClient(cfg)
	require.NoError(s.T(), err)

	s.orderRepo = postgres.NewOrderRepository(s.dbClient)
}

func (s *OrderRepositorySuite) TearDownSuite() {
	if s.dbClient != nil {
		_ = s.dbClient.Close()
	}
	if s.sqlDB != nil {
		_ = s.sqlDB.Close()
	}
	if s.pgContainer != nil {
		_ = s.pgContainer.Terminate(s.ctx)
	}
}

func makeTestOrder(id uint64) domain.Order {
	now := time.Now().UTC().Truncate(time.Second)

	return domain.Order{
		OrderID:        id,
		ReceiverID:     99,
		StorageUntil:   now.Add(48 * time.Hour),
		Status:         domain.StatusInStorage,
		AcceptTime:     now,
		LastUpdateTime: now,
		PackageType:    "box",
		Weight:         1.23,
		Price:          456.78,
	}
}

func (s *OrderRepositorySuite) Test_Save_And_GetByID() {
	ctx := s.ctx
	order := makeTestOrder(1)

	require.NoError(s.T(), s.orderRepo.Save(ctx, order))

	got, err := s.orderRepo.GetByID(ctx, order.OrderID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), order.OrderID, got.OrderID)
	assert.Equal(s.T(), order.Price, got.Price)
}

func (s *OrderRepositorySuite) Test_Exists() {
	ctx := s.ctx
	order := makeTestOrder(2)

	ok, err := s.orderRepo.Exists(ctx, order.OrderID)
	require.NoError(s.T(), err)
	assert.False(s.T(), ok)

	require.NoError(s.T(), s.orderRepo.Save(ctx, order))
	ok, err = s.orderRepo.Exists(ctx, order.OrderID)
	require.NoError(s.T(), err)
	assert.True(s.T(), ok)
}

func (s *OrderRepositorySuite) Test_Update() {
	ctx := s.ctx
	order := makeTestOrder(3)
	require.NoError(s.T(), s.orderRepo.Save(ctx, order))

	order.Status = domain.StatusGivenToCourier
	order.Price = 999.99
	order.LastUpdateTime = time.Now().UTC()

	require.NoError(s.T(), s.orderRepo.Update(ctx, order))

	got, err := s.orderRepo.GetByID(ctx, order.OrderID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), domain.StatusGivenToCourier, got.Status)
	assert.EqualValues(s.T(), 999.99, got.Price)
}

func (s *OrderRepositorySuite) Test_SaveHistory() {
	ctx := s.ctx
	order := makeTestOrder(4)
	require.NoError(s.T(), s.orderRepo.Save(ctx, order))

	h := domain.OrderHistory{
		OrderID:   order.OrderID,
		Status:    domain.StatusReturnedFromClient,
		ChangedAt: time.Now().UTC(),
	}
	require.NoError(s.T(), s.orderRepo.SaveHistory(ctx, h))

	history, err := s.orderRepo.GetHistoryByOrderID(ctx, order.OrderID)
	require.NoError(s.T(), err)
	require.Len(s.T(), history, 1)
	assert.Equal(s.T(), h.Status, history[0].Status)
}
