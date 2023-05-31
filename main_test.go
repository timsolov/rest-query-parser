package rqp

import (
	"net/url"
	"reflect"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
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

	q.AddField("test1")
	q.AddField("test2")
	assert.Equal(t, q.Select(), "test1, test2")
}

func TestSELECT(t *testing.T) {
	q := New()
	assert.Equal(t, q.SELECT(), "SELECT *")

	q.AddField("test1")
	q.AddField("test2")
	assert.Equal(t, q.SELECT(), "SELECT test1, test2")
}

func TestOrder(t *testing.T) {
	q := New()
	assert.Equal(t, q.Order(), "")
}

func TestHaveSortBy(t *testing.T) {
	q := New()
	assert.Equal(t, q.HaveSortBy("fake"), false)
}

func TestRemoveFilter(t *testing.T) {
	q := New()
	q.AddFilter("id", ILIKE, "id")
	q.AddFilter("test", ILIKE, "test")
	q.AddFilter("test2", ILIKE, "test2")
	assert.NoError(t, q.RemoveFilter("test"))
}

func TestGetFilter(t *testing.T) {
	q := New()
	q.AddFilter("id", ILIKE, "id")
	q.AddFilter("test", ILIKE, "test")
	q.AddFilter("test2", ILIKE, "test2")
	_, err := q.GetFilter("test")
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
			q := NewQV(URL.Query(), nil)
			assert.NoError(t, err)
			q.AddValidation("fields", c.v)
			err = q.Parse()
			if c.err != nil {
				assert.Equal(t, c.expected, q.FieldsString())
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
			AddValidation("offset", Max(10))
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
				AddValidation("limit", Multi(Min(2), Max(10)))
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
			q, err := NewParse(URL.Query(), Validations{"sort": In("id", "name")})
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
	assert.True(t, q.HaveSortBy("id"))

	// Test AddSortBy
	q.AddSortBy("email", true)
	assert.True(t, q.HaveSortBy("email"))
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
		// float
		{url: "?f[gte]=1.5&f[lte]=4.7", expected: " WHERE f >= ? AND f <= ?", expected2: " WHERE f <= ? AND f >= ?"},
		{url: "?f[gte]=1.5|f[lte]=4.7", expected: " WHERE (f >= ? OR f <= ?)", expected2: " WHERE (f <= ? OR f >= ?)"},
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
				"f:float": func(value interface{}) error {
					if value.(float32) > 8.5 {
						return errors.New("can't be greater then 8.5")
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
			}).IgnoreUnknownFilters(c.ignore)

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

			QueryEqual(t, q, q.Clone())
		})
	}
}

func TestWhere2(t *testing.T) {

	q := NewQV(nil, Validations{
		"id:int": func(value interface{}) error {
			if value.(int) > 10 {
				return errors.New("can't be greater then 10")
			}
			return nil
		},
		"f:float": func(value interface{}) error {
			if value.(float32) > 8.5 {
				return errors.New("can't be greater then 8.5")
			}
			return nil
		},
		"s": In(
			"super",
			"best",
		),
		"u:string": nil,
		"custom": func(value interface{}) error {
			return nil
		},
	})
	assert.NoError(t, q.SetUrlString("?id[eq]=10&f[gt]=4&s[like]=super|u[like]=*best*&id[gt]=1"))
	assert.NoError(t, q.Parse())
	//t.Log(q.SQL("tab"), q.Args())
	assert.NoError(t, q.SetUrlString("?id[eq]=10&s[like]=super|u[like]=&id[gt]=1"))
	assert.EqualError(t, q.Parse(), "u[like]: empty value")
}

func TestWhere3(t *testing.T) {
	q := NewQV(nil, Validations{
		"test1": nil,
		"test2": nil,
	})
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
		"fields":  In("id", "status"),
		"sort":    In("id"),
		"one:int": nil,
		"two":     nil,
		"three":   nil,
		"four":    nil,
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
		AddValidation("fields", In("id", "status")).
		AddValidation("sort", In("id"))
	q.IgnoreUnknownFilters(true)
	err = q.Parse()
	assert.NoError(t, err)
	assert.Equal(t, "SELECT id, status FROM test ORDER BY id OFFSET 10", q.SQL("test"))

	q.AddValidation("some:int", nil)
	err = q.Parse()
	assert.NoError(t, err)

	assert.Equal(t, "SELECT id, status FROM test WHERE some = ? ORDER BY id OFFSET 10", q.SQL("test"))
}

func TestReplaceFiltersNames(t *testing.T) {
	URL, err := url.Parse("?fields=one&sort=one&one=123&another=yes")
	assert.NoError(t, err)

	q, err := NewParse(URL.Query(), Validations{
		"fields":  In("one", "another", "two"),
		"sort":    In("one", "another", "two"),
		"one":     nil,
		"another": nil,
	})
	assert.NoError(t, err)
	assert.True(t, q.HaveFilter("one"))

	q.ReplaceNames(Replacer{
		"one": "two",
	})

	assert.Len(t, q.Filters, 2)
	assert.True(t, q.HaveFilter("two"))

	q.ReplaceNames(Replacer{
		"another":    "r.another",
		"nonpresent": "hello",
	})

	assert.Len(t, q.Filters, 2)
	assert.True(t, q.HaveFilter("two"))
	assert.True(t, q.HaveFilter("r.another"))
	assert.False(t, q.HaveFilter("one"))
	assert.False(t, q.HaveFilter("another"))
	assert.False(t, q.HaveFilter("nonpresent"))
	assert.False(t, q.HaveFilter("hello"))

	assert.NoError(t, q.RemoveFilter("r.another"))
	assert.Equal(t, q.RemoveFilter("r.another"), errors.Cause(ErrFilterNotFound))
	_, err = q.GetFilter("r.another")
	assert.Equal(t, err, errors.Cause(ErrFilterNotFound))
	f, _ := q.GetFilter("r.another")
	assert.IsType(t, &Filter{}, f)
}

func TestRequiredFilter(t *testing.T) {
	// required but not present
	URL, err := url.Parse("?")
	assert.NoError(t, err)

	_, err = NewParse(URL.Query(), Validations{"limit:required": nil})
	assert.EqualError(t, err, "limit: required")

	// required and present
	URL, err = url.Parse("?limit=10&one[eq]=1&count=4")
	assert.NoError(t, err)

	qp, err := NewParse(URL.Query(), Validations{
		"limit:required":     nil,
		"one:int":            nil,
		"count:int:required": nil,
	})
	assert.NoError(t, err)
	_, present := qp.validations["limit:required"]
	assert.False(t, present)
	_, present = qp.validations["limit"]
	assert.True(t, present)
}

func TestAddField(t *testing.T) {
	q := New()
	q.SetUrlString("?test=ok")
	q.AddField("test")
	assert.Len(t, q.Fields, 1)
	assert.True(t, q.HaveField("test"))
	assert.Equal(t, "test", q.FieldsString())
}

func TestAddFilter(t *testing.T) {
	q := New().AddFilter("test", EQ, "ok")
	assert.Len(t, q.Filters, 1)
	assert.True(t, q.HaveFilter("test"))
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

func TestRemoveValidation(t *testing.T) {
	q := New()

	// validation not found case
	assert.EqualError(t, q.RemoveValidation("fields"), ErrValidationNotFound.Error())

	// remove plain validation
	q.AddValidation("fields", In("id"))
	assert.NoError(t, q.RemoveValidation("fields"))

	// remove typed validation
	q.AddValidation("name:string", In("id"))
	assert.NoError(t, q.RemoveValidation("name"))
}

func Test_RemoveFilter(t *testing.T) {
	t.Run("?id[eq]=10|id[eq]=11", func(t *testing.T) {
		q := New()
		q.SetValidations(Validations{
			"id:int": nil,
			"u:int":  nil,
		})
		assert.NoError(t, q.SetUrlString("?id[eq]=10|id[eq]=11"))
		assert.NoError(t, q.Parse())
		assert.NoError(t, q.RemoveFilter("id"))
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

func Test_Date(t *testing.T) {

	q := New()

	q.SetValidations(Validations{
		"created_at": func(v interface{}) error {
			s, ok := v.(string)
			if !ok {
				return ErrBadFormat
			}
			return validation.Validate(s, validation.Date("2006-01-02"))
		},
	})

	cases := []struct {
		uri      string
		variant1 string
		variant2 string
	}{
		{
			uri:      "?created_at[eq]=2020-10-02",
			variant1: "SELECT * FROM test WHERE DATE(created_at) = ?",
		},
		{
			uri:      "?created_at[gt]=2020-10-01&created_at[lt]=2020-10-03",
			variant1: "SELECT * FROM test WHERE DATE(created_at) > ? AND DATE(created_at) < ?",
			variant2: "SELECT * FROM test WHERE DATE(created_at) < ? AND DATE(created_at) > ?",
		},
	}

	for _, tc := range cases {
		t.Run(tc.uri, func(t *testing.T) {
			q.SetUrlString(tc.uri)
			assert.NoError(t, q.Parse())
			q.ReplaceNames(Replacer{"created_at": "DATE(created_at)"})
			query := q.SQL("test")
			assert.Condition(t, func() bool {
				if tc.variant1 != query && tc.variant2 != query {
					t.Log(query)
					return false
				}
				return true
			})
		})
	}
}

func TestQuery_AddFilterRaw(t *testing.T) {
	q := New().AddFilter("test", EQ, "ok")
	q.AddFilterRaw("file_id != 'ec34d3b8-3013-43ee-ad7b-1d5d4a6d7213'")
	assert.Len(t, q.Filters, 2)
	assert.True(t, q.HaveFilter("test"))
	assert.Equal(t, "test = ? AND file_id != 'ec34d3b8-3013-43ee-ad7b-1d5d4a6d7213'", q.Where())
}

func TestEmptySliceFilterWithAnotherFilter(t *testing.T) {
	q := New().AddFilter("id", IN, []string{})
	q.AddFilter("another_id", EQ, uuid.New().String())
	t.Log(q.SQL("test"))
}

func TestQuery_AddORFilters(t *testing.T) {
	t.Run("2 OR conditions", func(t *testing.T) {
		q := New().AddFilter("test", EQ, "ok")
		q.AddORFilters(func(query *Query) {
			query.AddFilter("firstname", ILIKE, "*hello*")
			query.AddFilter("lastname", ILIKE, "*hello*")
		})
		out := q.SQL("table")
		t.Log(out)
		assert.Equal(t, `SELECT * FROM table WHERE test = ? AND (firstname ILIKE ? OR lastname ILIKE ?)`, out)
	})

	t.Run("3 OR conditions", func(t *testing.T) {
		q := New().AddFilter("test", EQ, "ok")
		q.AddORFilters(func(query *Query) {
			query.AddFilter("firstname", ILIKE, "*hello*")
			query.AddFilter("lastname", ILIKE, "*hello*")
			query.AddFilter("email", ILIKE, "*hello*")
		})
		out := q.SQL("table")
		t.Log(out)
		assert.Equal(t, `SELECT * FROM table WHERE test = ? AND (firstname ILIKE ? OR lastname ILIKE ? OR email ILIKE ?)`, out)
	})
}

func ExampleQuery_AddORFilters() {
	q := New().AddFilter("test", EQ, "ok")
	q.AddORFilters(func(query *Query) {
		query.AddFilter("firstname", ILIKE, "*hello*")
		query.AddFilter("lastname", ILIKE, "*hello*")
	})
	q.SQL("table") // SELECT * FROM table WHERE test = ? AND (firstname ILIKE ? OR lastname ILIKE ?)
}

func TestQuery_Clone(t *testing.T) {
	q := New()
	assert.NoError(t, q.SetUrlString("?offset=0&limit=10&fields=id&id=123"))
	q.AddValidation("id", func(value interface{}) error {
		return nil
	})

	QueryEqual(t, q, q.Clone())
}

func QueryEqual(t *testing.T, q, got *Query) {
	if !reflect.DeepEqual(q.query, got.query) {
		t.Errorf("q.query = %v , want = %v", got.query, q.query)
	}
	for k, origFunc := range q.validations {
		gotFunc, ok := got.validations[k]
		if assert.True(t, ok, "got.validations[%s] not present", k) {
			origPtr := reflect.ValueOf(origFunc).Pointer()
			gotPtr := reflect.ValueOf(gotFunc).Pointer()
			assert.Equal(t, origPtr, gotPtr)
		}
	}
	if !reflect.DeepEqual(q.Fields, got.Fields) {
		t.Errorf("q.Fields = %v , want = %v", got.Fields, q.Fields)
	}
	if !reflect.DeepEqual(q.Offset, got.Offset) {
		t.Errorf("q.Offset = %v , want = %v", got.Offset, q.Offset)
	}
	if !reflect.DeepEqual(q.Limit, got.Limit) {
		t.Errorf("q.Limit = %v , want = %v", got.Limit, q.Limit)
	}
	if !reflect.DeepEqual(q.Sorts, got.Sorts) {
		t.Errorf("q.Sorts = %v , want = %v", got.Sorts, q.Sorts)
	}
	if !reflect.DeepEqual(q.Filters, got.Filters) {
		t.Errorf("q.Filters = %v , want = %v", got.Filters, q.Filters)
	}
}
