package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (a *CLIAdapter) ScrollOrdersComm(cmd *cobra.Command, args []string) error {
	receiverID, err := cmd.Flags().GetUint64("user-id")
	if err != nil {
		return fmt.Errorf("flag.GetUint64: %w", err)
	}
	limit, err := cmd.Flags().GetUint64("limit")
	if err != nil {
		return fmt.Errorf("flag.GetUint64: %w", err)
	}

	if limit == 0 {
		limit = 20
	}

	var currentLastID uint64
	scanner := bufio.NewScanner(os.Stdin)
	orders, nextLastID, err := a.appService.GetReceiverOrdersScroll(receiverID, currentLastID, limit)
	if err != nil {
		return err
	}
	a.printScrollOrders(orders, nextLastID)
	currentLastID = nextLastID

	for {
		if currentLastID == 0 && len(orders) == 0 {
			fmt.Println("No more orders to display.")
			break
		}

		fmt.Print("> ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "next":
			if currentLastID == 0 {
				fmt.Println("No more orders to display.")
				continue
			}
			orders, nextLastID, err = a.appService.GetReceiverOrdersScroll(receiverID, currentLastID, limit)
			if err != nil {
				return err
			}
			a.printScrollOrders(orders, nextLastID)
			currentLastID = nextLastID
		case "exit":
			fmt.Println("Exiting scroll-orders.")
			return nil
		default:
			fmt.Println("Unknown command. Type 'next' to get more orders or 'exit' to quit.")
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner.Scan: %w", err)
	}
	return nil
}

func (a *CLIAdapter) printScrollOrders(orders []*domain.Order, nextLastID uint64) {
	if len(orders) == 0 {
		fmt.Println("No orders found in this batch.")
	} else {
		for _, order := range orders {
			packageType := order.PackageType
			if packageType == "" {
				packageType = "none"
			}
			fmt.Printf("ORDER: %d Receiver: %d Status: %s Storage Limit: %s Package: %s Weight: %.2f Price: %.2f\n",
				order.OrderID,
				order.ReceiverID,
				order.GetStatusString(),
				MapTimeToString(order.StorageUntil),
				packageType,
				order.Weight,
				order.Price,
			)
		}
	}
	if nextLastID > 0 {
		fmt.Printf("NEXT: %d\n", nextLastID)
	} else {
		fmt.Println("NEXT: 0 (End of orders)")
	}
}
