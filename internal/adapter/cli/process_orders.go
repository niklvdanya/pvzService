package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func (a *CLIAdapter) ProcessOrders(cmd *cobra.Command, args []string) error {
	receiverID, err := cmd.Flags().GetUint64("user-id")
	if err != nil {
		return fmt.Errorf("flag.GetUint64: %w", err)
	}
	action, err := cmd.Flags().GetString("action")
	if err != nil {
		return fmt.Errorf("flag.GetString: %w", err)
	}
	orderIDsStr, err := cmd.Flags().GetString("order-ids")
	if err != nil {
		return fmt.Errorf("flag.GetString: %w", err)
	}

	if action != "issue" && action != "return" {
		return fmt.Errorf("invalid action '%s'", action)
	}

	orderIDStrings := strings.Split(orderIDsStr, ",")
	orderIDs := make([]uint64, 0, len(orderIDStrings))
	for _, s := range orderIDStrings {
		orderID, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return fmt.Errorf("strconv.ParseUint: %w", err)
		}
		orderIDs = append(orderIDs, orderID)
	}

	var processingErr error
	if action == "issue" {
		processingErr = a.appService.IssueOrdersToClient(receiverID, orderIDs)
	} else {
		processingErr = a.appService.ReturnOrdersFromClient(receiverID, orderIDs)
	}

	if processingErr != nil {
		return processingErr
	}

	for _, orderID := range orderIDs {
		fmt.Printf("PROCESSED: %d\n", orderID)
	}
	return nil
}
