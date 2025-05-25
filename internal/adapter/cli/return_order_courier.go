package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (a *CLIAdapter) BackOrder(cmd *cobra.Command, args []string) error {
	orderID, err := cmd.Flags().GetUint64("order-id")
	if err != nil {
		return fmt.Errorf("flag.GetUint64: %w", err)
	}

	err = a.appService.ReturnOrderToDelivery(orderID)
	if err != nil {
		return err
	}
	fmt.Printf("ORDER_RETURNED: %d\n", orderID)
	return nil
}
