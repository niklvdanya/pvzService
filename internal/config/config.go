package config

import (
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/pkg/errors"
	"gitlab.ozon.dev/safariproxd/homework/internal/infra/telegram"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Service struct {
		GRPCAddress    string        `yaml:"grpc_address"`
		HTTPAddress    string        `yaml:"http_address"`
		SwaggerAddress string        `yaml:"swagger_address"`
		AdminAddress   string        `yaml:"admin_address"`
		Timeout        time.Duration `yaml:"timeout"`
		WorkerLimit    int           `yaml:"worker_limit"`
		QueueSize      int           `yaml:"queue_size"`
	} `yaml:"service"`

	DB struct {
		ReadHost  string `yaml:"read_host" env:"POSTGRES_READ_HOST"`
		WriteHost string `yaml:"write_host" env:"POSTGRES_WRITE_HOST"`
		Port      int    `yaml:"port" env:"POSTGRES_PORT"`
		Name      string `yaml:"name" env:"POSTGRES_DB"`
		User      string `yaml:"-" env:"POSTGRES_USER"`
		Pass      string `yaml:"-" env:"POSTGRES_PASSWORD"`
		SSL       string `yaml:"sslmode"`
		Pool      struct {
			MaxOpen int `yaml:"max_open"`
			MaxIdle int `yaml:"max_idle"`
		} `yaml:"pool"`
		MigrationsDir string `yaml:"migrations_dir"`
	} `yaml:"db"`

	Kafka struct {
		Brokers  []string `yaml:"brokers"`
		Topic    string   `yaml:"topic"`
		Producer struct {
			Timeout time.Duration `yaml:"timeout"`
			Retries int           `yaml:"retries"`
		} `yaml:"producer"`
	} `yaml:"kafka"`

	Outbox struct {
		WorkerInterval time.Duration `yaml:"worker_interval"`
		BatchSize      int           `yaml:"batch_size"`
	} `yaml:"outbox"`
	Telegram telegram.TelegramConfig `yaml:"telegram"`
}

func (c *Config) ReadDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DB.User, c.DB.Pass, c.DB.ReadHost, c.DB.Port, c.DB.Name, c.DB.SSL)
}

func (c *Config) WriteDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DB.User, c.DB.Pass, c.DB.WriteHost, c.DB.Port, c.DB.Name, c.DB.SSL)
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "read yaml")
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.Wrap(err, "parse yaml")
	}

	if err := env.Parse(&cfg); err != nil {
		return nil, errors.Wrap(err, "parse env")
	}
	return &cfg, nil
}
