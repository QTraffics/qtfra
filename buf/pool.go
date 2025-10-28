package buf

import (
	"github.com/QTraffics/qtfra/enhancements/pool"
	"github.com/QTraffics/qtfra/ex"
)

const maxBufferSize = 1 << 16

type Pool struct {
	size int
	pool *pool.Pool[*Buffer]
}

// Deprecated: use NewSize is Enough
func NewPool(size int) *Pool {
	if size <= 0 {
		panic("negative buffer pool size")
	}
	var source *pool.Pool[*Buffer]
	if size < maxBufferSize {
		source = pool.New(
			func() *Buffer {
				return NewSize(size)
			})
	}
	return &Pool{
		size: size,
		pool: source,
	}
}

func (p *Pool) Get() *Buffer {
	if !p.poolIsValid() {
		return As(make([]byte, p.size))
	}
	b := p.pool.Get()
	b.Reset()
	return b
}

func (p *Pool) GetSize(size int) *Buffer {
	if p.size <= size && p.poolIsValid() {
		b := p.Get()
		ex.Must(b.Resize(size))
		return b
	}
	return As(make([]byte, size))
}

func (p *Pool) Put(x *Buffer) {
	if !p.poolIsValid() || x.size > p.size || x.ref > 0 {
		return
	}
	p.pool.Put(x)
}

func (p *Pool) poolIsValid() bool {
	return p.size < maxBufferSize && p.pool != nil
}
