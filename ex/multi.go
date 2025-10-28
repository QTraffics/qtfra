package ex

import (
	"errors"
	"strings"

	"github.com/QTraffics/qtfra/enhancements/slicelib"
)

type multiError struct {
	errors []error
}

func (e *multiError) Error() string {
	return strings.Join(slicelib.Map(e.errors, func(it error) string {
		return it.Error()
	}), ";\n")
}

func (e *multiError) Unwrap() []error {
	return e.errors
}

func Errors(errors ...error) error {
	errors = slicelib.FilterNotNil(errors)
	errors = slicelib.FilterNotNil(errors)
	errors = slicelib.UniqBy(errors, error.Error)
	switch len(errors) {
	case 0:
		return nil
	case 1:
		return errors[0]
	}
	return &multiError{
		errors: errors,
	}
}

func IsMulti(err error, targetList ...error) bool {
	for _, target := range targetList {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
