package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (a *CLIAdapter) ListOrdersComm(cmd *cobra.Command, args []string) error {
	receiverID, err := cmd.Flags().GetUint64("user-id")
	if err != nil {
		return fmt.Errorf("flag.GetUint64: %w", err)
	}
	inPvz, err := cmd.Flags().GetBool("in-pvz")
	if err != nil {
		return fmt.Errorf("flag.GetBool: %w", err)
	}
	lastN, err := cmd.Flags().GetUint64("last")
	if err != nil {
		return fmt.Errorf("flag.GetUint64: %w", err)
	}
	page, err := cmd.Flags().GetUint64("page")
	if err != nil {
		return fmt.Errorf("flag.GetUint64: %w", err)
	}
	limit, err := cmd.Flags().GetUint64("limit")
	if err != nil {
		return fmt.Errorf("flag.GetUint64: %w", err)
	}

	if lastN > 0 && (page > 0 || limit > 0) {
		return fmt.Errorf("invalid flags combination")
	}

	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 10
	}

	orders, totalItems, err := a.appService.GetReceiverOrders(receiverID, inPvz, lastN, page, limit)
	if err != nil {
		return err
	}

	if len(orders) == 0 {
		fmt.Println("No orders found for this receiver with the given criteria.")
	} else {
		for _, order := range orders {
			fmt.Printf("Order: %d Receiver: %d Status: %s Storage Limit: %s Package: %s Weight: %.2f Price: %.2f\n",
				order.OrderID,
				order.ReceiverID,
				order.GetStatusString(),
				MapTimeToString(order.StorageUntil),
				MapPackageType(order.PackageType),
				order.Weight,
				order.Price,
			)
		}
	}
	fmt.Printf("TOTAL: %d\n", totalItems)
	return nil
}
