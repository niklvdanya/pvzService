package cli

import "github.com/spf13/cobra"

func (a *CLIAdapter) registerCommands(rootCmd *cobra.Command) {
	acceptOrderCmd := &cobra.Command{
		Use:   "accept-order",
		Short: "Accepts an order from a courier.",
		RunE:  a.AddComm,
	}
	acceptOrderCmd.Flags().Uint64P("order-id", "", 0, "ID of the order")
	acceptOrderCmd.Flags().Uint64P("user-id", "", 0, "ID of the receiver")
	acceptOrderCmd.Flags().StringP("expires", "", "", "Storage expiration date (YYYY-MM-DD)")
	// в  MapError обрабатываются ошибки, если флаг не указан
	_ = acceptOrderCmd.MarkFlagRequired("order-id")
	_ = acceptOrderCmd.MarkFlagRequired("user-id")
	_ = acceptOrderCmd.MarkFlagRequired("expires")
	rootCmd.AddCommand(acceptOrderCmd)

	returnOrderCmd := &cobra.Command{
		Use:   "return-order",
		Short: "Returns an order to the courier.",
		RunE:  a.BackOrder,
	}
	returnOrderCmd.Flags().Uint64P("order-id", "", 0, "ID of the order to return")
	_ = returnOrderCmd.MarkFlagRequired("order-id")
	rootCmd.AddCommand(returnOrderCmd)

	processOrdersCmd := &cobra.Command{
		Use:   "process-orders",
		Short: "Issues orders to a client or accepts returns from a client.",
		RunE:  a.ProcessOrders,
	}
	processOrdersCmd.Flags().Uint64P("user-id", "", 0, "ID of the receiver")
	processOrdersCmd.Flags().StringP("action", "", "", "Action to perform: 'issue' or 'return'")
	processOrdersCmd.Flags().StringP("order-ids", "", "", "Comma-separated list of order IDs")
	_ = processOrdersCmd.MarkFlagRequired("user-id")
	_ = processOrdersCmd.MarkFlagRequired("action")
	_ = processOrdersCmd.MarkFlagRequired("order-ids")
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
	_ = listOrdersCmd.MarkFlagRequired("user-id")
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
	_ = importOrdersCmd.MarkFlagRequired("file")
	rootCmd.AddCommand(importOrdersCmd)

	scrollOrdersCmd := &cobra.Command{
		Use:   "scroll-orders",
		Short: "Infinite orders scroll.",
		RunE:  a.ScrollOrdersComm,
	}
	scrollOrdersCmd.Flags().Uint64P("user-id", "", 0, "ID of the receiver")
	scrollOrdersCmd.Flags().Uint64P("limit", "", 20, "Number of orders to fetch at once")
	_ = scrollOrdersCmd.MarkFlagRequired("user-id")
	rootCmd.AddCommand(scrollOrdersCmd)
}
