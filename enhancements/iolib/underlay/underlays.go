package underlay

import (
	"io"
)

type Reader interface {
	UnderlayReader() io.Reader
}

type Writer interface {
	UnderlayWriter() io.Writer
}

func FindUnderlayReaderDeep(v any) io.Reader {
	if vv, ok := v.(Reader); ok {
		return FindUnderlayReaderDeep(vv.UnderlayReader())
	}
	if vv, ok := v.(io.Reader); ok {
		return vv
	}

	return nil
}

func FindUnderlayWriterDeep(v any) io.Writer {
	if vv, ok := v.(Writer); ok {
		return FindUnderlayWriterDeep(vv.UnderlayWriter())
	}
	if vv, ok := v.(io.Writer); ok {
		return vv
	}

	return nil
}
