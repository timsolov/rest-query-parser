package rqp

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Query contatins of all major data
type Query struct {
	query       map[string][]string
	validations Validations

	fields  []string
	offset  int
	limit   int
	sorts   []Sort
	filters []*Filter

	delimiterIN   string
	delimiterOR   string
	ignoreUnknown bool

	Error error
}

type Method string

var (
	NULL string = "NULL"

	EQ    Method = "EQ"
	NE    Method = "NE"
	GT    Method = "GT"
	LT    Method = "LT"
	GTE   Method = "GTE"
	LTE   Method = "LTE"
	LIKE  Method = "LIKE"
	ILIKE Method = "ILIKE"
	NOT   Method = "NOT"
	IN    Method = "IN"

	TranslateMethods map[Method]string = map[Method]string{
		EQ:    "=",
		NE:    "!=",
		GT:    ">",
		LT:    "<",
		GTE:   ">=",
		LTE:   "<=",
		LIKE:  "LIKE",
		ILIKE: "ILIKE",
		NOT:   "IS NOT",
		IN:    "IN",
	}
)

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

// Fields returns elements list separated by comma (",") for querying in SELECT statement or a star ("*") if nothing provided
func (p *Query) FieldsSQL() string {
	if len(p.fields) == 0 {
		return "*"
	}
	return strings.Join(p.fields, ", ")
}

// SelectSQL returns SELECT with fields from Filter "fields" from URL or a star ("*") if nothing provided
func (q *Query) SelectSQL() string {
	if len(q.fields) == 0 {
		return "*"
	}
	return fmt.Sprintf("SELECT %s", q.FieldsSQL())
}

// HaveField returns true if request asks for field
func (p *Query) HaveField(field string) bool {
	return stringInSlice(field, p.fields)
}

// AddField returns true if request asks for field
func (p *Query) AddField(field string) {
	p.fields = append(p.fields, field)
}

// OffsetSQL returns OFFSET statement
func (p *Query) OffsetSQL() string {
	if p.offset > 0 {
		return fmt.Sprintf(" OFFSET %d", p.offset)
	}
	return ""
}

// LimitSQL returns LIMIT statement
func (p *Query) LimitSQL() string {
	if p.limit > 0 {
		return fmt.Sprintf(" LIMIT %d", p.limit)
	}
	return ""
}

// Sort returns list of elements for ORDER BY statement
// you can use +/- prefix to specify direction of sorting (+ is default)
func (p *Query) Sort() string {
	if len(p.sorts) == 0 {
		return ""
	}

	var s string

	for i := 0; i < len(p.sorts); i++ {
		if i > 0 {
			s += ", "
		}
		if p.sorts[i].Desc {
			s += fmt.Sprintf("%s DESC", p.sorts[i].By)
		} else {
			s += p.sorts[i].By
		}
	}

	return s
}

// Sort returns ORDER BY statement with list of elements for sorting
// you can use +/- prefix to specify direction of sorting (+ is default)
func (q *Query) SortSQL() string {
	if len(q.sorts) == 0 {
		return ""
	}
	return fmt.Sprintf(" ORDER BY %s", q.Sort())
}

// HaveSortBy returns true if request contains some sorting
func (p *Query) HaveSortBy(by string) bool {

	for _, v := range p.sorts {
		if v.By == by {
			return true
		}
	}

	return false
}

// HaveFilter returns true if request contains some filter
func (p *Query) HaveFilter(name string) bool {

	for _, v := range p.filters {
		if v.Name == name {
			return true
		}
	}

	return false
}

// AddFilter adds a filter to Query
func (p *Query) AddFilter(name string, m Method, value interface{}) *Query {
	p.filters = append(p.filters, &Filter{
		Name:   name,
		Method: m,
		Value:  value,
	})
	return p
}

// RemoveFilter removes the filter by name
func (p *Query) RemoveFilter(name string) error {

	for i, v := range p.filters {
		if v.Name == name {
			// safe remove element from slice
			if i < len(p.filters)-1 {
				copy(p.filters[i:], p.filters[i+1:])
			}
			p.filters[len(p.filters)-1] = nil
			p.filters = p.filters[:len(p.filters)-1]

			return nil
		}
	}

	return ErrFilterNotFound
}

// AddValidation adds a validation to Query
func (q *Query) AddValidation(NameAndTags string, v ValidationFunc) *Query {
	if q.validations == nil {
		q.validations = Validations{}
	}
	q.validations[NameAndTags] = v
	return q
}

// AddValidation remove a validation from Query
// You can provide full name of filer with tags or only name of filter:
// RemoveValidation("id:int") and RemoveValidation("id") are same
func (q *Query) RemoveValidation(NameAndOrTags string) error {
	for k, _ := range q.validations {
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
func (p *Query) GetFilter(name string) (*Filter, error) {

	for _, v := range p.filters {
		if v.Name == name {
			return v, nil
		}
	}

	return nil, ErrFilterNotFound
}

// Replacer struct for ReplaceNames method
type Replacer map[string]string

// ReplaceNames replace all specified name to new names
// you can ask filter/field "user_id" but replace it which "id" for DB
// parameter is map[string]string which means map[currentName]newName
// usage: rqp.ReplaceFiltersNames(rqp.NamesReplacer{"oldName":"newName"})
func (p *Query) ReplaceNames(r Replacer) {

	for name, newname := range r {
		for i, v := range p.filters {
			if v.Name == name {
				p.filters[i].Name = newname
			}
		}
		for i, v := range p.fields {
			if v == name {
				p.fields[i] = newname
			}
		}
	}

}

// Where returns list of filters for WHERE statement
func (p *Query) Where() string {

	if len(p.filters) == 0 {
		return ""
	}

	var where string
	var OR bool = false

	for i := 0; i < len(p.filters); i++ {
		filter := p.filters[i]

		prefix := ""
		suffix := ""

		if filter.Or && !OR {
			if i == 0 {
				prefix = "("
			} else {
				prefix = " AND ("
			}
			OR = true
		} else if filter.Or && OR {
			prefix = " OR "
			// if last element of next element not OR method
			if i+1 == len(p.filters) || (i+1 < len(p.filters) && !p.filters[i+1].Or) {
				suffix = ")"
				OR = false
			}
		} else {
			if i > 0 {
				prefix = " AND "
			}
		}

		if a, err := filter.Where(); err == nil {
			where += fmt.Sprintf("%s%s%s", prefix, a, suffix)
		} else {
			continue
		}

	}

	return where
}

// WhereSQL returns list of filters for WHERE SQL statement
func (p *Query) WhereSQL() string {

	if len(p.filters) == 0 {
		return ""
	}

	return " WHERE " + p.Where()
}

// Args returns slice of arguments for WHERE statement
func (p *Query) Args() []interface{} {

	args := make([]interface{}, 0)

	if len(p.filters) == 0 {
		return args
	}

	for i := 0; i < len(p.filters); i++ {
		filter := p.filters[i]

		if a, err := filter.Args(); err == nil {
			args = append(args, a...)
		} else {
			continue
		}
	}

	return args
}

func (p *Query) SQL(table string) string {
	return fmt.Sprintf(
		"SELECT %s FROM %s%s%s%s%s",
		p.FieldsSQL(),
		table,
		p.WhereSQL(),
		p.SortSQL(),
		p.LimitSQL(),
		p.OffsetSQL(),
	)
}

// SetUrlQuery change url in the Query for parsing
// uses when you need provide Query from http.HandlerFunc(w http.ResponseWriter, r *http.Request)
// you can do q.SetUrlValues(r.URL.Query())
func (p *Query) SetUrlQuery(q url.Values) *Query {
	p.query = q
	return p
}

// SetUrlString change url in the Query for parsing
// uses when you would like to provide raw URL string to parsing
func (p *Query) SetUrlString(Url string) error {
	u, err := url.Parse(Url)
	if err != nil {
		return err
	}
	p.SetUrlQuery(u.Query())
	return err
}

// SetValidations change validations rules for the instance
func (p *Query) SetValidations(v Validations) *Query {
	p.validations = v
	return p
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
func (q *Query) Parse() error {

	// clean the filters slice
	if len(q.filters) > 0 {
		for i, _ := range q.filters {
			q.filters[i] = nil
		}
		q.filters = nil
	}

	// construct a slice with required names of filters

	var requiredNames []string

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
				requiredNames = append(requiredNames, low)
			default:
				requiredNames = append(requiredNames, name)
			}

			q.validations[newname] = f
			delete(q.validations, oldname)
		}
	}

	//fmt.Println("NEW QUERY:")

	for key, values := range q.query {

		low := strings.ToLower(key)

		switch low {
		case "fields", "fields[in]":
			low = strings.ReplaceAll(low, "[in]", "")
			if err := q.parseFields(values, q.validations[low]); err != nil {
				return errors.Wrap(err, key)
			}
			requiredNames = removeFromSlice(requiredNames, low)
		case "offset", "offset[in]":
			low = strings.ReplaceAll(low, "[in]", "")
			if err := q.parseOffset(values, q.validations[low]); err != nil {
				return errors.Wrap(err, key)
			}
			requiredNames = removeFromSlice(requiredNames, low)
		case "limit", "limit[in]":
			low = strings.ReplaceAll(low, "[in]", "")
			if err := q.parseLimit(values, q.validations[low]); err != nil {
				return errors.Wrap(err, key)
			}
			requiredNames = removeFromSlice(requiredNames, low)
		case "sort", "sort[in]":
			low = strings.ReplaceAll(low, "[in]", "")
			if err := q.parseSort(values, q.validations[low]); err != nil {
				return errors.Wrap(err, key)
			}
			requiredNames = removeFromSlice(requiredNames, low)
		default:
			if len(values) == 1 {

				//fmt.Println("new filter:")

				value := values[0]

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

						//fmt.Println("key:", key, "value:", v)
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
						filter.Or = true

						q.filters = append(q.filters, filter)
					}
				} else { // Single filter
					//fmt.Println("key:", key, "value:", value)
					filter, err := newFilter(key, value, q.delimiterIN, q.validations)
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

					q.filters = append(q.filters, filter)
				}

			} else {
				return errors.Wrap(ErrBadFormat, key)
			}
		}

	}

	// check required filters

	for _, requiredName := range requiredNames {
		if !q.HaveFilter(requiredName) {
			return errors.Wrap(ErrRequired, requiredName)
		}
	}

	return nil
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

	q.sorts = sort

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

	q.fields = list
	return nil
}

func (p *Query) parseOffset(value []string, validate ValidationFunc) error {

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
		return ErrBadFormat
	}

	if validate != nil {
		if err := validate(i); err != nil {
			return err
		}
	}

	p.offset = i

	return nil
}

func (p *Query) parseLimit(value []string, validate ValidationFunc) error {

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
		return ErrBadFormat
	}

	if validate != nil {
		if err := validate(i); err != nil {
			return err
		}
	}

	p.limit = i

	return nil
}
