package rqp

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Query the main struct of package
type Query struct {
	query       map[string][]string
	validations Validations

	Fields  []string
	Offset  int
	Limit   int
	Sorts   []Sort
	Filters []*Filter

	delimiterIN   string
	delimiterOR   string
	ignoreUnknown bool

	Error error
}

// Method is a compare method type
type Method string

// Compare methods:
var (
	EQ     Method = "EQ"
	NE     Method = "NE"
	GT     Method = "GT"
	LT     Method = "LT"
	GTE    Method = "GTE"
	LTE    Method = "LTE"
	LIKE   Method = "LIKE"
	ILIKE  Method = "ILIKE"
	NLIKE  Method = "NLIKE"
	NILIKE Method = "NILIKE"
	IS     Method = "IS"
	NOT    Method = "NOT"
	IN     Method = "IN"
	NIN    Method = "NIN"
	raw    Method = "raw" // internal usage
)

// NULL constant
const NULL = "NULL"

var (
	translateMethods map[Method]string = map[Method]string{
		EQ:     "=",
		NE:     "!=",
		GT:     ">",
		LT:     "<",
		GTE:    ">=",
		LTE:    "<=",
		LIKE:   "LIKE",
		ILIKE:  "ILIKE",
		NLIKE:  "NOT LIKE",
		NILIKE: "NOT ILIKE",
		IS:     "IS",
		NOT:    "IS NOT",
		IN:     "IN",
		NIN:    "NOT IN",
	}
)

// Sort is ordering struct
type Sort struct {
	By   string
	Desc bool
}

// IgnoreUnknownFilters set behavior for Parser to raise ErrFilterNotAllowed to undefined filters or not
func (q *Query) IgnoreUnknownFilters(i bool) *Query {
	q.ignoreUnknown = i
	return q
}

// SetDelimiterIN sets delimiter for values of filters
func (q *Query) SetDelimiterIN(d string) *Query {
	q.delimiterIN = d
	return q
}

// SetDelimiterOR sets delimiter for OR filters in query part of URL
func (q *Query) SetDelimiterOR(d string) *Query {
	q.delimiterOR = d
	return q
}

// FieldsString returns elements list separated by comma (",") for querying in SELECT statement or a star ("*") if nothing provided
//
// Return example:
//
// When "fields" empty or not provided: `*`.
//
// When "fields=id,email": `id, email`.
//
func (q *Query) FieldsString() string {
	if len(q.Fields) == 0 {
		return "*"
	}
	return strings.Join(q.Fields, ", ")
}

// Select returns elements list separated by comma (",") for querying in SELECT statement or a star ("*") if nothing provided
//
// Return examples:
//
// When "fields" empty or not provided: `*`
//
// When "fields=id,email": `id, email`
//
func (q *Query) Select() string {
	if len(q.Fields) == 0 {
		return "*"
	}
	return strings.Join(q.Fields, ", ")
}

// SELECT returns word SELECT with fields from Filter "fields" separated by comma (",") from URL-Query
// or word SELECT with star ("*") if nothing provided
//
// Return examples:
//
// When "fields" empty or not provided: `SELECT *`.
//
// When "fields=id,email": `SELECT id, email`.
//
func (q *Query) SELECT() string {
	if len(q.Fields) == 0 {
		return "SELECT *"
	}
	return fmt.Sprintf("SELECT %s", q.FieldsString())
}

// HaveField returns true if request asks for specified field
func (q *Query) HaveField(field string) bool {
	return stringInSlice(field, q.Fields)
}

// AddField adds field to SELECT statement
func (q *Query) AddField(field string) *Query {
	q.Fields = append(q.Fields, field)
	return q
}

// OFFSET returns word OFFSET with number
//
// Return example: ` OFFSET 0`
//
func (q *Query) OFFSET() string {
	if q.Offset > 0 {
		return fmt.Sprintf(" OFFSET %d", q.Offset)
	}
	return ""
}

// LIMIT returns word LIMIT with number
//
// Return example: ` LIMIT 100`
//
func (q *Query) LIMIT() string {
	if q.Limit > 0 {
		return fmt.Sprintf(" LIMIT %d", q.Limit)
	}
	return ""
}

// Order returns list of elements for ORDER BY statement
// you can use +/- prefix to specify direction of sorting (+ is default)
// return example: `id DESC, email`
func (q *Query) Order() string {
	if len(q.Sorts) == 0 {
		return ""
	}

	var s string

	for i := 0; i < len(q.Sorts); i++ {
		if i > 0 {
			s += ", "
		}
		if q.Sorts[i].Desc {
			s += fmt.Sprintf("%s DESC", q.Sorts[i].By)
		} else {
			s += q.Sorts[i].By
		}
	}

	return s
}

// ORDER returns words ORDER BY with list of elements for sorting
// you can use +/- prefix to specify direction of sorting (+ is default, apsent is +)
//
// Return example: ` ORDER BY id DESC, email`
func (q *Query) ORDER() string {
	if len(q.Sorts) == 0 {
		return ""
	}
	return fmt.Sprintf(" ORDER BY %s", q.Order())
}

// HaveSortBy returns true if request contains sorting by specified in by field name
func (q *Query) HaveSortBy(by string) bool {

	for _, v := range q.Sorts {
		if v.By == by {
			return true
		}
	}

	return false
}

// AddSortBy adds an ordering rule to Query
func (q *Query) AddSortBy(by string, desc bool) *Query {
	q.Sorts = append(q.Sorts, Sort{
		By:   by,
		Desc: desc,
	})
	return q
}

// HaveFilter returns true if request contains some filter
func (q *Query) HaveFilter(name string) bool {

	for _, v := range q.Filters {
		if v.Name == name {
			return true
		}
	}

	return false
}

// AddFilter adds a filter to Query
func (q *Query) AddFilter(name string, m Method, value interface{}) *Query {
	q.Filters = append(q.Filters, &Filter{
		Name:   name,
		Method: m,
		Value:  value,
	})
	return q
}

// AddORFilters adds multiple filter into one `OR` statement inside parenteses.
// E.g. (firstname ILIKE ? OR lastname ILIKE ?)
func (q *Query) AddORFilters(fn func(query *Query)) *Query {
	_q := New()

	fn(_q)

	if len(_q.Filters) < 2 {
		return q
	}

	firstIdx := 0
	lastIdx := len(_q.Filters) - 1

	for i := 0; i < len(_q.Filters); i++ {
		switch i {
		case firstIdx:
			_q.Filters[i].OR = StartOR
		case lastIdx:
			_q.Filters[i].OR = EndOR
		default:
			_q.Filters[i].OR = InOR
		}
	}

	q.Filters = append(q.Filters, _q.Filters...)
	return q
}

// AddFilterRaw adds a filter to Query as SQL condition.
// This function supports only single condition per one call.
// If you'd like add more then one conditions you should call this func several times.
func (q *Query) AddFilterRaw(condition string) *Query {
	q.Filters = append(q.Filters, &Filter{
		Name:   condition,
		Method: raw,
	})
	return q
}

// RemoveFilter removes the filter by name
func (q *Query) RemoveFilter(name string) error {
	var found bool
	for i := 0; i < len(q.Filters); i++ {
		v := q.Filters[i]

		// set next and previous Filter
		var next, prev *Filter
		if i+1 < len(q.Filters) {
			next = q.Filters[i+1]
		} else {
			next = nil
		}
		if i-1 >= 0 {
			prev = q.Filters[i-1]
		} else {
			prev = nil
		}

		if v.Name == name {
			// special cases for removing filters in OR statement
			if v.OR == StartOR && next != nil {
				if next.OR == EndOR {
					next.OR = NoOR
				} else {
					next.OR = StartOR
				}
			} else if v.OR == EndOR && prev != nil {
				if prev.OR == StartOR {
					prev.OR = NoOR
				} else {
					prev.OR = EndOR
				}
			}

			// safe remove element from slice
			if i < len(q.Filters)-1 {
				copy(q.Filters[i:], q.Filters[i+1:])
			}
			q.Filters[len(q.Filters)-1] = nil
			q.Filters = q.Filters[:len(q.Filters)-1]

			found = true
			i--
		}
	}
	if !found {
		return ErrFilterNotFound
	}
	return nil
}

// AddValidation adds a validation to Query
func (q *Query) AddValidation(NameAndTags string, v ValidationFunc) *Query {
	if q.validations == nil {
		q.validations = Validations{}
	}
	q.validations[NameAndTags] = v
	return q
}

// RemoveValidation remove a validation from Query
// You can provide full name of filter with tags or only name of filter:
// RemoveValidation("id:int") and RemoveValidation("id") are equal
func (q *Query) RemoveValidation(NameAndOrTags string) error {
	for k := range q.validations {
		if k == NameAndOrTags {
			delete(q.validations, k)
			return nil
		}
		if strings.Contains(k, ":") {
			parts := strings.Split(k, ":")
			if parts[0] == NameAndOrTags {
				delete(q.validations, k)
				return nil
			}
		}
	}
	return ErrValidationNotFound
}

// SetOffset sets Offset of query
func (q *Query) SetOffset(offset int) *Query {
	q.Offset = offset
	return q
}

// SetLimit sets Offset of query
func (q *Query) SetLimit(limit int) *Query {
	q.Limit = limit
	return q
}

// Clone makes copy of Query
func (q *Query) Clone() *Query {
	qNew := &Query{
		Offset:        q.Offset,
		Limit:         q.Limit,
		delimiterIN:   q.delimiterIN,
		delimiterOR:   q.delimiterOR,
		ignoreUnknown: q.ignoreUnknown,
		Error:         q.Error,
	}

	// copy query map
	if q.query != nil {
		qNew.query = make(map[string][]string)
		for key := range q.query {
			if len(q.query[key]) > 0 {
				qNew.query[key] = make([]string, len(q.query[key]), cap(q.query[key]))
				copy(qNew.query[key], q.query[key])
			}
		}
	}

	// copy validations
	if q.validations != nil {
		qNew.validations = make(Validations)
		for key := range q.validations {
			qNew.validations[key] = q.validations[key]
		}
	}

	// copy Fields
	if q.Fields != nil {
		qNew.Fields = make([]string, len(q.Fields), cap(q.Fields))
		copy(qNew.Fields, q.Fields)
	}
	// copy Sorts
	if q.Sorts != nil {
		qNew.Sorts = make([]Sort, len(q.Sorts), cap(q.Sorts))
		copy(qNew.Sorts, q.Sorts)
	}
	// copy Filters
	if q.Filters != nil {
		qNew.Filters = make([]*Filter, len(q.Filters), cap(q.Filters))
		copy(qNew.Filters, q.Filters)
	}

	return qNew
}

// GetFilter returns filter by name
func (q *Query) GetFilter(name string) (*Filter, error) {

	for _, v := range q.Filters {
		if v.Name == name {
			return v, nil
		}
	}

	return nil, ErrFilterNotFound
}

// Replacer struct for ReplaceNames method
type Replacer map[string]string

// ReplaceNames replace all specified name to new names
// Sometimes we've to hijack properties, for example when we do JOINs,
// so you can ask for filter/field "user_id" but replace it with "users.user_id".
// Parameter is a map[string]string which means map[currentName]newName.
// The library provide beautiful way by using special type rqp.Replacer.
// Example:
//   rqp.ReplaceNames(rqp.Replacer{
//	   "user_id": "users.user_id",
//   })
func (q *Query) ReplaceNames(r Replacer) {

	for name, newname := range r {
		for i, v := range q.Filters {
			if v.Name == name {
				q.Filters[i].Name = newname
			}
		}
		for i, v := range q.Fields {
			if v == name {
				q.Fields[i] = newname
			}
		}
		for i, v := range q.Sorts {
			if v.By == name {
				q.Sorts[i].By = newname
			}
		}
	}

}

// Where returns list of filters for WHERE statement
// return example: `id > 0 AND email LIKE 'some@email.com'`
func (q *Query) Where() string {

	if len(q.Filters) == 0 {
		return ""
	}

	var where string
	// var OR bool = false

	for i := 0; i < len(q.Filters); i++ {
		filter := q.Filters[i]

		prefix := ""
		suffix := ""

		if filter.OR == StartOR {
			if i == 0 {
				prefix = "("
			} else {
				prefix = " AND ("
			}
		} else if filter.OR == InOR {
			prefix = " OR "
		} else if filter.OR == EndOR {
			prefix = " OR "
			suffix = ")"
		} else if i > 0 && len(where) > 0 {
			prefix = " AND "
		}

		if a, err := filter.Where(); err == nil {
			where += fmt.Sprintf("%s%s%s", prefix, a, suffix)
		} else {
			continue
		}

	}

	return where
}

// WHERE returns list of filters for WHERE SQL statement with `WHERE` word
//
// Return example: ` WHERE id > 0 AND email LIKE 'some@email.com'`
//
func (q *Query) WHERE() string {

	if len(q.Filters) == 0 {
		return ""
	}

	return " WHERE " + q.Where()
}

// Args returns slice of arguments for WHERE statement
func (q *Query) Args() []interface{} {

	args := make([]interface{}, 0)

	if len(q.Filters) == 0 {
		return args
	}

	for i := 0; i < len(q.Filters); i++ {
		filter := q.Filters[i]
		if (filter.Method == IS || filter.Method == NOT) && filter.Value == NULL {
			continue
		}

		if a, err := filter.Args(); err == nil {
			args = append(args, a...)
		} else {
			continue
		}
	}

	return args
}

// SQL returns whole SQL statement
func (q *Query) SQL(table string) string {
	return fmt.Sprintf(
		"%s FROM %s%s%s%s%s",
		q.SELECT(),
		table,
		q.WHERE(),
		q.ORDER(),
		q.LIMIT(),
		q.OFFSET(),
	)
}

// SetUrlQuery change url in the Query for parsing
// uses when you need provide Query from http.HandlerFunc(w http.ResponseWriter, r *http.Request)
// you can do q.SetUrlValues(r.URL.Query())
func (q *Query) SetUrlQuery(query url.Values) *Query {
	q.query = query
	return q
}

// SetUrlString change url in the Query for parsing
// uses when you would like to provide raw URL string to parsing
func (q *Query) SetUrlString(Url string) error {
	u, err := url.Parse(Url)
	if err != nil {
		return err
	}
	q.SetUrlQuery(u.Query())
	return err
}

// SetValidations change validations rules for the instance
func (q *Query) SetValidations(v Validations) *Query {
	q.validations = v
	return q
}

// New creates new instance of Query
func New() *Query {
	return &Query{
		delimiterIN: ",",
		delimiterOR: "|",
	}
}

// NewQV creates new Query instance with parameters
func NewQV(q url.Values, v Validations) *Query {
	query := New().SetUrlQuery(q).SetValidations(v)
	return query
}

// NewParse creates new Query instance and Parse it
func NewParse(q url.Values, v Validations) (*Query, error) {
	query := New().SetUrlQuery(q).SetValidations(v)
	return query, query.Parse()
}

// Parse parses the query of URL
// as query you can use standart http.Request query by r.URL.Query()
func (q *Query) Parse() (err error) {

	// clean previously parsed filters
	q.cleanFilters()

	// construct a slice with required names of filters
	requiredNames := q.requiredNames()

	for key, values := range q.query {

		low := strings.ToLower(key)

		switch low {
		case "fields", "fields[in]":
			low = strings.ReplaceAll(low, "[in]", "")
			err = q.parseFields(values, q.validations[low])
			delete(requiredNames, low)
		case "offset", "offset[in]":
			low = strings.ReplaceAll(low, "[in]", "")
			err = q.parseOffset(values, q.validations[low])
			delete(requiredNames, low)
		case "limit", "limit[in]":
			low = strings.ReplaceAll(low, "[in]", "")
			err = q.parseLimit(values, q.validations[low])
			delete(requiredNames, low)
		case "sort", "sort[in]":
			low = strings.ReplaceAll(low, "[in]", "")
			err = q.parseSort(values, q.validations[low])
			delete(requiredNames, low)
		default:
			if len(values) == 0 {
				return errors.Wrap(ErrBadFormat, key)
			}
			for _, value := range values {
				err = q.parseFilter(key, value)
				if err != nil {
					return err
				}
			}
		}

		if err != nil {
			return errors.Wrap(err, key)
		}
	}

	// check required filters

	for requiredName := range requiredNames {
		if !q.HaveFilter(requiredName) {
			return errors.Wrap(ErrRequired, requiredName)
		}
	}

	return nil
}

// requiredNames returns list of required filters
func (q *Query) requiredNames() map[string]bool {
	required := make(map[string]bool)

	for name, f := range q.validations {
		if strings.Contains(name, ":required") {
			oldname := name
			// oldname = arg1:required
			// oldname = arg2:int:required
			newname := strings.Replace(name, ":required", "", 1)
			// newname = arg1
			// newname = arg2:int

			if strings.Contains(newname, ":") {
				parts := strings.Split(newname, ":")
				name = parts[0]
			} else {
				name = newname
			}
			// name = arg1
			// name = arg2

			low := strings.ToLower(name)
			switch low {
			case "fields", "fields[in]",
				"offset", "offset[in]",
				"limit", "limit[in]",
				"sort", "sort[in]":
				low = strings.ReplaceAll(low, "[in]", "")
				required[low] = true
			default:
				required[name] = true
			}

			q.validations[newname] = f
			delete(q.validations, oldname)
		}
	}
	return required
}

// parseFilter parses one filter
func (q *Query) parseFilter(key, value string) error {
	value = strings.TrimSpace(value)

	if len(value) == 0 {
		return errors.Wrap(ErrEmptyValue, key)
	}

	if strings.Contains(value, q.delimiterOR) { // OR multiple filter
		parts := strings.Split(value, q.delimiterOR)
		for i, v := range parts {
			if i > 0 {
				u := strings.Split(v, "=")
				if len(u) < 2 {
					return errors.Wrap(ErrBadFormat, key)
				}
				key = u[0]
				v = u[1]
			}

			v := strings.TrimSpace(v)
			if len(v) == 0 {
				return errors.Wrap(ErrEmptyValue, key)
			}

			filter, err := newFilter(key, v, q.delimiterIN, q.validations)

			if err != nil {
				if err == ErrValidationNotFound {
					if q.ignoreUnknown {
						continue
					} else {
						return errors.Wrap(ErrFilterNotFound, key)
					}
				}
				return errors.Wrap(err, key)
			}

			// set OR
			if i == 0 {
				filter.OR = StartOR
			} else if i == len(parts)-1 {
				filter.OR = EndOR
			} else {
				filter.OR = InOR
			}

			q.Filters = append(q.Filters, filter)
		}
	} else { // Single filter
		filter, err := newFilter(key, value, q.delimiterIN, q.validations)
		if err != nil {
			if err == ErrValidationNotFound {
				err = ErrFilterNotFound
				if q.ignoreUnknown {
					return nil
				}
			}
			return errors.Wrap(err, key)
		}

		q.Filters = append(q.Filters, filter)
	}

	return nil
}

// clean the filters slice
func (q *Query) cleanFilters() {
	if len(q.Filters) > 0 {
		for i := range q.Filters {
			q.Filters[i] = nil
		}
		q.Filters = nil
	}
}

func (q *Query) parseSort(value []string, validate ValidationFunc) error {
	if len(value) != 1 {
		return ErrBadFormat
	}

	if validate == nil {
		return ErrValidationNotFound
	}

	list := value
	if strings.Contains(value[0], q.delimiterIN) {
		list = strings.Split(value[0], q.delimiterIN)
	}

	list = cleanSliceString(list)

	sort := make([]Sort, 0)

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

		sort = append(sort, Sort{
			By:   by,
			Desc: desc,
		})
	}

	q.Sorts = sort

	return nil
}

func (q *Query) parseFields(value []string, validate ValidationFunc) error {
	if len(value) != 1 {
		return ErrBadFormat
	}

	if validate == nil {
		return ErrValidationNotFound
	}

	list := value
	if strings.Contains(value[0], q.delimiterIN) {
		list = strings.Split(value[0], q.delimiterIN)
	}

	list = cleanSliceString(list)

	if validate != nil {
		for _, v := range list {
			if err := validate(v); err != nil {
				return err
			}
		}
	}

	q.Fields = list
	return nil
}

func (q *Query) parseOffset(value []string, validate ValidationFunc) error {

	if len(value) != 1 {
		return ErrBadFormat
	}

	if len(value[0]) == 0 {
		return ErrBadFormat
	}

	var err error

	i, err := strconv.Atoi(value[0])
	if err != nil {
		return ErrBadFormat
	}

	if i < 0 {
		return errors.Wrapf(ErrNotInScope, "%d", i)
	}

	if validate != nil {
		if err := validate(i); err != nil {
			return err
		}
	}

	q.Offset = i

	return nil
}

func (q *Query) parseLimit(value []string, validate ValidationFunc) error {

	if len(value) != 1 {
		return ErrBadFormat
	}

	if len(value[0]) == 0 {
		return ErrBadFormat
	}

	var err error

	i, err := strconv.Atoi(value[0])
	if err != nil {
		return ErrBadFormat
	}

	if i <= 0 {
		return errors.Wrapf(ErrNotInScope, "%d", i)
	}

	if validate != nil {
		if err := validate(i); err != nil {
			return err
		}
	}

	q.Limit = i

	return nil
}
