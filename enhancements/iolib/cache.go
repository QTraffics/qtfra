package iolib

import (
	"io"

	"github.com/qtraffics/qtfra/buf"
	"github.com/qtraffics/qtfra/sys/sysvars"
)

type CacheReader interface {
	io.Reader
	ReadCache() (io.Reader, *buf.Buffer)
}

func PickReaderCache(r io.Reader) (io.Reader, *buf.Buffer) {
	if cr, ok := r.(CacheReader); ok {
		rr, buffer := cr.ReadCache()
		return rr, buffer
	}
	return r, nil
}

func PickReaderCacheList(r io.Reader) (io.Reader, []*buf.Buffer) {
	buffers := make([]*buf.Buffer, 0)
	for {
		var buffer *buf.Buffer
		r, buffer = PickReaderCache(r)
		if buffer == nil {
			break
		} else if buffer.Empty() {
			continue
		}
		buffers = append(buffers, buffer)
	}
	return r, buffers
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
	buffer := b.buf
	if buffer != nil {
		buffer.DecRef()
	}
	b.buf = nil
	return b.r, buffer
}

func (b *BufCachedReader) Read(p []byte) (n int, err error) {
	var offset int

	if b.buf != nil && !b.buf.Empty() {
		offset, err = b.buf.Read(p)
		// Buffer.Read should only return an error when buffer is empty.
		if sysvars.DebugEnabled && err != nil {
			panic("Buffer.Read returned an error when buffer not empty")
		}
		if b.buf.Empty() {
			b.buf.DecRef()
			b.buf.Free()
		}
		if offset == len(p) || err != nil {
			return offset, err
		}
	}
	if b.r == nil {
		return 0, io.EOF
	}

	n, err = b.r.Read(p[offset:])
	return n + offset, err
}

func NewCacheReader(r io.Reader, buffer *buf.Buffer) io.Reader {
	if buffer.Empty() {
		return r
	}
	buffer.IncRef()
	return &BufCachedReader{r: r, buf: buffer}
}

func NewCacheReaderList(r io.Reader, buffers []*buf.Buffer) io.Reader {
	for i := int(len(buffers)) - 1; i >= 0; i-- {
		r = NewCacheReader(r, buffers[i])
	}
	return r
}
