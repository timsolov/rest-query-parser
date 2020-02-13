package rqp

import "errors"

var (
	ErrRequired         error = errors.New("required")
	ErrBadFormat        error = errors.New("bad format")
	ErrUnknownMethod    error = errors.New("unknown method")
	ErrNotInScope       error = errors.New("not in scope")
	ErrSimilarNames     error = errors.New("similar names of keys are not allowed")
	ErrMethodNotAllowed error = errors.New("method are not allowed")
	ErrFilterNotAllowed error = errors.New("filter are not allowed")
	ErrFilterNotFound   error = errors.New("filter not found")
)
