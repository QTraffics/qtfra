package iolib

import (
	"io"
	"sync"

	"github.com/QTraffics/qtfra/buf"
	"github.com/QTraffics/qtfra/ex"
	"github.com/QTraffics/qtfra/threads"
)

type BufWriter struct {
	underlay io.WriteCloser
	buffer   *buf.Buffer
}

func (b *BufWriter) UnderlayWriter() io.Writer {
	return b.underlay
}

func (b *BufWriter) Write(p []byte) (n int, err error) {
	if len(p) > b.buffer.Cap() || len(p) > len(b.buffer.FreeBytes()) {
		err = b.Flush()
		if err != nil {
			return 0, ex.Cause(err, "flush")
		}
		n, err = b.underlay.Write(p)
		return
	}
	for n < len(p) {
		nn, e := b.buffer.Write(p[n:])
		n += nn
		if n == nn {
			return n, e
		}

		err = b.Flush()
		if err != nil {
			return
		}
	}
	return
}

func (b *BufWriter) WriteString(s string) (n int, err error) {
	n, err = b.buffer.WriteString(s)
	if err == nil || !ex.IsMulti(err, io.ErrShortBuffer) {
		return
	}
	err = b.Flush()
	if len(s) >= b.buffer.Cap() {
		return WriteString(b.underlay, s)
	}
	return b.buffer.WriteString(s)
}

func (b *BufWriter) WriteByte(bb byte) error {
	if len(b.buffer.FreeBytes()) == 0 {
		err := b.Flush()
		if err != nil {
			return ex.Cause(err, "flush")
		}
	}

	return b.buffer.WriteByte(bb)
}

func (b *BufWriter) ReadFrom(r io.Reader) (n int64, err error) {
	err = b.Flush()
	if err != nil {
		return 0, ex.Cause(err, "flush")
	}

	if rf, ok := b.underlay.(io.ReaderFrom); ok {
		return rf.ReadFrom(r)
	}

	for {
		nn, readErr := b.buffer.ReadFromOnce(r)
		n += int64(nn)

		flushErr := b.Flush()
		if flushErr != nil {
			return n, ex.Cause(flushErr, "flush")
		}

		if readErr != nil {
			if readErr == io.EOF {
				return n, nil
			}
			return n, readErr
		}
	}
}

func (b *BufWriter) Close() error {
	b.buffer.Free()
	return b.underlay.Close()
}

func (b *BufWriter) Free() {
	b.buffer.Free()
}

func (b *BufWriter) Flush() error {
	if b.buffer.Len() == 0 {
		return nil
	}
	_, err := b.buffer.WriteTo(b.underlay)
	if err != nil {
		return ex.Cause(err, "flush")
	}
	b.buffer.Reset()
	return nil
}

func NewBufWriter(w io.Writer, buffer *buf.Buffer) *BufWriter {
	if buffer == nil {
		buffer = buf.New()
	}
	return &BufWriter{WriteCloser(w), buffer}
}

func WriteString(w io.Writer, s string) (n int, err error) {
	if ws, ok := w.(io.StringWriter); ok {
		return ws.WriteString(s)
	}
	return w.Write([]byte(s))
}

type SafeWriter struct {
	sync.Mutex
	io.Writer
}

func NewSafeWriter(w io.Writer) io.Writer {
	if sf, ok := w.(threads.Safe); ok && sf.ThreadSafe() {
		return w
	}
	return &SafeWriter{Writer: w}
}

func (w *SafeWriter) Write(p []byte) (int, error) {
	w.Lock()
	defer w.Unlock()
	return w.Writer.Write(p)
}

func (w *SafeWriter) UnderlayWriter() io.Writer {
	return w.Writer
}

func (w *SafeWriter) ThreadSafe() bool {
	return true
}
