package rqp

type ValidationFunc func(value interface{}) error

type Validations map[string]ValidationFunc

func In(values ...interface{}) ValidationFunc {
	return func(value interface{}) error {

		in := false

		for _, v := range values {
			if v == value {
				in = true
				break
			}
		}

		if !in {
			return ErrNotInScope
		}

		return nil
	}
}
