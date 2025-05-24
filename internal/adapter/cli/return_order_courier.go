package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func RegisterReturnOrderCmd(rootCmd *cobra.Command, appService OrderService) {
	returnOrderCmd := &cobra.Command{
		Use:   "return-order",
		Short: "Returns an order to the courier.",
		RunE: func(cmd *cobra.Command, args []string) error {
			orderID, err := cmd.Flags().GetUint64("order-id")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetUint64: %w", err))
			}

			err = appService.ReturnOrderToDelivery(orderID)
			if err != nil {
				return MapError(err)
			}
			fmt.Printf("ORDER_RETURNED: %d\n", orderID)
			return nil
		},
	}
	returnOrderCmd.Flags().Uint64P("order-id", "", 0, "ID of the order to return")
	returnOrderCmd.MarkFlagRequired("order-id")
	rootCmd.AddCommand(returnOrderCmd)
}
