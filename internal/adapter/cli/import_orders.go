package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

func RegisterImportOrdersCmd(rootCmd *cobra.Command, appService OrderService) {
	importOrdersCmd := &cobra.Command{
		Use:   "import-orders",
		Short: "Imports orders from a JSON file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, err := cmd.Flags().GetString("file")
			if err != nil {
				return MapError(fmt.Errorf("flag.GetString: %w", err))
			}
			if filePath == "" {
				return MapError(fmt.Errorf("file path is empty"))
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				return MapError(fmt.Errorf("os.ReadFile: %w", err))
			}

			var ordersToImport []struct {
				OrderID      uint64 `json:"order_id"`
				ReceiverID   uint64 `json:"receiver_id"`
				StorageUntil string `json:"storage_until"`
			}

			if err := json.Unmarshal(data, &ordersToImport); err != nil {
				return MapError(fmt.Errorf("json.Unmarshal: %w", err))
			}

			importedCount, err := appService.ImportOrders(ordersToImport)
			if err != nil {
				multiErrors := multierr.Errors(err)
				for _, e := range multiErrors {
					fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", MapError(e))
				}
				if importedCount > 0 {
					fmt.Printf("IMPORTED: %d orders successfully.\n", importedCount)
				}
				return MapError(fmt.Errorf("import orders failed"))
			}

			fmt.Printf("IMPORTED: %d\n", importedCount)
			return nil
		},
	}
	importOrdersCmd.Flags().StringP("file", "", "", "Path to the JSON file with orders")
	importOrdersCmd.MarkFlagRequired("file")
	rootCmd.AddCommand(importOrdersCmd)
}
