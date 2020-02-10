package rqp

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrBadFormat        error = errors.New("bad format")
	ErrUnknownCmpMethod error = errors.New("unknown compare method")
	ErrMethodNotAllowed error = errors.New("not allowed method")
)

var (
	MethodEQ   string = "EQ"
	MethodNE   string = "NE"
	MethodGT   string = "GT"
	MethodLT   string = "LT"
	MethodLIKE string = "LIKE"
	MethodGTE  string = "GTE"
	MethodLTE  string = "LTE"
	MethodNOT  string = "NOT"
	MethodIN   string = "IN"

	TranslateMethods map[string]string = map[string]string{
		MethodEQ:   "=",
		MethodNE:   "!=",
		MethodGT:   ">",
		MethodLT:   "<",
		MethodGTE:  ">=",
		MethodLTE:  "<=",
		MethodNOT:  "NOT",
		MethodLIKE: "LIKE",
		MethodIN:   "IN",
	}
)

type ValidationFunc func(value interface{}) error

type QueryParser struct {
	Fields  []string
	Offset  int
	Limit   int
	Sort    []Sort
	Filters []*Filter

	delimiter string
}

type Sort struct {
	by   string
	desc bool
}

type Filter struct {
	name   string
	value  interface{}
	method string
}

func defaultQueryParser() *QueryParser {
	return &QueryParser{
		delimiter: ",",
	}
}

func (p *QueryParser) Delimiter(delimiter string) {
	p.delimiter = delimiter
}

func Parse(query map[string][]string, validations map[string]ValidationFunc) (p *QueryParser, err error) {

	p = defaultQueryParser()

	for key, value := range query {

		if strings.ToUpper(key) == "FIELDS" {
			if err = p.parseFields(value, validations[key]); err != nil {
				return nil, err
			}
		} else if strings.ToUpper(key) == "OFFSET" {
			if err = p.parseOffset(value, validations[key]); err != nil {
				return nil, err
			}
		} else if strings.ToUpper(key) == "LIMIT" {
			if err = p.parseLimit(value, validations[key]); err != nil {
				return nil, err
			}
		} else if strings.ToUpper(key) == "SORT" {
			if err = p.parseSort(value, validations[key]); err != nil {
				return nil, err
			}
		} else if len(key) > 7 && strings.ToUpper(key[:7]) == "FILTER[" {
			filter, err := parseFilterKey(key)
			if err != nil {
				return nil, err
			}
			validationFunc := validations[filter.name]
			_type := "string"
			for k, v := range validations {
				if strings.Contains(k, ":") {
					split := strings.Split(k, ":")
					if split[0] == filter.name {
						validationFunc = v
						_type = split[1]
					}
				}
			}

			if err = p.parseFilterValue(filter, _type, value, validationFunc); err != nil {
				return nil, err
			}

			fmt.Println("filter:", key, *filter, _type, validationFunc)
		}
	}

	return
}

func parseFilterKey(key string) (*Filter, error) {

	f := &Filter{}

	// skip "filter["
	key = key[7:]
	pos := strings.Index(key, "]")
	if pos == -1 {
		return nil, ErrBadFormat
	}

	// get variable name
	f.name = key[:pos]
	cmpMethod := MethodEQ
	// skip to text after "]"
	key = key[pos:]
	// get comparison method
	if len(key) > 0 && key[0] == '[' {
		// skip first "["
		key = key[1:]
		pos = strings.Index(key, "]")
		if pos == -1 {
			return nil, ErrBadFormat
		}
		cmpMethod = key[:pos]
	}

	if _, ok := TranslateMethods[strings.ToUpper(cmpMethod)]; !ok {
		return nil, ErrUnknownCmpMethod
	}
	f.method = cmpMethod

	return f, nil
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
			return err
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
				return err
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
			f.method != MethodLIKE {
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

func (p *QueryParser) parseFilterValue(filter *Filter, _type string, value []string, validate ValidationFunc) error {
	if len(value) != 1 {
		return ErrBadFormat
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

func (p *QueryParser) parseSort(value []string, validate ValidationFunc) error {
	if len(value) != 1 {
		return ErrBadFormat
	}

	list := value
	if strings.Contains(value[0], p.delimiter) {
		list = strings.Split(value[0], p.delimiter)
	}

	for _, v := range list {

		var (
			by   string
			desc bool
		)

		switch v[0] {
		case '-':
			by = v[1:]
			desc = true
		case '+':
			by = v[1:]
			desc = false
		default:
			by = v
			desc = false
		}

		if validate != nil {
			if err := validate(by); err != nil {
				return err
			}
		}

		p.Sort = append(p.Sort, Sort{
			by:   by,
			desc: desc,
		})
	}

	return nil
}

func (p *QueryParser) parseFields(value []string, validate ValidationFunc) error {
	if len(value) == 1 {
		list := value
		if strings.Contains(value[0], p.delimiter) {
			list = strings.Split(value[0], p.delimiter)
		}

		if validate != nil {
			for _, v := range list {
				if err := validate(v); err != nil {
					return err
				}
			}
		}

		p.Fields = list
		return nil
	}
	return ErrBadFormat
}

func (p *QueryParser) parseOffset(value []string, validate ValidationFunc) error {

	if len(value) != 1 {
		return ErrBadFormat
	}
	var err error

	p.Offset, err = strconv.Atoi(value[0])
	if err != nil {
		return err
	}

	if validate != nil {
		if err := validate(p.Offset); err != nil {
			return err
		}
	}

	return nil
}

func (p *QueryParser) parseLimit(value []string, validate ValidationFunc) error {

	if len(value) != 1 {
		return ErrBadFormat
	}

	var err error

	p.Limit, err = strconv.Atoi(value[0])
	if err != nil {
		return err
	}

	if validate != nil {
		if err := validate(p.Limit); err != nil {
			return err
		}
	}

	return nil
}

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
			return errors.New(fmt.Sprintf("%s: not in scope", value))
		}

		return nil
	}
}
