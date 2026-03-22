package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrConflict          = errors.New("conflict")
	ErrNotMergeQueueHead = errors.New("not at head of product merge queue")
	ErrInvalidInput      = errors.New("invalid input")
	ErrInvalidTransition = errors.New("invalid state transition")
	ErrBudgetExceeded    = errors.New("budget cap exceeded")
	ErrGateway           = errors.New("gateway error")
	ErrShipping          = errors.New("shipping error")
	// ErrShippingNonRetryable marks forge failures that should not be retried (e.g. bad auth). Often joined with ErrShipping.
	ErrShippingNonRetryable = errors.New("shipping error: non-retryable")
	ErrMergeConflict        = errors.New("merge conflict")
	ErrMergeShipBusy        = errors.New("merge queue lease held by another worker")
	ErrNotConfigured        = errors.New("not configured")
	ErrMergeGatesNotMet     = errors.New("merge gates not satisfied")
	// ErrProductAlreadyDeleted wraps ErrConflict when soft-delete is applied to an already-deleted product.
	ErrProductAlreadyDeleted = fmt.Errorf("%w: product already deleted", ErrConflict)
	// ErrProductNotDeleted wraps ErrConflict when restore is applied to an active product.
	ErrProductNotDeleted = fmt.Errorf("%w: product is not deleted", ErrConflict)
	// ErrStaleEntity wraps ErrConflict when an optimistic update sees a changed row version (e.g. updated_at).
	ErrStaleEntity = fmt.Errorf("%w: concurrent modification", ErrConflict)
	// ErrConvoyExists wraps ErrConflict when a second convoy is created for the same parent task (Mission Control rule).
	ErrConvoyExists = fmt.Errorf("%w: convoy already exists for this parent task", ErrConflict)
)
