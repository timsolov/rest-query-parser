package main

import (
	"errors"
	"fmt"
	"net/url"

	rqp "github.com/timsolov/rest-query-parser"
)

func main() {
	url, _ := url.Parse("http://localhost/?limit=10&id=1&i[eq]=5&s[eq]=one&email[like]=*tim*|name[like]=*tim*")
	q, _ := rqp.NewParse(url.Query(), rqp.Validations{
		// filter : validation func
		// filters will work if variable is provided in query
		// but is you add :required then parser raise error when variable is not in query
		// special system filters:
		"limit:required": rqp.MinMax(10, 100),  // limit must present in query and must be between 10 and 100
		"sort":           rqp.In("id", "name"), // sort could be or not in the query but if it is present it must be equal to "in" or "name"
		// user's filters:
		"s": rqp.In( // filter: s - string and equal
			"one",
			"two",
		),
		"id:int": nil, // filter: id is integer without additional validation
		"i:int": func(value interface{}) error { // filter: custom func for filtering
			if value.(int) > 10 {
				return errors.New("i: must be lower then 10")
			}
			return nil
		},
		"email": nil,
		"name":  nil,
	})

	fmt.Println(q.SQL("table")) // will print: SELECT * FROM table WHERE id = ? AND i = ? AND s = ? AND (email LIKE ? OR name LIKE ?) LIMIT 10
	fmt.Println(q.Args())       // will print: [one %tim% %tim% 1 5]
}
