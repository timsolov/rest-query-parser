package rqp

import (
	"github.com/pkg/errors"
)

type ValidationFunc func(value interface{}) error

type Validations map[string]ValidationFunc

func In(values ...interface{}) ValidationFunc {
	return func(value interface{}) error {

		var (
			v  interface{}
			in bool = false
		)

		for _, v = range values {
			if v == value {
				in = true
				break
			}
		}

		if !in {
			switch v.(type) {
			case string:
				return errors.Wrap(ErrNotInScope, value.(string))
			case int:
				return errors.Wrapf(ErrNotInScope, "%d", value.(int))
			default:
				return ErrNotInScope
			}
		}

		return nil
	}
}
