package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/config"
	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

const (
	numOrders           = 100
	baseReceiverID      = 100
	numReceiverIDs      = 10
	storageDurationDays = 365
)

var packageTypes = []string{"bag", "box", "film", "bag+film", "box+film"}

func main() {
	cfg := config.Default()
	outputFilePath := cfg.OrdersOutputFile
	fmt.Printf("Generating %d orders to %s...\n", numOrders, outputFilePath)

	orders := make([]domain.OrderToImport, numOrders)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	now := time.Now()

	for i := 0; i < numOrders; i++ {
		orderID := uint64(i + 1)
		receiverID := baseReceiverID + uint64(r.Intn(numReceiverIDs))
		randomDays := r.Intn(storageDurationDays)
		storageUntil := now.Add(time.Duration(randomDays) * 24 * time.Hour)

		weight := 0.1 + r.Float64()*19.9
		price := 100.0 + r.Float64()*4900.0
		packageType := packageTypes[r.Intn(len(packageTypes))]

		orders[i] = domain.OrderToImport{
			OrderID:      orderID,
			ReceiverID:   receiverID,
			StorageUntil: storageUntil.Format("2006-01-02"),
			PackageType:  packageType,
			Weight:       weight,
			Price:        price,
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
