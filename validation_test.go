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
