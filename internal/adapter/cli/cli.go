package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"go.uber.org/multierr"
)

type OrderService interface {
	AcceptOrder(receiverID, orderID uint64, storageUntil time.Time) error
	ReturnOrderToDelivery(orderID uint64) error
	IssueOrdersToClient(receiverID uint64, orderIDs []uint64) error
	ReturnOrdersFromClient(receiverID uint64, orderIDs []uint64) error
	GetReceiverOrders(receiverID uint64, inPVZ bool, lastN, page, limit uint64) ([]*domain.Order, uint64, error)
	GetReceiverOrdersScroll(receiverID uint64, lastID, limit uint64) ([]*domain.Order, uint64, error)
	GetReturnedOrders(page, limit uint64) ([]*domain.Order, uint64, error)
	GetOrderHistory() ([]*domain.Order, error)
	ImportOrders(orders []struct {
		OrderID      uint64 `json:"order_id"`
		ReceiverID   uint64 `json:"receiver_id"`
		StorageUntil string `json:"storage_until"`
	}) (uint64, error)
}

type CLIAdapter struct {
	appService OrderService
}

func NewCLIAdapter(appService OrderService) *CLIAdapter {
	return &CLIAdapter{appService: appService}
}

func (a *CLIAdapter) RegisterCommands(rootCmd *cobra.Command) {
	acceptOrderCmd := &cobra.Command{
		Use:   "accept-order",
		Short: "Accepts an order from a courier.",
		RunE:  a.AddComm,
	}
	acceptOrderCmd.Flags().Uint64P("order-id", "", 0, "ID of the order")
	acceptOrderCmd.Flags().Uint64P("user-id", "", 0, "ID of the receiver")
	acceptOrderCmd.Flags().StringP("expires", "", "", "Storage expiration date (YYYY-MM-DD)")
	acceptOrderCmd.MarkFlagRequired("order-id")
	acceptOrderCmd.MarkFlagRequired("user-id")
	acceptOrderCmd.MarkFlagRequired("expires")
	rootCmd.AddCommand(acceptOrderCmd)

	returnOrderCmd := &cobra.Command{
		Use:   "return-order",
		Short: "Returns an order to the courier.",
		RunE:  a.BackOrder,
	}
	returnOrderCmd.Flags().Uint64P("order-id", "", 0, "ID of the order to return")
	returnOrderCmd.MarkFlagRequired("order-id")
	rootCmd.AddCommand(returnOrderCmd)

	processOrdersCmd := &cobra.Command{
		Use:   "process-orders",
		Short: "Issues orders to a client or accepts returns from a client.",
		RunE:  a.ProcessOrders,
	}
	processOrdersCmd.Flags().Uint64P("user-id", "", 0, "ID of the receiver")
	processOrdersCmd.Flags().StringP("action", "", "", "Action to perform: 'issue' or 'return'")
	processOrdersCmd.Flags().StringP("order-ids", "", "", "Comma-separated list of order IDs")
	processOrdersCmd.MarkFlagRequired("user-id")
	processOrdersCmd.MarkFlagRequired("action")
	processOrdersCmd.MarkFlagRequired("order-ids")
	rootCmd.AddCommand(processOrdersCmd)

	listOrdersCmd := &cobra.Command{
		Use:   "list-orders",
		Short: "Lists orders for a specific receiver.",
		RunE:  a.ListOrdersComm,
	}
	listOrdersCmd.Flags().Uint64P("user-id", "", 0, "ID of the receiver")
	listOrdersCmd.Flags().BoolP("in-pvz", "", false, "Filter for orders currently in PVZ storage")
	listOrdersCmd.Flags().Uint64P("last", "", 0, "Show last N orders")
	listOrdersCmd.Flags().Uint64P("page", "", 0, "Page number for pagination")
	listOrdersCmd.Flags().Uint64P("limit", "", 0, "Items per page for pagination")
	listOrdersCmd.MarkFlagRequired("user-id")
	rootCmd.AddCommand(listOrdersCmd)

	listReturnsCmd := &cobra.Command{
		Use:   "list-returns",
		Short: "Lists all returned orders.",
		RunE:  a.GetReturnedOrders,
	}
	listReturnsCmd.Flags().Uint64P("page", "", 0, "Page number for pagination")
	listReturnsCmd.Flags().Uint64P("limit", "", 0, "Items per page for pagination")
	rootCmd.AddCommand(listReturnsCmd)

	orderHistoryCmd := &cobra.Command{
		Use:   "order-history",
		Short: "Shows the history of all order status changes (sorted by last update time).",
		RunE:  a.GetOrdersSortedByTime,
	}
	rootCmd.AddCommand(orderHistoryCmd)

	importOrdersCmd := &cobra.Command{
		Use:   "import-orders",
		Short: "Imports orders from a JSON file.",
		RunE:  a.ImportOrdersComm,
	}
	importOrdersCmd.Flags().StringP("file", "", "", "Path to the JSON file with orders")
	importOrdersCmd.MarkFlagRequired("file")
	rootCmd.AddCommand(importOrdersCmd)

	scrollOrdersCmd := &cobra.Command{
		Use:   "scroll-orders",
		Short: "Infinite orders scroll.",
		RunE:  a.ScrollOrdersComm,
	}
	scrollOrdersCmd.Flags().Uint64P("user-id", "", 0, "ID of the receiver")
	scrollOrdersCmd.Flags().Uint64P("limit", "", 20, "Number of orders to fetch at once")
	scrollOrdersCmd.MarkFlagRequired("user-id")
	rootCmd.AddCommand(scrollOrdersCmd)
}

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

func (a *CLIAdapter) BackOrder(cmd *cobra.Command, args []string) error {
	orderID, err := cmd.Flags().GetUint64("order-id")
	if err != nil {
		return mapError(fmt.Errorf("flag.GetUint64: %w", err))
	}

	err = a.appService.ReturnOrderToDelivery(orderID)
	if err != nil {
		return mapError(err)
	}
	fmt.Printf("ORDER_RETURNED: %d\n", orderID)
	return nil
}

func (a *CLIAdapter) ProcessOrders(cmd *cobra.Command, args []string) error {
	receiverID, err := cmd.Flags().GetUint64("user-id")
	if err != nil {
		return mapError(fmt.Errorf("flag.GetUint64: %w", err))
	}
	action, err := cmd.Flags().GetString("action")
	if err != nil {
		return mapError(fmt.Errorf("flag.GetString: %w", err))
	}
	orderIDsStr, err := cmd.Flags().GetString("order-ids")
	if err != nil {
		return mapError(fmt.Errorf("flag.GetString: %w", err))
	}

	if action != "issue" && action != "return" {
		return mapError(fmt.Errorf("invalid action '%s'", action))
	}

	orderIDStrings := strings.Split(orderIDsStr, ",")
	orderIDs := make([]uint64, 0, len(orderIDStrings))
	for _, s := range orderIDStrings {
		orderID, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return mapError(fmt.Errorf("strconv.ParseUint: %w", err))
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
		multiErrors := multierr.Errors(processingErr)
		for _, e := range multiErrors {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", mapError(e))
		}
		return mapError(fmt.Errorf("process orders failed"))
	}

	for _, orderID := range orderIDs {
		fmt.Printf("PROCESSED: %d\n", orderID)
	}
	return nil
}

func (a *CLIAdapter) ListOrdersComm(cmd *cobra.Command, args []string) error {
	receiverID, err := cmd.Flags().GetUint64("user-id")
	if err != nil {
		return mapError(fmt.Errorf("flag.GetUint64: %w", err))
	}
	inPvz, err := cmd.Flags().GetBool("in-pvz")
	if err != nil {
		return mapError(fmt.Errorf("flag.GetBool: %w", err))
	}
	lastN, err := cmd.Flags().GetUint64("last")
	if err != nil {
		return mapError(fmt.Errorf("flag.GetUint64: %w", err))
	}
	page, err := cmd.Flags().GetUint64("page")
	if err != nil {
		return mapError(fmt.Errorf("flag.GetUint64: %w", err))
	}
	limit, err := cmd.Flags().GetUint64("limit")
	if err != nil {
		return mapError(fmt.Errorf("flag.GetUint64: %w", err))
	}

	if lastN > 0 && (page > 0 || limit > 0) {
		return mapError(fmt.Errorf("invalid flags combination"))
	}

	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 10
	}

	orders, totalItems, err := a.appService.GetReceiverOrders(receiverID, inPvz, lastN, page, limit)
	if err != nil {
		return mapError(err)
	}

	if len(orders) == 0 {
		fmt.Println("No orders found for this receiver with the given criteria.")
	} else {
		for _, order := range orders {
			fmt.Printf("Order: %d Receiver: %d Status: %s Storage Limit: %s\n",
				order.OrderID,
				order.ReceiverID,
				order.GetStatusString(),
				order.StorageUntil.Format("2006-01-02"),
			)
		}
	}
	fmt.Printf("TOTAL: %d\n", totalItems)
	return nil
}

func (a *CLIAdapter) GetReturnedOrders(cmd *cobra.Command, args []string) error {
	page, err := cmd.Flags().GetUint64("page")
	if err != nil {
		return mapError(fmt.Errorf("flag.GetUint64: %w", err))
	}
	limit, err := cmd.Flags().GetUint64("limit")
	if err != nil {
		return mapError(fmt.Errorf("flag.GetUint64: %w", err))
	}

	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 10
	}

	returnedOrderList, totalItems, err := a.appService.GetReturnedOrders(page, limit)
	if err != nil {
		return mapError(err)
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
		return mapError(err)
	}

	if len(allOrders) == 0 {
		fmt.Println("No orders in the system.")
		return nil
	}

	for _, order := range allOrders {
		fmt.Printf("HISTORY: %d %s %s\n",
			order.OrderID,
			order.GetStatusString(),
			order.LastUpdateTime.Format("2006-01-02"),
		)
	}
	return nil
}

func (a *CLIAdapter) ImportOrdersComm(cmd *cobra.Command, args []string) error {
	filePath, err := cmd.Flags().GetString("file")
	if err != nil || filePath == "" {
		return mapError(fmt.Errorf("flag.GetString: %w", err))
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return mapError(fmt.Errorf("os.ReadFile: %w", err))
	}

	var ordersToImport []struct {
		OrderID      uint64 `json:"order_id"`
		ReceiverID   uint64 `json:"receiver_id"`
		StorageUntil string `json:"storage_until"`
	}

	if err := json.Unmarshal(data, &ordersToImport); err != nil {
		return mapError(fmt.Errorf("json.Unmarshal: %w", err))
	}

	importedCount, err := a.appService.ImportOrders(ordersToImport)
	if err != nil {
		multiErrors := multierr.Errors(err)
		for _, e := range multiErrors {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", mapError(e))
		}
		if importedCount > 0 {
			fmt.Printf("IMPORTED: %d orders successfully.\n", importedCount)
		}
		return mapError(fmt.Errorf("import orders failed"))
	}

	fmt.Printf("IMPORTED: %d\n", importedCount)
	return nil
}

func (a *CLIAdapter) ScrollOrdersComm(cmd *cobra.Command, args []string) error {
	receiverID, err := cmd.Flags().GetUint64("user-id")
	if err != nil {
		return mapError(fmt.Errorf("flag.GetUint64: %w", err))
	}
	limit, err := cmd.Flags().GetUint64("limit")
	if err != nil {
		return mapError(fmt.Errorf("flag.GetUint64: %w", err))
	}

	if limit == 0 {
		limit = 20
	}

	var currentLastID uint64
	scanner := bufio.NewScanner(os.Stdin)
	orders, nextLastID, err := a.appService.GetReceiverOrdersScroll(receiverID, currentLastID, limit)
	if err != nil {
		return mapError(err)
	}
	a.printScrollOrders(orders, nextLastID)
	currentLastID = nextLastID

	for {
		if currentLastID == 0 && len(orders) == 0 {
			fmt.Println("No more orders to display.")
			break
		}

		fmt.Print("> ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "next":
			if currentLastID == 0 {
				fmt.Println("No more orders to display.")
				continue
			}
			orders, nextLastID, err = a.appService.GetReceiverOrdersScroll(receiverID, currentLastID, limit)
			if err != nil {
				return mapError(err)
			}
			a.printScrollOrders(orders, nextLastID)
			currentLastID = nextLastID
		case "exit":
			fmt.Println("Exiting scroll-orders.")
			return nil
		default:
			fmt.Println("Unknown command. Type 'next' to get more orders or 'exit' to quit.")
		}
	}

	if err := scanner.Err(); err != nil {
		return mapError(fmt.Errorf("scanner.Scan: %w", err))
	}
	return nil
}

func (a *CLIAdapter) printScrollOrders(orders []*domain.Order, nextLastID uint64) {
	if len(orders) == 0 {
		fmt.Println("No orders found in this batch.")
	} else {
		for _, order := range orders {
			fmt.Printf("ORDER: %d Receiver: %d Status: %s Storage Limit: %s\n",
				order.OrderID,
				order.ReceiverID,
				order.GetStatusString(),
				order.StorageUntil.Format("2006-01-02"),
			)
		}
	}
	if nextLastID > 0 {
		fmt.Printf("NEXT: %d\n", nextLastID)
	} else {
		fmt.Println("NEXT: 0 (End of orders)")
	}
}
