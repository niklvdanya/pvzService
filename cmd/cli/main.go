package main

import (
	"log"
	"os"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/app"
	"gitlab.ozon.dev/safariproxd/homework/internal/config"
	"gitlab.ozon.dev/safariproxd/homework/internal/repository/file"

	"github.com/spf13/cobra"
)

var (
	debug   bool
	rootCmd = &cobra.Command{
		Short: "PVZ (Pickup Point) command-line interface",
		Long:  `A simple command-line interface for managing orders in a pickup point.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				log.Printf("Failed to show help: %v", err)
			}
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

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
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug mode")

	orderRepo, err := file.NewFileOrderRepository(cfg.OrderDataFile)
	if err != nil {
		log.Fatalf("Failed to initialize file order repository: %v", err)
	}
	pvzService := app.NewPVZService(orderRepo)

	cliAdapter := cli.NewCLIAdapter(pvzService, rootCmd, debug)

	if err := cliAdapter.Run(rootCmd); err != nil {
		log.Fatalf("CLI exited with error: %v", err)
	}
}
