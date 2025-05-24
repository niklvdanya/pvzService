package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"

	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

type OrderService interface {
	AcceptOrder(receiverID, orderID uint64, storageUntil time.Time) error
	ReturnOrderToDelivery(orderID uint64) error
	IssueOrdersToClient(receiverID uint64, orderIDs []uint64) error
	ReturnOrdersFromClient(receiverID uint64, orderIDs []uint64) error
	GetReceiverOrders(
		receiverID uint64,
		inPVZ bool,
		lastN, page, limit uint64,
	) ([]*domain.Order, uint64, error)
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

	rootCmd.AddCommand(&cobra.Command{
		Use:   "exit",
		Short: "Exits the PVZ system.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Exiting PVZ system.")
			// exit(0) не делаю, потому что main все равно завершит работу
		},
	})
}

func (a *CLIAdapter) AddComm(cmd *cobra.Command, args []string) error {
	receiverID, _ := cmd.Flags().GetUint64("user-id")
	orderID, _ := cmd.Flags().GetUint64("order-id")
	storageUntilStr, _ := cmd.Flags().GetString("expires")

	if receiverID == 0 {
		return fmt.Errorf("missing --user-id")
	}
	if orderID == 0 {
		return fmt.Errorf("missing --order-id")
	}
	if storageUntilStr == "" {
		return fmt.Errorf("missing --expires")
	}

	storageUntil, err := time.Parse("2006-01-02", storageUntilStr)
	if err != nil {
		return fmt.Errorf(
			"invalid storage until time format for Order %d. Expected 2006-01-02, got '%s': %w",
			orderID,
			storageUntilStr,
			err,
		)
	}

	err = a.appService.AcceptOrder(receiverID, orderID, storageUntil)
	if err != nil {
		return fmt.Errorf("failed to accept order: %w", err)
	}
	fmt.Printf("ORDER_ACCEPTED: %d\n", orderID)
	return nil
}

func (a *CLIAdapter) BackOrder(cmd *cobra.Command, args []string) error {
	orderID, _ := cmd.Flags().GetUint64("order-id")

	if orderID == 0 {
		return fmt.Errorf("missing --order-id")
	}

	err := a.appService.ReturnOrderToDelivery(orderID)
	if err != nil {
		return fmt.Errorf("failed to return order: %w", err)
	}
	fmt.Printf("ORDER_RETURNED: %d\n", orderID)
	return nil
}

func (a *CLIAdapter) ProcessOrders(cmd *cobra.Command, args []string) error {
	receiverID, _ := cmd.Flags().GetUint64("user-id")
	action, _ := cmd.Flags().GetString("action")
	orderIDsStr, _ := cmd.Flags().GetString("order-ids")

	if receiverID == 0 {
		return fmt.Errorf("missing --user-id")
	}
	if action == "" {
		return fmt.Errorf("missing --action")
	}
	if orderIDsStr == "" {
		return fmt.Errorf("missing --order-ids")
	}

	if action != "issue" && action != "return" {
		return fmt.Errorf("invalid action '%s': action must be 'issue' or 'return'", action)
	}

	orderIDStrings := strings.Split(orderIDsStr, ",")
	orderIDs := make([]uint64, 0, len(orderIDStrings))
	for _, s := range orderIDStrings {
		orderID, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid OrderID '%s': must be a number", s)
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
			if strings.HasPrefix(e.Error(), "order") {
				parts := strings.SplitN(e.Error(), ": ", 2)
				if len(parts) == 2 {
					orderPart := strings.TrimPrefix(parts[0], "order ")
					orderID, parseErr := strconv.ParseUint(orderPart, 10, 64)
					if parseErr == nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Order %d %s\n", orderID, parts[1])
						continue
					}
				}
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "ERROR: %v\n", e)
		}
		return fmt.Errorf("one or more orders failed to process")
	}

	for _, orderID := range orderIDs {
		fmt.Printf("PROCESSED: %d\n", orderID)
	}

	return nil
}

func (a *CLIAdapter) ListOrdersComm(cmd *cobra.Command, args []string) error {
	receiverID, _ := cmd.Flags().GetUint64("user-id")
	inPvz, _ := cmd.Flags().GetBool("in-pvz")
	lastN, _ := cmd.Flags().GetUint64("last")
	page, _ := cmd.Flags().GetUint64("page")
	limit, _ := cmd.Flags().GetUint64("limit")

	if lastN > 0 && (page > 0 || limit > 0) {
		return fmt.Errorf("cannot use --last with --page or --limit")
	}

	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 10
	}

	orders, totalItems, err := a.appService.GetReceiverOrders(receiverID, inPvz, lastN, page, limit)
	if err != nil {
		return err
	}

	if len(orders) == 0 {
		fmt.Println("No orders found for this receiver with the given criteria.")
	} else {
		for _, order := range orders {
			fmt.Printf("Order: %d Reciever: %d Status: %s Storage Limit: %s\n",
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
	page, _ := cmd.Flags().GetUint64("page")
	limit, _ := cmd.Flags().GetUint64("limit")

	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 10
	}

	returnedOrderList, totalItems, err := a.appService.GetReturnedOrders(page, limit)
	if err != nil {
		return fmt.Errorf("failed to list returned orders: %w", err)
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
		return fmt.Errorf("failed to get order history: %w", err)
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
	filePath, _ := cmd.Flags().GetString("file")
	if filePath == "" {
		return fmt.Errorf("missing --file path")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read JSON file '%s': %w", filePath, err)
	}

	var ordersToImport []struct {
		OrderID      uint64 `json:"order_id"`
		ReceiverID   uint64 `json:"receiver_id"`
		StorageUntil string `json:"storage_until"`
	}

	if err := json.Unmarshal(data, &ordersToImport); err != nil {
		return fmt.Errorf("failed to parse JSON from file '%s': %w", filePath, err)
	}

	importedCount, err := a.appService.ImportOrders(ordersToImport)

	if err != nil {
		if importedCount > 0 {
			fmt.Printf("IMPORTED: %d orders successfully.\n", importedCount)
		}
		multiErrors := multierr.Errors(err)
		for _, e := range multiErrors {
			fmt.Fprintf(cmd.ErrOrStderr(), "ERROR: %v\n", e)
		}
		return fmt.Errorf("import completed with errors")
	}

	fmt.Printf("IMPORTED: %d\n", importedCount)
	return nil
}

func (a *CLIAdapter) ScrollOrdersComm(cmd *cobra.Command, args []string) error {
	receiverID, _ := cmd.Flags().GetUint64("user-id")
	limit, _ := cmd.Flags().GetUint64("limit")

	if receiverID == 0 {
		return fmt.Errorf("missing --user-id")
	}
	if limit == 0 {
		limit = 20
	}

	var currentLastID uint64 = 0
	scanner := bufio.NewScanner(os.Stdin)
	// инициализация нашего цикла
	orders, nextLastID, err := a.appService.GetReceiverOrdersScroll(receiverID, currentLastID, limit)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "ERROR: %v\n", err)
		return nil
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
			// благодаря GetReceiverOrdersScroll находим n = limit заказов и заодно id последнего полученного заказа
			orders, nextLastID, err = a.appService.GetReceiverOrdersScroll(
				receiverID,
				currentLastID,
				limit,
			)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "ERROR: %v\n", err)
				return nil
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
		return fmt.Errorf("error reading input: %w", err)
	}
	return nil
}

func (a *CLIAdapter) printScrollOrders(orders []*domain.Order, nextLastID uint64) {
	if len(orders) == 0 {
		fmt.Println("No orders found in this batch.")
	} else {
		for _, order := range orders {
			fmt.Printf("ORDER: %d Reciever: %d Status: %s Storage Limit: %s\n",
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
