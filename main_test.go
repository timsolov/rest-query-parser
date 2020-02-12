package rqp

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFields(t *testing.T) {

	// Fields:
	cases := []struct {
		url      string
		expected string
		err      error
	}{
		{url: "?", expected: "*", err: nil},
		{url: "?fields=", expected: "*", err: nil},
		{url: "?fields=*", expected: "*", err: nil},
		{url: "?fields=id", expected: "id", err: nil},
		{url: "?fields=id,name", expected: "id, name", err: nil},
	}

	for _, c := range cases {
		URL, err := url.Parse(c.url)
		assert.NoError(t, err)
		q, err := NewParse(URL.Query(), nil)
		assert.Equal(t, c.err, err)
		assert.Equal(t, c.expected, q.FieldsSQL())
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
		{url: "?offset=", expected: ""},
		{url: "?offset=10", expected: " OFFSET 10"},
	}
	for _, c := range cases {
		URL, err := url.Parse(c.url)
		assert.NoError(t, err)
		q, err := NewParse(URL.Query(), nil)
		assert.Equal(t, c.err, err)
		assert.Equal(t, c.expected, q.OffsetSQL())
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
		{url: "?limit=", expected: ""},
		{url: "?limit=10", expected: " LIMIT 10"},
	}
	for _, c := range cases {
		URL, err := url.Parse(c.url)
		assert.NoError(t, err)
		q, err := NewParse(URL.Query(), nil)
		assert.Equal(t, c.err, err)
		assert.Equal(t, c.expected, q.LimitSQL())
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
		URL, err := url.Parse(c.url)
		assert.NoError(t, err)
		q, err := NewParse(URL.Query(), nil)
		assert.Equal(t, c.err, err)
		assert.Equal(t, c.expected, q.SortSQL())
	}
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
		{url: "?id", expected: "", err: "id: bad format"},
		{url: "?id=", expected: "", err: "id: bad format"},
		{url: "?id=1,2", expected: "", err: "id: method are not allowed"},
		{url: "?id=4", expected: " WHERE id = ?"},
		{url: "?id=1&name=superman", expected: " WHERE id = ?", ignore: true},
		{url: "?id=1&name=superman&s[like]=super", expected: " WHERE id = ? AND s LIKE ?", expected2: " WHERE s LIKE ? AND id = ?", ignore: true},
		{url: "?s=super", expected: " WHERE s = ?"},
		{url: "?s=puper", expected: "", err: "s: puper: not in scope"},
		{url: "?id[in]=1,2", expected: " WHERE id IN (?, ?)"},
		{url: "?id[eq]=1&id[eq]=4", err: "id[eq]: similar names of keys are not allowed"},
		{url: "?id[gte]=1&id[lte]=4", expected: " WHERE id >= ? AND id <= ?", expected2: " WHERE id <= ? AND id >= ?"},
	}
	for _, c := range cases {
		//t.Log(c)
		URL, err := url.Parse(c.url)
		assert.NoError(t, err)

		q := New(URL.Query(), Validations{
			"id:int": nil,
			"s": In(
				"super",
				"best",
			),
			"custom": func(value interface{}) error {
				return nil
			},
		})
		q.IgnoreUnknownFilters(c.ignore)
		err = q.Parse()

		if len(c.err) > 0 {
			assert.EqualError(t, err, c.err)
		}
		where := q.WhereSQL()
		if len(c.expected2) > 0 {
			//t.Log(where)
			assert.True(t, c.expected == where || c.expected2 == where)
		} else {
			//t.Log(where)
			assert.True(t, c.expected == where)
		}

	}

}

func TestSQL(t *testing.T) {
	URL, err := url.Parse("?fields=id,status&sort=id&offset=10&some=123")
	assert.NoError(t, err)

	q := New(URL.Query(), nil)
	q.IgnoreUnknownFilters(true)
	err = q.Parse()
	assert.NoError(t, err)
	assert.Equal(t, "SELECT id, status FROM test ORDER BY id OFFSET 10", q.SQL("test"))

	q.SetValidations(Validations{
		"some:int": nil,
	})
	err = q.Parse()
	assert.NoError(t, err)

	assert.Equal(t, "SELECT id, status FROM test WHERE some = ? ORDER BY id OFFSET 10", q.SQL("test"))
}

func TestReplaceFiltersNames(t *testing.T) {
	URL, err := url.Parse("?one=123&another=yes")
	assert.NoError(t, err)

	q, err := NewParse(URL.Query(), Validations{
		"one": nil, "another": nil,
	})
	assert.NoError(t, err)

	q.ReplaceFiltersNames(FiltersNamesReplacer{
		"one": "another",
	})

	assert.Len(t, q.Filters, 2)
	assert.True(t, q.HaveFilter("one"))
	assert.True(t, q.HaveFilter("another"))

	q.ReplaceFiltersNames(FiltersNamesReplacer{
		"one":        "two",
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
}

func TestRequiredFilter(t *testing.T) {
	// required but not present
	URL, err := url.Parse("?one=1")
	assert.NoError(t, err)

	qp, err := NewParse(URL.Query(), Validations{"limit:required": nil})
	assert.EqualError(t, err, "limit: required")

	// required and present
	URL, err = url.Parse("?limit=10&one[eq]=1")
	assert.NoError(t, err)

	qp, err = NewParse(URL.Query(), Validations{"limit:required": nil, "one:int": nil})
	assert.NoError(t, err)
	_, present := qp.validations["limit:required"]
	assert.False(t, present)
	_, present = qp.validations["limit"]
	assert.True(t, present)
}
