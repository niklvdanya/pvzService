package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func RegisterScrollOrdersCmd(rootCmd *cobra.Command, appService OrderService) {
	scrollOrdersCmd := &cobra.Command{
		Use:   "scroll-orders",
		Short: "Infinite orders scroll.",
		RunE: func(cmd *cobra.Command, args []string) error {
			receiverID, err := cmd.Flags().GetUint64("user-id")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetUint64: %w", err))
			}
			limit, err := cmd.Flags().GetUint64("limit")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetUint64: %w", err))
			}

			if limit == 0 {
				limit = 20
			}

			var currentLastID uint64
			scanner := bufio.NewScanner(os.Stdin)
			orders, nextLastID, err := appService.GetReceiverOrdersScroll(receiverID, currentLastID, limit)
			if err != nil {
				return MapError(err)
			}
			printScrollOrders(orders, nextLastID)
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
					orders, nextLastID, err = appService.GetReceiverOrdersScroll(receiverID, currentLastID, limit)
					if err != nil {
						return MapError(err)
					}
					printScrollOrders(orders, nextLastID)
					currentLastID = nextLastID
				case "exit":
					fmt.Println("Exiting scroll-orders.")
					return nil
				default:
					fmt.Println("Unknown command. Type 'next' to get more orders or 'exit' to quit.")
				}
			}

			if err := scanner.Err(); err != nil {
				return MapError(fmt.Errorf("scanner.Scan: %w", err))
			}
			return nil
		},
	}
	scrollOrdersCmd.Flags().Uint64P("user-id", "", 0, "ID of the receiver")
	scrollOrdersCmd.Flags().Uint64P("limit", "", 20, "Number of orders to fetch at once")
	scrollOrdersCmd.MarkFlagRequired("user-id")
	rootCmd.AddCommand(scrollOrdersCmd)
}

func printScrollOrders(orders []*domain.Order, nextLastID uint64) {
	if len(orders) == 0 {
		fmt.Println("No orders found in this batch.")
	} else {
		for _, order := range orders {
			fmt.Printf("ORDER: %d Receiver: %d Status: %s Storage Limit: %s\n",
				order.OrderID,
				order.ReceiverID,
				order.GetStatusString(),
				order.StorageUntil.Format("2006-01-02"),
			)
		}
	}
	if nextLastID > 0 {
		fmt.Printf("NEXT: %d\n", nextLastID)
	} else {
		fmt.Println("NEXT: 0 (End of orders)")
	}
}
