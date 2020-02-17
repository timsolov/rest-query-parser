package rqp

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_parseFilterValue(t *testing.T) {
	filter1, _ := parseFilterKey("is_admin[eq]")
	err := New().parseFilterValue(filter1, "bool", []string{"false"}, nil)
	assert.NoError(t, err)

	filter2, _ := parseFilterKey("is_admin[eq]")
	err = New().parseFilterValue(filter2, "bool", []string{"q1qwe"}, nil)
	assert.Equal(t, ErrBadFormat, errors.Cause(err))
}
