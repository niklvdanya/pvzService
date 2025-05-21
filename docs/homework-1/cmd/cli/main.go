package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
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
	StatusReturnedFromClient
)

// добавил часы и минуты, чтобы было легче отлаживать работу функций, где есть проверка с time.Now()
// для синхронизации с московским временем добавил
var moscowTime, _ = time.LoadLocation("Europe/Moscow")

type Order struct {
	OrderID      uint64
	ReceiverID   uint64
	StorageUntil time.Time
	Status       OrderStatus
	AcceptTime   time.Time
}

var (
	ordersByID       map[uint64]*Order              = make(map[uint64]*Order)
	ordersByReceiver map[uint64]map[uint64]struct{} = make(map[uint64]map[uint64]struct{})
	//надо наверное чуть более подробную структуру завести
	returnedOrders map[uint64]struct{} = make(map[uint64]struct{})
)

var rootCommand cobra.Command = cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {},
}

func GiveOrdersToClient(receiverID uint64, orderIDs []uint64) error {
	var combinedErr error

	for _, orderID := range orderIDs {
		var currentOrderErr error

		order, exists := ordersByID[orderID]
		if !exists {
			currentOrderErr = fmt.Errorf("order %d not found", orderID)
		} else if order.ReceiverID != receiverID {
			currentOrderErr = fmt.Errorf("order %d belongs to a different receiver (expected %d, got %d)", orderID, receiverID, order.ReceiverID)
		} else if order.Status == StatusGivenToClient {
			currentOrderErr = fmt.Errorf("order %d already given to client", orderID)
		} else if time.Now().In(moscowTime).After(order.StorageUntil) {
			currentOrderErr = fmt.Errorf("order %d storage period expired, order cannot be given", orderID)
		}

		if currentOrderErr != nil {
			combinedErr = multierr.Append(combinedErr, currentOrderErr)
			continue
		}
		// ничего кроме изменения статуса мы не делаем, ибо клиент может вернуть заказ, соответственно его еще надо хранить
		order.Status = StatusGivenToClient
	}

	return combinedErr
}

func ReturnOrderFromClient(receiverID uint64, orderIDs []uint64) error {
	var combinedErr error

	for _, orderID := range orderIDs {
		var currentOrderErr error

		order, exists := ordersByID[orderID]
		if !exists {
			currentOrderErr = fmt.Errorf("order %d not found", orderID)
		} else if order.ReceiverID != receiverID {
			currentOrderErr = fmt.Errorf("order %d belongs to a different receiver (expected %d, got %d)", orderID, receiverID, order.ReceiverID)
		} else if order.Status == StatusInStorage {
			currentOrderErr = fmt.Errorf("order %d already in storage", orderID)
		} else {
			currentTimeInMoscow := time.Now().In(moscowTime)
			timeInStorage := currentTimeInMoscow.Sub(order.AcceptTime)

			twoDaysLimit := 48 * time.Hour

			if timeInStorage > twoDaysLimit {
				currentOrderErr = fmt.Errorf("order %d has been in storage for too long (%.1f hours), cannot be given",
					orderID, timeInStorage.Hours())
			}
		}

		if currentOrderErr != nil {
			combinedErr = multierr.Append(combinedErr, currentOrderErr)
			continue
		}
		delete(ordersByID, orderID)
		delete(ordersByReceiver[order.ReceiverID], orderID)
		returnedOrders[orderID] = struct{}{}
	}

	return combinedErr
}
func GetReturnedOrders(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("expected number of args - %d, got - %d", 0, len(args))
	}
	for orderId, _ := range returnedOrders {
		fmt.Println(orderId)
	}
	return nil
}
func ProcessOrders(cmd *cobra.Command, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("expected number of args - %d, got - %d", 2, len(args))
	}
	ReceiverID, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("wrong string conversion in ProcessOrders: %s", err)
	}
	action := args[1]
	if action != "issue" && action != "return" {
		return fmt.Errorf("action must be 'issue' or 'return'")
	}
	orderIDs := make([]uint64, 0, len(args)-2)
	for i := 2; i < len(args); i++ {
		orderID, err := strconv.ParseUint(args[i], 10, 64)
		if err != nil {
			return fmt.Errorf("wrong string conversion in ProcessOrders: %s", err)
		}
		orderIDs = append(orderIDs, orderID)
	}

	if action == "issue" {
		err := GiveOrdersToClient(ReceiverID, orderIDs)
		if err != nil {
			return fmt.Errorf("cannot issue orders: %s", err)
		}
	} else {
		err := ReturnOrderFromClient(ReceiverID, orderIDs)
		if err != nil {
			return fmt.Errorf("cannot return orders: %s", err)
		}
	}

	return nil
}
func View(cmd *cobra.Command, args []string) error {
	ReceiverID, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	fmt.Println(ReadAll(ReceiverID))
	return nil
}

func AddComm(cmd *cobra.Command, args []string) error {
	if len(args) != expectedArgCnt {
		return fmt.Errorf("expected number of arguments - %d, got - %d", expectedArgCnt, len(args))
	}
	ReceiverID, err1 := strconv.ParseUint(args[0], 10, 64)
	OrderID, err2 := strconv.ParseUint(args[1], 10, 64)

	if err1 != nil || err2 != nil {
		return fmt.Errorf("wrong string conversion")
	}
	StorageUntil := args[2]
	err := Add(ReceiverID, OrderID, StorageUntil)
	if err != nil {
		return fmt.Errorf("cannot add to pvz: %s", err)
	}
	return nil

}
func BackOrder(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected number of args - %d, got - %d", 1, len(args))
	}
	orderID, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("wrong string conversation in BackOrder: %s", err)
	}
	err2 := BackToDelivery(orderID)
	if err2 != nil {
		return fmt.Errorf("cannot back order: %s", err2)
	}
	return nil
}

func main() {

	rootCommand.AddCommand(
		&cobra.Command{
			Use:     "accept-order",
			Short:   "a",
			Example: "",
			Args:    cobra.ExactArgs(3),
			RunE:    AddComm,
		},
	)
	rootCommand.AddCommand(
		&cobra.Command{
			Use:     "list-orders",
			Short:   "lo",
			Example: "",
			Args:    cobra.ExactArgs(1),
			RunE:    View,
		},
	)
	rootCommand.AddCommand(
		&cobra.Command{
			Use:     "return-order",
			Short:   "ro",
			Example: "",
			Args:    cobra.ExactArgs(1),
			RunE:    BackOrder,
		},
	)
	rootCommand.AddCommand(
		&cobra.Command{
			Use:     "process-orders",
			Short:   "po",
			Example: "",
			Args:    cobra.MinimumNArgs(3),
			RunE:    ProcessOrders,
		},
	)
	rootCommand.AddCommand(
		&cobra.Command{
			Use:     "list-returns",
			Short:   "lr",
			Example: "",
			Args:    cobra.NoArgs,
			RunE:    GetReturnedOrders,
		},
	)
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
			fmt.Fprintln(os.Stderr, "Command failed, Input Error: %s", err)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error: %s", err)
	}

}

func Add(ReceiverID, OrderID uint64, storageUntilStr string) error {
	storageUntil, err := time.ParseInLocation("2006-01-02_15:04", storageUntilStr, moscowTime)
	if err != nil {
		return fmt.Errorf("invalid time format, expected YYYY-MM-DD_HH:MM: %v", err)
	}

	currentTimeInMoscow := time.Now().In(moscowTime)
	// москвоское время ставим, иначе будут проблемы с поясами
	if storageUntil.Before(currentTimeInMoscow) {
		return fmt.Errorf("storage period already expired")
	}
	if _, exists := ordersByID[OrderID]; exists {
		return fmt.Errorf("order %d already exists", OrderID)
	}
	ordersByID[OrderID] = &Order{
		OrderID:      OrderID,
		ReceiverID:   ReceiverID,
		StorageUntil: storageUntil,
		Status:       StatusInStorage,
		AcceptTime:   currentTimeInMoscow,
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
		return fmt.Errorf("order %d not found", OrderID)
	}

	if order.Status != StatusInStorage && order.Status != StatusReturnedFromClient {
		return fmt.Errorf("order %d cannot be returned (wrong status)", OrderID)
	}
	if time.Now().In(moscowTime).Before(order.StorageUntil) {
		return fmt.Errorf("order %d cannot be returned (storage period not expired)", OrderID)
	}

	delete(ordersByID, OrderID)

	delete(ordersByReceiver[order.ReceiverID], OrderID)

	return nil
}

func ReadOnlyPvz(ReceiverID uint64) string {
	var output strings.Builder
	orders := ordersByReceiver[ReceiverID]

	for orderID, _ := range orders {
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

	for orderID, _ := range orders {
		order := ordersByID[orderID]
		statusStr := ""
		switch order.Status {
		case StatusInStorage:
			statusStr = "In Storage"
		case StatusGivenToClient:
			statusStr = "Given to Client"
		case StatusReturnedFromClient:
			statusStr = "Returned from Client"
		// по идее такого быть не может, но на всякий случай
		default:
			statusStr = "Unknown Status"
		}

		fmt.Fprintf(&output, "Order: %d, Time Limit: %s, Status: %s\n",
			order.OrderID,
			order.StorageUntil.Format("2006-01-02_15:04"),
			statusStr)
	}
	return output.String()
}
