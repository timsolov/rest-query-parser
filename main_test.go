package rqp

import (
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestSetDelimiterOR(t *testing.T) {
	q := New()
	q.SetDelimiterOR("!")
	assert.Equal(t, q.delimiterOR, "!")
}

func TestSelect(t *testing.T) {
	q := New()
	assert.Equal(t, q.Select(), "*")

	q.AddQueryField("test1")
	q.AddQueryField("test2")
	assert.Equal(t, q.Select(), "test1, test2")
}

func TestSELECT(t *testing.T) {
	q := New()
	assert.Equal(t, q.SELECT(), "SELECT *")

	q.AddQueryField("test1")
	q.AddQueryField("test2")
	assert.Equal(t, q.SELECT(), "SELECT test1, test2")
}

func TestOrder(t *testing.T) {
	q := New()
	assert.Equal(t, q.Order(), "")
}

func TestHaveSortBy(t *testing.T) {
	q := New()
	assert.Equal(t, q.HaveQuerySortBy("fake"), false)
}

func TestRemoveQueryFilter(t *testing.T) {
	q := New()
	q.AddQueryFilter("id", ILIKE, "id")
	q.AddQueryFilter("test", ILIKE, "test")
	q.AddQueryFilter("test2", ILIKE, "test2")
	assert.NoError(t, q.RemoveQueryFilter("test"))
}

func TestGetQueryFilter(t *testing.T) {
	q := New()
	q.AddQueryFilter("id", ILIKE, "id")
	q.AddQueryFilter("test", ILIKE, "test")
	q.AddQueryFilter("test2", ILIKE, "test2")
	_, err := q.GetQueryFilter("test")
	assert.NoError(t, err)
}

func TestFields(t *testing.T) {

	// mockValidation := func(value interface{}) error { return nil }
	validate := In("id", "name")

	// Fields:
	cases := []struct {
		url      string
		expected string
		v        ValidationFunc
		err      error
	}{
		{url: "?", expected: "*", v: validate, err: nil},
		{url: "?fields=", expected: "*", v: validate, err: nil},
		{url: "?fields=id", expected: "id", v: validate, err: nil},
		{url: "?fields=id,name", expected: "id, name", v: validate, err: nil},
		{"?fields=", "*", nil, ErrValidationNotFound},
	}

	for _, c := range cases {
		t.Run(c.url, func(t *testing.T) {
			URL, err := url.Parse(c.url)
			assert.NoError(t, err)
			q := NewQV(URL.Query(), nil, nil)
			assert.NoError(t, err)
			q.AddQueryValidation("fields", c.v)
			err = q.Parse()
			if c.err != nil {
				assert.Equal(t, c.expected, q.QueryFields)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.err, errors.Cause(err))
			}
		})
	}
}

func TestOffset(t *testing.T) {

	// Offset:
	cases := []struct {
		url      string
		expected string
		err      error
	}{
		{url: "?", expected: ""},
		{url: "?offset=", expected: "", err: ErrBadFormat},
		{url: "?offset=-1", expected: "", err: ErrNotInScope},
		{url: "?offset=num", expected: "", err: ErrBadFormat},
		{url: "?offset=11", expected: "", err: ErrNotInScope},
		{url: "?offset[in]=10", expected: " OFFSET 10"},
	}
	for _, c := range cases {
		//t.Log(c)
		URL, err := url.Parse(c.url)
		assert.NoError(t, err)
		q := New().
			SetUrlQuery(URL.Query()).
			AddQueryValidation("page", Max(10))
		err = q.Parse()
		assert.Equal(t, c.err, errors.Cause(err))
		assert.Equal(t, c.expected, q.OFFSET())
	}
}

func TestLimit(t *testing.T) {
	// Limit
	cases := []struct {
		url      string
		expected string
		err      error
	}{
		{url: "?", expected: ""},
		{url: "?limit=", expected: "", err: ErrBadFormat},
		{url: "?limit=1,2", expected: "", err: ErrBadFormat},
		{url: "?limit=11", expected: "", err: ErrNotInScope},
		{url: "?limit=-1", expected: "", err: ErrNotInScope},
		{url: "?limit=1", expected: "", err: ErrNotInScope},
		{url: "?limit=q", expected: "", err: ErrBadFormat},
		{url: "?limit=10", expected: " LIMIT 10"},
	}
	for _, c := range cases {
		t.Run(c.url, func(t *testing.T) {
			URL, err := url.Parse(c.url)
			assert.NoError(t, err)
			q := New().
				SetUrlQuery(URL.Query()).
				AddQueryValidation("page_size", Multi(Min(2), Max(10)))
			err = q.Parse()
			assert.Equal(t, c.err, errors.Cause(err))
			assert.Equal(t, c.expected, q.LIMIT())
		})
	}
}

func TestSort(t *testing.T) {

	cases := []struct {
		url      string
		expected string
		err      error
	}{
		{url: "?", expected: ""},
		{url: "?sort=", expected: ""},
		{url: "?sort=id", expected: " ORDER BY id"},
		{url: "?sort=+id", expected: " ORDER BY id"},
		{url: "?sort=-id", expected: " ORDER BY id DESC"},
		{url: "?sort=id,-name", expected: " ORDER BY id, name DESC"},
	}
	for _, c := range cases {
		t.Run(c.url, func(t *testing.T) {
			URL, err := url.Parse(c.url)
			assert.NoError(t, err)
			q, err := NewParse(URL.Query(), Validations{"sort": In("id", "name")}, nil)
			assert.Equal(t, c.err, err)
			assert.Equal(t, c.expected, q.ORDER())
		})
	}

	q := New().SetValidations(Validations{"sort": In("id")})
	err := q.SetUrlString("://")
	assert.Error(t, err)

	err = q.SetUrlString("?sort=id")
	assert.NoError(t, err)

	err = q.Parse()
	assert.NoError(t, err)
	assert.True(t, q.HaveQuerySortBy("id"))

	// Test AddSortBy
	q.AddQuerySortBy("email", true)
	assert.True(t, q.HaveQuerySortBy("email"))
}

func TestWhere(t *testing.T) {

	cases := []struct {
		url       string
		expected  string
		expected2 string
		err       string
		ignore    bool
	}{
		{url: "?", expected: ""},
		{url: "?id", expected: "", err: "id: empty value"},
		{url: "?id=", expected: "", err: "id: empty value"},
		{url: "?u=", expected: "", err: "u: empty value"},
		{url: "?id=1.2", expected: "", err: "id: bad format"},
		{url: "?id[in]=1.2", expected: "", err: "id[in]: bad format"},
		{url: "?id[in]=1.2,1.2", expected: "", err: "id[in]: bad format"},
		{url: "?id[nin]=1.2", expected: "", err: "id[nin]: bad format"},
		{url: "?id[nin]=1.2,1.2", expected: "", err: "id[nin]: bad format"},
		{url: "?id[test]=1", expected: "", err: "id[test]: unknown method"},
		{url: "?id[like]=1", expected: "", err: "id[like]: method are not allowed"},
		{url: "?id=1,2", expected: "", err: "id: method are not allowed"},
		{url: "?id=4", expected: " WHERE id = ?"},

		{url: "?id=100", err: "id: can't be greater then 10"},
		{url: "?id[in]=100,200", err: "id[in]: can't be greater then 10"},
		{url: "?id[nin]=100,200", err: "id[nin]: can't be greater then 10"},

		// not like, not ilike:
		{url: "?u[nlike]=superman", expected: " WHERE u NOT LIKE ?"},
		{url: "?u[nilike]=superman", expected: " WHERE u NOT ILIKE ?"},

		{url: "?id=1&name=superman", expected: " WHERE id = ?", ignore: true},
		{url: "?id=1&name=superman&s[like]=super", expected: " WHERE id = ? AND s LIKE ?", expected2: " WHERE s LIKE ? AND id = ?", ignore: true},
		{url: "?s=super", expected: " WHERE s = ?"},
		{url: "?s[in]=super,puper", err: "s[in]: puper: not in scope"},
		{url: "?s[in]=super,best", expected: " WHERE s IN (?, ?)"},
		{url: "?s[nin]=super,puper", err: "s[nin]: puper: not in scope"},
		{url: "?s[nin]=super,best", expected: " WHERE s NOT IN (?, ?)"},
		{url: "?s=puper", expected: "", err: "s: puper: not in scope"},
		{url: "?u=puper", expected: " WHERE u = ?"},
		{url: "?u[eq]=1,2", expected: "", err: "u[eq]: method are not allowed"},
		{url: "?u[gt]=1", expected: " WHERE u > ?"},
		{url: "?id[in]=1,2", expected: " WHERE id IN (?, ?)"},
		{url: "?id[eq]=1&id[eq]=4", expected: " WHERE id = ? AND id = ?"},
		{url: "?id[gte]=1&id[lte]=4", expected: " WHERE id >= ? AND id <= ?", expected2: " WHERE id <= ? AND id >= ?"},
		{url: "?id[gte]=1|id[lte]=4", expected: " WHERE (id >= ? OR id <= ?)", expected2: " WHERE (id <= ? OR id >= ?)"},
		// null:
		{url: "?u[not]=NULL", expected: " WHERE u IS NOT NULL"},
		{url: "?u[is]=NULL", expected: " WHERE u IS NULL"},
		// bool:
		{url: "?b=true", expected: " WHERE b = ?"},
		{url: "?b=true1", err: "b: bad format"},
		{url: "?b[not]=true", err: "b[not]: method are not allowed"},
		{url: "?b[eq]=true,false", err: "b[eq]: method are not allowed"},
	}
	for _, c := range cases {
		t.Run(c.url, func(t *testing.T) {
			URL, err := url.Parse(c.url)
			assert.NoError(t, err)

			q := NewQV(URL.Query(), Validations{
				"id:int": func(value interface{}) error {
					if value.(int) > 10 {
						return errors.New("can't be greater then 10")
					}
					return nil
				},
				"s": In(
					"super",
					"best",
				),
				"u:string": nil,
				"b:bool":   nil,
				"custom": func(value interface{}) error {
					return nil
				},
			}, nil).IgnoreUnknownFilters(c.ignore)

			err = q.Parse()

			if len(c.err) > 0 {
				assert.EqualError(t, err, c.err)
			} else {
				assert.NoError(t, err)
			}
			where := q.WHERE()
			//t.Log(q.SQL("table"), q.Args())
			if len(c.expected2) > 0 {
				//t.Log("expected:", c.expected, "or:", c.expected2, "got:", where)
				assert.True(t, c.expected == where || c.expected2 == where)
			} else {
				assert.Equal(t, c.expected, where)
			}
		})
	}
}

func TestWhere2(t *testing.T) {

	q := NewQV(nil, Validations{
		"id": func(value interface{}) error {
			if value.(int) > 10 {
				return errors.New("can't be greater then 10")
			}
			return nil
		},
		"s": In(
			"super",
			"best",
		),
		"u": nil,
		"custom": func(value interface{}) error {
			return nil
		},
	}, nil)
	assert.NoError(t, q.SetUrlString("?id[eq]=10&s[like]=super|u[like]=*best*&id[gt]=1"))
	assert.NoError(t, q.Parse())
	//t.Log(q.SQL("tab"), q.Args())
	assert.NoError(t, q.SetUrlString("?id[eq]=10&s[like]=super|u[like]=&id[gt]=1"))
	assert.EqualError(t, q.Parse(), "u[like]: empty value")
}

func TestWhere3(t *testing.T) {
	q := NewQV(nil, Validations{
		"test1": nil,
		"test2": nil,
	}, nil)
	URL, err := url.Parse("?test1[eq]=test10|test2[eq]=test20&test1[eq]=test11|test2[eq]=test21")
	assert.NoError(t, err)
	assert.NoError(t, q.SetUrlQuery(URL.Query()).Error)
	assert.NoError(t, q.Parse())
	where := q.Where()
	assert.Equal(t, where, "(test1 = ? OR test2 = ?) AND (test1 = ? OR test2 = ?)")
}

func TestArgs(t *testing.T) {
	q := New()
	q.SetDelimiterIN("!")
	assert.Len(t, q.Args(), 0)
	// setup url
	URL, err := url.Parse("?fields=id!status&sort=id!+id!-id&offset=10&one=123&two=test&three[like]=*www*&three[in]=www1!www2&four[not]=NULL")
	assert.NoError(t, err)

	err = q.SetUrlQuery(URL.Query()).SetValidations(Validations{
		"fields": In("id", "status"),
		"sort":   In("id"),
		"one":    nil,
		"two":    nil,
		"three":  nil,
		"four":   nil,
	}).Parse()
	assert.NoError(t, err)

	assert.Len(t, q.Args(), 5)
	assert.Contains(t, q.Args(), 123)
	assert.Contains(t, q.Args(), "test")
	assert.Contains(t, q.Args(), "%www%")
	assert.Contains(t, q.Args(), "www1")
	assert.Contains(t, q.Args(), "www2")
}

func TestSQL(t *testing.T) {
	URL, err := url.Parse("?fields=id,status&sort=id&offset=10&some=123")
	assert.NoError(t, err)

	q := New().SetUrlQuery(URL.Query()).
		AddQueryValidation("fields", In("id", "status")).
		AddQueryValidation("sort", In("id"))
	q.IgnoreUnknownFilters(true)
	err = q.Parse()
	assert.NoError(t, err)
	assert.Equal(t, "SELECT id, status FROM test ORDER BY id OFFSET 10", q.SQL("test"))

	q.AddQueryValidation("some", nil)
	err = q.Parse()
	assert.NoError(t, err)

	assert.Equal(t, "SELECT id, status FROM test WHERE some = ? ORDER BY id OFFSET 10", q.SQL("test"))
}

func TestAddQueryField(t *testing.T) {
	q := New()
	q.SetUrlString("?test=ok")
	q.AddQueryField("test")
	assert.Len(t, q.QueryFields, 1)
	assert.True(t, q.HaveQueryField("test"))
}

func TestAddQueryFilter(t *testing.T) {
	q := New().AddQueryFilter("test", EQ, "ok")
	assert.Len(t, q.Filters, 1)
	assert.True(t, q.HaveQueryFilter("test"))
	assert.Equal(t, "test = ?", q.Where())
}

func Test_ignoreUnknown(t *testing.T) {
	q := New()
	q.SetUrlString("?id=10")
	q.IgnoreUnknownFilters(true)
	assert.NoError(t, q.Parse())

	q.IgnoreUnknownFilters(false)
	assert.Equal(t, ErrFilterNotFound, errors.Cause(q.Parse()))

	q.SetUrlString("?id[gt]=10|id[lt]=10")
	q.IgnoreUnknownFilters(true)
	assert.NoError(t, q.Parse())

	q.IgnoreUnknownFilters(false)
	assert.Equal(t, ErrFilterNotFound, errors.Cause(q.Parse()))

}

func Test_RemoveQueryFilter(t *testing.T) {
	t.Run("?id[eq]=10|id[eq]=11", func(t *testing.T) {
		q := New()
		q.SetValidations(Validations{
			"id:int": nil,
			"u:int":  nil,
		})
		assert.NoError(t, q.SetUrlString("?id[eq]=10|id[eq]=11"))
		assert.NoError(t, q.Parse())
		assert.NoError(t, q.RemoveQueryFilter("id"))
		assert.Equal(t, "SELECT * FROM test", q.SQL("test"))
	})
}

func Test_MultipleUsageOfUniqueFilters(t *testing.T) {

	q := New()
	q.SetValidations(Validations{
		"id:int": nil,
		"u:int":  nil,
	})
	assert.NoError(t, q.SetUrlString("?id[eq]=10|u[eq]=10&id[eq]=11|u[eq]=11"))
	assert.NoError(t, q.Parse())
	assert.Equal(t, "SELECT * FROM test WHERE (id = ? OR u = ?) AND (id = ? OR u = ?)", q.SQL("test"))
	t.Log(q.SQL("test"))
}

func TestEmptySliceFilterWithAnotherFilter(t *testing.T) {
	q := New().AddQueryFilter("id", IN, []string{})
	q.AddQueryFilter("another_id", EQ, uuid.New().String())
	t.Log(q.SQL("test"))
}

func TestQuery_AddORFilters(t *testing.T) {
	t.Run("2 OR conditions", func(t *testing.T) {
		q := New().AddQueryFilter("test", EQ, "ok")
		q.AddORFilters(func(query *Query) {
			query.AddQueryFilter("firstname", ILIKE, "*hello*")
			query.AddQueryFilter("lastname", ILIKE, "*hello*")
		})
		out := q.SQL("table")
		t.Log(out)
		assert.Equal(t, `SELECT * FROM table WHERE test = ? AND (firstname ILIKE ? OR lastname ILIKE ?)`, out)
	})

	t.Run("3 OR conditions", func(t *testing.T) {
		q := New().AddQueryFilter("test", EQ, "ok")
		q.AddORFilters(func(query *Query) {
			query.AddQueryFilter("firstname", ILIKE, "*hello*")
			query.AddQueryFilter("lastname", ILIKE, "*hello*")
			query.AddQueryFilter("email", ILIKE, "*hello*")
		})
		out := q.SQL("table")
		t.Log(out)
		assert.Equal(t, `SELECT * FROM table WHERE test = ? AND (firstname ILIKE ? OR lastname ILIKE ? OR email ILIKE ?)`, out)
	})
}

func ExampleQuery_AddORFilters() {
	q := New().AddQueryFilter("test", EQ, "ok")
	q.AddORFilters(func(query *Query) {
		query.AddQueryFilter("firstname", ILIKE, "*hello*")
		query.AddQueryFilter("lastname", ILIKE, "*hello*")
	})
	q.SQL("table") // SELECT * FROM table WHERE test = ? AND (firstname ILIKE ? OR lastname ILIKE ?)
}
