package iolib

import (
	"io"

	"github.com/qtraffics/qtfra/enhancements/iolib/underlay"
)

type nopReadCloser struct{ r io.Reader }

func (n *nopReadCloser) Read(p []byte) (int, error) { return n.r.Read(p) }
func (n *nopReadCloser) Close() error               { return nil }
func (n *nopReadCloser) UnderlayReader() io.Reader  { return n.r }

type nopReadWriteToCloser struct{ r io.Reader }

func (n *nopReadWriteToCloser) Read(p []byte) (int, error) { return n.r.Read(p) }
func (n *nopReadWriteToCloser) Close() error               { return nil }
func (n *nopReadWriteToCloser) UnderlayReader() io.Reader  { return n.r }
func (n *nopReadWriteToCloser) WriteTo(w io.Writer) (int64, error) {
	return n.r.(io.WriterTo).WriteTo(w)
}

func NopReadCloser(r io.Reader) io.ReadCloser {
	if _, ok := r.(io.WriterTo); ok {
		return &nopReadWriteToCloser{r: r}
	}

	return &nopReadCloser{r: r}
}

func ReadCloser(r io.Reader) io.ReadCloser {
	if rc, ok := r.(io.ReadCloser); ok {
		return rc
	}

	return readCloser(r, r)
}

func readCloser(original io.Reader, r io.Reader) io.ReadCloser {
	if c, ok := r.(io.Closer); ok {
		return struct {
			io.Reader
			io.Closer
		}{original, c}
	}
	if rr, ok := r.(underlay.Reader); ok {
		return readCloser(original, rr.UnderlayReader())
	}

	return NopReadCloser(original)
}

type nopWriteCloser struct{ w io.Writer }

func (n *nopWriteCloser) Write(p []byte) (int, error) { return n.w.Write(p) }
func (n *nopWriteCloser) Close() error                { return nil }
func (n *nopWriteCloser) UnderlayWriter() io.Writer   { return n.w }

type nopWriteReadFromCloser struct{ w io.Writer }

func (n *nopWriteReadFromCloser) Write(p []byte) (int, error) { return n.w.Write(p) }
func (n *nopWriteReadFromCloser) Close() error                { return nil }
func (n *nopWriteReadFromCloser) UnderlayWriter() io.Writer   { return n.w }
func (n *nopWriteReadFromCloser) ReadFrom(r io.Reader) (int64, error) {
	return n.w.(io.ReaderFrom).ReadFrom(r)
}

func NopWriteCloser(w io.Writer) io.WriteCloser {
	if _, ok := w.(io.ReaderFrom); ok {
		return &nopWriteReadFromCloser{w: w}
	}

	return &nopWriteCloser{w: w}
}

func WriteCloser(w io.Writer) io.WriteCloser {
	if wc, ok := w.(io.WriteCloser); ok {
		return wc
	}

	return writeCloser(w, w)
}

func writeCloser(original io.Writer, w io.Writer) io.WriteCloser {
	if c, ok := w.(io.Closer); ok {
		return struct {
			io.Writer
			io.Closer
		}{original, c}
	}
	if ww, ok := w.(underlay.Writer); ok {
		return writeCloser(original, ww.UnderlayWriter())
	}

	return NopWriteCloser(original)
}

func Close(v any) error {
	switch cc := v.(type) {
	case io.Closer:
		return cc.Close()
	case underlay.Writer:
		return Close(cc.UnderlayWriter())
	case underlay.Reader:
		return Close(cc.UnderlayReader())
	default:
		return nil
	}
}
