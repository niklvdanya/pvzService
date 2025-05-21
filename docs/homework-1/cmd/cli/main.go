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
	expectedArgCnt              = 3
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

var (
	ordersByID       map[uint64]*Order              = make(map[uint64]*Order)
	ordersByReceiver map[uint64]map[uint64]struct{} = make(map[uint64]map[uint64]struct{})
	//надо наверное чуть более подробную структуру завести для возвратов
	returnedOrders map[uint64]struct{} = make(map[uint64]struct{})
)

var rootCommand cobra.Command = cobra.Command{
	Short: "PVZ (Pickup Point) command-line interface",
	Long: `A simple command-line interface for managing orders in a pickup point.
	Type 'help' for a list of commands, or 'help <command>' for specific command usage.`,
	Run: func(cmd *cobra.Command, args []string) {},
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
	// по сути в default никогда не зайдем, но добавил на всякий случай
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
		// ничего кроме изменения статуса мы не делаем, ибо клиент может вернуть заказ, соответственно его еще надо хранить
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
			//простая проверка на то, что заказ не был выдан больше 2 дней назад
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
		// не уверен, что для возвратов именно такая логика верная, но судя по сообщениям в чате
		// есть смысл хранить в ПВЗ только те товары, что мы можем в перспективе выдать клиенту
		delete(ordersByID, orderID)
		if _, exists := ordersByReceiver[order.ReceiverID]; exists {
			delete(ordersByReceiver[order.ReceiverID], orderID)
			if len(ordersByReceiver[order.ReceiverID]) == 0 {
				delete(ordersByReceiver, order.ReceiverID)
			}
		}
		// добавляем в возвраты, храним их в отдельном месте
		returnedOrders[orderID] = struct{}{}
	}

	return combinedErr
}

func GetReturnedOrders(cmd *cobra.Command, args []string) error {
	currentPage := uint64(1)
	itemsPerPage := uint64(10)

	if len(args) > 0 {
		parsedPage, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid page number: %w", err)
		}
		if parsedPage == 0 {
			return fmt.Errorf("page number cannot be 0")
		}
		currentPage = parsedPage
	}

	if len(args) > 1 {
		parsedLimit, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid limit: %w", err)
		}
		if parsedLimit == 0 {
			return fmt.Errorf("limit cannot be 0")
		}
		itemsPerPage = parsedLimit
	}

	var returnedOrderIDs []uint64
	for orderID := range returnedOrders {
		returnedOrderIDs = append(returnedOrderIDs, orderID)
	}

	sort.Slice(returnedOrderIDs, func(i, j int) bool {
		return returnedOrderIDs[i] < returnedOrderIDs[j]
	})

	paginatedIDs := paginate(returnedOrderIDs, currentPage, itemsPerPage)

	totalItems := uint64(len(returnedOrderIDs))

	if len(paginatedIDs) == 0 {
		fmt.Println("No returns found on this page or no returns in total.")
	} else {
		fmt.Println("List of Returned Orders:")
		for _, orderID := range paginatedIDs {
			fmt.Printf("RETURN: %d\n", orderID)
		}
	}

	fmt.Printf("PAGE: %d LIMIT: %d TOTAL: %d\n", currentPage, itemsPerPage, totalItems)

	return nil
}

func GetOrdersSortedByTime(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("this command does not accept arguments")
	}
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
	fmt.Println(len(allOrders))
	fmt.Println("All Orders (sorted by Last Update Time, newest first):")
	fmt.Println("-----------------------------------------------------")
	for _, order := range allOrders {
		statusStr := getStatusString(order.Status)
		fmt.Printf("Order: %d | Receiver: %d | Status: %s | Last Update: %s\n",
			order.OrderID,
			order.ReceiverID,
			statusStr,
			order.LastUpdateTime.Format("2006-01-02 15:04"),
		)
	}
	fmt.Println("-----------------------------------------------------")

	return nil
}

func ProcessOrders(cmd *cobra.Command, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("missing arguments. Usage: process-orders <ReceiverID> <action> <OrderID1> [OrderID2...]")
	}
	ReceiverID, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ReceiverID '%s': must be a number", args[0])
	}
	action := strings.ToLower(args[1])
	if action != "issue" && action != "return" {
		return fmt.Errorf("invalid action '%s': action must be 'issue' or 'return'", args[1])
	}
	orderIDs := make([]uint64, 0, len(args)-2)
	for i := 2; i < len(args); i++ {
		orderID, err := strconv.ParseUint(args[i], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid OrderID '%s': must be a number", args[i])
		}
		orderIDs = append(orderIDs, orderID)
	}

	var processingErr error

	if action == "issue" {
		processingErr = GiveOrdersToClient(ReceiverID, orderIDs)
	} else {
		processingErr = ReturnOrderFromClient(ReceiverID, orderIDs)
	}

	if processingErr != nil {
		multiErrors := multierr.Errors(processingErr)
		for _, e := range multiErrors {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", e)
		}
		return fmt.Errorf("one or more orders failed to process. See above for details")
	}

	for _, orderID := range orderIDs {
		fmt.Printf("PROCESSED: %d\n", orderID)
	}

	return nil
}

func ListOrdersComm(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing ReceiverID. Usage: list-orders <ReceiverID> [in-pvz] [page] [limit]")
	}

	ReceiverID, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ReceiverID '%s': must be a number", args[0])
	}

	inPvz := false
	remainingArgs := []string{}

	if len(args) > 1 && args[1] == "in-pvz" {
		inPvz = true
		remainingArgs = args[2:]
	} else {
		remainingArgs = args[1:]
	}

	currentPage := uint64(1)
	itemsPerPage := uint64(10)
	if len(remainingArgs) > 0 {
		parsedPage, err := strconv.ParseUint(remainingArgs[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid page number '%s': must be a number", remainingArgs[0])
		}
		if parsedPage == 0 {
			return fmt.Errorf("page number cannot be 0")
		}
		currentPage = parsedPage
	}

	if len(remainingArgs) > 1 {
		parsedLimit, err := strconv.ParseUint(remainingArgs[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid limit '%s': must be a number", remainingArgs[1])
		}
		if parsedLimit == 0 {
			return fmt.Errorf("limit cannot be 0")
		}
		itemsPerPage = parsedLimit
	}

	var filteredOrders []*Order
	if userOrders, exists := ordersByReceiver[ReceiverID]; exists {
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

	if len(filteredOrders) == 0 {
		fmt.Println("No orders found for this receiver with the given criteria.")
		fmt.Printf("TOTAL: %d\n", 0)
		return nil
	}

	sort.Slice(filteredOrders, func(i, j int) bool {
		return filteredOrders[i].OrderID < filteredOrders[j].OrderID
	})
	paginatedOrders := paginate(filteredOrders, currentPage, itemsPerPage)

	totalItems := uint64(len(filteredOrders))

	if len(paginatedOrders) == 0 {
		fmt.Println("No orders found on this page for the given receiver and filters.")
	} else {
		fmt.Printf("Orders for Receiver ID %d:\n", ReceiverID)
		fmt.Println("-----------------------------------------------------")
		for _, order := range paginatedOrders {
			statusStr := getStatusString(order.Status)

			fmt.Printf("ORDER: %d %d %s %s\n",
				order.OrderID,
				order.ReceiverID,
				statusStr,
				order.StorageUntil.Format("2006-01-02_15:04"),
			)
		}
		fmt.Println("-----------------------------------------------------")
	}
	fmt.Printf("TOTAL: %d\n", totalItems)

	return nil
}

func AddComm(cmd *cobra.Command, args []string) error {
	if len(args) != expectedArgCnt {
		return fmt.Errorf("missing arguments. Usage: accept-order <ReceiverID> <OrderID> <StorageUntil>")
	}
	ReceiverID, err1 := strconv.ParseUint(args[0], 10, 64)
	OrderID, err2 := strconv.ParseUint(args[1], 10, 64)

	if err1 != nil {
		return fmt.Errorf("invalid ReceiverID '%s': must be a number", args[0])
	}
	if err2 != nil {
		return fmt.Errorf("invalid OrderID '%s': must be a number", args[1])
	}
	StorageUntil := args[2]
	err := Add(ReceiverID, OrderID, StorageUntil)
	if err != nil {
		return fmt.Errorf("failed to accept order: %w", err)
	}
	fmt.Printf("ORDER_ACCEPTED: %d\n", OrderID)
	return nil
}

func BackOrder(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("missing argument. Usage: return-order <OrderID>")
	}
	orderID, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid OrderID '%s': must be a number", args[0])
	}
	err2 := BackToDelivery(orderID)
	if err2 != nil {
		return fmt.Errorf("failed to return order: %w", err2)
	}
	fmt.Printf("ORDER_RETURNED: %d\n", orderID)
	return nil
}

func main() {
	// наверное есть смысл реализовать консольку через флаги, но я реализовал как в воркшопе
	rootCommand.AddCommand(
		&cobra.Command{
			Use:     "accept-order <ReceiverID> <OrderID> <YYYY-MM-DD_HH:MM>",
			Short:   "Accepts an order from a courier.",
			Example: "pvz> accept-order 123 101 2025-12-31_15:00",
			Args:    cobra.ExactArgs(3),
			RunE:    AddComm,
		},
	)
	rootCommand.AddCommand(
		&cobra.Command{
			Use:     "list-orders <ReceiverID> [in-pvz] [page] [limit]",
			Short:   "Lists orders for a specific receiver.",
			Example: "pvz> list-orders 123\npvz> list-orders 456 1 5",
			Args:    cobra.MinimumNArgs(1),
			RunE:    ListOrdersComm,
		},
	)
	rootCommand.AddCommand(
		&cobra.Command{
			Use:     "return-order <OrderID>",
			Short:   "Returns an order to the courier.",
			Example: "pvz> return-order 101",
			Args:    cobra.ExactArgs(1),
			RunE:    BackOrder,
		},
	)
	rootCommand.AddCommand(
		&cobra.Command{
			Use:     "process-orders <ReceiverID> <action> <OrderID1> [OrderID2...]",
			Short:   "Issues orders to a client or accepts returns from a client.",
			Example: "pvz> process-orders 123 issue 101 102\npvz> process-orders 123 return 103",
			Args:    cobra.MinimumNArgs(3),
			RunE:    ProcessOrders,
		},
	)
	rootCommand.AddCommand(
		&cobra.Command{
			Use:     "list-returns [page] [limit]",
			Short:   "Lists all returned orders.",
			Example: "pvz> list-returns\npvz> list-returns 1 5",
			Args:    cobra.MaximumNArgs(2),
			RunE:    GetReturnedOrders,
		},
	)
	rootCommand.AddCommand(
		&cobra.Command{
			Use:     "order-history",
			Short:   "Shows the history of all order status changes (sorted by last update time).",
			Example: "pvz> order-history",
			Args:    cobra.NoArgs,
			RunE:    GetOrdersSortedByTime,
		},
	)

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

		parts := strings.Fields(line)
		rootCommand.SetArgs(parts)

		if err := rootCommand.Execute(); err != nil {
			fmt.Fprintf(os.Stderr, "Command failed: %v\n", err)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error: %s", err)
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

func ReadOnlyPvz(ReceiverID uint64) string {
	var output strings.Builder
	orders := ordersByReceiver[ReceiverID]

	for orderID := range orders {
		order := ordersByID[orderID]
		if order.Status == StatusInStorage {
			fmt.Fprintf(&output, "Order: %d, Time Limit: %s\n",
				order.OrderID,
				order.StorageUntil.Format("2006-01-02_15:04"))
		}
	}
	return output.String()
}

func ReadAll(ReceiverID uint64) string {
	var output strings.Builder
	orders := ordersByReceiver[ReceiverID]

	for orderID := range orders {
		order := ordersByID[orderID]
		statusStr := getStatusString(order.Status)

		fmt.Fprintf(&output, "Order: %d, Time Limit: %s, Status: %s\n",
			order.OrderID,
			order.StorageUntil.Format("2006-01-02_15:04"),
			statusStr)
	}
	return output.String()
}
