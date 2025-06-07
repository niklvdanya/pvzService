package config

type Config struct {
	OrderDataFile     string
	OrdersOutputFile  string
	LogFile           string
	PackageConfigFile string
	GRPCAddress       string
	HTTPAddress       string
	SwaggerAddress    string
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
	}
}
