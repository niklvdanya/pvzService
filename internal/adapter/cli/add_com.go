package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func (a *CLIAdapter) AddComm(cmd *cobra.Command, args []string) error {
	receiverID, err := cmd.Flags().GetUint64("user-id")
	if err != nil {
		return mapError(fmt.Errorf("flag.GetUint64: %w", err))
	}
	orderID, err := cmd.Flags().GetUint64("order-id")
	if err != nil {
		return mapError(fmt.Errorf("flag.GetUint64: %w", err))
	}
	storageUntilStr, err := cmd.Flags().GetString("expires")
	if err != nil {
		return mapError(fmt.Errorf("flag.GetString: %w", err))
	}

	storageUntil, err := time.Parse("2006-01-02", storageUntilStr)
	if err != nil {
		return mapError(fmt.Errorf("time.Parse: %w", err))
	}

	err = a.appService.AcceptOrder(receiverID, orderID, storageUntil)
	if err != nil {
		return mapError(err)
	}
	fmt.Printf("ORDER_ACCEPTED: %d\n", orderID)
	return nil
}
