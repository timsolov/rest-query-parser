package rqp

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type DatabaseField struct {
	Name  string
	Table string
	Type  string
}

type QueryDbMap map[string]DatabaseField

// Query the main struct of package
type Query struct {
	query       map[string][]string
	validations Validations

	queryDbFieldMap QueryDbMap

	QueryFields []string
	Offset      int
	Limit       int
	Sorts       []Sort
	Filters     []*Filter

	delimiterIN           string
	delimiterOR           string
	ignoreUnknown         bool
	allowedSpecialFilters []string

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
	QuerySortBy     string
	ByParameterized string
	Desc            bool
}

// IgnoreUnknownFilters set behavior for Parser to raise ErrFilterNotAllowed to undefined filters or not
func (q *Query) IgnoreUnknownFilters(i bool) *Query {
	q.ignoreUnknown = i
	return q
}

// set behavior for Parser to not raise ErrFilterNotAllowed for these undefined filters or not
func (q *Query) AllowSpecialFilters(names ...string) *Query {
	q.allowedSpecialFilters = names
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
func (q *Query) FieldsString(tables ...string) string {
	return q.Select(tables...)
}

func (q *Query) FieldsNames() []string {
	fieldNames := []string{}
	for _, f := range q.QueryFields {
		fieldNames = append(fieldNames, f)
	}
	return fieldNames
}

// Select returns elements list separated by comma (",") for querying in SELECT statement or a star ("*") if nothing provided
//
// Return examples:
//
// When "fields" empty or not provided: `*`
//
// When "fields=id,email": `id, email`
//
func (q *Query) Select(tables ...string) string {
	fieldNames := []string{}
	if len(q.QueryFields) == 0 {
		for _, table := range tables {
			fieldNames = append(fieldNames, fmt.Sprintf("%s.*", table))
		}
	} else {
		for _, qf := range q.QueryFields {
			for k, v := range q.queryDbFieldMap {
				newFieldNames := []string{}
				if stringInSlice(v.Table, tables) {
					baseQueryField := strings.Split(k, ".")[0]
					if baseQueryField == qf {
						if v.Type == "custom" {
							fieldNames = append(fieldNames, fmt.Sprintf("%s.%s", v.Table, baseQueryField))
							break
						} else {
							newFieldNames = append(newFieldNames, fmt.Sprintf("%s.%s", v.Table, v.Name))
						}
					}
				}
				fieldNames = append(fieldNames, newFieldNames...)
			}
		}
	}
	return strings.Join(fieldNames, ", ")
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
func (q *Query) SELECT(tables ...string) string {
	if len(q.QueryFields) == 0 {
		return "SELECT *"
	}
	return fmt.Sprintf("SELECT %s", q.Select(tables...))
}

// HaveQueryField returns true if request asks for specified field
func (q *Query) HaveQueryField(name string) bool {
	for _, b := range q.QueryFields {
		if b == name {
			return true
		}
	}
	return false
}

// HaveQueryFieldsOnTable returns true if request asks for specified table
func (q *Query) HaveQueryFieldsOnTable(table string) bool {
	for _, qf := range q.QueryFields {
		if dbf, ok := q.queryDbFieldMap[qf]; ok {
			if dbf.Table == table {
				return true
			}
		}
	}
	return false
}

// AddField adds field to SELECT statement
func (q *Query) AddQueryField(queryName string) *Query {
	q.QueryFields = append(q.QueryFields, queryName)
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

func (q *Query) SetOffset(offset int) *Query {
	q.Offset = offset
	return q
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

func (q *Query) SetLimit(limit int) *Query {
	q.Limit = limit
	return q
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
			s += fmt.Sprintf("%s DESC", q.Sorts[i].ByParameterized)
		} else {
			s += q.Sorts[i].ByParameterized
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
func (q *Query) HaveQuerySortBy(querySortBy string) bool {
	for _, v := range q.Sorts {
		if v.QuerySortBy == querySortBy {
			return true
		}
	}
	return false
}

// AddSortBy adds an ordering rule to Query
func (q *Query) AddQuerySortBy(querySortBy string, desc bool) *Query {
	q.Sorts = append(q.Sorts, Sort{
		QuerySortBy:     querySortBy,
		ByParameterized: getParameterizedName(querySortBy, q.queryDbFieldMap),
		Desc:            desc,
	})
	return q
}

// HaveFilter returns true if request contains some filter
func (q *Query) HaveQueryFiltersOnTable(table string) bool {
	for _, v := range q.Filters {
		if v.DbField.Table == table {
			return true
		}
	}
	return false
}

// HaveFilter returns true if request contains some filter
func (q *Query) HaveQueryFilter(queryName string) bool {
	for _, v := range q.Filters {
		if v.QueryName == queryName {
			return true
		}
	}
	return false
}

// AddFilter adds a filter to Query
func (q *Query) AddQueryFilter(queryName string, m Method, value interface{}) *Query {
	dbField, err := detectDbField(queryName, q.queryDbFieldMap)
	if err != nil {
		panic("could not find db field for filter")
	}
	q.Filters = append(q.Filters, &Filter{
		QueryName:         queryName,
		ParameterizedName: getParameterizedName(dbField.Name, q.queryDbFieldMap),
		Method:            m,
		Value:             value,
		DbField:           dbField,
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

// // AddFilterRaw adds a filter to Query as SQL condition.
// // This function supports only single condition per one call.
// // If you'd like add more then one conditions you should call this func several times.
// func (q *Query) AddFilterRaw(table, name, type, condition string) *Query {
// 	q.Filters = append(q.Filters, &Filter{
// 		ParameterizedName: condition,
// 		Method:            raw,
// 		DbField: DatabaseField{},
// 	})
// 	return q
// }

// RemoveFilter removes the filter by name
func (q *Query) RemoveQueryFilter(name string) error {
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

		if v.QueryName == name {
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
func (q *Query) AddQueryValidation(NameAndTags string, v ValidationFunc) *Query {
	if q.validations == nil {
		q.validations = Validations{}
	}
	q.validations[NameAndTags] = v
	return q
}

// RemoveValidation remove a validation from Query
// You can provide full name of filter with tags or only name of filter:
// RemoveValidation("id:int") and RemoveValidation("id") are equal
func (q *Query) RemoveQueryValidation(NameAndOrTags string) error {
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

// GetFilter returns filter by name
func (q *Query) GetQueryFilter(queryName string) (*Filter, error) {
	for _, v := range q.Filters {
		if v.QueryName == queryName {
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
		pName := getParameterizedName(newname, q.queryDbFieldMap)
		for i, v := range q.Filters {
			if v.QueryName == name {
				q.Filters[i].QueryName = newname
				q.Filters[i].ParameterizedName = pName
			}
		}
		for i, v := range q.QueryFields {
			if v == name {
				q.QueryFields[i] = newname
			}
		}
		for i, v := range q.Sorts {
			if v.QuerySortBy == name {
				q.Sorts[i].QuerySortBy = newname
				q.Sorts[i].ByParameterized = pName
			}
		}
	}

}

// Where returns list of filters for WHERE statement
// return example: `id > 0 AND email LIKE 'some@email.com'`
func (q *Query) Where(tables ...string) string {

	var where string
	// var OR bool = false

	for i := 0; i < len(q.Filters); i++ {
		filter := q.Filters[i]

		if !stringInSlice(filter.DbField.Table, tables) {
			continue
		}

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
func (q *Query) WHERE(tables ...string) string {
	where := q.Where(tables...)
	if where == "" {
		return where
	} else {
		return " WHERE " + where
	}
}

// Args returns slice of arguments for WHERE statement
func (q *Query) Args(tables ...string) []interface{} {

	args := make([]interface{}, 0)

	if len(q.Filters) == 0 {
		return args
	}

	for i := 0; i < len(q.Filters); i++ {
		filter := q.Filters[i]

		if !stringInSlice(filter.DbField.Table, tables) {
			continue
		}

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
		q.SELECT(table),
		table,
		q.WHERE(table),
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
		delimiterIN:     ",",
		delimiterOR:     "|",
		queryDbFieldMap: QueryDbMap{},
	}
}

// NewQV creates new Query instance with parameters
func NewQV(q url.Values, v Validations, qdbm QueryDbMap) *Query {
	query := New().SetUrlQuery(q).SetValidations(v).SetQueryDbFieldsMap(qdbm)
	return query
}

// NewParse creates new Query instance and Parse it
func NewParse(q url.Values, v Validations, qdbm QueryDbMap) (*Query, error) {
	query := New().SetUrlQuery(q).SetValidations(v).SetQueryDbFieldsMap(qdbm)
	return query, query.Parse()
}

func (q *Query) SetQueryDbFieldsMap(m QueryDbMap) *Query {
	q.queryDbFieldMap = m
	return q
}

func (q *Query) AddQueryDbFieldMapping(queryName string, dbField DatabaseField) *Query {
	q.queryDbFieldMap[queryName] = dbField
	return q
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
			err = q.parseFields(values)
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
		if !q.HaveQueryFilter(requiredName) {
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
	var f *Filter

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

			filter, err := newFilter(key, v, q.delimiterIN, q.validations, q.queryDbFieldMap)

			if err != nil {
				if err == ErrValidationNotFound {
					if q.ignoreUnknown || stringInSlice(filter.QueryName, q.allowedSpecialFilters) {
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
			f = filter
		}
	} else { // Single filter
		filter, err := newFilter(key, value, q.delimiterIN, q.validations, q.queryDbFieldMap)
		if err != nil {
			if err == ErrValidationNotFound {
				err = ErrFilterNotFound
				if q.ignoreUnknown || stringInSlice(filter.QueryName, q.allowedSpecialFilters) {
					return nil
				}
			}
			return errors.Wrap(err, key)
		}
		f = filter
	}

	q.Filters = append(q.Filters, f)

	return nil
}

// allow support for filters on nested custom/json properties,
// e.g. pace.pacing_strategy
func getParameterizedName(name string, qdbm QueryDbMap) string {
	elems := strings.Split(name, ".")
	cur := elems[0]
	curFormatted := elems[0]
	var jsonElems []string
	if len(elems) > 1 {
		for i, el := range elems[1:] {
			t := detectType(cur, qdbm)
			if t == "json" {
				jsonElems = append(jsonElems, curFormatted)
				cur = el
				curFormatted = el
			} else if t == "custom" {
				if i == 0 {
					curFormatted = fmt.Sprintf("(%s).%s", cur, el)
				} else {
					curFormatted = fmt.Sprintf("%s.%s", curFormatted, el)
				}
				cur = fmt.Sprintf("%s.%s", cur, el)
			} else {
				return name
			}
		}
		jsonElems = append(jsonElems, curFormatted)
		if len(jsonElems) > 1 {
			t := detectType(name, qdbm)
			return getParameterizedNameJsonHelper(t, jsonElems...)
		}
	}
	return curFormatted
}

// recursive helper for extracting json
func getParameterizedNameJsonHelper(outerType string, elems ...string) string {
	if len(elems) == 1 {
		switch outerType {
		case "custom", "json":
			return elems[0]
		case "string":
			return fmt.Sprintf("%s::text", elems[0])
		case "bool":
			return fmt.Sprintf("%s::text::boolean", elems[0])
		case "time":
			return fmt.Sprintf("%s::text::timestamp with time zone", elems[0])
		// int, float
		default:
			return fmt.Sprintf("%s::text::%s", elems[0], outerType)
		}
	}
	newElems := []string{
		fmt.Sprintf("json_extract_path(json_strip_nulls(%s), '%s')", elems[0], elems[1]),
	}
	if len(elems) > 2 {
		newElems = append(newElems, elems[2:]...)
	}
	return getParameterizedNameJsonHelper(outerType, newElems...)
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
			QuerySortBy:     by,
			ByParameterized: getParameterizedName(by, q.queryDbFieldMap),
			Desc:            desc,
		})
	}

	q.Sorts = sort

	return nil
}

func (q *Query) parseFields(value []string) error {
	if len(value) != 1 {
		return ErrBadFormat
	}

	validate := q.validations["fields"]
	if validate == nil {
		return ErrValidationNotFound
	}

	list := value
	if strings.Contains(value[0], q.delimiterIN) {
		list = strings.Split(value[0], q.delimiterIN)
	}

	list = cleanSliceString(list)

	for _, v := range list {
		if validate != nil {
			if err := validate(v); err != nil {
				return err
			}
		}
		q.QueryFields = append(q.QueryFields, v)
	}

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
