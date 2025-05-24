package cli

import (
	"time"

	"github.com/spf13/cobra"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
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
