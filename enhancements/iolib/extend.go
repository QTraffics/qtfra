package iolib

import "io"

type nopReaderCloser struct {
	r io.Reader
}

type nopWriterCloser struct {
	w io.Writer
}

func (n *nopReaderCloser) Read(p []byte) (int, error) {
	return n.r.Read(p)
}

func (n *nopWriterCloser) Write(p []byte) (int, error) {
	return n.w.Write(p)
}

func (n *nopReaderCloser) Close() error {
	return nil
}

func (n *nopWriterCloser) Close() error {
	return nil
}

func (n *nopWriterCloser) UnderlayWriter() io.Writer {
	return n.w
}

func (n *nopReaderCloser) UnderlayReader() io.Reader {
	return n.r
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
	if rr, ok := r.(UnderlayReader); ok {
		return readCloser(original, rr.UnderlayReader())
	}
	return &nopReaderCloser{r: original}
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
	if ww, ok := w.(UnderlayWriter); ok {
		return writeCloser(original, ww.UnderlayWriter())
	}

	return &nopWriterCloser{w: w}
}
