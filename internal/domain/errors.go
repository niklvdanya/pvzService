package domain

import "fmt"

type ErrorCode int64

const (
	ErrorCodeNotFound               ErrorCode = 1
	ErrorCodeAlreadyExists          ErrorCode = 2
	ErrorCodeStorageExpired         ErrorCode = 3
	ErrorCodeValidationFailed       ErrorCode = 4
	ErrorCodeAlreadyGiven           ErrorCode = 5
	ErrorCodeBelongsToOtherReceiver ErrorCode = 6
	ErrorCodeAlreadyInStorage       ErrorCode = 7
	ErrorCodeReturnPeriodExpired    ErrorCode = 8
	ErrorCodeStorageNotExpired      ErrorCode = 9
	ErrorCodeUnavaliableReturned    ErrorCode = 10
	ErrorCodeNilOrder               ErrorCode = 11
)

type Error struct {
	Code    ErrorCode
	Message string
}

func (e Error) Error() string {
	return e.Message
}

func EntityNotFoundError(entityName, entityID string) error {
	return Error{
		Code:    ErrorCodeNotFound,
		Message: fmt.Sprintf("Entity %q with ID %s not found", entityName, entityID),
	}
}

func OrderAlreadyExistsError(orderID uint64) error {
	return Error{
		Code:    ErrorCodeAlreadyExists,
		Message: fmt.Sprintf("Order %d already exists", orderID),
	}
}

func StorageExpiredError(orderID uint64, storageUntil string) error {
	return Error{
		Code:    ErrorCodeStorageExpired,
		Message: fmt.Sprintf("Order %d storage period expired (%s)", orderID, storageUntil),
	}
}

func ValidationFailedError(message string) error {
	return Error{
		Code:    ErrorCodeValidationFailed,
		Message: message,
	}
}

func OrderAlreadyGivenError(orderID uint64) error {
	return Error{
		Code:    ErrorCodeAlreadyGiven,
		Message: fmt.Sprintf("Order %d already given to client", orderID),
	}
}

func BelongsToDifferentReceiverError(orderID, expectedReceiverID, actualReceiverID uint64) error {
	return Error{
		Code:    ErrorCodeBelongsToOtherReceiver,
		Message: fmt.Sprintf("Order %d belongs to a different receiver (expected %d, got %d)", orderID, expectedReceiverID, actualReceiverID),
	}
}

func AlreadyInStorageError(orderID uint64) error {
	return Error{
		Code:    ErrorCodeAlreadyInStorage,
		Message: fmt.Sprintf("Order %d is already in storage", orderID),
	}
}

func ReturnPeriodExpiredError(orderID uint64, hoursSinceGiven float64) error {
	return Error{
		Code:    ErrorCodeReturnPeriodExpired,
		Message: fmt.Sprintf("Order %d return period expired (%.1f hours)", orderID, hoursSinceGiven),
	}
}

func StorageNotExpiredError(orderID uint64, storageUntil string) error {
	return Error{
		Code:    ErrorCodeStorageNotExpired,
		Message: fmt.Sprintf("Order %d storage period not expired (until %s)", orderID, storageUntil),
	}
}

func UnavaliableReturnedOrderError(orderID uint64) error {
	return Error{
		Code:    ErrorCodeUnavaliableReturned,
		Message: fmt.Sprintf("Order %d is an unavailable returned order", orderID),
	}
}

func NilOrderError(orderID uint64) error {
	return Error{
		Code:    ErrorCodeNilOrder,
		Message: fmt.Sprintf("Order %d is nil", orderID),
	}
}
