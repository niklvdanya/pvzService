package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/internal/adapter/file"
	"gitlab.ozon.dev/safariproxd/homework/internal/app"

	"github.com/spf13/cobra"
)

var (
	debug   bool
	rootCmd = &cobra.Command{
		Short:         "PVZ (Pickup Point) command-line interface",
		Long:          `A simple command-line interface for managing orders in a pickup point.`,
		Run:           func(cmd *cobra.Command, args []string) { cmd.Help() },
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

func initLogging() {
	file, err := os.OpenFile("pvz.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	log.SetOutput(file)
}

func main() {
	initLogging()
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug mode")

	orderRepo, err := file.NewFileOrderRepository()
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
			fmt.Fprintf(os.Stderr, "%s\n", err)
			if debug {
				fmt.Fprintf(os.Stderr, "DEBUG: %v\n", err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading from stdin: %v", err)
	}
}
