package iolib

import (
	"io"
	"net"

	"github.com/qtraffics/qtfra/ex"
)

type UnderlayConn interface {
	UnderlayConn() net.Conn
}

type UnderlayReader interface {
	UnderlayReader() io.Reader
}

type UnderlayWriter interface {
	UnderlayWriter() io.Writer
}

func FindUnderlayReader(v any) io.Reader {
	if v == nil {
		return nil
	}

	if vv, ok := v.(UnderlayReader); ok {
		return FindUnderlayReader(vv.UnderlayReader())
	}
	if r, ok := v.(io.Reader); ok {
		return r
	}
	return nil
}

func FindUnderlayWriter(v any) io.Writer {
	if v == nil {
		return nil
	}

	if vv, ok := v.(UnderlayWriter); ok {
		return FindUnderlayWriter(vv.UnderlayWriter())
	}
	if w, ok := v.(io.Writer); ok {
		return w
	}
	return nil
}

func Close(v any) error {
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

func CloseAll(v ...any) error {
	var err error
	for _, cc := range v {
		err = ex.Errors(err, Close(cc))
	}
	return err
}
