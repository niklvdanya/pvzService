package app

import "errors"

var (
	ErrOrderNotFound              = errors.New("order not found")
	ErrOrderAlreadyExists         = errors.New("order already exists")
	ErrOrderAlreadyGiven          = errors.New("order already given to client")
	ErrStorageExpired             = errors.New("storage period expired")
	ErrBelongsToDifferentReceiver = errors.New("order belongs to a different receiver")
	ErrAlreadyInStorage           = errors.New("order is already in storage")
	ErrReturnPeriodExpired        = errors.New("return period expired")
	ErrStorageNotExpired          = errors.New("storage period not expired yet")
	ErrUnavaliableReturnedOrder   = errors.New("returned orders are not available")
	// надо добавить еще
)
