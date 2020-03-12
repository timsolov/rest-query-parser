package rqp

import "errors"

type Error error

func NewError(msg string) Error {
	return errors.New(msg)
}

var (
	ErrRequired           = NewError("required")
	ErrBadFormat          = NewError("bad format")
	ErrUnknownMethod      = NewError("unknown method")
	ErrNotInScope         = NewError("not in scope")
	ErrSimilarNames       = NewError("similar names of keys are not allowed")
	ErrMethodNotAllowed   = NewError("method are not allowed")
	ErrFilterNotAllowed   = NewError("filter are not allowed")
	ErrFilterNotFound     = NewError("filter not found")
	ErrValidationNotFound = NewError("validation not found")
)
