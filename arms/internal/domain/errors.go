package domain

import (
	"errors"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrNotMergeQueueHead  = errors.New("not at head of product merge queue")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidTransition  = errors.New("invalid state transition")
	ErrBudgetExceeded     = errors.New("budget cap exceeded")
	ErrGateway            = errors.New("gateway error")
	ErrShipping           = errors.New("shipping error")
	ErrMergeConflict      = errors.New("merge conflict")
	ErrMergeShipBusy      = errors.New("merge queue lease held by another worker")
)
