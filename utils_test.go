package rqp

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_stringInSlice(t *testing.T) {
	t.Run("RETURN FALSE", func(t *testing.T) {
		assert.Equal(t, false, stringInSlice("", nil))
	})
}
