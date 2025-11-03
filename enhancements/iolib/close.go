package iolib

import (
	"io"

	"github.com/qtraffics/qtfra/ex"
)

func CloseDeep(v any) error {
	if v == nil {
		return nil
	}
	if cc, ok := v.(io.Closer); ok {
		return cc.Close()
	}
	if rr, ok := v.(io.Reader); ok {
		return ReadCloser(rr).Close()
	}
	if ww, ok := v.(io.Writer); ok {
		return WriteCloser(ww).Close()
	}
	return nil
}

func Close(v ...any) error {
	err := &ex.JoinError{}
	for _, cc := range v {
		err.NewError(CloseDeep(cc))
	}
	return err.Err
}

func CloseFast(v ...any) error {
	for _, cc := range v {
		e := CloseDeep(cc)
		if e != nil {
			return e
		}
	}
	return nil
}
