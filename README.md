# Query Parser for REST
Query Parser is a library for easy building dynamic SQL queries to Database. It provides a simple API for web-applications which needs to do some filtering throught GET queries. It is a connector between the HTTP handler and the DB engine, and manages validations and translations for user inputs.

## Installation
    go get -u github.com/timsolov/rest-query-parser

## Fast start
See cmd/main.go and tests for more examples.

```go
    package main
    
    import (
    	"errors"
    	"fmt"
    	"net/url"
    
    	rqp "github.com/timsolov/rest-query-parser"
    )
    
    func main() {
    	url, _ := url.Parse("http://localhost/?sort=+name,-id&limit=10&id=1&i[eq]=5&s[eq]=one&email[like]=*tim*|name[like]=*tim*")
		q, _ := rqp.NewParse(url.Query(), rqp.Validations{
			"limit:required": rqp.MinMax(10, 100),  // limit must present in the Query part and must be between 10 and 100 (default: Min(1))
			"sort":           rqp.In("id", "name"), // sort could be or not in the query but if it is present it must be equal to "in" or "name"
			"s":      rqp.In("one", "two"), // filter: s - string and equal
			"id:int": nil,                  // filter: id is integer without additional validation
			"i:int": func(value interface{}) error { // filter: custom func for validating
				if value.(int) > 1 && value.(int) < 10 {
					return nil
				}
				return errors.New("i: must be greater then 1 and lower then 10")
			},
			"email": nil,
			"name":  nil,
		})

		fmt.Println(q.SQL("table")) // SELECT * FROM table WHERE id = ? AND i = ? AND s = ? AND (email LIKE ? OR name LIKE ?) ORDER BY name, id DESC LIMIT 10
		fmt.Println(q.Where())      // id = ? AND i = ? AND s = ? AND (email LIKE ? OR name LIKE ?)
		fmt.Println(q.Args())       // [1 5 one %tim% %tim%]

		q.AddValidation("fields", rqp.In("id", "name"))
		q.SetUrlString("http://localhost/?fields=id,name&limit=10")
		q.Parse()

		fmt.Println(q.SQL("table")) // SELECT id, name FROM table ORDER BY id LIMIT 10
		fmt.Println(q.FieldsSQL())  // id, name
		fmt.Println(q.Args())       // []
    }
```

## Top level fields:
* `fields` - fields for SELECT clause separated by comma (",") Eg. `&fields=id,name`. If nothing provided use "\*" by default. Attention! If you want to use this filter you have to defaine validation func for it. Use `rqp.In()` func for limit fields for your query.
* `sort` - sorting fields list separated by comma (","). Must be validated too. Could include prefix +/- which means ASC/DESC sorting. Eg. `&sort=+id,-name` will print `ORDER BY id, name  DESC`. You have to filter fields in this parameter by adding `rqp.In("id", "name")`.
* `limit` - is limit for LIMIT clause. Should be greater then 0 by default. Definition of the validation for `limit` is not required.
* `offset` - is offset for OFFSET clause. Should be greater then or equal to 0 by default. Definition of the validation for `offset` is not required.


## Supported types
- `strings` - the default type for all provided filters if not specified another. Could be compared by `eq, ne, like, ilkie, in` methods.
- `int` - integer type. Must be specified with tag ":int". Could be compared by `eq, ne, gt, lt, gte, lte, in` methods.
- `bool` - boolean type. Must be specified with tag ":bool". Could be compared by `eq` method.

## TODO:
- new type `time`.