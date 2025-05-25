package cli

import (
	"errors"
	"fmt"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
)

func NotFoundErrorError(message string) error {
	return fmt.Errorf("ERROR: ORDER_NOT_FOUND: %s", message)
}

func AlreadyExistsError(message string) error {
	return fmt.Errorf("ERROR: ORDER_ALREADY_EXISTS: %s", message)
}

func StorageExpiredError(message string) error {
	return fmt.Errorf("ERROR: STORAGE_EXPIRED: %s", message)
}

func ValidationFailedError(message string) error {
	return fmt.Errorf("ERROR: VALIDATION_FAILED: %s", message)
}

func StorageNotExpiredError(message string) error {
	return fmt.Errorf("ERROR: STORAGE_NOT_EXPIRED: %s", message)
}

func InternalError(err error) error {
	return fmt.Errorf("ERROR: unexpected error: %w", err)
}

func mapError(err error) error {
	var domainErr domain.Error
	if errors.As(err, &domainErr) {
		switch domainErr.Code {
		case domain.ErrorCodeNotFound:
			return NotFoundErrorError(domainErr.Message)
		case domain.ErrorCodeAlreadyExists:
			return AlreadyExistsError(domainErr.Message)
		case domain.ErrorCodeStorageExpired:
			return StorageExpiredError(domainErr.Message)
		case domain.ErrorCodeValidationFailed:
			return ValidationFailedError(domainErr.Message)
		case domain.ErrorCodeAlreadyGiven:
			return ValidationFailedError(domainErr.Message)
		case domain.ErrorCodeBelongsToOtherReceiver:
			return ValidationFailedError(domainErr.Message)
		case domain.ErrorCodeAlreadyInStorage:
			return ValidationFailedError(domainErr.Message)
		case domain.ErrorCodeReturnPeriodExpired:
			return ValidationFailedError(domainErr.Message)
		case domain.ErrorCodeStorageNotExpired:
			return StorageNotExpiredError(domainErr.Message)
		case domain.ErrorCodeUnavaliableReturned:
			return ValidationFailedError(domainErr.Message)
		default:
			return InternalError(err)
		}
	}
	return InternalError(err)
}
