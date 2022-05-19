package rqp

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
)

type StateOR byte

const (
	NoOR StateOR = iota
	StartOR
	InOR
	EndOR
)

// Filter represents a filter defined in the query part of URL
type Filter struct {
	Key               string // key from URL (eg. "id[eq]")
	ParameterizedName string // after applying enhancements to allow nesting
	QueryName         string // name of filter, takes from Key (eg. "id")
	Method            Method // compare method, takes from Key (eg. EQ)
	Value             interface{}
	OR                StateOR
	DbField           DatabaseField
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
func detectType(queryName string, qdbm QueryDbMap) FieldType {
	dbf, err := detectDbField(queryName, qdbm)
	if err == nil {
		return dbf.Type
	}
	// for special filters, assume json
	// also safe bc json allowed methods are a subset of other types'
	return FieldTypeJson //, errors.New("could not find type")
}

// detectDbField
func detectDbField(queryName string, qdbm QueryDbMap) (DatabaseField, error) {
	if dbf, ok := qdbm[queryName]; ok {
		return dbf, nil
	}
	return DatabaseField{}, errors.New("could not find table")
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
func newFilter(rawKey string, value string, delimiter string, validations Validations, qdbm QueryDbMap) (*Filter, error) {
	f := &Filter{
		Key: rawKey,
	}

	// set Key, Name, Method
	if err := f.parseKey(rawKey); err != nil {
		return nil, err
	}

	// detect have we validator func definition on this parameter or not
	validate, _ := detectValidation(f.QueryName, validations)
	// if !ok {
	// 	return nil, ErrValidationNotFound
	// }

	// detect type by key names in validations
	valueType := detectType(f.QueryName, qdbm)

	if err := f.parseValue(valueType, value, delimiter); err != nil {
		return nil, err
	}

	if !isNotNull(f) && validate != nil {
		if err := f.validate(validate); err != nil {
			return nil, err
		}
	}

	dbField, err := detectDbField(f.QueryName, qdbm)
	if err != nil {
		return f, ErrValidationNotFound
	}
	f.ParameterizedName = getParameterizedName(dbField, qdbm)
	f.DbField = dbField

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

// parseKey parses key to set f.Name and f.Method
//   id[eq] -> f.Name = "id", f.Method = EQ
func (f *Filter) parseKey(key string) error {

	// default Method is EQ
	f.Method = EQ

	spos := strings.Index(key, "[")
	if spos != -1 {
		f.QueryName = key[:spos]
		epos := strings.Index(key[spos:], "]")
		if epos != -1 {
			// go inside brekets
			spos = spos + 1
			epos = spos + epos - 1

			if epos-spos > 0 {
				f.Method = Method(strings.ToUpper(string(key[spos:epos])))
				if _, ok := translateMethods[f.Method]; !ok {
					return ErrUnknownMethod
				}
			}
		}
	} else {
		f.QueryName = key
	}

	return nil
}

// parseValue parses value depends on its type
func (f *Filter) parseValue(valueType FieldType, value string, delimiter string) error {

	var list []string

	if strings.Contains(value, delimiter) {
		list = strings.Split(value, delimiter)
	} else {
		list = append(list, value)
	}

	switch valueType {
	case FieldTypeInt:
		err := f.setInt(list)
		if err != nil {
			return err
		}
	case FieldTypeBool:
		err := f.setBool(list)
		if err != nil {
			return err
		}
	case FieldTypeFloat:
		err := f.setFloat(list)
		if err != nil {
			return err
		}
	case FieldTypeCustom, FieldTypeJson:
		err := f.setCustom(list)
		if err != nil {
			return err
		}
	case FieldTypeTime:
		err := f.setTime(list)
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
	case EQ, NE, GT, LT, GTE, LTE, LIKE, ILIKE, NLIKE, NILIKE:
		exp = fmt.Sprintf("%s %s ?", f.ParameterizedName, translateMethods[f.Method])
		return exp, nil
	case IS, NOT:
		if f.Value == NULL {
			exp = fmt.Sprintf("%s %s NULL", f.ParameterizedName, translateMethods[f.Method])
			return exp, nil
		}
		return exp, ErrUnknownMethod
	case IN, NIN:
		exp = fmt.Sprintf("%s %s (?)", f.ParameterizedName, translateMethods[f.Method])
		exp, _, _ = in(exp, f.Value)
		return exp, nil
	case raw:
		return f.ParameterizedName, nil
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
	case IS, NOT:
		if f.Value == NULL {
			args = append(args, f.Value)
			return args, nil
		}
		return nil, ErrUnknownMethod
	case LIKE, ILIKE, NLIKE, NILIKE:
		value := f.Value.(string)
		if len(value) >= 2 && strings.HasPrefix(value, "*") {
			value = "%" + value[1:]
		}
		if len(value) >= 2 && strings.HasSuffix(value, "*") {
			value = value[:len(value)-1] + "%"
		}
		args = append(args, value)
		return args, nil
	case IN, NIN:
		_, params, _ := in("?", f.Value)
		args = append(args, params...)
		return args, nil
	case raw:
		return args, nil
	default:
		return nil, ErrUnknownMethod
	}
}

func (f *Filter) setInt(list []string) error {
	if len(list) == 1 {
		switch f.Method {
		case EQ, NE, GT, LT, GTE, LTE, IN, NIN:
			i, err := strconv.Atoi(list[0])
			if err != nil {
				return ErrBadFormat
			}
			f.Value = i
		case IS, NOT:
			if strings.Compare(strings.ToUpper(list[0]), NULL) == 0 {
				f.Value = NULL
				return nil
			} else {
				return ErrBadFormat
			}
		default:
			return ErrMethodNotAllowed
		}
	} else {
		if f.Method != IN && f.Method != NIN {
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

func (f *Filter) setFloat(list []string) error {
	if len(list) == 1 {
		switch f.Method {
		case EQ, NE, GT, LT, GTE, LTE, IN, NIN:
			i, err := strconv.ParseFloat(list[0], 64)
			if err != nil {
				return ErrBadFormat
			}
			f.Value = i
		case IS, NOT:
			if strings.Compare(strings.ToUpper(list[0]), NULL) == 0 {
				f.Value = NULL
				return nil
			} else {
				return ErrBadFormat
			}
		default:
			return ErrMethodNotAllowed
		}
	} else {
		if f.Method != IN && f.Method != NIN {
			return ErrMethodNotAllowed
		}
		floatSlice := make([]float64, len(list))
		for i, s := range list {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return ErrBadFormat
			}
			floatSlice[i] = v
		}
		f.Value = floatSlice
	}
	return nil
}

func (f *Filter) setBool(list []string) error {
	if len(list) == 1 {
		switch f.Method {
		case EQ, NE:
			i, err := strconv.ParseBool(list[0])
			if err != nil {
				return ErrBadFormat
			}
			f.Value = i
		case IS, NOT:
			if strings.Compare(strings.ToUpper(list[0]), NULL) == 0 {
				f.Value = NULL
				return nil
			} else {
				return ErrBadFormat
			}
		default:
			return ErrMethodNotAllowed
		}
	} else {
		return ErrMethodNotAllowed
	}
	return nil
}

func (f *Filter) setString(list []string) error {
	if len(list) == 1 {
		switch f.Method {
		case EQ, NE, GT, LT, GTE, LTE, LIKE, ILIKE, NLIKE, NILIKE, IN, NIN:
			f.Value = list[0]
			return nil
		case IS, NOT:
			if strings.Compare(strings.ToUpper(list[0]), NULL) == 0 {
				f.Value = NULL
				return nil
			} else {
				return ErrBadFormat
			}
		default:
			return ErrMethodNotAllowed
		}
	} else {
		switch f.Method {
		case IN, NIN:
			f.Value = list
			return nil
		}
	}
	return ErrMethodNotAllowed
}

func (f *Filter) setCustom(list []string) error {
	if len(list) == 1 {
		switch f.Method {
		case IS, NOT:
			if strings.Compare(strings.ToUpper(list[0]), NULL) == 0 {
				f.Value = NULL
				return nil
			} else {
				return ErrBadFormat
			}
		}
	}
	return ErrMethodNotAllowed
}

func (f *Filter) setTime(list []string) error {
	if len(list) == 1 {
		switch f.Method {
		case EQ, NE, GT, LT, GTE, LTE, IN, NIN:
			t, err := dateparse.ParseAny(list[0])
			if err != nil {
				return ErrBadFormat
			}
			f.Value = t.UTC().Format(time.RFC3339)
			return nil
		case IS, NOT:
			if strings.Compare(strings.ToUpper(list[0]), NULL) == 0 {
				f.Value = NULL
				return nil
			} else {
				return ErrBadFormat
			}
		default:
			return ErrMethodNotAllowed
		}
	} else {
		switch f.Method {
		case IN, NIN:
			f.Value = list
			return nil
		}
	}
	return ErrMethodNotAllowed
}
