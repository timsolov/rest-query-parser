package rqp

import (
	"errors"
	"fmt"
	"reflect"
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
func (q *Query) detectType(queryName string) FieldType {
	dbf, err := q.detectDbField(queryName)
	if err == nil {
		return dbf.Type
	}
	if val, ok := q.allowedNonDbFields[queryName]; ok {
		return val
	}

	// assume string
	// safe bc string allowed methods are a subset of other types'
	// and parse string will always work
	return FieldTypeString //, errors.New("could not find type")
}

// detectDbField
func (q *Query) detectDbField(queryName string) (DatabaseField, error) {
	if dbf, ok := q.queryDbFieldMap[queryName]; ok {
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
func (q *Query) newFilter(rawKey string, value string, delimiter string, validations Validations, qdbm QueryDbMap, allowedNonDbFields map[string]FieldType) (*Filter, error) {
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
	valueType := q.detectType(f.QueryName)

	if err := f.parseValue(valueType, value, delimiter); err != nil {
		return nil, err
	}

	if !isNotNull(f) && validate != nil {
		if err := f.validate(validate); err != nil {
			return nil, err
		}
	}

	dbField, err := q.detectDbField(f.QueryName)
	if err != nil {
		return f, ErrValidationNotFound
	}
	f.ParameterizedName = q.getParameterizedName(dbField)
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
//
//	id[eq] -> f.Name = "id", f.Method = EQ
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
		if err := f.setInt(list); err != nil {
			return err
		}
	case FieldTypeBool:
		if err := f.setBool(list); err != nil {
			return err
		}
	case FieldTypeFloat:
		if err := f.setFloat(list); err != nil {
			return err
		}
	case FieldTypeObjectArray:
		if err := f.setObjectArray(list); err != nil {
			return err
		}
	case FieldTypeIntArray:
		if err := f.setIntArray(list); err != nil {
			return err
		}
	case FieldTypeStringArray:
		if err := f.setStringArray(list); err != nil {
			return err
		}
	case FieldTypeFloatArray:
		if err := f.setFloatArray(list); err != nil {
			return err
		}
	case FieldTypeCustom, FieldTypeJson, FieldTypeObject:
		if err := f.setNullCheckable(list); err != nil {
			return err
		}
	case FieldTypeTime:
		if err := f.setTime(list); err != nil {
			return err
		}
	default: // str, string and all other unknown types will handle as string
		if err := f.setString(list); err != nil {
			return err
		}
	}

	return nil
}

func (f *Filter) buildStringArrayStr() string {
	vs := f.Value.([]string)
	sb := &strings.Builder{}
	sb.WriteString("'{")
	for i, v := range vs {
		if i > 0 {
			sb.WriteRune(',')
		}
		sb.WriteString(v)
	}
	sb.WriteString("}'")
	return sb.String()
}

func (f *Filter) buildIntArrayStr() string {
	vs := f.Value.([]int)
	sb := &strings.Builder{}
	sb.WriteString("'{")
	for i, v := range vs {
		if i > 0 {
			sb.WriteRune(',')
		}
		sb.WriteString(strconv.Itoa(v))
	}
	sb.WriteString("}'")
	return sb.String()
}

func (f *Filter) buildFloatArrayStr() string {
	vs := f.Value.([]float64)
	sb := &strings.Builder{}
	sb.WriteString("'{")
	for i, v := range vs {
		if i > 0 {
			sb.WriteRune(',')
		}
		sb.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
	}
	sb.WriteString("}'")
	return sb.String()
}

// Where returns condition expression
func (f *Filter) Where() (string, error) {
	var exp string

	switch f.Method {
	case EQ, NE:
		switch f.DbField.Type {
		case FieldTypeStringArray:
			arrayStr := f.buildStringArrayStr()
			exp = fmt.Sprintf("%s @> %s AND %s <@ %s", f.ParameterizedName, arrayStr, f.ParameterizedName, arrayStr)
		case FieldTypeIntArray:
			arrayStr := f.buildIntArrayStr()
			exp = fmt.Sprintf("%s @> %s AND %s <@ %s", f.ParameterizedName, arrayStr, f.ParameterizedName, arrayStr)
		case FieldTypeFloatArray:
			arrayStr := f.buildFloatArrayStr()
			exp = fmt.Sprintf("%s @> %s AND %s <@ %s", f.ParameterizedName, arrayStr, f.ParameterizedName, arrayStr)
		default:
			exp = fmt.Sprintf("%s %s ?", f.ParameterizedName, translateMethods[f.Method])
		}
		return exp, nil
	case GT, LT, GTE, LTE, LIKE, ILIKE, NLIKE, NILIKE:
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
	case EQ, NE:
		switch reflect.ValueOf(f.Value).Kind() {
		case reflect.Slice:
			// arrays are handled uniquely in Where(), without args
			return args, nil
		default:
			args = append(args, f.Value)
			return args, nil
		}
	case GT, LT, GTE, LTE:
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
		switch f.Method {
		case IN, NIN, EQ, NE:
			intSlice := make([]int, len(list))
			for i, s := range list {
				v, err := strconv.Atoi(s)
				if err != nil {
					return ErrBadFormat
				}
				intSlice[i] = v
			}
			f.Value = intSlice
		default:
			return ErrMethodNotAllowed
		}
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
		switch f.Method {
		case IN, NIN, EQ, NE:
			floatSlice := make([]float64, len(list))
			for i, s := range list {
				v, err := strconv.ParseFloat(s, 64)
				if err != nil {
					return ErrBadFormat
				}
				floatSlice[i] = v
			}
			f.Value = floatSlice
		default:
			return ErrMethodNotAllowed
		}
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
		case IN, NIN, EQ, NE:
			f.Value = list
		default:
			return ErrMethodNotAllowed
		}
	}
	return nil
}

func (f *Filter) setIntArray(list []string) error {
	switch f.Method {
	case EQ, NE:
		intSlice := make([]int, len(list))
		for i, s := range list {
			v, err := strconv.Atoi(s)
			if err != nil {
				return ErrBadFormat
			}
			intSlice[i] = v
		}
		f.Value = intSlice
		return nil
	case IS, NOT:
		if len(list) == 1 && strings.Compare(strings.ToUpper(list[0]), NULL) == 0 {
			f.Value = NULL
			return nil
		} else {
			return ErrBadFormat
		}
	default:
		return ErrMethodNotAllowed
	}
}

func (f *Filter) setStringArray(list []string) error {
	switch f.Method {
	case EQ, NE:
		f.Value = list
		return nil
	case IS, NOT:
		if len(list) == 1 && strings.Compare(strings.ToUpper(list[0]), NULL) == 0 {
			f.Value = NULL
			return nil
		} else {
			return ErrBadFormat
		}
	default:
		return ErrMethodNotAllowed
	}
}

func (f *Filter) setFloatArray(list []string) error {
	switch f.Method {
	case EQ, NE:
		floatSlice := make([]float64, len(list))
		for i, s := range list {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return ErrBadFormat
			}
			floatSlice[i] = v
		}
		f.Value = floatSlice
		return nil
	case IS, NOT:
		if len(list) == 1 && strings.Compare(strings.ToUpper(list[0]), NULL) == 0 {
			f.Value = NULL
			return nil
		} else {
			return ErrBadFormat
		}
	default:
		return ErrMethodNotAllowed
	}
}

func (f *Filter) setObjectArray(list []string) error {
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

func (f *Filter) setNullCheckable(list []string) error {
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
		case IN, NIN, EQ, NE:
			f.Value = list
			return nil
		}
	}
	return ErrMethodNotAllowed
}
