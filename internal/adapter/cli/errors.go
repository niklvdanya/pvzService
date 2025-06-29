package cli

import (
	"errors"
	"fmt"
	"strings"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"go.uber.org/multierr"
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

func WeightTooHeavyError(message string) error {
	return fmt.Errorf("ERROR: WEIGHT_TOO_HEAVY: %s", message)
}

func InternalError(err error) error {
	return fmt.Errorf("INTERNAL ERROR: %w", err)
}

func mapError(err error) error {
	if err == nil {
		return nil
	}

	// для вложенных ошибок multierr
	if errs := multierr.Errors(err); len(errs) > 1 {
		var userMsgs []string
		for _, e := range errs {
			userMsgs = append(userMsgs, processSingleError(e).Error())
		}
		return fmt.Errorf("%s", strings.Join(userMsgs, "\n"))
	}

	return processSingleError(err)
}

func processSingleError(err error) error {
	if err == nil {
		return nil
	}

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
		case domain.ErrorCodeNilOrder:
			return ValidationFailedError(domainErr.Message)
		case domain.ErrorCodeInvalidPackage:
			return ValidationFailedError(domainErr.Message)
		case domain.ErrorCodeWeightTooHeavy:
			return WeightTooHeavyError(domainErr.Message)
		default:
			return InternalError(err)
		}
	}

	// для "сломанных" запросов по типу
	// accept-order --user-id --order-id=123 --expires 2025-05-31 --weight=10kg --price=1000
	msg := err.Error()
	switch {
	case strings.Contains(msg, "invalid argument") && strings.Contains(msg, "--user-id"):
		return ValidationFailedError("Invalid value for flag --user-id")
	case strings.Contains(msg, "invalid argument") && strings.Contains(msg, "--order-id"):
		return ValidationFailedError("Invalid value for flag --order-id")
	case strings.Contains(msg, "invalid argument") && strings.Contains(msg, "--weight"):
		return ValidationFailedError("Invalid value for flag --weight")
	case strings.Contains(msg, "invalid argument") && strings.Contains(msg, "--price"):
		return ValidationFailedError("Invalid value for flag --price")
	case strings.Contains(msg, "invalid argument"):
		return ValidationFailedError("Invalid value for one of the flags")
	case strings.Contains(msg, "flag needs an argument"):
		return ValidationFailedError("Missing value for a required flag")
	case strings.Contains(msg, "unknown flag"):
		return ValidationFailedError("Unknown flag")
	case strings.Contains(msg, "parsing"):
		return ValidationFailedError("Failed to parse flag value")
	}

	return InternalError(err)
}
