package rqp

import (
	"strconv"
	"strings"
)

type Filter struct {
	Name       string
	Value      interface{}
	Method     string
	Expression string
}

func (f *Filter) setInt(list []string) error {
	if len(list) == 1 {
		if f.Method != MethodEQ &&
			f.Method != MethodNE &&
			f.Method != MethodGT &&
			f.Method != MethodLT &&
			f.Method != MethodGTE &&
			f.Method != MethodLTE &&
			f.Method != MethodIN {
			return ErrMethodNotAllowed
		}

		i, err := strconv.Atoi(list[0])
		if err != nil {
			return ErrBadFormat
		}
		f.Value = i
	} else {
		if f.Method != MethodIN {
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

func (f *Filter) setString(list []string) error {
	if len(list) == 1 {
		if f.Method != MethodEQ &&
			f.Method != MethodNE &&
			f.Method != MethodLIKE &&
			f.Method != MethodIN {
			return ErrMethodNotAllowed
		}
		f.Value = list[0]
	} else {
		if f.Method != MethodIN {
			return ErrMethodNotAllowed
		}
		f.Value = list
	}
	return nil
}

func parseFilterKey(key string) (Filter, error) {

	f := Filter{
		Method: MethodEQ,
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
				f.Method = strings.ToUpper(key[spos:epos])
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

func (p *Query) parseFilterValue(f Filter, fType string, value []string, validate ValidationFunc) error {
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
	default: // str, string and all other unknown types will handle like string
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
