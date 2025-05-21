package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

// TODO: error handling, paging, Postgre - ??, interface segregation, type aliases

type OrderStatus uint8

const (
	StatusInStorage OrderStatus = iota
	StatusGivenToClient
)

// добавил часы и минуты, чтобы было легче отлаживать работу функций, где есть проверка с time.Now()
// для синхронизации с московским временем добавил time.LoadLocation
var moscowTime, _ = time.LoadLocation("Europe/Moscow")

type Order struct {
	OrderID        uint64
	ReceiverID     uint64
	StorageUntil   time.Time
	Status         OrderStatus
	AcceptTime     time.Time
	LastUpdateTime time.Time
}

type ReturnedOrder struct {
	OrderID    uint64
	ReceiverID uint64
	ReturnedAt time.Time
}

var (
	ordersByID       map[uint64]*Order              = make(map[uint64]*Order)
	ordersByReceiver map[uint64]map[uint64]struct{} = make(map[uint64]map[uint64]struct{})
	returnedOrders   map[uint64]*ReturnedOrder      = make(map[uint64]*ReturnedOrder)
)

var rootCommand = &cobra.Command{
	Short: "PVZ (Pickup Point) command-line interface",
	Long: `A simple command-line interface for managing orders in a pickup point.
    Type 'help' for a list of commands, or 'help <command>' for specific command usage.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func paginate[T any](items []T, currentPage, itemsPerPage uint64) []T {
	totalItems := uint64(len(items))

	if itemsPerPage == 0 {
		return []T{}
	}
	if currentPage == 0 {
		currentPage = 1
	}

	startIndex := (currentPage - 1) * itemsPerPage
	endIndex := startIndex + itemsPerPage

	if startIndex >= totalItems {
		return []T{}
	}
	if endIndex > totalItems {
		endIndex = totalItems
	}

	return items[startIndex:endIndex]
}

func getStatusString(status OrderStatus) string {
	switch status {
	case StatusInStorage:
		return "In Storage"
	case StatusGivenToClient:
		return "Given to Client"
	default:
		return "Unknown Status"
	}

}

func GiveOrdersToClient(receiverID uint64, orderIDs []uint64) error {
	var combinedErr error

	for _, orderID := range orderIDs {
		var currentOrderErr error
		order, exists := ordersByID[orderID]
		if !exists {
			currentOrderErr = fmt.Errorf("order %d: not found", orderID)
		} else if order.ReceiverID != receiverID {
			currentOrderErr = fmt.Errorf("order %d: belongs to a different receiver (expected %d, got %d)", orderID, receiverID, order.ReceiverID)
		} else if order.Status == StatusGivenToClient {
			currentOrderErr = fmt.Errorf("order %d: already given to client", orderID)
		} else if time.Now().In(moscowTime).After(order.StorageUntil) {
			currentOrderErr = fmt.Errorf("order %d: storage period expired (%s), cannot be given", orderID, order.StorageUntil.Format("2006-01-02 15:04"))
		}

		if currentOrderErr != nil {
			combinedErr = multierr.Append(combinedErr, currentOrderErr)
			continue
		}
		order.Status = StatusGivenToClient
		order.LastUpdateTime = time.Now().In(moscowTime)
	}

	return combinedErr
}

func ReturnOrderFromClient(receiverID uint64, orderIDs []uint64) error {
	var combinedErr error

	for _, orderID := range orderIDs {
		var currentOrderErr error

		order, exists := ordersByID[orderID]
		if !exists {
			currentOrderErr = fmt.Errorf("order %d: not found", orderID)
		} else if order.ReceiverID != receiverID {
			currentOrderErr = fmt.Errorf("order %d: belongs to a different receiver (expected %d, got %d)", orderID, receiverID, order.ReceiverID)
		} else if order.Status == StatusInStorage {
			currentOrderErr = fmt.Errorf("order %d: already in storage, cannot be returned as client return", orderID)
		} else {
			currentTimeInMoscow := time.Now().In(moscowTime)
			timeSinceGiven := currentTimeInMoscow.Sub(order.LastUpdateTime)

			twoDaysLimit := 48 * time.Hour

			if timeSinceGiven > twoDaysLimit {
				currentOrderErr = fmt.Errorf("order %d: too much time has passed (%.1f hours) since it was given to client. Return period expired (2-day limit)",
					orderID, timeSinceGiven.Hours())
			}
		}

		if currentOrderErr != nil {
			combinedErr = multierr.Append(combinedErr, currentOrderErr)
			continue
		}
		returnedOrders[orderID] = &ReturnedOrder{
			OrderID:    order.OrderID,
			ReceiverID: order.ReceiverID,
			ReturnedAt: time.Now().In(moscowTime),
		}

		delete(ordersByID, orderID)
		if _, exists := ordersByReceiver[order.ReceiverID]; exists {
			delete(ordersByReceiver[order.ReceiverID], orderID)
			if len(ordersByReceiver[order.ReceiverID]) == 0 {
				delete(ordersByReceiver, order.ReceiverID)
			}
		}
	}

	return combinedErr
}

func GetReturnedOrders(cmd *cobra.Command, args []string) error {
	page, _ := cmd.Flags().GetUint64("page")
	limit, _ := cmd.Flags().GetUint64("limit")

	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 10
	}

	var returnedOrderList []*ReturnedOrder
	for _, order := range returnedOrders {
		returnedOrderList = append(returnedOrderList, order)
	}

	sort.Slice(returnedOrderList, func(i, j int) bool {
		return returnedOrderList[i].OrderID < returnedOrderList[j].OrderID
	})

	paginatedIDs := paginate(returnedOrderList, page, limit)
	if len(paginatedIDs) == 0 {
		fmt.Println("No returns found on this page or no returns in total.")
	} else {
		for _, order := range paginatedIDs {
			fmt.Printf("RETURN: %d %d %s\n", order.OrderID, order.ReceiverID, order.ReturnedAt.Format("2006-01-02 15:04"))
		}
	}

	fmt.Printf("PAGE: %d LIMIT: %d\n", page, limit)

	return nil
}

func GetOrdersSortedByTime(cmd *cobra.Command, args []string) error {
	var allOrders []*Order
	for _, order := range ordersByID {
		allOrders = append(allOrders, order)
	}

	if len(allOrders) == 0 {
		fmt.Println("No orders in the system.")
		return nil
	}
	sort.Slice(allOrders, func(i, j int) bool {
		return allOrders[i].LastUpdateTime.After(allOrders[j].LastUpdateTime)
	})

	for _, order := range allOrders {
		statusStr := getStatusString(order.Status)
		fmt.Printf("HISTORY: %d %s %s\n",
			order.OrderID,
			statusStr,
			order.LastUpdateTime.Format("2006-01-02 15:04"),
		)
	}

	return nil
}

func ProcessOrders(cmd *cobra.Command, args []string) error {
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
		processingErr = GiveOrdersToClient(receiverID, orderIDs)
	} else {
		processingErr = ReturnOrderFromClient(receiverID, orderIDs)
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
						fmt.Fprintf(os.Stderr, "Order %d %s\n", orderID, parts[1])
						continue
					}
				}
			}
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", e)
		}
		return fmt.Errorf("one or more orders failed to process")
	}

	for _, orderID := range orderIDs {
		fmt.Printf("PROCESSED: %d\n", orderID)
	}

	return nil
}

func ListOrdersComm(cmd *cobra.Command, args []string) error {
	receiverID, _ := cmd.Flags().GetUint64("user-id")
	inPvz, _ := cmd.Flags().GetBool("in-pvz")
	lastN, _ := cmd.Flags().GetUint64("last")
	page, _ := cmd.Flags().GetUint64("page")
	limit, _ := cmd.Flags().GetUint64("limit")

	if receiverID == 0 {
		return fmt.Errorf("missing --user-id")
	}

	if lastN > 0 && (page > 0 || limit > 0) {
		return fmt.Errorf("cannot use --last with --page or --limit")
	}

	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 10
	}

	var filteredOrders []*Order
	if userOrders, exists := ordersByReceiver[receiverID]; exists {
		for orderID := range userOrders {
			order := ordersByID[orderID]
			if order == nil {
				continue
			}
			if inPvz && order.Status != StatusInStorage {
				continue
			}
			filteredOrders = append(filteredOrders, order)
		}
	}

	sort.Slice(filteredOrders, func(i, j int) bool {
		return filteredOrders[i].OrderID < filteredOrders[j].OrderID
	})

	var paginatedOrders []*Order
	totalItems := uint64(len(filteredOrders))

	if lastN > 0 {
		if totalItems > lastN {
			paginatedOrders = filteredOrders[totalItems-lastN:]
		} else {
			paginatedOrders = filteredOrders
		}
	} else {
		paginatedOrders = paginate(filteredOrders, page, limit)
	}

	if len(paginatedOrders) == 0 {
		fmt.Println("No orders found for this receiver with the given criteria.")
	} else {
		for _, order := range paginatedOrders {
			statusStr := getStatusString(order.Status)

			fmt.Printf("Order: %d Reciever: %d Status: %s Storage Limit: %s\n",
				order.OrderID,
				order.ReceiverID,
				statusStr,
				order.StorageUntil.Format("2006-01-02_15:04"),
			)
		}
	}
	fmt.Printf("TOTAL: %d\n", totalItems)

	return nil
}

func AddComm(cmd *cobra.Command, args []string) error {
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

	err := Add(receiverID, orderID, storageUntilStr)
	if err != nil {
		return fmt.Errorf("failed to accept order: %w", err)
	}
	fmt.Printf("ORDER_ACCEPTED: %d\n", orderID)
	return nil
}

func BackOrder(cmd *cobra.Command, args []string) error {
	orderID, _ := cmd.Flags().GetUint64("order-id")

	if orderID == 0 {
		return fmt.Errorf("missing --order-id")
	}

	err := BackToDelivery(orderID)
	if err != nil {
		return fmt.Errorf("failed to return order: %w", err)
	}
	fmt.Printf("ORDER_RETURNED: %d\n", orderID)
	return nil
}

func main() {
	acceptOrderCmd := &cobra.Command{
		Use:   "accept-order",
		Short: "Accepts an order from a courier.",
		RunE:  AddComm,
	}
	acceptOrderCmd.Flags().Uint64P("order-id", "", 0, "ID of the order")
	acceptOrderCmd.Flags().Uint64P("user-id", "", 0, "ID of the receiver")
	acceptOrderCmd.Flags().StringP("expires", "", "", "Storage expiration date (YYYY-MM-DD_HH:MM)")
	acceptOrderCmd.MarkFlagRequired("order-id")
	acceptOrderCmd.MarkFlagRequired("user-id")
	acceptOrderCmd.MarkFlagRequired("expires")
	rootCommand.AddCommand(acceptOrderCmd)

	returnOrderCmd := &cobra.Command{
		Use:   "return-order",
		Short: "Returns an order to the courier.",
		RunE:  BackOrder,
	}
	returnOrderCmd.Flags().Uint64P("order-id", "", 0, "ID of the order to return")
	returnOrderCmd.MarkFlagRequired("order-id")
	rootCommand.AddCommand(returnOrderCmd)

	processOrdersCmd := &cobra.Command{
		Use:   "process-orders",
		Short: "Issues orders to a client or accepts returns from a client.",
		RunE:  ProcessOrders,
	}
	processOrdersCmd.Flags().Uint64P("user-id", "", 0, "ID of the receiver")
	processOrdersCmd.Flags().StringP("action", "", "", "Action to perform: 'issue' or 'return'")
	processOrdersCmd.Flags().StringP("order-ids", "", "", "Comma-separated list of order IDs")
	processOrdersCmd.MarkFlagRequired("user-id")
	processOrdersCmd.MarkFlagRequired("action")
	processOrdersCmd.MarkFlagRequired("order-ids")
	rootCommand.AddCommand(processOrdersCmd)

	listOrdersCmd := &cobra.Command{
		Use:   "list-orders",
		Short: "Lists orders for a specific receiver.",
		RunE:  ListOrdersComm,
	}
	listOrdersCmd.Flags().Uint64P("user-id", "", 0, "ID of the receiver")
	listOrdersCmd.Flags().BoolP("in-pvz", "", false, "Filter for orders currently in PVZ storage")
	listOrdersCmd.Flags().Uint64P("last", "", 0, "Show last N orders")
	listOrdersCmd.Flags().Uint64P("page", "", 0, "Page number for pagination")
	listOrdersCmd.Flags().Uint64P("limit", "", 0, "Items per page for pagination")
	listOrdersCmd.MarkFlagRequired("user-id")
	rootCommand.AddCommand(listOrdersCmd)

	listReturnsCmd := &cobra.Command{
		Use:   "list-returns",
		Short: "Lists all returned orders.",
		RunE:  GetReturnedOrders,
	}
	listReturnsCmd.Flags().Uint64P("page", "", 0, "Page number for pagination")
	listReturnsCmd.Flags().Uint64P("limit", "", 0, "Items per page for pagination")
	rootCommand.AddCommand(listReturnsCmd)

	orderHistoryCmd := &cobra.Command{
		Use:   "order-history",
		Short: "Shows the history of all order status changes (sorted by last update time).",
		RunE:  GetOrdersSortedByTime,
	}
	rootCommand.AddCommand(orderHistoryCmd)

	rootCommand.AddCommand(&cobra.Command{
		Use:   "exit",
		Short: "Exits the PVZ system.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Exiting PVZ system.")
			os.Exit(0)
		},
	})
	fmt.Println("welcome")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("pvz>")

		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		rootCommand.SetArgs(strings.Fields(line))
		if err := rootCommand.Execute(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading from stdin: %s", err)
	}

}

func Add(ReceiverID, OrderID uint64, storageUntilStr string) error {
	storageUntil, err := time.ParseInLocation("2006-01-02_15:04", storageUntilStr, moscowTime)
	if err != nil {
		return fmt.Errorf("invalid storage until time format for Order %d. Expected 2006-01-02_15:04, got '%s': %w", OrderID, storageUntilStr, err)
	}

	currentTimeInMoscow := time.Now().In(moscowTime)
	if storageUntil.Before(currentTimeInMoscow) {
		return fmt.Errorf("cannot accept order %d: storage period already expired. Current time: %s, Provided until: %s", OrderID, currentTimeInMoscow.Format("2006-01-02 15:04"), storageUntil.Format("2006-01-02 15:04"))
	}
	if _, exists := ordersByID[OrderID]; exists {
		return fmt.Errorf("cannot accept order %d: order with this ID already exists", OrderID)
	}

	ordersByID[OrderID] = &Order{
		OrderID:        OrderID,
		ReceiverID:     ReceiverID,
		StorageUntil:   storageUntil,
		Status:         StatusInStorage,
		AcceptTime:     currentTimeInMoscow,
		LastUpdateTime: currentTimeInMoscow,
	}
	if _, exists := ordersByReceiver[ReceiverID]; !exists {
		ordersByReceiver[ReceiverID] = make(map[uint64]struct{})
	}
	ordersByReceiver[ReceiverID][OrderID] = struct{}{}
	return nil
}

func BackToDelivery(OrderID uint64) error {
	order, exists := ordersByID[OrderID]
	if !exists {
		return fmt.Errorf("cannot return order %d to delivery: order not found", OrderID)
	}

	if order.Status != StatusInStorage {
		return fmt.Errorf("cannot return order %d to delivery: order is not in storage (current status: %s)", OrderID, getStatusString(order.Status))
	}
	if time.Now().In(moscowTime).Before(order.StorageUntil) {
		return fmt.Errorf("cannot return order %d to delivery: storage period has not yet expired (until: %s)", OrderID, order.StorageUntil.Format("2006-01-02 15:04"))
	}

	delete(ordersByID, OrderID)
	delete(ordersByReceiver[order.ReceiverID], OrderID)

	return nil
}
