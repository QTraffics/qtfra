package ex

import (
	"errors"
	"fmt"
)

func New(vv ...any) error {
	if len(vv) == 0 {
		return nil
	}

	return errors.New(fmt.Sprint(vv...))
}

func Cause(err error, due string) error {
	return fmt.Errorf("%s : %w", due, err)
}

func Zone(zone string, err error) error {
	return Cause(err, zone)
}
