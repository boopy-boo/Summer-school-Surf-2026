package domain

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrConflict        = errors.New("conflict")
	ErrGone            = errors.New("gone")
	ErrTooManyRequests = errors.New("too many requests")
	ErrInvalidOTP      = errors.New("invalid otp")
	ErrInternal        = errors.New("internal error")
	ErrBadRequest      = errors.New("bad request")
)