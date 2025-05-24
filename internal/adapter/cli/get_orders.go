package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func RegisterGetReturnedOrdersCmd(rootCmd *cobra.Command, appService OrderService) {
	getReturnedOrdersCmd := &cobra.Command{
		Use:   "list-orders --returned",
		Short: "Lists returned orders with pagination.",
		RunE: func(cmd *cobra.Command, args []string) error {
			page, err := cmd.Flags().GetUint64("page")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetUint64: %w", err))
			}
			limit, err := cmd.Flags().GetUint64("limit")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetUint64: %w", err))
			}

			if page == 0 {
				page = 1
			}
			if limit == 0 {
				limit = 10
			}

			orders, totalItems, err := appService.GetReturnedOrders(page, limit)
			if err != nil {
				return MapError(err)
			}

			if len(orders) == 0 {
				fmt.Println("No returned orders found.")
			} else {
				for _, order := range orders {
					fmt.Printf("Order: %d Receiver: %d Status: %s Storage Limit: %s\n",
						order.OrderID,
						order.ReceiverID,
						order.GetStatusString(),
						order.StorageUntil.Format("2006-01-02"),
					)
				}
			}
			fmt.Printf("TOTAL: %d\n", totalItems)
			return nil
		},
	}
	getReturnedOrdersCmd.Flags().Uint64P("page", "", 0, "Page number for pagination")
	getReturnedOrdersCmd.Flags().Uint64P("limit", "", 0, "Items per page for pagination")
	getReturnedOrdersCmd.Flags().BoolP("returned", "", true, "Filter for returned orders")
	rootCmd.AddCommand(getReturnedOrdersCmd)
}
