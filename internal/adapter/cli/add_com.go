package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func (a *CLIAdapter) AddComm(cmd *cobra.Command, args []string) error {
	receiverID, err := cmd.Flags().GetUint64("user-id")
	if err != nil {
		return fmt.Errorf("flag.GetUint64: %w", err)
	}
	orderID, err := cmd.Flags().GetUint64("order-id")
	if err != nil {
		return fmt.Errorf("flag.GetUint64: %w", err)
	}
	storageUntilStr, err := cmd.Flags().GetString("expires")
	if err != nil {
		return fmt.Errorf("flag.GetString: %w", err)
	}
	weight, err := cmd.Flags().GetFloat64("weight")
	if err != nil {
		return fmt.Errorf("flag.GetFloat64: %w", err)
	}
	price, err := cmd.Flags().GetFloat64("price")
	if err != nil {
		return fmt.Errorf("flag.GetFloat64: %w", err)
	}
	packageType, err := cmd.Flags().GetString("package")
	if err != nil {
		return fmt.Errorf("flag.GetString: %w", err)
	}

	storageUntil, err := time.Parse("2006-01-02", storageUntilStr)
	if err != nil {
		return fmt.Errorf("time.Parse: %w", err)
	}

	totalPrice, err := a.appService.AcceptOrder(receiverID, orderID, storageUntil, weight, price, packageType)
	if err != nil {
		return err
	}

	fmt.Printf("ORDER_ACCEPTED: %d\n", orderID)
	fmt.Printf("PACKAGE: %s\n", packageType)
	fmt.Printf("TOTAL_PRICE: %.2f\n", totalPrice)
	return nil
}
