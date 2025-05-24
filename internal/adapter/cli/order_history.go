package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func RegisterOrderHistoryCmd(rootCmd *cobra.Command, appService OrderService) {
	orderHistoryCmd := &cobra.Command{
		Use:   "order-history",
		Short: "Shows the history of all order status changes (sorted by last update time).",
		RunE: func(cmd *cobra.Command, args []string) error {
			allOrders, err := appService.GetOrderHistory()
			if err != nil {
				return MapError(err)
			}

			if len(allOrders) == 0 {
				fmt.Println("No orders in the system.")
				return nil
			}

			for _, order := range allOrders {
				fmt.Printf("HISTORY: %d %s %s\n",
					order.OrderID,
					order.GetStatusString(),
					order.LastUpdateTime.Format("2006-01-02"),
				)
			}
			return nil
		},
	}
	rootCmd.AddCommand(orderHistoryCmd)
}
