package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"gitlab.ozon.dev/safariproxd/homework/docs/homework-1/internal/adapter/cli"
	"gitlab.ozon.dev/safariproxd/homework/docs/homework-1/internal/adapter/inmemory"
	"gitlab.ozon.dev/safariproxd/homework/docs/homework-1/internal/app"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Short: "PVZ (Pickup Point) command-line interface",
	Long: `A simple command-line interface for managing orders in a pickup point.
    Type 'help' for a list of commands, or 'help <command>' for specific command usage.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func main() {
	orderRepo := inmemory.NewInMemoryOrderRepository()
	returnedOrderRepo := inmemory.NewInMemoryReturnedOrderRepository()

	pvzService := app.NewPVZService(orderRepo, returnedOrderRepo)

	cliAdapter := cli.NewCLIAdapter(pvzService)

	cliAdapter.RegisterCommands(rootCmd)

	fmt.Println("Welcome to PVZ system.")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("pvz>")

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
			fmt.Fprintf(os.Stderr, "Command Error: %v\n", err)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading from stdin: %s", err)
	}
}
