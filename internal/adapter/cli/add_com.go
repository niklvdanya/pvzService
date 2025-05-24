package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func RegisterAcceptOrderCmd(rootCmd *cobra.Command, appService OrderService) {
	acceptOrderCmd := &cobra.Command{
		Use:   "accept-order",
		Short: "Accepts an order from a courier.",
		RunE: func(cmd *cobra.Command, args []string) error {
			receiverID, err := cmd.Flags().GetUint64("user-id")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetUint64: %w", err))
			}
			orderID, err := cmd.Flags().GetUint64("order-id")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetUint64: %w", err))
			}
			storageUntilStr, err := cmd.Flags().GetString("expires")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetString: %w", err))
			}

			storageUntil, err := time.Parse("2006-01-02", storageUntilStr)
			if err != nil {
				return MapError(fmt.Errorf("invalid date format: %w", err))
			}

			err = appService.AcceptOrder(receiverID, orderID, storageUntil)
			if err != nil {
				return MapError(err)
			}
			fmt.Printf("ORDER_ACCEPTED: %d\n", orderID)
			return nil
		},
	}
	acceptOrderCmd.Flags().Uint64P("order-id", "", 0, "ID of the order")
	acceptOrderCmd.Flags().Uint64P("user-id", "", 0, "ID of the receiver")
	acceptOrderCmd.Flags().StringP("expires", "", "", "Storage expiration date (YYYY-MM-DD)")
	acceptOrderCmd.MarkFlagRequired("order-id")
	acceptOrderCmd.MarkFlagRequired("user-id")
	acceptOrderCmd.MarkFlagRequired("expires")
	rootCmd.AddCommand(acceptOrderCmd)
}
