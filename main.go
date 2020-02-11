package rqp

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	MethodEQ   string = "EQ"
	MethodNE   string = "NE"
	MethodGT   string = "GT"
	MethodLT   string = "LT"
	MethodGTE  string = "GTE"
	MethodLTE  string = "LTE"
	MethodLIKE string = "LIKE"
	MethodNOT  string = "NOT"
	MethodIN   string = "IN"

	TranslateMethods map[string]string = map[string]string{
		MethodEQ:   "=",
		MethodNE:   "!=",
		MethodGT:   ">",
		MethodLT:   "<",
		MethodGTE:  ">=",
		MethodLTE:  "<=",
		MethodLIKE: "LIKE",
		MethodNOT:  "NOT",
		MethodIN:   "IN",
	}
)

type Sort struct {
	by   string
	desc bool
}

// QueryParser contatins of all major data
type QueryParser struct {
	query       map[string][]string
	validations Validations

	fields  []string
	offset  int
	limit   int
	sorts   []Sort
	filters []Filter

	delimiter     string
	ignoreUnknown bool

	ErrorMessage string
}

// Delimiter sets delimiter for values of multiple filter
func (p *QueryParser) Delimiter(delimiter string) *QueryParser {
	p.delimiter = delimiter
	return p
}

// IgnoreUnknownFilters set behavior for Parser to raise ErrFilterNotAllowed to undefined filters or not
func (p *QueryParser) IgnoreUnknownFilters(i bool) *QueryParser {
	p.ignoreUnknown = i
	return p
}

// Fields returns elements for querying in SELECT statement or * if fields parameter not specified
func (p *QueryParser) Fields() string {
	if len(p.fields) == 0 {
		return "*"
	}
	return strings.Join(p.fields, ", ")
}

// GetFields getter for fields
func (p *QueryParser) GetFields() []string {
	return p.fields
}

// SetFields setter for fields
func (p *QueryParser) SetFields(fields []string) {
	p.fields = fields
}

// AskField returns true if request asks for field
func (p *QueryParser) HaveField(field string) bool {
	return stringInSlice(field, p.fields)
}

// Offset returns OFFSET statement
func (p *QueryParser) Offset() string {
	if p.offset > 0 {
		return fmt.Sprintf(" OFFSET %d", p.offset)
	}
	return ""
}

// GetOffset getter for offset
func (p *QueryParser) GetOffset() int {
	return p.offset
}

// SetOffset setter for offset
func (p *QueryParser) SetOffset(offset int) {
	p.offset = offset
}

// Limit returns LIMIT statement
func (p *QueryParser) Limit() string {
	if p.limit > 0 {
		return fmt.Sprintf(" LIMIT %d", p.limit)
	}
	return ""
}

// GetLimit getter for limit
func (p *QueryParser) GetLimit() int {
	return p.limit
}

// GetLimit setter for limit
func (p *QueryParser) SetLimit(limit int) {
	p.limit = limit
}

// Sort returns ORDER BY statement
// you can use +/- prefix to specify direction of sorting (+ is default)
func (p *QueryParser) Sort() string {
	if len(p.sorts) == 0 {
		return ""
	}

	s := " ORDER BY "

	for i := 0; i < len(p.sorts); i++ {
		if i > 0 {
			s += ", "
		}
		if p.sorts[i].desc {
			s += fmt.Sprintf("%s DESC", p.sorts[i].by)
		} else {
			s += p.sorts[i].by
		}
	}

	return s
}

// GetSorts getter for sort
func (p *QueryParser) GetSorts() []Sort {
	return p.sorts
}

// SetSorts setter for sort
func (p *QueryParser) SetSorts(sort []Sort) {
	p.sorts = sort
}

// HaveSortBy returns true if request contains some sorting
func (p *QueryParser) HaveSortBy(by string) bool {

	for _, v := range p.sorts {
		if v.by == by {
			return true
		}
	}

	return false
}

// GetFilters getter for filters
func (p *QueryParser) GetFilters() []Filter {
	return p.filters
}

// SetFilters setter for filters
func (p *QueryParser) SetFilters(filters []Filter) {
	p.filters = filters
}

// HaveFilter returns true if request contains some filter
func (p *QueryParser) HaveFilter(key string) bool {

	for _, v := range p.filters {
		if v.name == key {
			return true
		}
	}

	return false
}

// Where returns list of filters for WHERE statement
func (p *QueryParser) Where() string {

	if len(p.filters) == 0 {
		return ""
	}

	var where []string

	for i := 0; i < len(p.filters); i++ {
		filter := p.filters[i]

		var exp string
		switch filter.method {
		case MethodEQ, MethodNE, MethodGT, MethodLT, MethodGTE, MethodLTE, MethodLIKE:
			exp = fmt.Sprintf("%s %s ?", filter.name, TranslateMethods[filter.method])
		case MethodIN:
			exp = fmt.Sprintf("%s %s (?)", filter.name, TranslateMethods[filter.method])
			exp, _, _ = in(exp, filter.value)
		default:
			continue
		}

		where = append(where, exp)
	}

	return " WHERE " + strings.Join(where, " AND ")
}

// Args returns slice of arguments for WHERE statement
func (p *QueryParser) Args() []interface{} {

	args := make([]interface{}, 0)

	if len(p.filters) == 0 {
		return args
	}

	for i := 0; i < len(p.filters); i++ {
		filter := p.filters[i]

		switch filter.method {
		case MethodEQ, MethodNE, MethodGT, MethodLT, MethodGTE, MethodLTE:
			args = append(args, filter.value)
		case MethodLIKE:
			value := filter.value.(string)
			value = strings.Replace(value, "*", "%", -1)
			args = append(args, value)
		case MethodIN:
			_, params, _ := in("?", filter.value)
			args = append(args, params...)
		default:
			continue
		}
	}

	return args
}

func (p *QueryParser) SQL(table string) string {
	return fmt.Sprintf(
		"SELECT %s FROM %s%s%s%s%s",
		p.Fields(),
		table,
		p.Where(),
		p.Sort(),
		p.Limit(),
		p.Offset(),
	)
}

func defaults() *QueryParser {
	return &QueryParser{
		delimiter: ",",
	}
}

// SetQuery change url query for the instance
func (p *QueryParser) SetQuery(query map[string][]string) *QueryParser {
	p.query = query
	return p
}

// SetValidations change validations rules for the instance
func (p *QueryParser) SetValidations(validations Validations) *QueryParser {
	p.validations = validations
	return p
}

func New(query map[string][]string, validations Validations) *QueryParser {
	return defaults().SetQuery(query).SetValidations(validations)
}

func NewParse(query map[string][]string, validations Validations) (*QueryParser, error) {
	q := New(query, validations)
	return q, q.Parse()
}

// Parse parses the query of URL
// as query you can use standart http.Request query by r.URL.Query()
func (p *QueryParser) Parse() error {

	for key, value := range p.query {

		if strings.ToUpper(key) == "FIELDS" {
			if err := p.parseFields(value, p.validations[key]); err != nil {
				return err
			}
		} else if strings.ToUpper(key) == "OFFSET" {
			if err := p.parseOffset(value, p.validations[key]); err != nil {
				return err
			}
		} else if strings.ToUpper(key) == "LIMIT" {
			if err := p.parseLimit(value, p.validations[key]); err != nil {
				return err
			}
		} else if strings.ToUpper(key) == "SORT" {
			if err := p.parseSort(value, p.validations[key]); err != nil {
				return err
			}
		} else {
			filter, err := parseFilterKey(key)
			if err != nil {
				return err
			}

			allowed := false
			validationFunc := p.validations[filter.name]
			_type := "string"

			for k, v := range p.validations {
				if strings.Contains(k, ":") {
					split := strings.Split(k, ":")
					if split[0] == filter.name {
						allowed = true
						validationFunc = v
						_type = split[1]
						break
					}
				} else if k == filter.name {
					allowed = true
					break
				}
			}

			if !allowed {
				if p.ignoreUnknown {
					continue
				} else {
					return ErrFilterNotAllowed
				}
			}

			if err = p.parseFilterValue(filter, _type, value, validationFunc); err != nil {
				p.ErrorMessage = fmt.Sprintf("%s: %v", key, err)
				return err
			}
		}
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
			by:   by,
			desc: desc,
		})
	}

	p.sorts = sort

	return nil
}

func (p *QueryParser) parseFields(value []string, validate ValidationFunc) error {
	if len(value) == 1 {
		list := value
		if strings.Contains(value[0], p.delimiter) {
			list = strings.Split(value[0], p.delimiter)
		}

		list = cleanSliceString(list)

		if validate != nil {
			for _, v := range list {
				if err := validate(v); err != nil {
					return err
				}
			}
		}

		p.fields = list
		return nil
	}
	return ErrBadFormat
}

func (p *QueryParser) parseOffset(value []string, validate ValidationFunc) error {

	if len(value) != 1 {
		return ErrBadFormat
	}

	if len(value[0]) == 0 {
		return nil
	}

	var err error

	p.offset, err = strconv.Atoi(value[0])
	if err != nil {
		return err
	}

	if validate != nil {
		if err := validate(p.offset); err != nil {
			return err
		}
	}

	return nil
}

func (p *QueryParser) parseLimit(value []string, validate ValidationFunc) error {

	if len(value) != 1 {
		return ErrBadFormat
	}

	if len(value[0]) == 0 {
		return nil
	}

	var err error

	p.limit, err = strconv.Atoi(value[0])
	if err != nil {
		return err
	}

	if validate != nil {
		if err := validate(p.limit); err != nil {
			return err
		}
	}

	return nil
}
