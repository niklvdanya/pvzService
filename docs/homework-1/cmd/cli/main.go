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

type OrderStatus uint8

const (
	expectedArgCnt              = 3
	StatusInStorage OrderStatus = iota
	StatusGivenToClient
	StatusReturnedToCourier
)

type Order struct {
	OrderID      uint64
	ReceiverID   uint64
	StorageUntil time.Time
	Status       OrderStatus
	AcceptTime   time.Time
}

var (
	ordersByID       map[uint64]*Order   = make(map[uint64]*Order)
	ordersByReceiver map[uint64][]uint64 = make(map[uint64][]uint64)
)

var rootCommand cobra.Command = cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {},
}

func View(cmd *cobra.Command, args []string) error {
	RecieverID, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	fmt.Println(ReadAll(RecieverID))
	return nil
}
func main() {

	rootCommand.AddCommand(
		&cobra.Command{
			Use:     "add",
			Short:   "a",
			Example: "",
			Args:    cobra.ExactArgs(3),
			RunE:    Add,
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

func Add(cmd *cobra.Command, args []string) error {
	if len(args) != expectedArgCnt {
		return nil
	}

	ReceiverID, err1 := strconv.ParseUint(args[0], 10, 64)
	OrderID, err2 := strconv.ParseUint(args[1], 10, 64)
	storageUntilStr := args[2]
	if err1 != nil || err2 != nil {

	}
	storageUntil, err := time.Parse("2006-01-02", storageUntilStr)
	if err != nil {
		return fmt.Errorf("invalid time format, expected YYYY-MM-DD: %v", err)
	}
	if storageUntil.Before(time.Now()) {
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
		AcceptTime:   time.Now(),
	}
	ordersByReceiver[ReceiverID] = append(ordersByReceiver[ReceiverID], OrderID)
	return nil
}

func ReadOnlyPvz(ReceiverID uint64) string {
	var output strings.Builder
	orders := ordersByReceiver[ReceiverID]

	for _, orderID := range orders {
		order := ordersByID[orderID]
		if order.Status == StatusInStorage {
			fmt.Fprintf(&output, "Order: %d, Time Limit: %s\n",
				order.OrderID,
				order.StorageUntil.Format("2006-01-02"))
		}
	}
	return output.String()
}

func ReadAll(RecieverID uint64) string {
	var output strings.Builder
	orders := ordersByReceiver[RecieverID]

	for _, orderID := range orders {
		order := ordersByID[orderID]
		statusStr := ""
		switch order.Status {
		case StatusInStorage:
			statusStr = "In Storage"
		case StatusGivenToClient:
			statusStr = "Given to Client"
		case StatusReturnedToCourier:
			statusStr = "Returned to Courier"
		}

		fmt.Fprintf(&output, "Order: %d, Time Limit: %s, Status: %s\n",
			order.OrderID,
			order.StorageUntil.Format("2006-01-02"),
			statusStr)
	}
	return output.String()
}

func BackToDelivery(OrderID uint64) error {
	order, exists := ordersByID[OrderID]
	if !exists {
		return fmt.Errorf("order %d not found", OrderID)
	}

	if order.Status != StatusInStorage {
		return fmt.Errorf("order %d cannot be returned (wrong status)", OrderID)
	}
	if time.Now().Before(order.StorageUntil) {
		return fmt.Errorf("order %d cannot be returned (storage period not expired)", OrderID)
	}

	delete(ordersByID, OrderID)
	orders := ordersByReceiver[order.ReceiverID]
	for i, id := range orders {
		if id == OrderID {
			ordersByReceiver[order.ReceiverID] = append(orders[:i], orders[i+1:]...)
			break
		}
	}

	return nil
}
