package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
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

	storageUntil, err := MapStringToTime(storageUntilStr)
	if err != nil {
		return fmt.Errorf("time.Parse: %w", err)
	}
	req := domain.AcceptOrderRequest{
		ReceiverID:   receiverID,
		OrderID:      orderID,
		StorageUntil: storageUntil,
		Weight:       weight,
		Price:        price,
		PackageType:  packageType,
	}
	totalPrice, err := a.appService.AcceptOrder(req)
	if err != nil {
		return err
	}

	fmt.Printf("ORDER_ACCEPTED: %d\n", orderID)
	fmt.Printf("PACKAGE: %s\n", packageType)
	fmt.Printf("TOTAL_PRICE: %.2f\n", totalPrice)
	return nil
}
