package rqp

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestIn(t *testing.T) {
	err := In("one", "two")("three")
	assert.Equal(t, errors.Cause(err), ErrNotInScope)
	assert.EqualError(t, err, "three: not in scope")

	err = In(1, 2)(3)
	assert.Equal(t, errors.Cause(err), ErrNotInScope)
	assert.EqualError(t, err, "3: not in scope")

	err = In(true)(false)
	assert.Equal(t, errors.Cause(err), ErrNotInScope)
	assert.EqualError(t, err, "false: not in scope")
}

func TestMinMax(t *testing.T) {
	err := Max(100)(101)
	assert.Equal(t, errors.Cause(err), ErrNotInScope)
	assert.EqualError(t, err, "101: not in scope")

	err = Max(100)(100)
	assert.NoError(t, err)

	err = Min(100)(100)
	assert.NoError(t, err)

	err = MinMax(10, 100)(9)
	assert.Equal(t, errors.Cause(err), ErrNotInScope)
	assert.EqualError(t, err, "9: not in scope")

	err = MinMax(10, 100)(101)
	assert.Equal(t, errors.Cause(err), ErrNotInScope)
	assert.EqualError(t, err, "101: not in scope")

	err = MinMax(10, 100)(50)
	assert.NoError(t, err)

	err = Multi(Min(10), Max(100))(50)
	assert.NoError(t, err)

	err = Multi(Min(10), Max(100))(101)
	assert.Equal(t, errors.Cause(err), ErrNotInScope)

	err = MinMax(10, 100)("one")
	assert.Equal(t, errors.Cause(err), ErrNotInScope)
	assert.EqualError(t, err, "one: not in scope")

}

func TestMinMaxFloat(t *testing.T) {
	err := MaxFloat(100)(101)
	assert.Equal(t, errors.Cause(err), ErrNotInScope)
	assert.EqualError(t, err, "101: not in scope")

	err = MaxFloat(100)(100)
	assert.NoError(t, err)

	err = MinFloat(100)(100)
	assert.NoError(t, err)

	err = MinMaxFloat(10, 100)(9)
	assert.Equal(t, errors.Cause(err), ErrNotInScope)
	assert.EqualError(t, err, "9: not in scope")

	err = MinMaxFloat(10, 100)(101)
	assert.Equal(t, errors.Cause(err), ErrNotInScope)
	assert.EqualError(t, err, "101: not in scope")

	err = MinMaxFloat(10, 100)(50)
	assert.NoError(t, err)

	err = Multi(MinFloat(10), MaxFloat(100))(50)
	assert.NoError(t, err)

	err = Multi(MinFloat(10), MaxFloat(100))(101)
	assert.Equal(t, errors.Cause(err), ErrNotInScope)

	err = MinMaxFloat(10, 100)("one")
	assert.Equal(t, errors.Cause(err), ErrNotInScope)
	assert.EqualError(t, err, "one: not in scope")
}

func TestNotEmpty(t *testing.T) {
	// good case
	err := NotEmpty()("test")
	assert.NoError(t, err)
	// bad case
	err = NotEmpty()("")
	assert.Equal(t, errors.Cause(err), ErrNotInScope)
}
