package main

import (
	"errors"
	"fmt"
	"net/url"

	rqp "github.com/nfidel/rest-query-parser"
)

func main() {

	// Filter is parameter provided in the Query part of the URL
	//   The lib handles system filters:
	//     * fields - list of fields separated by comma (",") for SELECT statement. Should be validated.
	//     * sort   - list of fields separated by comma (",") for ORDER BY statement. Should be validated. Could includes prefix +/- which means ASC/DESC sorting. Eg. &sort=-id will be ORDER BY id DESC.
	//     * limit  - number for LIMIT statement. Should be greater then 0 by default.
	//     * offset - number for OFFSET statement. Should be greater then or equal to 0 by default.
	//   and user defined filters.
	//
	// Validation is a function for validate some Filter
	//
	// Field is enumerated in the Filter "fields" field which lib must put into SELECT statement.

	url, _ := url.Parse("http://localhost/?sort=+name,-id&limit=10&id=1&i[eq]=5&s[eq]=one&email[like]=*tim*|name[like]=*tim*")
	q, err := rqp.NewParse(url.Query(), rqp.Validations{
		// FORMAT: [field name] : [ ValidationFunc | nil ]

		// validation will work if field will be provided in the Query part of the URL
		// but if you add ":required" tag the Parser raise an Error if the field won't be in the Query part

		// special system fields: fields, limit, offset, sort
		// filters "fields" and "sort" must be always validated
		// If you won't define ValidationFunc but include "fields" or "sort" parameter to the URL the Parser raises an Error
		"limit:required": rqp.MinMax(10, 100),        // limit must present in the Query part and must be between 10 and 100 (default: Min(1))
		"sort":           rqp.InString("id", "name"), // sort could be or not in the query but if it is present it must be equal to "in" or "name"

		"s":      rqp.InString("one", "two"), // filter: s - string and equal
		"id:int": nil,                        // filter: id is integer without additional validation
		"i:int": func(value interface{}) error { // filter: custom func for validating
			if value.(int) > 1 && value.(int) < 10 {
				return nil
			}
			return errors.New("i: must be greater then 1 and lower then 10")
		},
		"email": nil,
		"name":  nil,
	})

	if err != nil {
		panic(err)
	}

	fmt.Println(q.SQL("table")) // SELECT * FROM table WHERE id = ? AND i = ? AND s = ? AND (email LIKE ? OR name LIKE ?) ORDER BY name, id DESC LIMIT 10
	fmt.Println(q.Where())      // id = ? AND i = ? AND s = ? AND (email LIKE ? OR name LIKE ?)
	fmt.Println(q.Args())       // [1 5 one %tim% %tim%]

	q.AddValidation("fields", rqp.InString("id", "name"))
	q.SetUrlString("http://localhost/?fields=id,name&limit=10")
	q.Parse()

	fmt.Println(q.SQL("table")) // SELECT id, name FROM table ORDER BY id LIMIT 10
	fmt.Println(q.Select())     // id, name
	fmt.Println(q.Args())       // []
}
