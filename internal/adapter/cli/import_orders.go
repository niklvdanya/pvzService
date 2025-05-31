package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func (a *CLIAdapter) ImportOrdersComm(cmd *cobra.Command, args []string) error {
	filePath, err := cmd.Flags().GetString("file")
	if err != nil || filePath == "" {
		return fmt.Errorf("flag.GetString: %w", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("os.ReadFile: %w", err)
	}

	var ordersToImport []domain.OrderToImport
	if err := json.Unmarshal(data, &ordersToImport); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}

	importedCount, err := a.appService.ImportOrders(ordersToImport)
	if err != nil {
		if importedCount > 0 {
			fmt.Printf("IMPORTED: %d orders successfully.\n", importedCount)
		}
		return err
	}

	fmt.Printf("IMPORTED: %d\n", importedCount)
	return nil
}
