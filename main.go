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

	Fields  []string
	Offset  int
	Limit   int
	Sorts   []Sort
	Filters []*Filter

	delimiter     string
	ignoreUnknown bool

	ErrorMessage string
}

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
	By   string
	Desc bool
}

// Delimiter sets delimiter for values of multiple filter
func (p *Query) Delimiter(delimiter string) *Query {
	p.delimiter = delimiter
	return p
}

// IgnoreUnknownFilters set behavior for Parser to raise ErrFilterNotAllowed to undefined filters or not
func (p *Query) IgnoreUnknownFilters(i bool) *Query {
	p.ignoreUnknown = i
	return p
}

// FieldsSQL returns elements for querying in SELECT statement or * if fields parameter not specified
func (p *Query) FieldsSQL() string {
	if len(p.Fields) == 0 {
		return "*"
	}
	return strings.Join(p.Fields, ", ")
}

// HaveField returns true if request asks for field
func (p *Query) HaveField(field string) bool {
	return stringInSlice(field, p.Fields)
}

// AddField returns true if request asks for field
func (p *Query) AddField(field string) {
	p.Fields = append(p.Fields, field)
}

// OffsetSQL returns OFFSET statement
func (p *Query) OffsetSQL() string {
	if p.Offset > 0 {
		return fmt.Sprintf(" OFFSET %d", p.Offset)
	}
	return ""
}

// LimitSQL returns LIMIT statement
func (p *Query) LimitSQL() string {
	if p.Limit > 0 {
		return fmt.Sprintf(" LIMIT %d", p.Limit)
	}
	return ""
}

// SortSQL returns ORDER BY statement
// you can use +/- prefix to specify direction of sorting (+ is default)
func (p *Query) SortSQL() string {
	if len(p.Sorts) == 0 {
		return ""
	}

	s := " ORDER BY "

	for i := 0; i < len(p.Sorts); i++ {
		if i > 0 {
			s += ", "
		}
		if p.Sorts[i].Desc {
			s += fmt.Sprintf("%s DESC", p.Sorts[i].By)
		} else {
			s += p.Sorts[i].By
		}
	}

	return s
}

// HaveSortBy returns true if request contains some sorting
func (p *Query) HaveSortBy(by string) bool {

	for _, v := range p.Sorts {
		if v.By == by {
			return true
		}
	}

	return false
}

// HaveFilter returns true if request contains some filter
func (p *Query) HaveFilter(name string) bool {

	for _, v := range p.Filters {
		if v.Name == name {
			return true
		}
	}

	return false
}

// RemoveFilter removes the filter by name
func (p *Query) RemoveFilter(name string) error {

	for i, v := range p.Filters {
		if v.Name == name {
			// safe remove element from slice
			if i < len(p.Filters)-1 {
				copy(p.Filters[i:], p.Filters[i+1:])
			}
			p.Filters[len(p.Filters)-1] = nil
			p.Filters = p.Filters[:len(p.Filters)-1]

			return nil
		}
	}

	return ErrFilterNotFound
}

// GetFilter returns filter by name
func (p *Query) GetFilter(name string) (*Filter, error) {

	for _, v := range p.Filters {
		if v.Name == name {
			return v, nil
		}
	}

	return nil, ErrFilterNotFound
}

// FiltersNamesReplacer struct for ReplaceFiltersNames method
type FiltersNamesReplacer map[string]string

// ReplaceFiltersNames replace all specified name to new names
// parameter is map[string]string which means map[currentName]newName
// usage: rqp.ReplaceFiltersNames(rqp.FiltersNamesReplacer{"oldName":"newName"})
func (p *Query) ReplaceFiltersNames(r FiltersNamesReplacer) {

	for name, newname := range r {
		for i, v := range p.Filters {
			if v.Name == name && !p.HaveFilter(newname) {
				p.Filters[i].Name = newname
			}
		}
	}

}

// Where returns list of filters for WHERE statement
func (p *Query) Where() string {

	if len(p.Filters) == 0 {
		return ""
	}

	var where []string

	for i := 0; i < len(p.Filters); i++ {
		filter := p.Filters[i]

		if a, err := filter.SQL(); err == nil {
			where = append(where, a)
		} else {
			continue
		}
	}

	return strings.Join(where, " AND ")
}

// WhereSQL returns list of filters for WHERE SQL statement
func (p *Query) WhereSQL() string {

	if len(p.Filters) == 0 {
		return ""
	}

	return " WHERE " + p.Where()
}

// Args returns slice of arguments for WHERE statement
func (p *Query) Args() []interface{} {

	args := make([]interface{}, 0)

	if len(p.Filters) == 0 {
		return args
	}

	for i := 0; i < len(p.Filters); i++ {
		filter := p.Filters[i]

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

// SetUrlQuery change url query for the instance
func (p *Query) SetUrlQuery(q url.Values) *Query {
	p.query = q
	return p
}

// SetUrlRaw change url query for the instance
func (p *Query) SetUrlRaw(Url string) error {
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
func New(q url.Values, v Validations) *Query {
	QP := &Query{
		delimiter: ",",
	}
	return QP.SetUrlQuery(q).SetValidations(v)
}

// NewParse creates new Query instance and Parse it
func NewParse(q url.Values, v Validations) (*Query, error) {
	query := New(q, v)
	return query, query.Parse()
}

// Parse parses the query of URL
// as query you can use standart http.Request query by r.URL.Query()
func (p *Query) Parse() error {

	// check if required
	for name, f := range p.validations {
		if strings.Contains(name, ":required") {
			oldname := name
			newname := strings.Replace(name, ":required", "", 1)

			if strings.Contains(newname, ":") {
				parts := strings.Split(newname, ":")
				name = parts[0]
			} else {
				name = newname
			}

			found := false
			for key, _ := range p.query {
				filter, err := parseFilterKey(key)
				if err != nil {
					return errors.Wrap(err, name)
				}
				if filter.Name == name {
					found = true
					break
				}
			}

			if !found {
				return errors.Wrap(ErrRequired, name)
			} else {
				p.validations[newname] = f
				delete(p.validations, oldname)
			}
		}
	}

	for key, value := range p.query {

		if strings.ToUpper(key) == "FIELDS" {
			if err := p.parseFields(value, p.validations[key]); err != nil {
				return errors.Wrap(err, key)
			}
		} else if strings.ToUpper(key) == "OFFSET" {
			if err := p.parseOffset(value, p.validations[key]); err != nil {
				return errors.Wrap(err, key)
			}
		} else if strings.ToUpper(key) == "LIMIT" {
			if err := p.parseLimit(value, p.validations[key]); err != nil {
				return errors.Wrap(err, key)
			}
		} else if strings.ToUpper(key) == "SORT" {
			if err := p.parseSort(value, p.validations[key]); err != nil {
				return errors.Wrap(err, key)
			}
		} else {
			filter, err := parseFilterKey(key)
			if err != nil {
				return errors.Wrap(err, key)
			}

			allowed := false
			validationFunc := p.validations[filter.Name]
			_type := "string"

			for k, v := range p.validations {
				if strings.Contains(k, ":") {
					split := strings.Split(k, ":")
					if split[0] == filter.Name {
						allowed = true
						validationFunc = v
						_type = split[1]
						break
					}
				} else if k == filter.Name {
					allowed = true
					break
				}
			}

			if !allowed {
				if p.ignoreUnknown {
					continue
				} else {
					return errors.Wrap(ErrFilterNotAllowed, key)
				}
			}

			if err = p.parseFilterValue(filter, _type, value, validationFunc); err != nil {
				return errors.Wrap(err, key)
			}
		}
	}

	return nil
}

func (p *Query) parseSort(value []string, validate ValidationFunc) error {
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
			By:   by,
			Desc: desc,
		})
	}

	p.Sorts = sort

	return nil
}

func (p *Query) parseFields(value []string, validate ValidationFunc) error {
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

		p.Fields = list
		return nil
	}
	return ErrBadFormat
}

func (p *Query) parseOffset(value []string, validate ValidationFunc) error {

	if len(value) != 1 {
		return ErrBadFormat
	}

	if len(value[0]) == 0 {
		return nil
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

func (p *Query) parseLimit(value []string, validate ValidationFunc) error {

	if len(value) != 1 {
		return ErrBadFormat
	}

	if len(value[0]) == 0 {
		return nil
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
