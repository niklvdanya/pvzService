package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

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
	cliAdapter := cli.NewCLIAdapter(pvzService)

	cliAdapter.RegisterCommands(rootCmd)

	fmt.Println("Welcome to PVZ system.")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("pvz> ")

		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if line == "exit" {
			fmt.Println("Exiting PVZ system.")
			os.Exit(0)
		}

		rootCmd.SetArgs(strings.Fields(line))
		if err := rootCmd.Execute(); err != nil {
			log.Printf("Command execution error: %v", err)
			fmt.Fprintf(os.Stderr, "%s\n", cli.MapError(err))
			if debug {
				fmt.Fprintf(os.Stderr, "DEBUG: %v\n", err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading from stdin: %v", err)
	}
}
