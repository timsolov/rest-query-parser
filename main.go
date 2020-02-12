package rqp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// QueryParser contatins of all major data
type QueryParser struct {
	query       map[string][]string
	validations Validations

	Fields  []string
	Offset  int
	Limit   int
	Sorts   []Sort
	Filters []Filter

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
func (p *QueryParser) Delimiter(delimiter string) *QueryParser {
	p.delimiter = delimiter
	return p
}

// IgnoreUnknownFilters set behavior for Parser to raise ErrFilterNotAllowed to undefined filters or not
func (p *QueryParser) IgnoreUnknownFilters(i bool) *QueryParser {
	p.ignoreUnknown = i
	return p
}

// FieldsSQL returns elements for querying in SELECT statement or * if fields parameter not specified
func (p *QueryParser) FieldsSQL() string {
	if len(p.Fields) == 0 {
		return "*"
	}
	return strings.Join(p.Fields, ", ")
}

// HaveField returns true if request asks for field
func (p *QueryParser) HaveField(field string) bool {
	return stringInSlice(field, p.Fields)
}

// AddField returns true if request asks for field
func (p *QueryParser) AddField(field string) {
	p.Fields = append(p.Fields, field)
}

// OffsetSQL returns OFFSET statement
func (p *QueryParser) OffsetSQL() string {
	if p.Offset > 0 {
		return fmt.Sprintf(" OFFSET %d", p.Offset)
	}
	return ""
}

// LimitSQL returns LIMIT statement
func (p *QueryParser) LimitSQL() string {
	if p.Limit > 0 {
		return fmt.Sprintf(" LIMIT %d", p.Limit)
	}
	return ""
}

// SortSQL returns ORDER BY statement
// you can use +/- prefix to specify direction of sorting (+ is default)
func (p *QueryParser) SortSQL() string {
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

// GetSorts getter for sort
func (p *QueryParser) GetSorts() []Sort {
	return p.Sorts
}

// SetSorts setter for sort
func (p *QueryParser) SetSorts(sort []Sort) {
	p.Sorts = sort
}

// HaveSortBy returns true if request contains some sorting
func (p *QueryParser) HaveSortBy(by string) bool {

	for _, v := range p.Sorts {
		if v.By == by {
			return true
		}
	}

	return false
}

// HaveFilter returns true if request contains some filter
func (p *QueryParser) HaveFilter(name string) bool {

	for _, v := range p.Filters {
		if v.Name == name {
			return true
		}
	}

	return false
}

// FiltersNamesReplacer struct for ReplaceFiltersNames method
type FiltersNamesReplacer map[string]string

// ReplaceFiltersNames replace all specified name to new names
// parameter is map[string]string which means map[currentName]newName
// usage: rqp.ReplaceFiltersNames(rqp.FiltersNamesReplacer{"oldName":"newName"})
func (p *QueryParser) ReplaceFiltersNames(replacer FiltersNamesReplacer) {

	for name, newname := range replacer {
		for i, v := range p.Filters {
			if v.Name == name && !p.HaveFilter(newname) {
				p.Filters[i].Name = newname
			}
		}
	}

}

// Where returns list of filters for WHERE statement
func (p *QueryParser) Where() string {

	if len(p.Filters) == 0 {
		return ""
	}

	var where []string

	for i := 0; i < len(p.Filters); i++ {
		filter := p.Filters[i]

		var exp string
		switch filter.Method {
		case MethodEQ, MethodNE, MethodGT, MethodLT, MethodGTE, MethodLTE, MethodLIKE:
			exp = fmt.Sprintf("%s %s ?", filter.Name, TranslateMethods[filter.Method])
		case MethodIN:
			exp = fmt.Sprintf("%s %s (?)", filter.Name, TranslateMethods[filter.Method])
			exp, _, _ = in(exp, filter.Value)
		default:
			continue
		}

		where = append(where, exp)
	}

	return strings.Join(where, " AND ")
}

// WhereSQL returns list of filters for WHERE SQL statement
func (p *QueryParser) WhereSQL() string {

	if len(p.Filters) == 0 {
		return ""
	}

	return " WHERE " + p.Where()
}

// Args returns slice of arguments for WHERE statement
func (p *QueryParser) Args() []interface{} {

	args := make([]interface{}, 0)

	if len(p.Filters) == 0 {
		return args
	}

	for i := 0; i < len(p.Filters); i++ {
		filter := p.Filters[i]

		switch filter.Method {
		case MethodEQ, MethodNE, MethodGT, MethodLT, MethodGTE, MethodLTE:
			args = append(args, filter.Value)
		case MethodLIKE:
			value := filter.Value.(string)
			value = strings.Replace(value, "*", "%", -1)
			args = append(args, value)
		case MethodIN:
			_, params, _ := in("?", filter.Value)
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
		p.FieldsSQL(),
		table,
		p.WhereSQL(),
		p.SortSQL(),
		p.LimitSQL(),
		p.OffsetSQL(),
	)
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
	QP := &QueryParser{
		delimiter: ",",
	}
	return QP.SetQuery(query).SetValidations(validations)
}

func NewParse(query map[string][]string, validations Validations) (*QueryParser, error) {
	q := New(query, validations)
	return q, q.Parse()
}

// Parse parses the query of URL
// as query you can use standart http.Request query by r.URL.Query()
func (p *QueryParser) Parse() error {

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
					return err
				}
				if filter.Name == name {
					found = true
					break
				}
			}

			if !found {
				return errors.New(fmt.Sprintf("%s: required", name))
			} else {
				p.validations[newname] = f
				delete(p.validations, oldname)
			}
		}
	}

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
					return ErrFilterNotAllowed
				}
			}

			if err = p.parseFilterValue(filter, _type, value, validationFunc); err != nil {
				return errors.Wrap(err, key)
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
			By:   by,
			Desc: desc,
		})
	}

	p.Sorts = sort

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

		p.Fields = list
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
