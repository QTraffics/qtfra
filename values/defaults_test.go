package values

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaults(t *testing.T) {
	type testStruct struct {
		v int
	}

	var (
		exceptedString        = "excepted"
		exceptedFunc          = func() { fmt.Print() /* test only */ }
		exceptedStruct        = testStruct{v: 10}
		exceptedStructPointer = &testStruct{v: 11}
		emptyString           string
		emptyFunc             func()
		emptyStruct           testStruct
		emptyStructPointer    *testStruct
	)

	assert.Equal(t, exceptedString, UseDefault(emptyString, exceptedString))
	// Invalid operation: (func())(0x5ad920) == (func())(0x5ad920) (cannot take func type as argument)
	// assert.Equal(t, exceptedFunc,UseDefaultNil(emptyFunc, exceptedFunc))
	defaultNilFunc := UseDefaultNil(emptyFunc, exceptedFunc)
	assert.True(t, IsNil(emptyFunc))
	assert.False(t, IsNil(exceptedFunc))
	assert.False(t, IsNil(defaultNilFunc))

	assert.Equal(t, exceptedStruct.v, UseDefault(emptyStruct, exceptedStruct).v)
	assert.Equal(t, exceptedStructPointer.v, UseDefaultNil(emptyStructPointer, exceptedStructPointer).v)
}
