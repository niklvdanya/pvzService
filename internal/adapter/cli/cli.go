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
	RegisterAcceptOrderCmd(rootCmd, a.appService)
	RegisterReturnOrderCmd(rootCmd, a.appService)
	RegisterProcessOrdersCmd(rootCmd, a.appService)
	RegisterListOrdersCmd(rootCmd, a.appService)
	RegisterGetReturnedOrdersCmd(rootCmd, a.appService)
	RegisterOrderHistoryCmd(rootCmd, a.appService)
	RegisterImportOrdersCmd(rootCmd, a.appService)
	RegisterScrollOrdersCmd(rootCmd, a.appService)
}
