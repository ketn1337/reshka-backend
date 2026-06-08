package domain

import "errors"

// Доменные ошибки. handler-маппер переводит их в HTTP-коды.
var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrForbidden    = errors.New("forbidden")
	ErrUnauthorized = errors.New("unauthorized")
	ErrValidation   = errors.New("validation failed")
	ErrBadStatus    = errors.New("invalid status transition")
)
