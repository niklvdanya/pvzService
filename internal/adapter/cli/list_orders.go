package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func RegisterListOrdersCmd(rootCmd *cobra.Command, appService OrderService) {
	listOrdersCmd := &cobra.Command{
		Use:   "list-orders",
		Short: "Lists orders for a specific receiver.",
		RunE: func(cmd *cobra.Command, args []string) error {
			receiverID, err := cmd.Flags().GetUint64("user-id")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetUint64: %w", err))
			}
			inPvz, err := cmd.Flags().GetBool("in-pvz")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetBool: %w", err))
			}
			lastN, err := cmd.Flags().GetUint64("last")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetUint64: %w", err))
			}
			page, err := cmd.Flags().GetUint64("page")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetUint64: %w", err))
			}
			limit, err := cmd.Flags().GetUint64("limit")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetUint64: %w", err))
			}

			if lastN > 0 && (page > 0 || limit > 0) {
				return MapError(fmt.Errorf("invalid flags combination"))
			}

			if page == 0 {
				page = 1
			}
			if limit == 0 {
				limit = 10
			}

			orders, totalItems, err := appService.GetReceiverOrders(receiverID, inPvz, lastN, page, limit)
			if err != nil {
				return MapError(err)
			}

			if len(orders) == 0 {
				fmt.Println("No orders found for this receiver with the given criteria.")
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
	listOrdersCmd.Flags().Uint64P("user-id", "", 0, "ID of the receiver")
	listOrdersCmd.Flags().BoolP("in-pvz", "", false, "Filter for orders currently in PVZ storage")
	listOrdersCmd.Flags().Uint64P("last", "", 0, "Show last N orders")
	listOrdersCmd.Flags().Uint64P("page", "", 0, "Page number for pagination")
	listOrdersCmd.Flags().Uint64P("limit", "", 0, "Items per page for pagination")
	listOrdersCmd.MarkFlagRequired("user-id")
	rootCmd.AddCommand(listOrdersCmd)
}
