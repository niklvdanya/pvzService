package cli

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

type OrderService interface {
	AcceptOrder(receiverID, orderID uint64, storageUntil time.Time) error
	ReturnOrderToDelivery(orderID uint64) error
	IssueOrdersToClient(receiverID uint64, orderIDs []uint64) error
	ReturnOrdersFromClient(receiverID uint64, orderIDs []uint64) error
	GetReceiverOrders(receiverID uint64, inPVZ bool, lastN, page, limit uint64) ([]*domain.Order, uint64, error)
	GetReceiverOrdersScroll(receiverID uint64, lastID, limit uint64) ([]*domain.Order, uint64, error)
	GetReturnedOrders(page, limit uint64) ([]*domain.Order, uint64, error)
	GetOrderHistory() ([]*domain.Order, error)
	ImportOrders(orders []struct {
		OrderID      uint64 `json:"order_id"`
		ReceiverID   uint64 `json:"receiver_id"`
		StorageUntil string `json:"storage_until"`
	}) (uint64, error)
}

type CLIAdapter struct {
	appService OrderService
	debug      bool
}

func NewCLIAdapter(appService OrderService, rootCmd *cobra.Command, debugMode bool) *CLIAdapter {
	a := &CLIAdapter{
		appService: appService,
		debug:      debugMode,
	}
	a.registerCommands(rootCmd)
	return a
}

func (a *CLIAdapter) Run(rootCmd *cobra.Command) error {
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
			fmt.Fprintf(os.Stderr, "%s\n", mapError(err))
			if a.debug {
				fmt.Fprintf(os.Stderr, "DEBUG: %v\n", err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading from stdin: %w", err)
	}
	return nil
}
