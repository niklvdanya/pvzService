package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (a *CLIAdapter) GetReturnedOrders(cmd *cobra.Command, args []string) error {
	page, err := cmd.Flags().GetUint64("page")
	if err != nil {
		return fmt.Errorf("flag.GetUint64: %w", err)
	}
	limit, err := cmd.Flags().GetUint64("limit")
	if err != nil {
		return fmt.Errorf("flag.GetUint64: %w", err)
	}

	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 10
	}

	returnedOrderList, totalItems, err := a.appService.GetReturnedOrders(page, limit)
	if err != nil {
		return err
	}

	if len(returnedOrderList) == 0 {
		fmt.Println("No returns found on this page or no returns in total.")
	} else {
		for _, order := range returnedOrderList {
			fmt.Printf("RETURN: %d %d %s\n", order.OrderID, order.ReceiverID, order.LastUpdateTime.Format("2006-01-02"))
		}
	}
	fmt.Printf("PAGE: %d LIMIT: %d\n", page, limit)
	fmt.Printf("TOTAL: %d\n", totalItems)
	return nil
}

func (a *CLIAdapter) GetOrdersSortedByTime(cmd *cobra.Command, args []string) error {
	allOrders, err := a.appService.GetOrderHistory()
	if err != nil {
		return err
	}

	if len(allOrders) == 0 {
		fmt.Println("No orders in the system.")
		return nil
	}

	for _, order := range allOrders {
		fmt.Printf("HISTORY: %d %s %s\n",
			order.OrderID,
			order.GetStatusString(),
			MapTimeToString(order.LastUpdateTime),
		)
	}
	return nil
}
