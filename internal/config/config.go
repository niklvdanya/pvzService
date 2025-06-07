package config

import "time"

type Config struct {
	OrderDataFile     string
	OrdersOutputFile  string
	LogFile           string
	PackageConfigFile string
	GRPCAddress       string
	HTTPAddress       string
	SwaggerAddress    string
	ServiceTimeout    time.Duration
}

func Default() *Config {
	return &Config{
		OrderDataFile:     "data/orders.json",
		OrdersOutputFile:  "data/new_orders.json",
		LogFile:           "pvz.log",
		PackageConfigFile: "internal/config/packages.json",
		GRPCAddress:       ":50051",
		HTTPAddress:       ":8081",
		SwaggerAddress:    ":8082",
		ServiceTimeout:    2 * time.Second,
	}
}
