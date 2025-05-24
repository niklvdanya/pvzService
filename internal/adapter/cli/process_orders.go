package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

func RegisterProcessOrdersCmd(rootCmd *cobra.Command, appService OrderService) {
	processOrdersCmd := &cobra.Command{
		Use:   "process-orders",
		Short: "Issues orders to a client or accepts returns from a client.",
		RunE: func(cmd *cobra.Command, args []string) error {
			receiverID, err := cmd.Flags().GetUint64("user-id")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetUint64: %w", err))
			}
			action, err := cmd.Flags().GetString("action")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetString: %w", err))
			}
			orderIDsStr, err := cmd.Flags().GetString("order-ids")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetString: %w", err))
			}

			if action != "issue" && action != "return" {
				return MapError(fmt.Errorf("invalid action: %s", action))
			}

			orderIDStrings := strings.Split(orderIDsStr, ",")
			orderIDs := make([]uint64, 0, len(orderIDStrings))
			for _, s := range orderIDStrings {
				orderID, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
				if err != nil {
					return MapError(fmt.Errorf("invalid order ID format: %w", err))
				}
				orderIDs = append(orderIDs, orderID)
			}

			var processingErr error
			if action == "issue" {
				processingErr = appService.IssueOrdersToClient(receiverID, orderIDs)
			} else {
				processingErr = appService.ReturnOrdersFromClient(receiverID, orderIDs)
			}

			if processingErr != nil {
				multiErrors := multierr.Errors(processingErr)
				for _, e := range multiErrors {
					fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", MapError(e))
				}
				return MapError(fmt.Errorf("process orders failed"))
			}

			for _, orderID := range orderIDs {
				fmt.Printf("PROCESSED: %d\n", orderID)
			}
			return nil
		},
	}
	processOrdersCmd.Flags().Uint64P("user-id", "", 0, "ID of the receiver")
	processOrdersCmd.Flags().StringP("action", "", "", "Action to perform: 'issue' or 'return'")
	processOrdersCmd.Flags().StringP("order-ids", "", "", "Comma-separated list of order IDs")
	processOrdersCmd.MarkFlagRequired("user-id")
	processOrdersCmd.MarkFlagRequired("action")
	processOrdersCmd.MarkFlagRequired("order-ids")
	rootCmd.AddCommand(processOrdersCmd)
}
