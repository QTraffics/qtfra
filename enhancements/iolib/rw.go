package iolib

import (
	"io"
	"sync"

	"github.com/qtraffics/qtfra/buf"
	"github.com/qtraffics/qtfra/ex"
	"github.com/qtraffics/qtfra/log"
	"github.com/qtraffics/qtfra/sys/sysvars"
	"github.com/qtraffics/qtfra/threads"
)

type BufWriter struct {
	underlay io.WriteCloser
	buffer   *buf.Buffer
}

func (b *BufWriter) UnderlayWriter() io.Writer {
	err := b.Flush()
	if sysvars.DebugEnabled && err != nil {
		log.Error("failed to flush buffer when accessing underlay writer", log.AttrError(err))
	}
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
	if err != nil {
		return n, err
	}
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

type CacheReader interface {
	io.Reader
	ReadCache() (io.Reader, *buf.Buffer)
}

func PickReaderCache(r io.Reader) (io.Reader, *buf.Buffer) {
	if cr, ok := r.(CacheReader); ok {
		rr, buffer := cr.ReadCache()
		if buffer.Empty() {
			buffer.Free()
			return rr, nil
		}
		return rr, buffer
	}
	return r, nil
}

var (
	_ io.Reader   = (*BufCachedReader)(nil)
	_ CacheReader = (*BufCachedReader)(nil)
)

// BufCachedReader is the default CacheReader implement
// Note: do not implement underlay.Reader here , Because there a some data still store in cache.
type BufCachedReader struct {
	r   io.Reader
	buf *buf.Buffer
}

func (b *BufCachedReader) ReadCache() (io.Reader, *buf.Buffer) {
	return b.r, b.buf
}

func (b *BufCachedReader) Read(p []byte) (n int, err error) {
	var offset int
	if !b.buf.Empty() {
		offset, err = b.buf.Read(p)
		// Buffer.Read should only return an error when buffer is empty.
		if sysvars.DebugEnabled && err != nil {
			panic("Buffer.Read returned an error when buffer is empty")
		}
		if b.buf.Empty() {
			b.buf.Free()
		}
		if offset == len(p) || err != nil {
			return offset, err
		}
	}

	n, err = b.r.Read(p[offset:])
	return n + offset, err
}

func NewCacheReader(r io.Reader, buffer *buf.Buffer) io.Reader {
	if buffer.Empty() {
		return r
	}
	return &BufCachedReader{r: r, buf: buffer}
}
