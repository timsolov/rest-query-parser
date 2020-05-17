package rqp

import (
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MyValuer struct{}

func (v MyValuer) Value() (driver.Value, error) {
	return "NULL", nil
}

func Test_in(t *testing.T) {
	t.Run("ALL OK", func(t *testing.T) {
		q, args, err := in("id IN (?)", []string{"1", "2"})
		assert.NoError(t, err)
		assert.Equal(t, "id IN (?, ?)", q)
		assert.Equal(t, []interface{}{"1", "2"}, args)
	})

	t.Run("Valuer", func(t *testing.T) {
		q, args, err := in("id IN (?)", []sql.NullString{{String: "1", Valid: true}, {String: "2"}})
		assert.NoError(t, err)
		assert.Equal(t, "id IN (?, ?)", q)
		assert.Equal(t, []interface{}{sql.NullString{String: "1", Valid: true}, sql.NullString{String: "2", Valid: false}}, args)
	})

	t.Run("MyValuer", func(t *testing.T) {
		q, args, err := in("id IN (?)", MyValuer{})
		assert.NoError(t, err)
		assert.Equal(t, "id IN (?)", q)
		assert.Equal(t, []interface{}{MyValuer{}}, args)
	})

	t.Run("More arguments", func(t *testing.T) {
		_, _, err := in("id IN (?), id2 = ?", []string{"1", "2"})
		assert.EqualError(t, err, "number of bindVars exceeds arguments")
	})

	t.Run("Less arguments", func(t *testing.T) {
		s := "2"
		sPtr := &s
		_, _, err := in("id = ?", []string{"1", "2"}, sPtr)
		assert.EqualError(t, err, "number of bindVars less than number arguments")
	})

	t.Run("No slice", func(t *testing.T) {
		_, _, err := in("id IN (?)", "1")
		assert.NoError(t, err)
	})

	t.Run("Empty slice", func(t *testing.T) {
		_, _, err := in("id IN (?)", []string{})
		assert.Error(t, err, "empty slice passed to 'in' query")
	})

	t.Run("Skip not slice", func(t *testing.T) {
		_, _, err := in("id IN (?), id2 = ?", "1", []interface{}{"2"})
		assert.NoError(t, err)
	})
}
