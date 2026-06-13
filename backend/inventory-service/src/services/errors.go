package services

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrInvalidInput       = errors.New("invalid input")
	ErrConflict           = errors.New("conflict")
	ErrInsufficientStock  = errors.New("insufficient stock")
	ErrInactiveIngredient = errors.New("inactive ingredient")
	ErrMissingConversion  = errors.New("missing unit conversion")
	ErrReadOnlyGlobalUOM  = errors.New("global uoms are read-only")
	ErrProductNotFound    = errors.New("product not found")
)
