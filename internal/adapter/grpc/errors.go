package server

import (
	"errors"
	"fmt"
	"strings"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	"gitlab.ozon.dev/safariproxd/homework/pkg/api"
	"go.uber.org/multierr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func MapErrorToGRPCStatus(err error) error {
	var domainErr domain.Error
	if errors.As(err, &domainErr) {
		switch domainErr.Code {
		case domain.ErrorCodeNotFound:
			return status.Error(codes.NotFound, domainErr.Message)
		case domain.ErrorCodeAlreadyExists:
			return status.Error(codes.AlreadyExists, domainErr.Message)
		case domain.ErrorCodeStorageExpired, domain.ErrorCodeStorageNotExpired:
			return status.Error(codes.FailedPrecondition, domainErr.Message)
		case domain.ErrorCodeValidationFailed, domain.ErrorCodeInvalidPackage, domain.ErrorCodeWeightTooHeavy:
			return status.Error(codes.InvalidArgument, domainErr.Message)
		default:
			return status.Error(codes.Internal, domainErr.Message)
		}
	}
	if multierr.Errors(err) != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return status.Error(codes.Internal, err.Error())
}

func processErrors(err error, orderIDs []uint64) (*api.ProcessResult, error) {
	var processed, errors []uint64
	multiErrs := multierr.Errors(err)
	if len(multiErrs) == 0 {
		return &api.ProcessResult{Processed: orderIDs}, nil
	}
	for _, orderID := range orderIDs {
		found := false
		for _, e := range multiErrs {
			if strings.Contains(e.Error(), fmt.Sprintf("Order %d", orderID)) {
				errors = append(errors, orderID)
				found = true
				break
			}
		}
		if !found {
			processed = append(processed, orderID)
		}
	}
	return &api.ProcessResult{Processed: processed, Errors: errors}, status.Error(codes.InvalidArgument, err.Error())
}

func processImportErrors(err error, orders []domain.OrderToImport) *api.ImportResult {
	var errors []uint64
	multiErrs := multierr.Errors(err)
	for _, order := range orders {
		for _, e := range multiErrs {
			if strings.Contains(e.Error(), fmt.Sprintf("Order %d", order.OrderID)) {
				errors = append(errors, order.OrderID)
				break
			}
		}
	}
	return &api.ImportResult{Imported: int32(len(orders) - len(errors)), Errors: errors}
}
