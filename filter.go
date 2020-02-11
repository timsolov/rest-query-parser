package rqp

import (
	"strconv"
	"strings"
)

type Filter struct {
	name       string
	value      interface{}
	method     string
	expression string
}

func (f *Filter) setInt(list []string) error {
	if len(list) == 1 {
		if f.method != MethodEQ &&
			f.method != MethodNE &&
			f.method != MethodGT &&
			f.method != MethodLT &&
			f.method != MethodGTE &&
			f.method != MethodLTE {
			return ErrMethodNotAllowed
		}

		i, err := strconv.Atoi(list[0])
		if err != nil {
			return ErrBadFormat
		}
		f.value = i
	} else {
		if f.method != MethodIN {
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
		f.value = intSlice
	}
	return nil
}

func (f *Filter) setString(list []string) error {
	if len(list) == 1 {
		if f.method != MethodEQ &&
			f.method != MethodNE &&
			f.method != MethodLIKE &&
			f.method != MethodIN {
			return ErrMethodNotAllowed
		}
		f.value = list[0]
	} else {
		if f.method != MethodIN {
			return ErrMethodNotAllowed
		}
		f.value = list
	}
	return nil
}

func parseFilterKey(key string) (Filter, error) {

	f := Filter{
		method: MethodEQ,
	}

	spos := strings.Index(key, "[")
	if spos != -1 {
		f.name = key[:spos]
		epos := strings.Index(key[spos:], "]")
		if epos != -1 {
			// go inside brekets
			spos = spos + 1
			epos = spos + epos - 1

			if epos-spos > 0 {
				f.method = strings.ToUpper(key[spos:epos])
				if _, ok := TranslateMethods[f.method]; !ok {
					return f, ErrUnknownMethod
				}
			}
		}
	} else {
		f.name = key
	}

	return f, nil
}

func (p *QueryParser) parseFilterValue(filter Filter, _type string, value []string, validate ValidationFunc) error {
	if len(value) != 1 {
		return ErrSimilarNames
	}

	list := value
	if strings.Contains(value[0], p.delimiter) {
		list = strings.Split(value[0], p.delimiter)
	}

	switch _type {
	case "int":
		err := filter.setInt(list)
		if err != nil {
			return err
		}
		if validate != nil {
			if err := validate(filter.value); err != nil {
				return err
			}
		}
		p.Filters = append(p.Filters, filter)
	default: // str, string and all other unknown types will handle like string
		err := filter.setString(list)
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
		p.Filters = append(p.Filters, filter)
	}

	return nil
}
