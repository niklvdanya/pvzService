package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// для генерации тестовых данных
type OrderData struct {
	OrderID      uint64 `json:"order_id"`
	ReceiverID   uint64 `json:"receiver_id"`
	StorageUntil string `json:"storage_until"`
}

const (
	numOrders           = 100
	outputFilePath      = "data/new_orders.json"
	baseReceiverID      = 100
	numReceiverIDs      = 10
	storageDurationDays = 365
)

func main() {
	fmt.Printf("Generating %d orders to %s...\n", numOrders, outputFilePath)

	orders := make([]OrderData, numOrders)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	now := time.Now()
	moscowLoc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		fmt.Printf("Warning: Failed to load Europe/Moscow location, using UTC: %v\n", err)
		moscowLoc = time.UTC
	}

	for i := 0; i < numOrders; i++ {
		orderID := uint64(i + 1)
		receiverID := baseReceiverID + uint64(r.Intn(numReceiverIDs))
		randomDays := r.Intn(storageDurationDays)
		storageUntil := now.Add(time.Duration(randomDays) * 24 * time.Hour)
		storageUntil = storageUntil.In(moscowLoc)

		orders[i] = OrderData{
			OrderID:      orderID,
			ReceiverID:   receiverID,
			StorageUntil: storageUntil.Format("2006-01-02_15:04"),
		}
	}

	if err := os.MkdirAll("data", 0755); err != nil {
		fmt.Printf("Error creating data directory: %v\n", err)
		os.Exit(1)
	}

	file, err := os.Create(outputFilePath)
	if err != nil {
		fmt.Printf("Error creating file %s: %v\n", outputFilePath, err)
		os.Exit(1)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(orders); err != nil {
		fmt.Printf("Error writing JSON to file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Data generation complete!")
}
