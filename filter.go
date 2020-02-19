package rqp

import (
	"fmt"
	"strconv"
	"strings"
)

type Filter struct {
	Key    string // key from URL (eg. "id[eq]")
	Name   string // name of filter, takes from Key (eg. "id")
	Method Method // compare method, takes from Key (eg. EQ)
	Value  interface{}
	Or     bool
}

// detectValidation
// name - only name without method
// validations - must be q.validations
func detectValidation(name string, validations Validations) (ValidationFunc, bool) {

	for k, v := range validations {
		if strings.Contains(k, ":") {
			split := strings.Split(k, ":")
			if split[0] == name {
				return v, true
			}
		} else if k == name {
			return v, true
		}
	}

	return nil, false
}

// detectType
func detectType(name string, validations Validations) string {

	for k, _ := range validations {
		if strings.Contains(k, ":") {
			split := strings.Split(k, ":")
			if split[0] == name {
				switch split[1] {
				case "int", "i":
					return "int"
				case "bool", "b":
					return "bool"
				default:
					return "string"
				}
			}
		}
	}

	return "string"
}

func isNotNull(f *Filter) bool {
	s, ok := f.Value.(string)
	if !ok {
		return false
	}
	return f.Method == NOT && strings.ToUpper(s) == NULL
}

// rawKey - url key
// value - must be one value (if need IN method then values must be separated by comma (,))
func newFilter(rawKey string, value string, delimiter string, validations Validations) (*Filter, error) {
	f := &Filter{
		Key: rawKey,
	}

	// set Key, Name, Method
	if err := f.parseKey(rawKey); err != nil {
		return nil, err
	}

	// detect have we validate definition on this parameter name or not
	validate, ok := detectValidation(f.Name, validations)
	if !ok {
		return nil, ErrValidationNotFound
	}

	// detect type by key names in validations
	valueType := detectType(f.Name, validations)

	if err := f.parseValue(valueType, value, delimiter); err != nil {
		return nil, err
	}

	if !isNotNull(f) && validate != nil {
		if err := f.validate(validate); err != nil {
			return nil, err
		}
	}

	return f, nil
}

func (f *Filter) validate(validate ValidationFunc) error {

	switch f.Value.(type) {
	case []int:
		for _, v := range f.Value.([]int) {
			err := validate(v)
			if err != nil {
				return err
			}
		}
	case []string:
		for _, v := range f.Value.([]string) {
			err := validate(v)
			if err != nil {
				return err
			}
		}
	case int, bool, string:
		err := validate(f.Value)
		if err != nil {
			return err
		}
	}

	return nil
}

// parseKey parses key from f.Key which must be set from url
// sets Name, Method
func (f *Filter) parseKey(key string) error {

	// default Method is EQ
	f.Method = EQ

	spos := strings.Index(key, "[")
	if spos != -1 {
		f.Name = key[:spos]
		epos := strings.Index(key[spos:], "]")
		if epos != -1 {
			// go inside brekets
			spos = spos + 1
			epos = spos + epos - 1

			if epos-spos > 0 {
				f.Method = Method(strings.ToUpper(string(key[spos:epos])))
				if _, ok := TranslateMethods[f.Method]; !ok {
					return ErrUnknownMethod
				}
			}
		}
	} else {
		f.Name = key
	}

	return nil
}

// parseValue parses value depends on its type
func (f *Filter) parseValue(valueType string, value string, delimiter string) error {

	var list []string

	if strings.Contains(value, delimiter) {
		list = strings.Split(value, delimiter)
	} else {
		list = append(list, value)
	}

	switch valueType {
	case "int":
		err := f.setInt(list)
		if err != nil {
			return err
		}
	case "bool":
		err := f.setBool(list)
		if err != nil {
			return err
		}
	default: // str, string and all other unknown types will handle as string
		err := f.setString(list)
		if err != nil {
			return err
		}
	}

	return nil
}

// Where returns condition expression
func (f *Filter) Where() (string, error) {
	var exp string

	switch f.Method {
	case EQ, NE, GT, LT, GTE, LTE, LIKE, ILIKE:
		exp = fmt.Sprintf("%s %s ?", f.Name, TranslateMethods[f.Method])
		return exp, nil
	case NOT:
		if f.Value == NULL {
			exp = fmt.Sprintf("%s %s NULL", f.Name, TranslateMethods[f.Method])
			return exp, nil
		}
		return exp, ErrUnknownMethod
	case IN:
		exp = fmt.Sprintf("%s %s (?)", f.Name, TranslateMethods[f.Method])
		exp, _, _ = in(exp, f.Value)
		return exp, nil
	default:
		return exp, ErrUnknownMethod
	}
}

// Args returns arguments slice depending on filter condition
func (f *Filter) Args() ([]interface{}, error) {

	args := make([]interface{}, 0)

	switch f.Method {
	case EQ, NE, GT, LT, GTE, LTE:
		args = append(args, f.Value)
		return args, nil
	case NOT:
		if f.Value == NULL {
			return args, nil
		}
		return nil, ErrUnknownMethod
	case LIKE, ILIKE:
		value := f.Value.(string)
		if len(value) >= 2 && strings.HasPrefix(value, "*") {
			value = "%" + value[1:]
		}
		if len(value) >= 2 && strings.HasSuffix(value, "*") {
			value = value[:len(value)-1] + "%"
		}
		args = append(args, value)
		return args, nil
	case IN:
		_, params, _ := in("?", f.Value)
		args = append(args, params...)
		return args, nil
	default:
		return nil, ErrUnknownMethod
	}
}

func (f *Filter) setInt(list []string) error {
	if len(list) == 1 {
		if f.Method != EQ &&
			f.Method != NE &&
			f.Method != GT &&
			f.Method != LT &&
			f.Method != GTE &&
			f.Method != LTE &&
			f.Method != IN {
			return ErrMethodNotAllowed
		}

		i, err := strconv.Atoi(list[0])
		if err != nil {
			return ErrBadFormat
		}
		f.Value = i
	} else {
		if f.Method != IN {
			return ErrMethodNotAllowed
		}
		intSlice := make([]int, len(list))
		for i, s := range list {
			v, err := strconv.Atoi(s)
			if err != nil {
				return ErrBadFormat
			}
			intSlice[i] = v
		}
		f.Value = intSlice
	}
	return nil
}

func (f *Filter) setBool(list []string) error {
	if len(list) == 1 {
		if f.Method != EQ {
			return ErrMethodNotAllowed
		}

		i, err := strconv.ParseBool(list[0])
		if err != nil {
			return ErrBadFormat
		}
		f.Value = i
	} else {
		return ErrMethodNotAllowed
	}
	return nil
}

func (f *Filter) setString(list []string) error {
	if len(list) == 1 {
		switch f.Method {
		case EQ, NE, LIKE, ILIKE, IN:
			f.Value = list[0]
			return nil
		case NOT:
			if strings.Compare(strings.ToUpper(list[0]), NULL) == 0 {
				f.Value = NULL
				return nil
			}
		default:
			return ErrMethodNotAllowed
		}
	} else {
		switch f.Method {
		case IN:
			f.Value = list
			return nil
		}
	}
	return ErrMethodNotAllowed
}
