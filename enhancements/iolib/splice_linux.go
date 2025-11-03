//go:build linux

package iolib

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/qtraffics/qtfra/enhancements/iolib/counter"

	"golang.org/x/sys/unix"
)

const (
	// spliceNonblock doesn't make the splice itself necessarily nonblocking
	// (because the actual file descriptors that are spliced from/to may block
	// unless they have the O_NONBLOCK flag set), but it makes the splice pipe
	// operations nonblocking.
	spliceNonblock = 0x2

	// maxSpliceSize is the maximum amount of data Splice asks
	// the kernel to move in a single call to splice(2).
	// We use 1MB as Splice writes data through a pipe, and 1MB is the default maximum pipe buffer size,
	// which is determined by /proc/sys/fs/pipe-max-size.
	maxSpliceSize = 1 << 20
)

type splicePipeFields struct {
	rfd  int
	wfd  int
	data int
}

type splicePipe struct {
	splicePipeFields

	// We want to use a finalizer, so ensure that the size is
	// large enough to not use the tiny allocator.
	_ [24 - unsafe.Sizeof(splicePipeFields{})%24]byte
}

// splicePipePool caches pipes to avoid high-frequency construction and destruction of pipe buffers.
// The garbage collector will free all pipes in the sync.Pool periodically, thus we need to set up
// a finalizer for each pipe to close its file descriptors before the actual GC.
var splicePipePool = sync.Pool{New: newPoolPipe}

var CloseFunc func(int) error = syscall.Close

func newPoolPipe() any {
	// Discard the error which occurred during the creation of pipe buffer,
	// redirecting the data transmission to the conventional way utilizing read() + write() as a fallback.
	p := newPipe()
	if p == nil {
		return nil
	}
	runtime.SetFinalizer(p, destroyPipe)
	return p
}

// getPipe tries to acquire a pipe buffer from the pool or create a new one with newPipe() if it gets nil from the cache.
func getPipe() (*splicePipe, error) {
	v := splicePipePool.Get()
	if v == nil {
		return nil, syscall.EINVAL
	}
	return v.(*splicePipe), nil
}

func putPipe(p *splicePipe) {
	// If there is still data left in the pipe,
	// then close and discard it instead of putting it back into the pool.
	if p.data != 0 {
		runtime.SetFinalizer(p, nil)
		destroyPipe(p)
		return
	}
	splicePipePool.Put(p)
}

// newPipe sets up a pipe for a splice operation.
func newPipe() *splicePipe {
	var fds [2]int
	if err := syscall.Pipe2(fds[:], syscall.O_CLOEXEC|syscall.O_NONBLOCK); err != nil {
		return nil
	}

	// Splice will loop writing maxSpliceSize bytes from the source to the pipe,
	// and then write those bytes from the pipe to the destination.
	// Set the pipe buffer size to maxSpliceSize to optimize that.
	// Ignore errors here, as a smaller buffer size will work,
	// although it will require more system calls.
	_, _ = unix.FcntlInt(uintptr(fds[0]), syscall.F_SETPIPE_SZ, maxSpliceSize)

	return &splicePipe{splicePipeFields: splicePipeFields{rfd: fds[0], wfd: fds[1]}}
}

// destroyPipe destroys a pipe.
func destroyPipe(p *splicePipe) {
	_ = CloseFunc(p.rfd)
	_ = CloseFunc(p.wfd)
}

func splice(source syscall.RawConn, destination syscall.RawConn,
	readCounters []counter.Func, writeCounters []counter.Func,
) (handed bool, n int64, err error) {
	handed = true
	var pipe *splicePipe
	pipe, err = getPipe()
	if err != nil {
		return
	}
	defer putPipe(pipe)
	var readN int
	var readErr error
	var writeSize int
	var writeErr error
	readFunc := func(fd uintptr) (done bool) {
		p0, p1 := unix.Splice(int(fd), nil, pipe.wfd, nil, maxSpliceSize, unix.SPLICE_F_NONBLOCK)
		readN = int(p0)
		readErr = p1
		return readErr != unix.EAGAIN
	}
	writeFunc := func(fd uintptr) (done bool) {
		for writeSize > 0 {
			p0, p1 := unix.Splice(pipe.rfd, nil, int(fd), nil, writeSize, unix.SPLICE_F_NONBLOCK|unix.SPLICE_F_MOVE)
			writeN := int(p0)
			n += p0
			writeErr = p1
			if writeErr != nil {
				return writeErr != unix.EAGAIN
			}
			writeSize -= writeN
		}
		return true
	}
	for {
		err = source.Read(readFunc)
		if err != nil {
			readErr = err
		}
		if readErr != nil {
			if readErr == unix.EINVAL || readErr == unix.ENOSYS {
				handed = false
				return
			}
			err = fmt.Errorf("splice read: %w", readErr)
			return
		}
		if readN == 0 {
			return
		}
		writeSize = readN
		err = destination.Write(writeFunc)
		if err != nil {
			writeErr = err
		}
		if writeErr != nil {
			err = fmt.Errorf("splice write: %w", writeErr)
			return
		}
		for _, readCounter := range readCounters {
			readCounter(int64(readN))
		}
		for _, writeCounter := range writeCounters {
			writeCounter(int64(readN))
		}
	}
}
