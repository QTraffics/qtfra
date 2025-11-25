package values

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNil(t *testing.T) {
	type nilTestStruct struct{}
	type nilTestIf interface{}
	type nilTestFunc func()

	var (
		testsP *nilTestStruct
		tests  nilTestStruct
		testif nilTestIf
		testfn nilTestFunc
	)
	assert.True(t, testsP == nil)

	assert.False(t, IsNil(&tests))
	assert.True(t, IsNil(nil))
	assert.True(t, IsNil(testsP))
	assert.True(t, IsNil(testif))
	assert.True(t, IsNil(testfn))
}
