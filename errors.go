package rqp

type Error struct {
	s string
}

func (e *Error) Error() string {
	return e.s
}

func NewError(msg string) *Error {
	return &Error{msg}
}

var (
	ErrRequired           = NewError("required")
	ErrBadFormat          = NewError("bad format")
	ErrEmptyValue         = NewError("empty value")
	ErrUnknownMethod      = NewError("unknown method")
	ErrNotInScope         = NewError("not in scope")
	ErrSimilarNames       = NewError("similar names of keys are not allowed")
	ErrMethodNotAllowed   = NewError("method are not allowed")
	ErrFilterNotAllowed   = NewError("filter are not allowed")
	ErrFilterNotFound     = NewError("filter not found")
	ErrValidationNotFound = NewError("validation not found")
)
