package buf

import (
	"errors"
	"fmt"
	"io"
)

var (
	ErrNegativeCount = fmt.Errorf("buffer: negative count")
	ErrOverflow      = fmt.Errorf("buffer: overflow")
)

const (
	MaxManagedSize = int(64*1024) - 1
)

// Buffer
// Note: buffer is thread-unsafe
type Buffer struct {
	data    []byte
	size    int
	managed bool
	r       int
	w       int

	ref int
}

func New() *Buffer {
	return NewSize(4096)
}

func NewSize(size int) *Buffer {
	if size < 0 {
		panic("negative buffer size")
	}

	if size == 0 {
		return &Buffer{}
	} else if size > MaxManagedSize {
		return &Buffer{
			data: make([]byte, size),
			size: size,
		}
	}
	return &Buffer{
		data:    get(size),
		size:    size,
		managed: true,
	}
}

func As(bs []byte) *Buffer {
	return &Buffer{
		data: bs,
		size: len(bs),
		r:    0,
		w:    len(bs),
	}
}

func (b *Buffer) Resize(n int) error {
	if n > len(b.data) {
		return ErrOverflow
	}
	b.size = n
	return nil
}

func (b *Buffer) Reset() {
	b.r = 0
	b.w = 0
}

func (b *Buffer) IncRef() {
	b.ref++
}

func (b *Buffer) DecRef() {
	b.ref--
}

func (b *Buffer) Free() {
	if b == nil || b.data == nil || b.ref > 0 {
		return
	}

	if b.managed {
		put(b.data)
	}
	b.data = nil
}

func (b *Buffer) Bytes() []byte {
	return b.data[b.r:b.w]
}

func (b *Buffer) ReadFrom(r io.Reader) (n int64, err error) {
	if b.Full() {
		return 0, io.ErrShortBuffer
	}
	var (
		nn    int
		retry int
	)
	for {
		nn, err = r.Read(b.data[b.w:b.size])
		b.w += nn
		n += int64(nn)
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			}
			break
		}
		if b.Full() {
			err = io.ErrShortBuffer
			break
		}
		if nn == 0 {
			retry++
			if retry > 100 {
				err = io.ErrNoProgress
				break
			}
			continue
		}
		retry = 0
	}
	return
}

func (b *Buffer) ReadFromOnce(r io.Reader) (n int, err error) {
	if b.Full() {
		return 0, io.ErrShortWrite
	}
	nn, err := r.Read(b.FreeBytes())
	b.w += nn
	return nn, err
}

func (b *Buffer) WriteToOnce(w io.Writer) (n int, err error) {
	if b.Empty() {
		return 0, io.EOF
	}
	nn, err := w.Write(b.Bytes())
	b.r += nn
	return nn, err
}

func (b *Buffer) WriteTo(w io.Writer) (n int64, err error) {
	if b.Empty() {
		return 0, io.EOF
	}
	var (
		nn    int
		retry int
	)
	for {
		nn, err = w.Write(b.Bytes())
		b.r += nn
		n += int64(nn)
		if err != nil || b.Empty() {
			break
		}
		if nn == 0 {
			retry++
			if retry > 100 {
				err = io.ErrNoProgress
				break
			}
			continue
		}
		retry = 0
	}
	return
}

func (b *Buffer) ReadFull(r io.Reader, length int) (n int, err error) {
	end := b.w + length
	if end > b.size {
		return 0, io.ErrShortBuffer
	}
	n, err = io.ReadFull(r, b.data[b.w:end])
	b.w += n
	return n, err
}

func (b *Buffer) Read(bs []byte) (n int, err error) {
	if b.Empty() {
		return 0, io.EOF
	}
	n = copy(bs, b.data[b.r:b.w])
	b.r += n
	return n, nil
}

func (b *Buffer) ReadByte() (byte, error) {
	if b.Empty() {
		return 0, io.EOF
	}
	b.r++
	return b.data[b.r-1], nil
}

func (b *Buffer) WriteString(s string) (n int, err error) {
	if len(s) == 0 {
		return 0, nil
	}
	end := b.w + len(s)
	if end > b.size {
		return 0, io.ErrShortBuffer
	}
	n = copy(b.data[b.w:end], s)
	b.w += n
	return n, nil
}

func (b *Buffer) WriteByte(by byte) error {
	if b.Full() {
		return io.ErrShortBuffer
	}
	b.data[b.w] = by
	b.w++
	return nil
}

func (b *Buffer) Discard(n int) (nn int, err error) {
	if n < 0 {
		return 0, ErrNegativeCount
	}
	if b.Empty() {
		return 0, io.EOF
	}
	end := b.r + n
	if end > b.w {
		end = b.w
	}
	discarded := end - b.r
	b.r = end
	return discarded, nil
}

func (b *Buffer) Peek(n int) ([]byte, error) {
	if n < 0 {
		return nil, ErrNegativeCount
	}

	if b.Empty() {
		return nil, io.EOF
	}
	if b.r+n > b.w {
		return nil, ErrOverflow
	}
	return b.data[b.r : b.r+n], nil
}

func (b *Buffer) Write(bs []byte) (n int, err error) {
	if b.Full() {
		return 0, io.ErrShortBuffer
	}
	if len(bs) == 0 {
		return 0, nil
	}
	n = copy(b.FreeBytes(), bs[:])
	b.w += n
	return n, nil
}

func (b *Buffer) FreeBytes() []byte {
	return b.data[b.w:b.size]
}

func (b *Buffer) Truncated(n int) {
	if n < 0 {
		panic("negative count")
	}
	b.w = b.r + n
}

func (b *Buffer) Len() int {
	return b.w - b.r
}

func (b *Buffer) Full() bool {
	return b.w == b.size
}

func (b *Buffer) Empty() bool {
	return b.r == b.w
}

func (b *Buffer) CopyBytes() []byte {
	return append([]byte{}, b.Bytes()...)
}

func (b *Buffer) Size() int {
	return b.size
}

func (b *Buffer) Cap() int {
	return len(b.data)
}
