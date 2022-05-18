package main

import (
	"fmt"
	"net/url"

	rqp "github.com/nfidel/rest-query-parser/v2"
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

	url, _ := url.Parse("http://localhost/?fields=flights,pace,frequency_cap&pace.pace[gt]=6&flights[not]=null&pace=poo&pace.pacing_strategy=asap") //&flights[is]=null&frequency_cap[is]=null&frequency_cap.impressions.test[is]=NULL")
	q := rqp.NewQV(url.Query(),
		rqp.Validations{
			"fields": rqp.In("pace", "frequency_cap", "flights"),
		},
		rqp.QueryDbMap{
			"pace.pace":                 {Name: "global_bid_rate", Table: "campaign_pace", Type: "float"},
			"pace.pacing_strategy":      {Name: "pacing_strategy", Table: "campaign", Type: "string"},
			"frequency_cap":             {Name: "frequency_cap", Table: "campaign", Type: "custom"},
			"frequency_cap.impressions": {Name: "frequency_cap.impressions", Table: "campaign", Type: "float"},
		})
	q.IgnoreUnknownFilters(false)
	q.AllowSpecialFilters("flights", "pace")
	//q.Allow
	err := q.Parse()
	if err != nil {
		panic(err)
	}

	fmt.Println(q.SQL("campaign_pace"))                // SELECT * FROM table WHERE id = ? AND i = ? AND s = ? AND (email LIKE ? OR name LIKE ?) ORDER BY name, id DESC LIMIT 10
	fmt.Println(q.SQL("campaign"))                     // SELECT * FROM table WHERE id = ? AND i = ? AND s = ? AND (email LIKE ? OR name LIKE ?) ORDER BY name, id DESC LIMIT 10
	fmt.Println(q.Select("campaign", "campaign_pace")) // SELECT * FROM table WHERE id = ? AND i = ? AND s = ? AND (email LIKE ? OR name LIKE ?) ORDER BY name, id DESC LIMIT 10
	//fmt.Println(q.Where()) // id = ? AND i = ? AND s = ? AND (email LIKE ? OR name LIKE ?)
	// fmt.Println(q.Args()) // [1 5 one %tim% %tim%]
	fmt.Println(q.HaveQueryFilter("pace.pace"))
	fmt.Println(q.HaveQueryFilter("global_bid_rate"))
	fmt.Println(q.HaveQueryField("pace"))
	fmt.Println(q.HaveQueryFilter("pace"))
	fmt.Println(q.HaveQueryFilter("flights"))
	fmt.Println(q.HaveQueryField("flights"))

	// q.AddValidation("fields", rqp.In("id", "name"))
	// q.SetUrlString("http://localhost/?fields=id,name&price.goal=10&inventory_targeting.test[is]=null&inventory_targeting[is]=null&flights[is]=null")
	// err = q.Parse()
	// if err != nil {
	// 	panic(err)
	// }
	// //fmt.Println(q.SQL("table")) // SELECT id, name FROM table ORDER BY id LIMIT 10
	// fmt.Println(q.Select()) // id, name
	// fmt.Println(q.Where())  // id, name
	// fmt.Println(q.Args())   // []
}
