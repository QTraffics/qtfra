package iolib

import (
	"io"
	"syscall"

	"github.com/qtraffics/qtfra/buf"
	"github.com/qtraffics/qtfra/enhancements/iolib/counter"
	"github.com/qtraffics/qtfra/ex"
	"github.com/qtraffics/qtfra/sys/sysvars"
)

var testSpliceTriggered = false

func Copy(destination io.Writer, source io.Reader) (n int64, err error) {
	var (
		readCounter  []counter.Func
		writeCounter []counter.Func
	)

	source, readCounter = counter.UnwrapReadCounter(source)
	destination, writeCounter = counter.UnwrapWriterCounter(destination)

	return CopyCounters(destination, source, writeCounter, readCounter)
}

func CopyCounters(destination io.Writer, source io.Reader, writeCounters []counter.Func, readCounters []counter.Func) (n int64, err error) {
	var (
		earlyCopied int
		pureCopied  int64
	)
	earlyCopied, source, err = copyEarly(destination, source, writeCounters, readCounters)
	n += int64(earlyCopied)
	if err != nil {
		return n, ex.Cause(err, "copyEarly")
	}

	pureCopied, err = copyPure(destination, source, writeCounters, readCounters)
	n += pureCopied
	if err != nil && !ex.IsMulti(err, io.EOF) {
		return n, ex.Cause(err, "copyPure")
	}

	return n, nil
}

func copyPure(destination io.Writer, source io.Reader, writeCounters []counter.Func, readCounters []counter.Func) (n int64, err error) {
	sourceSysConn, sourceIsSysConn := source.(syscall.Conn)
	destinationSysConn, destinationIsSysConn := destination.(syscall.Conn)
	if sourceIsSysConn && destinationIsSysConn {
		var (
			internalErr error

			sourceRawConn      syscall.RawConn
			destinationRawConn syscall.RawConn
		)

		if sourceRawConn, internalErr = sourceSysConn.SyscallConn(); internalErr != nil {
			goto genericCopy
		}
		if destinationRawConn, internalErr = destinationSysConn.SyscallConn(); internalErr != nil {
			goto genericCopy
		}
		var spliceHanded bool
		spliceHanded, n, err = splice(sourceRawConn, destinationRawConn, readCounters, writeCounters)
		if spliceHanded {
			// for test only
			// See: copy_test.go
			testSpliceTriggered = true
			return n, err
		}
	}

genericCopy:
	return copyGeneric(destination, source, writeCounters, readCounters)
}

func copyEarly(destination io.Writer, source io.Reader, writeCounter []counter.Func, readCounter []counter.Func) (n int, _ io.Reader, err error) {
	if needHandshake, ok := destination.(NeedHandshake); ok && needHandshake.NeedHandshake() {
		bufferHandshaker, isBufferHandshaker := destination.(HandshakeBuffer)
		if !isBufferHandshaker {
			err = destination.(Handshaker).Handshake()
			if err != nil {
				return n, source, ex.Cause(err, "handshake")
			}
		} else {
			handshakeBuffer := buf.New()
			defer handshakeBuffer.Free()

			handshakeReadN, handshakeReadErr := handshakeBuffer.ReadFromOnce(source)

			for _, c := range readCounter {
				c(int64(handshakeReadN))
			}

			if handshakeReadErr != nil {
				return handshakeReadN, source, ex.Cause(handshakeReadErr, "handshake: read")
			}

			handshakeWriteN, handshakeWriteErr := bufferHandshaker.Handshake(handshakeBuffer.Bytes())
			n += handshakeWriteN

			for _, c := range writeCounter {
				c(int64(handshakeWriteN))
			}

			if handshakeWriteErr != nil {
				return n, source, ex.Cause(handshakeWriteErr, "handshake: write")
			}
		}
	}

	var buffers []*buf.Buffer
	source, buffers = PickReaderCacheList(source)
	if len(buffers) > 0 {
		defer func() {
			for len(buffers) > 0 {
				buffers[0].Free()
				buffers[0] = nil
				buffers = buffers[1:]
			}
		}()
	}
	for len(buffers) > 0 {
		to, err := buffers[0].WriteTo(destination)
		n += int(to)
		if err != nil {
			buffers[0].Free()
			return n, source, ex.Cause(err, "writeCache")
		}

		for _, c := range readCounter {
			c(to)
		}

		for _, c := range writeCounter {
			c(to)
		}

		if sysvars.DebugEnabled && !buffers[0].Empty() {
			buffers[0].Free()
			panic("buffer not WriteTo fully")
		}

		// Next
		buffers[0].Free()
		buffers[0] = nil
		buffers = buffers[1:]
	}
	return n, source, nil
}

func copyGeneric(destination io.Writer, source io.Reader, writeCounter []counter.Func, readCounter []counter.Func) (n int64, err error) {
	buffer := buf.NewHuge()
	defer buffer.Free()

	source = counter.NewReader(source, readCounter)
	destination = counter.NewWriter(destination, writeCounter)

	return io.CopyBuffer(destination, source, buffer.FreeBytes())
}
