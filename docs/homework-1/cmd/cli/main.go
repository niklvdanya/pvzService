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
)

// TODO: error handling, paging, Postgre - ??, interface segregation, type aliases

type OrderStatus uint8

const (
	expectedArgCnt              = 3
	StatusInStorage OrderStatus = iota
	StatusGivenToClient
)

// для синхронизации с московским временем
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
)

var rootCommand cobra.Command = cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {},
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
			Use:     "add",
			Short:   "a",
			Example: "",
			Args:    cobra.ExactArgs(3),
			RunE:    AddComm,
		},
	)
	rootCommand.AddCommand(
		&cobra.Command{
			Use:     "view",
			Short:   "v",
			Example: "",
			Args:    cobra.ExactArgs(1),
			RunE:    View,
		},
	)
	rootCommand.AddCommand(
		&cobra.Command{
			Use:     "backDel",
			Short:   "bD",
			Example: "",
			Args:    cobra.ExactArgs(1),
			RunE:    BackOrder,
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

	if order.Status != StatusInStorage {
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
		}

		fmt.Fprintf(&output, "Order: %d, Time Limit: %s, Status: %s\n",
			order.OrderID,
			order.StorageUntil.Format("2006-01-02_15:04"),
			statusStr)
	}
	return output.String()
}
