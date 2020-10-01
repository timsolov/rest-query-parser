package rqp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Where(t *testing.T) {
	t.Run("ErrUnknownMethod", func(t *testing.T) {
		filter := Filter{
			Key:    "id[not]",
			Name:   "id",
			Method: NOT,
			Or:     true,
		}
		_, err := filter.Where()
		assert.Equal(t, err, ErrUnknownMethod)

		filter = Filter{
			Key:    "id[fake]",
			Name:   "id",
			Method: "fake",
			Or:     true,
		}
		_, err = filter.Where()
		assert.Equal(t, err, ErrUnknownMethod)
	})
}

func Test_Args(t *testing.T) {
	t.Run("ErrUnknownMethod", func(t *testing.T) {
		filter := Filter{
			Key:    "id[not]",
			Name:   "id",
			Method: NOT,
			Or:     true,
			Value:  "id",
		}
		_, err := filter.Args()
		assert.Equal(t, err, ErrUnknownMethod)

		filter = Filter{
			Key:    "id[fake]",
			Name:   "id",
			Method: "fake",
			Or:     true,
		}
		_, err = filter.Args()
		assert.Equal(t, err, ErrUnknownMethod)
	})
}
