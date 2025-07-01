package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// в domain особо нечего тестировать, но по условию ДЗ надо 40% покрытия слоев usecase и entity
func Test_Order_GetStatusString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		st   OrderStatus
		want string
	}{
		{StatusInStorage, "In Storage"},
		{StatusGivenToClient, "Given to client"},
		{StatusReturnedFromClient, "Returned from client"},
		{StatusGivenToCourier, "Given to courier"},
		{StatusReturnedWithoutClient, "Given to courier without client"},
		{99, "Unknown Status"},
	}

	for _, row := range tests {
		got := Order{Status: row.st}.GetStatusString()
		assert.Equal(t, row.want, got)
	}
}

func Test_Domain_ErrorHelpers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		code   ErrorCode
		substr string
	}{
		{"NotFound", EntityNotFoundError("order", "42"), ErrorCodeNotFound, "42"},
		{"AlreadyExists", OrderAlreadyExistsError(7), ErrorCodeAlreadyExists, "7"},
		{"StorageExpired", StorageExpiredError(9, "2025-06-30"), ErrorCodeStorageExpired, "2025-06-30"},
		{"WeightTooHeavy", WeightTooHeavyError("box", 12.5, 10), ErrorCodeWeightTooHeavy, "12.50"},
	}

	for _, tt := range tests {
		e := tt.err.(Error)
		assert.Equal(t, tt.code, e.Code, tt.name)
		assert.Contains(t, e.Message, tt.substr, tt.name)
	}
}
