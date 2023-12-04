package rqp

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Where(t *testing.T) {
	t.Run("ErrUnknownMethod", func(t *testing.T) {
		filter := Filter{
			Key:               "id[not]",
			ParameterizedName: "id",
			Method:            NOT,
		}
		_, err := filter.Where()
		assert.Equal(t, err, ErrUnknownMethod)

		filter = Filter{
			Key:               "id[fake]",
			ParameterizedName: "id",
			Method:            "fake",
		}
		_, err = filter.Where()
		assert.Equal(t, err, ErrUnknownMethod)
	})
}

func Test_Args(t *testing.T) {
	t.Run("ErrUnknownMethod", func(t *testing.T) {
		filter := Filter{
			Key:               "id[not]",
			ParameterizedName: "id",
			Method:            NOT,
			Value:             "id",
		}
		_, err := filter.Args()
		assert.Equal(t, err, ErrUnknownMethod)

		filter = Filter{
			Key:               "id[fake]",
			ParameterizedName: "id",
			Method:            "fake",
		}
		_, err = filter.Args()
		assert.Equal(t, err, ErrUnknownMethod)
	})
}

func Test_RemoveOrEntries(t *testing.T) {
	type testCase struct {
		name           string
		urlQuery       string
		filterToRemove string
		wantWhere      string
	}
	tests := []testCase{
		{
			name:           "should fix OR statements after removing EndOR filter with 2 items",
			urlQuery:       "?test1[eq]=test10|test2[eq]=test10",
			filterToRemove: "test2",
			wantWhere:      " WHERE test1 = ?",
		},
		{
			name:           "should fix OR statements after removing StartOR filter with 2 items",
			urlQuery:       "?test1[eq]=test10|test2[eq]=test10",
			filterToRemove: "test1",
			wantWhere:      " WHERE test2 = ?",
		},
		{
			name:           "should fix OR statements after removing StartOR filter with 3 items",
			urlQuery:       "?test1[eq]=test10|test2[eq]=test10|test3[eq]=test10",
			filterToRemove: "test1",
			wantWhere:      " WHERE (test2 = ? OR test3 = ?)",
		},
		{
			name:           "should fix OR statements after removing EndOR filter with 3 items",
			urlQuery:       "?test1[eq]=test10|test2[eq]=test10|test3[eq]=test10",
			filterToRemove: "test3",
			wantWhere:      " WHERE (test1 = ? OR test2 = ?)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			URL, _ := url.Parse(tt.urlQuery)
			q := NewQV(nil, Validations{
				"test1": nil,
				"test2": nil,
				"test3": nil,
			}, nil)
			_ = q.SetUrlQuery(URL.Query()).Parse()

			// Act
			_ = q.RemoveQueryFilter(tt.filterToRemove)

			// Assert
			assert.Equal(t, tt.wantWhere, q.WHERE())
		})
	}
}
