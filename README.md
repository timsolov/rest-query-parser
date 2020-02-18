# Query Parser for REST
Query Parser is a library for easy make dynamic SQL queries to database.  It provides a simple API for web-applications which needs to do some filtering throught GET queries. It is a connector between the HTTP handler and the DB engine, and manages validations and translations for user inputs.

## Installation
    go get -u github.com/timsolov/rest-query-parser

## Fast start
```go
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
```

## Top level fields:
* `fields` - fields for SELECT clause separated by comma (",") Eg. `&fields=id,name`. If nothing provided use "\*" by default. Attention! Use `rqp.In()` func for limit fields for your table.
* `limit` - is limit for LIMIT clause. Adds to SQL if > 0.
* `offset` - is offset for OFFSET clause.
* `sort` - sorting fields list separated by comma (","). Could include prefix +/- which means ASC/DESC sorting. Eg. `&sort=+id,-name` will print `ORDER BY id, name DESC`. You could filter fields in this parameter by adding `rqp.In("id", "name")` in validation.

## Supported compare methods
- `eq`
- `ne`
- `gt`
- `lt`
- `gte`
- `lte`
- `like`
- `ilike`
- `not`
- `in`
