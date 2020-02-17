package rqp

import (
	"fmt"
	"strconv"
	"strings"
)

type Filter struct {
	Name   string
	Value  interface{}
	Method Method
}

// Where returns condition expression
func (f *Filter) Where() (string, error) {
	var exp string

	switch f.Method {
	case EQ, NE, GT, LT, GTE, LTE, LIKE, ILIKE:
		exp = fmt.Sprintf("%s %s ?", f.Name, TranslateMethods[f.Method])
		return exp, nil
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
		if f.Method != EQ &&
			f.Method != NE &&
			f.Method != LIKE &&
			f.Method != ILIKE &&
			f.Method != IN {
			return ErrMethodNotAllowed
		}
		f.Value = list[0]
	} else {
		if f.Method != IN {
			return ErrMethodNotAllowed
		}
		f.Value = list
	}
	return nil
}

func parseFilterKey(key string) (*Filter, error) {

	f := &Filter{
		Method: EQ,
	}

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
					return f, ErrUnknownMethod
				}
			}
		}
	} else {
		f.Name = key
	}

	return f, nil
}

func (p *Query) parseFilterValue(f *Filter, fType string, value []string, validate ValidationFunc) error {
	if len(value) != 1 {
		return ErrSimilarNames
	}

	list := value
	if strings.Contains(value[0], p.delimiter) {
		list = strings.Split(value[0], p.delimiter)
	}

	switch fType {
	case "int":
		err := f.setInt(list)
		if err != nil {
			return err
		}
		if validate != nil {
			switch f.Value.(type) {
			case int:
				if err := validate(f.Value); err != nil {
					return err
				}
			case []int:
				for _, v := range f.Value.([]int) {
					if err := validate(v); err != nil {
						return err
					}
				}
			}
		}
		p.Filters = append(p.Filters, f)
	case "bool":
		err := f.setBool(list)
		if err != nil {
			return err
		}
		if validate != nil {
			switch f.Value.(type) {
			case bool:
				if err := validate(f.Value); err != nil {
					return err
				}
			}
		}
		p.Filters = append(p.Filters, f)
	default: // str, string and all other unknown types will handle as string
		err := f.setString(list)
		if err != nil {
			return err
		}
		if validate != nil {
			for _, v := range list {
				if err := validate(v); err != nil {
					return err
				}
			}
		}
		p.Filters = append(p.Filters, f)
	}

	return nil
}
